/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    api_explain_usage_detail
 *@date    2026/4/1 14:54
 */

package app

import (
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (engine *threadMap) explainUsage(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"url": fmt.Sprintf("http://%s:%d", util.H.Ip, util.Read.Server.Port+30)})
}
