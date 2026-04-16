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
	"github.com/gin-gonic/gin"
	"net/http"
)

func (engine *threadMap) showErrOptimze(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}

	beginTime := c.GetHeader("Begintime")
	endTime := c.GetHeader("Endtime")

	appid = setdefault(appid)

	var result []showErr
	if val, ok := apicache.Get(appid + "optimize-errors"); ok {
		for _, item := range val.([]util.TaskOptimizeErr) {
			ok, err := isInrange(beginTime, endTime, item.CreateTime.(string))
			if err != nil {
				util.Loggrs.Error(err.Error())
				continue
			}
			if !ok {
				continue
			}
			var stmt string
			if item.Operation != nil {
				stmt = item.Operation.(string)
			}
			result = append(result, showErr{
				Starttime: item.CreateTime.(string),
				User:      item.TableName.(string),
				Queryid:   item.TransactionId.(string),
				Errmsg:    item.Msg.(string),
				Errinfo:   item.State.(string),
				Stmt:      stmt,
				Category:  categoryModel(item.Msg.(string)),
			})
		}
	}
	c.JSON(http.StatusOK, result)
}
