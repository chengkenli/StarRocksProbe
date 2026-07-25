/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    api_memory_usage_jemalloc
 *@date    2026/1/6 14:28
 */

package app

import (
	"StarRocksProbe/util"
	"fmt"
	"github.com/antchfx/htmlquery"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"net/http"
	"strings"
)

type jemallocData struct {
	IP     string           `json:"ip"`
	Total  int              `json:"total"`
	Memory int64            `json:"memory"`
	Body   []jemallocTablet `json:"body"`
}
type jemallocTablet struct {
	Tabletid int `json:"tabletid"`
	Size     int `json:"size"`
}

func (engine *threadMap) jemalloc(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
}

func Jemalloc() {
	get_backend_jemalloc("10.128.8.101")
}

func get_backend_jemalloc(ip string) jemallocData {
	uri := fmt.Sprintf("http://%s:8040/memz", ip)
	//发送POST请求并处理响应
	response, err := resty.New().R().
		SetHeader("Content-Type", "application/json;charset=utf-8").
		Get(uri)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return jemallocData{}
	}
	// 解析 HTML
	doc, err := htmlquery.Parse(strings.NewReader(string(response.Body())))
	if err != nil {
		util.Loggrs.Error(err.Error())
		return jemallocData{}
	}
	// 使用 XPath 定位表格的 tbody
	tbody := htmlquery.FindOne(doc, "/html/body/pre[3]/text()")
	if tbody == nil {
		util.Loggrs.Error("未找到表格 tbody")
		return jemallocData{}
	}
	util.Loggrs.Info(tbody.Data)
	for _, item := range strings.Split(tbody.Data, "\n") {
		util.Loggrs.Info(item)
	}
	return jemallocData{}
}
