/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_err_optimze
 *@date    2025/9/17 10:01
 */

package app

import (
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"time"
)

func (engine *threadMap) showErrStream2load(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}

	beginTime := c.GetHeader("Begintime")
	endTime := c.GetHeader("Endtime")

	appid = setdefault(appid)
	db, err := engine.getmapConnect(appid)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return
	}
	c.JSON(http.StatusOK, showstream2load(appid, db, beginTime, endTime))
}

func showstream2load(app string, db *gorm.DB, beginTime, endTime string) []showErr {
	var m []map[string]interface{}
	stmt := fmt.Sprintf("select * from information_schema.stream_loads where CREATE_TIME_MS >= '%s' and CREATE_TIME_MS < '%s' and state ='CANCELLED' order by CREATE_TIME_MS desc", beginTime, endTime)
	util.Loggrs.Debug(stmt)
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil
	}

	if len(m) == 0 {
		return []showErr{}
	}

	var srcData []showErr
	for _, item := range m {

		d := showErr{
			Starttime: item["CREATE_TIME_MS"].(time.Time).Format("2006-01-02 15:04:05"),
			User:      fmt.Sprintf("%d（sink.properties.timeout = %d）", item["TXN_ID"].(int64), item["TIMEOUT_SECOND"].(int64)),
			Queryid:   fmt.Sprintf("%s.%s", item["DB_NAME"].(string), item["TABLE_NAME"].(string)),
			Errmsg:    fmt.Sprintf("%v", item["ERROR_MSG"]),
			Errinfo:   fmt.Sprintf("%v", item["TRACKING_URL"]),
			Stmt:      fmt.Sprintf("%v", item["TRACKING_SQL"]),
			Category:  0,
		}

		srcData = append(srcData, d)
	}
	return srcData
}
