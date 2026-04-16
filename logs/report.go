/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package logs
 *@file    report
 *@date    2025/12/16 16:09
 */

package logs

import (
	"StarRocksProbe/util"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"net/http"
)

func reports(c *gin.Context) {
	logFile := c.Param("path")
	util.Loggrs.Info(logFile)

	// 直接读取HTML文件并返回
	content, err := ioutil.ReadFile(logFile)
	if err != nil {
		c.String(http.StatusBadRequest, "File not found")
		return
	}
	// 设置Content-Type为HTML
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, string(content))
}
