/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    main
 *@date    2025/2/12 14:03
 */

package app

import (
	"StarRocksProbe/logs"
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"net/http"
)

func App() {
	//Jemalloc()
	//return
	go assistance()
	r := gin.Default()
	// 初始化 session 中间件（使用 cookie 存储）
	store := cookie.NewStore([]byte("starrocks-probe")) // 替换 "secret" 为你的密钥
	r.Use(sessions.Sessions("visited", store))
	// 内部启动日志访问器
	go logs.Logserver(r)
	// 加载HTML模板
	r.LoadHTMLGlob(util.Read.Server.Loadhtmlglob)
	r.Static("/static", util.Read.Server.Loadstatic)
	r.POST("/api/verify-token", verify)
	r.POST("/verify-token", verifytoken)
	// 定义路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index3.html", nil)
	})
	r.GET("/getstmtid", engine.metriStmtId)
	r.GET("/getbrokid", engine.metriBrokerId)
	r.GET("/getsubmit", engine.metriSubmitId)
	r.GET("/getstreamloadid", engine.metriStreamloadId)
	r.GET("/getstream", engine.metriStreamId)
	r.POST("/setcatch", engine.apicatch)
	r.GET("/appstate", engine.getState)
	r.GET("/ugapi", pluginuserApi)
	r.GET("/html/*path", viewHtml)
	// 受保护的管理员路由
	admin := r.Group("/api")
	admin.Use(tokenAuth()) // 应用管理员认证中间件
	{
		admin.POST("/kills", engine.meticSleep)
		admin.POST("/killall", engine.metriKillall)
		admin.POST("/killw", engine.metriKillw)
		admin.POST("/killc", engine.metriKillc)
		admin.POST("/killone", engine.metriKillone)
		admin.POST("/showerr", engine.ShowErr)
		admin.POST("/broker-err", engine.showBrokerErr)
		admin.GET("/broker", engine.showBroker)
		admin.POST("/cancel-labelId", engine.cancelBroker)
		admin.GET("/submit", engine.showSubmit)
		admin.GET("/optimize", engine.metriOptimize)
		admin.POST("/optimize-err", engine.showErrOptimze)
		admin.POST("/cancel-optimize", engine.cancelOptimize)
		admin.POST("/cancel-task", engine.cancelSubmit)
		admin.POST("/submit-err", engine.showSubmitErr)
		admin.GET("/query", engine.processlist)
		admin.GET("/queries", engine.metriQuery)
		admin.GET("/querysleep", engine.metriQuerysleep)
		admin.GET("/appids", engine.metriGetAppid)
		admin.POST("/disconnect", engine.disconnect)
		admin.GET("/beads", engine.metriErrbead)
		admin.POST("/pie", engine.pie)
		admin.GET("/getgrafan", engine.metrigrafana)
		admin.POST("/stream-err", engine.showStreamErr)
		admin.GET("/stream2load", engine.stream2load)
		admin.POST("/stream2load-err", engine.showErrStream2load)
		admin.GET("/catch", engine.catch)
		admin.GET("/security", engine.getsecurity)
		admin.POST("/dropsecurity", engine.dropsecurity)
		admin.GET("/metadata", engine.metadata)
		admin.GET("/replica", engine.replica)
		admin.GET("/report_path", engine.report)
		admin.GET("/memory_usage", engine.memoryUsage)
		admin.GET("/explain_usage", engine.explainUsage)
	}
	err := r.Run(fmt.Sprintf(":%d", util.Read.Server.Port))
	if err != nil {
		util.Loggrs.Error(err.Error())
		return
	}
}

func viewHtml(c *gin.Context) {
	logFile := c.Param("path")
	util.Loggrs.Info(fmt.Sprintf("request: %s, read: %s", c.ClientIP(), logFile))
	c.File(logFile)
	return
}
