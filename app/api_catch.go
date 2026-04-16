/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    api_catch
 *@date    2025/11/4 16:33
 */

package app

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"net/http"
)

type catchdata struct {
	App      string `json:"app"`
	Ts       string `json:"ts"`
	Table    string `json:"table"`
	Count    int64  `json:"count"`
	Before   int    `json:"before"`
	Last     int    `json:"last"`
	Progress int    `json:"progress"`
	Class    string `json:"class"`
	Comment  string `json:"comment"`
	State    string `json:"state"`
}

var slice = NewExpiringCatchDataSlice()

func (engine *threadMap) apicatch(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	data, err := c.GetRawData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	var catch catchdata
	json.Unmarshal(data, &catch)

	if slice.GetByTable(catch.Table) != nil {
		switch catch.State {
		case "RUNNING":
			/*还没完成的，已经存在，就不重复加了，重置时间*/
			slice.ResetExp(catch.Table)
		case "FINISHED":
			/*已经完成了，删除目前的*/
			slice.Del(catch.Table)
		}
	} else {
		slice.Add(catch)
	}
}

func (engine *threadMap) catch(c *gin.Context) {
	//defer slice.Stop()
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	data := slice.GetAll()

	var r, p, f []string
	var catch []catchdata
	for _, item := range data {
		if item.App == appid {
			catch = append(catch, item)
			switch item.State {
			case "RUNNING":
				r = append(r, item.Table)
			case "PENDING":
				p = append(p, item.Table)
			case "FINISHED":
				f = append(f, item.Table)
			}
		}
	}
	var dts []catchdata
	if len(catch) >= 10 {
		dts = catch[0:10]
	} else {
		dts = catch
	}
	c.JSON(http.StatusOK, gin.H{"data": dts, "running": len(r), "pending": len(p), "total": len(catch)})
}
