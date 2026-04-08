/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    map
 *@date    2025/6/6 16:06
 */

package app

import (
	"StarRocksProbe/conn"
	"StarRocksProbe/util"
	"errors"
	"github.com/go-resty/resty/v2"
	"gorm.io/gorm"
	"sync"
)

// 全局连接池（使用 sync.Map 或普通 Map + 互斥锁）
//var syncmap sync.Map // key: 指标名（如 "adhoc"）, value: *gorm.DB

type threadMap struct {
	Pool sync.Map
}

func newPool() *threadMap {
	return &threadMap{Pool: sync.Map{}}
}

func (engine *threadMap) Store(key any, value any) {
	engine.Pool.Store(key, value)
}

func (engine *threadMap) Load(key any) (any, bool) {
	return engine.Pool.Load(key)
}

func (engine *threadMap) Delete(key any) {
	engine.Pool.Delete(key)
}

func (engine *threadMap) setConnectMap(app string) error {
	//starrocks
	db, err := conn.StarRocks(app)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return err
	}
	engine.Store(app+"db", db)
	return nil
}
func (engine *threadMap) setRestyMap(app, user, password string) error {
	//resty
	restys := resty.New().SetDisableWarn(true).SetLogger(&util.CustomLogger{}).SetBasicAuth(user, password)
	engine.Store(app+"resty", restys)
	return nil
}

func (engine *threadMap) getmapConnect(app string) (*gorm.DB, error) {
	// getConnectMap 获取指定指标的数据库连接
	if db, ok := engine.Load(app + "db"); ok {
		return db.(*gorm.DB), nil
	}
	return nil, errors.New("sync.Map connect db is err")
}
func (engine *threadMap) getmapResty(app string) (*resty.Client, error) {
	// getConnectMap 获取指定指标的数据库连接
	if restys, ok := engine.Load(app + "resty"); ok {
		return restys.(*resty.Client), nil
	}
	return nil, errors.New("sync.Map connect resty is err")
}

func (engine *threadMap) _init() {
	for _, m := range util.MetaLink {
		err := engine.setConnectMap(m["app"].(string))
		if err != nil {
			util.Loggrs.Error(err.Error())
		}
		err = engine.setRestyMap(m["app"].(string), m["user"].(string), m["password"].(string))
		if err != nil {
			util.Loggrs.Error(err.Error())
		}
	}
}

// 匹配当前第一个集群
func _app() string {
	return "sr-adhoc"
	if util.MetaLink != nil {
		return util.MetaLink[0]["app"].(string)
	}
	return ""
}
