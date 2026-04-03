/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_get_streamload
 *@date    2025/9/1 14:32
 */

package app

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"net/http"
	"sort"
	"strconv"
	"time"
)

func (engine *threadMap) stream2load(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}

	appid = setdefault(appid)
	db, err := engine.getmapConnect(appid)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return
	}

	var itemData []util.Stream2Data
	if val, ok := second30cache.Get(appid + "stream2load"); ok {
		itemData = val.([]util.Stream2Data)
	} else {
		itemData = mergedata(stream2txn(appid, db), stream2schema(appid, db))
		if _, ok := second30cache.Get(appid + "stream2load"); !ok {
			second30cache.Set(appid+"stream2load", itemData, cache.DefaultExpiration)
		}
	}
	c.JSON(http.StatusOK, gin.H{"data": itemData, "total": len(itemData), "threshold": getfc2key(appid, "max_running_txn_num_per_db", db)})
}

// get frontends config 2 key
func getfc2key(app, key string, db *gorm.DB) interface{} {
	if val, ok := lastcache.Get(app + key); ok {
		return val.(interface{})
	}
	var m map[string]interface{}
	r := db.Raw(fmt.Sprintf("ADMIN SHOW FRONTEND CONFIG LIKE '%s'", key)).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return ""
	}
	if m["Value"] == nil {
		r := db.Raw(fmt.Sprintf("SHOW VARIABLES LIKE '%s'", key)).Scan(&m)
		if r.Error != nil {
			util.Loggrs.Error(r.Error.Error())
			return ""
		}
	}
	lastcache.Set(app+key, m["Value"], cache.DefaultExpiration)
	return m["Value"]
}

func mergedata(txnData, schemaData []util.Stream2Data) []util.Stream2Data {
	// 使用 map 以 TxnId 为主键去重
	uniqueMap := make(map[string]util.Stream2Data)
	// 合并两个切片，保留每个 TxnId 最后一次出现的记录
	for _, data := range append(txnData, schemaData...) {
		uniqueMap[data.TxnId] = data
	}
	// 将 map 转换回切片
	result := make([]util.Stream2Data, 0, len(uniqueMap))
	for _, value := range uniqueMap {
		result = append(result, value)
	}
	// 降序
	sortedData := make([]util.Stream2Data, len(result))
	copy(sortedData, result)
	sort.Slice(sortedData, func(i, j int) bool {
		// 解析时间字符串，格式必须与 "2006-01-02 15:04:05" 严格对应
		t1, err1 := time.Parse("2006-01-02 15:04:05", sortedData[i].CreateTimeMs)
		t2, err2 := time.Parse("2006-01-02 15:04:05", sortedData[j].CreateTimeMs)
		// 如果解析出错，把错误的时间放到后面（视为很早的时间）
		if err1 != nil {
			t1 = time.Time{} // 0001-01-01 00:00:00
		}
		if err2 != nil {
			t2 = time.Time{}
		}
		// 降序：时间晚的排在前面
		return t1.After(t2)
	})

	return sortedData
}

// 从交易事件中获取总数
func stream2txn(app string, db *gorm.DB) []util.Stream2Data {
	var m []map[string]interface{}
	r := db.Raw("show proc '/transactions'").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil
	}
	var srcData []util.Stream2Data
	for _, m2 := range m {
		dbname := m2["DbName"].(string)
		var data []map[string]interface{}
		r := db.Raw(fmt.Sprintf("show proc '/transactions/%s/running'", dbname)).Scan(&data)
		if r.Error != nil {
			util.Loggrs.Error(r.Error.Error())
			continue
		}
		for _, item := range data {
			TimeoutMs, _ := strconv.Atoi(item["TimeoutMs"].(string))
			// 解析字符串为 time.Time 类型
			PrepareTime, _ := time.ParseInLocation("2006-01-02 15:04:05", item["PrepareTime"].(string), time.Local)
			d := util.Stream2Data{
				Label:            item["Label"].(string),
				LoadId:           item["Label"].(string),
				TxnId:            item["TransactionId"].(string),
				DbName:           dbname,
				TableName:        "source: " + item["Label"].(string),
				State:            item["TransactionStatus"].(string),
				TimeoutSecond:    fmt.Sprintf("%d(%s)", TimeoutMs/1000, tools.GetHour(int(time.Now().Sub(PrepareTime).Seconds()))),
				CreateTimeMs:     item["PrepareTime"].(string),
				BeforeLoadTimeMs: time.Now().Format("2006-01-02 15:04:05"),
			}
			srcData = append(srcData, d)
		}
	}
	return srcData
}

func stream2schema(app string, db *gorm.DB) []util.Stream2Data {
	var m []map[string]interface{}
	r := db.Raw("select * from information_schema.stream_loads where state in('PENDING','BEGIN','QUEUEING','BEFORE_LOAD','LOADING','PREPARING','PREPARED','COMMITED') order by CREATE_TIME_MS desc").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil
	}

	var srcData []util.Stream2Data
	for _, item := range m {
		var CREATE_TIME_MS, BEFORE_LOAD_TIME_MS string
		var CREATE_TIME time.Time
		if item["CREATE_TIME_MS"] != nil {
			CREATE_TIME_MS = item["CREATE_TIME_MS"].(time.Time).Format("2006-01-02 15:04:05")
			CREATE_TIME = item["CREATE_TIME_MS"].(time.Time)
		}
		if item["BEFORE_LOAD_TIME_MS"] != nil {
			BEFORE_LOAD_TIME_MS = item["BEFORE_LOAD_TIME_MS"].(time.Time).Format("2006-01-02 15:04:05")
		}
		d := util.Stream2Data{
			Label:            item["LABEL"].(string),
			LoadId:           item["LOAD_ID"].(string),
			TxnId:            fmt.Sprintf("%d", item["TXN_ID"].(int64)),
			DbName:           item["DB_NAME"].(string),
			TableName:        item["TABLE_NAME"].(string),
			State:            item["STATE"].(string),
			TimeoutSecond:    fmt.Sprintf("%d(%s)", item["TIMEOUT_SECOND"].(int64), tools.GetHour(int(time.Now().Sub(CREATE_TIME).Seconds()))),
			CreateTimeMs:     CREATE_TIME_MS,
			BeforeLoadTimeMs: BEFORE_LOAD_TIME_MS,
		}
		srcData = append(srcData, d)
	}
	return srcData
}
