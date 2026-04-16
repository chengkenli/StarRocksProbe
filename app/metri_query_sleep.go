/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri
 *@date    2025/5/27 20:23
 */

package app

import (
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"sort"
)

func (engine *threadMap) metriQuerysleep(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	if val, ok := second30cache.Get(appid + "sleepConnect"); ok {
		item := countProces(val.([]util.Process))
		c.JSON(http.StatusOK, item)
		return
	}
	c.JSON(http.StatusOK, nil)
}

func countProces(processes []util.Process) []util.CountProcess {
	// 使用 map 进行分组统计
	userCountMap := make(map[string]int)
	for _, process := range processes {
		userCountMap[process.User]++
	}
	// 将 map 转换为切片
	var result []util.CountProcess
	for user, count := range userCountMap {
		result = append(result, util.CountProcess{
			User:    user,
			Count:   count,
			Command: fmt.Sprintf(`<button id="kill-disconnect" class="btn btn-light btn-sm ms-2" data-disconnect="%s">⛔</button>`, user),
		})
	}
	// 按 Count 降序排序
	sort.Slice(result, func(i, j int) bool {
		return result[i].Count > result[j].Count
	})
	return result
}
