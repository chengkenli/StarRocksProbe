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
	"context"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"regexp"
	"strings"
	"sync"
	"time"
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

// ResourceClassifiers
// @资源组信息获取
func ResourceClassifiers(app string, db *gorm.DB) {
	var m []map[string]interface{}
	r := db.Raw("SHOW RESOURCE GROUPS ALL").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Warn(r.Error.Error())
		return
	}
	var wg sync.WaitGroup
	for _, m2 := range m {
		wg.Add(1)
		go func(m2 map[string]interface{}) {
			defer wg.Done()

			matches := regexp.MustCompile(`user=([a-zA-Z0-9_]+)`).FindStringSubmatch(m2["classifiers"].(string))
			//util.Loggrs.Infof("%v -> %v -> %v", len(matches), m2["classifiers"].(string), matches)
			if len(matches) >= 2 {
				user := matches[1]
				if user != "" {
					//util.Loggrs.Infof("%v %v", app, user)
					apicache.Set(app+user+"resource_group", m2, cache.NoExpiration)
				}
			}
		}(m2)
	}
	wg.Wait()
}

// 将 "2min:48s"、"10s"、"36.519 s" 转换为秒数（int）
func parseCustomDuration(durationStr string) (int, error) {
	// 预处理字符串：
	// 1. 替换 "min:" 为 "m"（Go 的 Duration 格式要求）
	// 2. 去掉空格和多余的冒号
	normalized := strings.ReplaceAll(durationStr, "min:", "m")
	normalized = strings.ReplaceAll(normalized, " ", "")
	normalized = strings.ReplaceAll(normalized, ":", "")

	// 解析为 time.Duration
	duration, err := time.ParseDuration(normalized)
	if err != nil {
		return 0, fmt.Errorf("解析时间失败: %v", err)
	}
	// 转换为秒（int），四舍五入
	return int(duration.Seconds() + 0.5), nil
}

// TimeoutQueries
// goroutine中的timeout
func TimeoutQueries(ctx context.Context, wg *sync.WaitGroup) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-ctx.Done():
		return false
	}
}
