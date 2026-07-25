/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_get_grafana
 *@date    2025/7/28 11:24
 */

package app

import (
	"StarRocksProbe/util"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"net/http"
)

func (engine *threadMap) metrigrafana(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	dbnum, ok := apicache.Get(appid + "DbName")
	if !ok {
		if _, ok := apicache.Get("grafana"); !ok {
			util.Loggrs.Infof("grafana response:real")
			grafana()
		}
		apicache.Set("grafana", "real", cache.DefaultExpiration)
		dbnum, _ = apicache.Get(appid + "DbName")
	}
	tablenum, _ := apicache.Get(appid + "TableNum")
	partitionnum, _ := apicache.Get(appid + "PartitionNum")
	tabletnum, _ := apicache.Get(appid + "TabletNum")
	replicanum, _ := apicache.Get(appid + "ReplicaNum")
	unhealthytabletnum, _ := apicache.Get(appid + "UnhealthyTabletNum")
	c.JSON(http.StatusOK, gin.H{
		"dbnum":              dbnum,
		"tablenum":           tablenum,
		"partitionnum":       partitionnum,
		"unhealthytabletnum": unhealthytabletnum,
		"tabletNum":          tabletnum,
		"replicaNum":         replicanum,
	})
}
