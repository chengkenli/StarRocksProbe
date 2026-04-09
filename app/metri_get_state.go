/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_get_state
 *@date    2025/10/27 9:21
 */

package app

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (engine *threadMap) getState(c *gin.Context) {
	appid, _ := c.GetQuery("app")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	db, err := engine.getmapConnect(appid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	var m1 []map[string]interface{}
	r := db.Raw("show frontends").Scan(&m1)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		c.JSON(http.StatusInternalServerError, r.Error.Error())
		return
	}
	var front []util.Frontend
	var status []string
	for _, item := range m1 {
		front = append(front, util.Frontend{
			IP:    item["IP"].(string),
			Alive: item["Alive"].(string),
		})
		status = append(status, fmt.Sprintf("%v", item["Alive"].(string)))
	}

	var m2 []map[string]interface{}
	r = db.Raw("show backends").Scan(&m2)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		c.JSON(http.StatusInternalServerError, r.Error.Error())
		return
	}
	var backd []util.Backend
	for _, item := range m2 {
		backd = append(backd, util.Backend{
			IP:    item["IP"].(string),
			Alive: item["Alive"].(string),
		})
		status = append(status, fmt.Sprintf("%v", item["Alive"].(string)))
	}
	var state int
	if tools.IsUniform(status, "true") {
		state = 1
	} else if tools.IsUniform(status, "false") {
		state = -1
	} else if tools.StrInSlice("false", status) || tools.StrInSlice("true", status) {
		state = 2
	} else {
		state = 0
	}

	c.JSON(http.StatusOK, util.AppStates{
		ServiceDetectStatus: state,
		Frontends:           front,
		Backends:            backd,
	})
}
