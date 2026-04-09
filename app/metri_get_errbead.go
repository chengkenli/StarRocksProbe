/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_get_errbead
 *@date    2025/6/20 13:34
 */

package app

import (
	"StarRocksProbe/conn"
	"StarRocksProbe/util"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"net/http"
	"strings"
	"sync"
)

func (engine *threadMap) metriErrbead(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	type data struct {
		Errquery  int64 `json:"errquery"`
		Errbroker int64 `json:"errbroker"`
		Errsubmit int64 `json:"errsubmit"`
		Errstream int64 `json:"errstream"`
	}

	if val, ok := escache.Get(appid + "errors"); ok {
		marshal, _ := json.Marshal(val.(data))
		util.Loggrs.Info(string(marshal))
		c.JSON(http.StatusOK, val.(data))
		return
	}

	beginTime := c.GetHeader("Begintime")
	endTime := c.GetHeader("Endtime")

	db, err := engine.getmapConnect(appid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	d := data{
		Errquery:  queryerrs(appid, db, beginTime, endTime),
		Errbroker: brokererrs(appid, db, beginTime, endTime),
		Errsubmit: submiterrs(appid, db, beginTime, endTime),
		Errstream: errstreamload(appid, db, beginTime, endTime),
	}
	go escache.Set(appid+"errors", d, cache.DefaultExpiration)
	c.JSON(http.StatusOK, d)
}

// 统计stream load
func errstreamload(app string, db *gorm.DB, begin, end string) int64 {
	var m map[string]interface{}
	stmt := fmt.Sprintf("select count(*) as count from information_schema.stream_loads where CREATE_TIME_MS >= '%s' and CREATE_TIME_MS < '%s' and state ='CANCELLED'", begin, end)
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(app, " ", r.Error.Error())
		return -1
	}
	return m["count"].(int64)
}

// 统计query erros
func queryerrs(app string, db *gorm.DB, begin, end string) int64 {
	// 优先从mysql中取
	avg := init_meta(app)
	m := metainfoSum(avg, begin, end)
	if m >= 1 {
		return m
	}
	if util.Read.Schema.Auditops == "" {
		return -1
	}
	var queryes map[string]interface{}
	stmt := fmt.Sprintf("select count(*) as count from %s where timestamp>='%s' and timestamp<'%s' and user != 'root' and state='ERR'", util.Read.Schema.Auditops, begin, end)
	r := db.Raw(stmt).Scan(&queryes)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return -1
	}
	return queryes["count"].(int64)
}

// 从mysql数据库中获取信息
func metainfoSum(item util.ConnectParms, begin, end string) int64 {
	if item.MetaUser == "" || item.MetaPass == "" || item.MetaHost == "" {
		return -1
	}
	db, err := conn.ConnectItemMySQL(item)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return -1
	}
	var database string
	switch item.App {
	case "sr-app":
		database = "starrocksemr"
	case "sr-scct":
		database = "manager_console_scct"
	default:
		database = "manager_console"
	}
	var m map[string]interface{}
	var stmt string
	if util.Read.Schema.FilterErrmsg == "" {
		stmt = fmt.Sprintf("select count(*) as count from %s.query_record where start_time>='%s' and start_time<'%s' and state = 'FAILED'", database, begin, end)
	} else {
		whor := fmt.Sprintf("and NOT (lower(error_message) REGEXP '%s')", util.Read.Schema.FilterErrmsg)
		stmt = fmt.Sprintf("select count(*) as count from %s.query_record where start_time>='%s' and start_time<'%s' and state = 'FAILED' %v ", database, begin, end, whor)
	}
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return -1
	}
	return m["count"].(int64)
}

// 统计broker errors
func brokererrs(app string, db *gorm.DB, begin, end string) int64 {
	var brokeres []map[string]interface{}
	if val, ok := apicache.Get(app + "dbs"); ok {
		brokeres = val.([]map[string]interface{})
	} else {
		r := db.Raw("show proc '/dbs'").Scan(&brokeres)
		if r.Error != nil {
			util.Loggrs.Error(r.Error.Error())
			return -1
		}
	}

	var err_broker []map[string]interface{}
	done := make(chan struct{}, 5)
	var wg sync.WaitGroup
	for _, m2 := range brokeres {
		wg.Add(1)
		go func(m2 map[string]interface{}) {
			defer func() {
				<-done
				wg.Done()
			}()

			done <- struct{}{}
			database := strings.ReplaceAll(m2["DbName"].(string), "default_cluster:", "")
			var tbLoads []map[string]interface{}
			r := db.Raw(fmt.Sprintf("show load from %s where STATE = 'CANCELLED'", database)).Scan(&tbLoads)
			if r.Error != nil {
				util.Loggrs.Error(r.Error.Error())
				return
			}
			for _, load := range tbLoads {

				ok, err := isInrange(begin, end, load["CreateTime"].(string))
				if err != nil {
					util.Loggrs.Error(err.Error())
					continue
				}
				if !ok {
					continue
				}

				err_broker = append(err_broker, load)
			}
		}(m2)
	}
	wg.Wait()

	go apicache.Set(app+"dbs", brokeres, cache.DefaultExpiration)
	return int64(len(err_broker))
}

// 统计submit task errors
func submiterrs(app string, db *gorm.DB, begin, end string) int64 {
	var submites []map[string]interface{}
	stmt := fmt.Sprintf("select * from information_schema.task_runs where state='FAILED' and CREATE_TIME >='%s' and CREATE_TIME<'%s'", begin, end)
	//stmt := fmt.Sprintf("select * from information_schema.task_runs where state='FAILED' and CREATE_TIME >='%s' and CREATE_TIME<'%s'", begin, end)
	r := db.Raw(stmt).Scan(&submites)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return -1
	}
	return int64(len(submites))
}
