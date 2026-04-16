/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    init
 *@date    2026/1/7 14:35
 */

package ins

import (
	"StarRocksProbe/util"
	"fmt"
	"gorm.io/gorm"
	"os"
	"strings"
	"sync"
)

var (
	once   sync.Once
	insdir string
)

func init() {
	insdir = strings.NewReplacer("*.html", "ins").Replace(util.Read.Server.Loadhtmlglob)
	once.Do(func() {
		os.MkdirAll(insdir, 0755)
	})
}

// 获取fe和be的信息，并返回
// @avgs *gorm.DB 数据库连接对象
// @result []map[string]interface{}
func backend(db *gorm.DB) ([]map[string]interface{}, []map[string]interface{}) {
	var fenode []map[string]interface{}
	r := db.Raw("show frontends").Scan(&fenode)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return fenode, nil
	}
	var benode []map[string]interface{}
	r = db.Raw("show backends").Scan(&benode)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return fenode, benode
	}
	return fenode, benode
}

// 统计集群用户数
func statisuser(db *gorm.DB) int {
	var m []map[string]interface{}
	r := db.Raw("show users").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return -1
	}
	return len(m)
}

// 统计每天集群访问量
func statisrequest(db *gorm.DB) int64 {
	if util.Read.Schema.Auditops == "" {
		return -1
	}
	var m map[string]interface{}
	r := db.Raw(fmt.Sprintf("select count(*) as count from %s where timestamp >= date_sub(now(), INTERVAL 24 HOUR) ", util.Read.Schema.Auditops)).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return -1
	}
	return m["count"].(int64)
}
