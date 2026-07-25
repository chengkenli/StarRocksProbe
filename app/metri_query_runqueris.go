/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_get_runqueris
 *@date    2026/7/25 18:53
 */

package app

import (
	"encoding/json"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"StarRocksProbe/util"

	"github.com/gin-gonic/gin"
)

func (engine *threadMap) numRun(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	if val, ok := lastcache.Get(appid + "numRun"); ok {
		item := numRunQueris(val)
		marshal, _ := json.Marshal(item)
		util.Loggrs.Info(string(marshal))
		c.JSON(http.StatusOK, item)
		return
	}

	c.JSON(http.StatusOK, nil)
}

func numRunQueris(val interface{}) (numRunProcess []util.NumRunProcess) {
	for _, item := range val.(util.Backends) {
		numRunningQueries, _ := strconv.Atoi(item.NumRunningQueries)
		dataUsedPct, _ := strconv.ParseFloat(strings.Split(item.DataUsedPct, " ")[0], 64)
		cpuUsedPct, _ := strconv.ParseFloat(strings.Split(item.CpuUsedPct, " ")[0], 64)
		memUsedPct, _ := strconv.ParseFloat(strings.Split(item.MemUsedPct, " ")[0], 64)
		numRunProcess = append(numRunProcess,
			util.NumRunProcess{
				IP:                item.IP,
				Alive:             item.Alive,
				NumRunningQueries: numRunningQueries,
				DataUsedPct:       dataUsedPct,
				CpuUsedPct:        cpuUsedPct,
				MemUsedPct:        memUsedPct,
			})
	}
	// 按 Count 降序排序
	sort.Slice(numRunProcess, func(i, j int) bool {
		return numRunProcess[i].MemUsedPct > numRunProcess[j].MemUsedPct
	})
	return
}
