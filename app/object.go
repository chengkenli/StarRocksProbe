/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    node_info
 *@date    2025/10/29 14:28
 */

package app

import (
	"StarRocksProbe/util"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/patrickmn/go-cache"
)

// ExecDBOperation
// @args appid string
// @args command string
// @args fe node ...[]string
// @result response
// 通过接口的形式，执行sql
func ExecDBOperation(appid, command string, feitem ...string) (response *resty.Response) {
	restys, err := engine.getmapResty(appid)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return nil
	}
	if feitem == nil {
		feitem = fronendNodes(appid)
	}
	for _, fe := range feitem {
		uri := fmt.Sprintf(`http://%s:8030/api/v1/catalogs/default_catalog/databases/information_schema/sql`, fe)
		//发送POST请求并处理响应
		util.Loggrs.Infof("%s uri: %v", appid, uri)
		var err error
		response, err = restys.R().SetBody(fmt.Sprintf(`{"query": "%s"}`, command)).Post(uri)
		if err != nil {
			util.Loggrs.Error(err.Error())
			continue
		}
		util.Loggrs.Infof("%s %s uri: %v request: %v, response: %v", appid, fe, uri, command, string(response.Body()))
		break
	}
	return
}

// 获取某集群的fe信息
func fronendNodes(app string) []string {
	if val, ok := lastcache.Get(app + "frontends"); ok {
		return val.([]string)
	}
	db, err := engine.getmapConnect(app)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return nil
	}
	var m []map[string]interface{}
	var a []string
	db.Raw("show frontends").Scan(&m)
	for _, item := range m {
		a = append(a, item["IP"].(string))
	}
	lastcache.Set(app+"frontends", a, cache.DefaultExpiration)
	return a
}
