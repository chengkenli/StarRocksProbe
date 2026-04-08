/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_cancel_submit
 *@date    2025/9/1 10:29
 */

package app

import (
	"StarRocksProbe/util"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

func (engine *threadMap) cancelOptimize(c *gin.Context) {
	util.Loggrs.Infof("%s cancel job", c.ClientIP())
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	db, err := engine.getmapConnect(appid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	type taskmsg struct {
		Taskname []string `json:"taskname"`
	}
	var ids taskmsg
	data, _ := c.GetRawData()
	json.Unmarshal(data, &ids)
	util.Loggrs.Info("cancel label:", strings.Join(ids.Taskname, ","))

	if ids.Taskname == nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "success:nil,failed:nil",
		})
		return
	}
	tag := strings.Split(ids.Taskname[0], ":")[0]
	taskname := strings.Split(ids.Taskname[0], ":")[1]

	var stmt string
	switch tag {
	case "COLUMN":
		stmt = "CANCEL ALTER TABLE COLUMN FROM " + taskname
	case "ROLLUP":
		stmt = "CANCEL ALTER TABLE ROLLUP FROM " + taskname
	case "OPTIMIZE":
		stmt = "CANCEL ALTER TABLE OPTIMIZE FROM " + taskname
	}

	r := db.Exec(stmt)
	if r.Error != nil {
		c.JSON(http.StatusOK, gin.H{
			"message": "success:nil,failed:db exec stmt is failed",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("success:%d,failed:%d", 1, 0),
	})
}
