/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_get_optimize
 *@date    2025/9/8 14:36
 */

package app

import (
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"net/http"
)

type optimizeData struct {
	data                    []util.TaskOptimize
	running, pending, total int
}

func (engine *threadMap) metriOptimize(c *gin.Context) {
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
	opts := optimize(db, appid)
	c.JSON(http.StatusOK, gin.H{"data": opts.data, "running": opts.running, "pending": opts.pending, "total": len(opts.data)})
}

func optimize(db *gorm.DB, app string) optimizeData {
	var m []map[string]interface{}
	r := db.Raw("show proc '/dbs'").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return optimizeData{}
	}
	var running, pending []string
	var result []util.TaskOptimize
	var errResult []util.TaskOptimizeErr
	for _, m2 := range m {
		database := m2["DbName"].(string)
		/*
			SHOW ALTER TABLE COLUMN FROM MBRSHIP_SECURE;
		*/
		var column []map[string]interface{}
		r := db.Raw(fmt.Sprintf("SHOW ALTER TABLE COLUMN FROM %s", database)).Scan(&column)
		if r.Error != nil {
			util.Loggrs.Error(r.Error.Error())
		}
		for _, c1 := range column {
			switch c1["State"].(string) {
			case "RUNNING":
				running = append(running, c1["JobId"].(string))
			case "PENDING":
				pending = append(pending, c1["JobId"].(string))
			case "FINISHED":
				continue
			case "CANCELLED":
				errResult = append(errResult,
					util.TaskOptimizeErr{
						JobId:         c1["JobId"],
						TableName:     fmt.Sprintf("%s.%v", database, c1["TableName"]),
						CreateTime:    c1["CreateTime"],
						Operation:     c1["Operation"],
						TransactionId: c1["TransactionId"],
						State:         c1["State"],
						Msg:           c1["Msg"],
						Progress:      c1["Progress"],
						Timeout:       c1["Timeout"],
						Tag:           "COLUMN",
					})
				continue
			}
			result = append(result,
				util.TaskOptimize{
					JobId:         c1["JobId"],
					TableName:     fmt.Sprintf("%s.%v", database, c1["TableName"]),
					CreateTime:    c1["CreateTime"],
					Operation:     c1["Operation"],
					TransactionId: c1["TransactionId"],
					State:         c1["State"],
					Msg:           c1["Msg"],
					Progress:      c1["Progress"],
					Timeout:       c1["Timeout"],
					Tag:           "COLUMN",
					Command:       fmt.Sprintf(`<button id="cancel-optimize" class="btn btn-light btn-sm ms-2" data-optimize="%s">❌</button>`, fmt.Sprintf("COLUMN:%s.%v", database, c1["TableName"])),
				})
		}

		/*
			SHOW ALTER TABLE ROLLUP FROM MBRSHIP_SECURE;
		*/
		var rollup []map[string]interface{}
		r = db.Raw(fmt.Sprintf("SHOW ALTER TABLE ROLLUP FROM %s", database)).Scan(&rollup)
		if r.Error != nil {
			util.Loggrs.Error(r.Error.Error())
		}
		for _, c1 := range rollup {
			switch c1["State"].(string) {
			case "RUNNING":
				running = append(running, c1["JobId"].(string))
			case "PENDING":
				pending = append(pending, c1["JobId"].(string))
			case "FINISHED":
				continue
			case "CANCELLED":
				errResult = append(errResult,
					util.TaskOptimizeErr{
						JobId:         c1["JobId"],
						TableName:     fmt.Sprintf("%s.%v", database, c1["TableName"]),
						CreateTime:    c1["CreateTime"],
						Operation:     c1["Operation"],
						TransactionId: c1["TransactionId"],
						State:         c1["State"],
						Msg:           c1["Msg"],
						Progress:      c1["Progress"],
						Timeout:       c1["Timeout"],
						Tag:           "ROLLUP",
					})
				continue
			}
			result = append(result,
				util.TaskOptimize{
					JobId:         c1["JobId"],
					TableName:     fmt.Sprintf("%s.%v", database, c1["TableName"]),
					CreateTime:    c1["CreateTime"],
					Operation:     c1["IndexName"],
					TransactionId: c1["TransactionId"],
					State:         c1["State"],
					Msg:           c1["Msg"],
					Progress:      c1["Progress"],
					Timeout:       c1["Timeout"],
					Tag:           "ROLLUP",
					Command:       fmt.Sprintf(`<button id="cancel-optimize" class="btn btn-light btn-sm ms-2" data-optimize="%s">❌</button>`, fmt.Sprintf("ROLLUP:%s.%v", database, c1["TableName"])),
				})
		}
		/*
			SHOW ALTER TABLE OPTIMIZE FROM <DATABASE>;
		*/
		var optimize []map[string]interface{}
		r = db.Raw(fmt.Sprintf("SHOW ALTER TABLE OPTIMIZE FROM %s", database)).Scan(&optimize)
		if r.Error != nil {
			util.Loggrs.Error(r.Error.Error())
		}
		for _, c1 := range optimize {
			switch c1["State"].(string) {
			case "RUNNING":
				running = append(running, c1["JobId"].(string))
			case "PENDING":
				pending = append(pending, c1["JobId"].(string))
			case "FINISHED":
				continue
			case "CANCELLED":
				errResult = append(errResult,
					util.TaskOptimizeErr{
						JobId:         c1["JobId"],
						TableName:     fmt.Sprintf("%s.%v", database, c1["TableName"]),
						CreateTime:    c1["CreateTime"],
						Operation:     c1["Operation"],
						TransactionId: c1["TransactionId"],
						State:         c1["State"],
						Msg:           c1["Msg"],
						Progress:      c1["Progress"],
						Timeout:       c1["Timeout"],
						Tag:           "OPTIMIZE",
					})
				continue
			}
			result = append(result,
				util.TaskOptimize{
					JobId:         c1["JobId"],
					TableName:     fmt.Sprintf("%s.%v", database, c1["TableName"]),
					CreateTime:    c1["CreateTime"],
					Operation:     c1["Operation"],
					TransactionId: c1["TransactionId"],
					State:         c1["State"],
					Msg:           c1["Msg"],
					Progress:      c1["Progress"],
					Timeout:       c1["Timeout"],
					Tag:           "OPTIMIZE",
					Command:       fmt.Sprintf(`<button id="cancel-optimize" class="btn btn-light btn-sm ms-2" data-optimize="%s">❌</button>`, fmt.Sprintf("OPTIMIZE:%s.%v", database, c1["TableName"])),
				})
		}
	}

	go apicache.Set(app+"optimize-errors", sortByStartTimeDesc5(errResult), cache.DefaultExpiration)

	return optimizeData{
		data:    sortByStartTimeDesc4(result),
		running: len(running),
		pending: len(pending),
		total:   len(running) + len(pending),
	}
}
