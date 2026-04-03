/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    applica
 *@date    2026/1/6 16:37
 */

package ins

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
)

// Ins
// 巡检报告模块
func Ins(app string, db *gorm.DB) {
	defer func() {
		util.Lastcache.Delete(app + "ins")
		util.Lastcache.Set(app+"report", true, cache.DefaultExpiration)
	}()
	fs, bs := backend(db)
	var felogpath, belogpath string
	for _, m := range util.MetaLink {
		if m["app"].(string) == "app" {
			if m["fe_log_path"] != nil {
				felogpath = m["fe_log_path"].(string)
			}
			if m["be_log_path"] != nil {
				belogpath = m["be_log_path"].(string)
			}
			break
		}
	}
	var availdateJson []byte
	// 首页
	version := tools.Version(app, db)
	SubjectIndex(app, fmt.Sprintf("%v", version), felogpath, belogpath, db, fs, bs, availdateJson)
	util.Loggrs.Info("巡检完成！")
}
