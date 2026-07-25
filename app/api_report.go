/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    api_report
 *@date    2025/12/16 17:19
 */

package app

import (
	"StarRocksProbe/ins"
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"net/http"
	"strings"
)

// 巡检报告
func (engine *threadMap) report(c *gin.Context) {
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
	var state string

	if _, ok := util.Lastcache.Get(appid + "report"); ok {
		state = "FINISHED"
	} else {
		if _, ok := util.Lastcache.Get(appid + "ins"); !ok {
			util.Loggrs.Info("开始巡检！")
			state = "RUNNING"
			go ins.Ins(appid, db)
			util.Lastcache.Set(appid+"ins", true, cache.DefaultExpiration)
		} else {
			util.Loggrs.Info("巡检队列中已经存在该任务！指标置为FINISHED代表巡检完成！")
			if util.Read.Server.Sshuser == "" {
				util.Loggrs.Info("由于没配置免密账号，本次巡检不对[配置文件]、[进程日志]进行巡检！")
			}
			state = "PENDING"
		}
	}
	uri := fmt.Sprintf("%s:%d/html%s", util.H.Ip, util.Read.Server.Port, strings.NewReplacer("*.html", fmt.Sprintf("ins/%s.html", appid)).Replace(util.Read.Server.Loadhtmlglob))
	c.JSON(http.StatusOK, gin.H{"url": uri, "state": state})
}
