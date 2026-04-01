/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    analy_processlist
 *@date    2025/5/26 13:40
 */

package app

import (
	"StarRocksProbe/conn"
	"StarRocksProbe/meta"
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type queryResult struct {
	Id         string   `json:"id"`
	User       string   `json:"user"`
	Userurl    string   `json:"userurl"`
	Host       string   `json:"host"`
	Clientuser string   `json:"clientuser"`
	Db         string   `json:"db"`
	Command    string   `json:"command"`
	Time       int      `json:"time"`
	State      string   `json:"state"`
	IsPending  bool     `json:"isPending"`
	Info       string   `json:"info"`
	Warehouse  string   `json:"warehouse"`
	IsSlow     bool     `json:"isSlow"`     // 标记慢查询(600-1500ms)
	IsCritical bool     `json:"isCritical"` // 标记高耗时查询(>1500ms)
	Uri        string   `json:"uri"`
	Whitelist  bool     `json:"whitelist"`
	Feip       string   `json:"feip"`
	Shortlist  bool     `json:"shortlist"`
	Ctxip      []string `json:"ctxip"`
	Gethour    string   `json:"gethour"`
	Getmin     string   `json:"getmin"`
	Admin      bool     `json:"admin"`
	Point      bool     `json:"point"`
	//自定义
	Operational      string
	OperationalMsg   string
	Concurrencylimit int
}
type dataItem struct {
	Data      []queryResult `json:"data"`
	Run       int           `json:"run"`
	Pend      int           `json:"pend"`
	Sleep     int           `json:"sleep"`
	Count     int           `json:"count"`
	Fe        []string      `json:"fe"`
	Title     string        `json:"title"`
	Threshold interface{}   `json:"threshold"`
}

func (engine *threadMap) processlist(c *gin.Context) {

	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	db, err := engine.getmapConnect(appid)
	if err != nil {
		util.Loggrs.Error(err.Error())
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	// 系统第一次加载，触发预热功能
	go func() {
		once.Do(func() {
			for _, m := range util.MetaLink {
				app := m["app"].(string)
				db, err := engine.getmapConnect(app)
				if err != nil {
					util.Loggrs.Error(err.Error())
					return
				}
				tools.GetUserOperational(db, app, errcache)
			}
		})
	}()
	// get
	if v, ok := querycache.Get(appid + "data"); ok {
		pend, _ := querycache.Get(appid + "pend")
		sleep, _ := querycache.Get(appid + "sleep")
		fe, _ := querycache.Get(appid + "fe")

		c.JSON(http.StatusOK, gin.H{
			"data":      sortResult(v.([]queryResult)),
			"run":       len(v.([]queryResult)),
			"pend":      pend.(int),
			"sleep":     sleep.(int),
			"count":     len(v.([]queryResult)) + sleep.(int),
			"fe":        fe.([]string),
			"threshold": getfc2key(appid, "query_queue_max_queued_queries", db),
		})
		util.Loggrs.Info(c.ClientIP(), " buff response:cache")
		return
	}

	var felist []string
	var m []map[string]interface{}
	r := db.Raw("show frontends").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Warn(r.Error.Error())
		return
	}

	var mfe []map[string]string
	for _, item := range m {
		if item["Alive"].(string) != "true" {
			continue
		}
		mfe = append(mfe, map[string]string{
			item["IP"].(string): item["Role"].(string),
		})
		felist = append(felist, item["IP"].(string))
	}

	// http sql api
	if tools.Version(db) >= 3.3 {
		c.JSON(http.StatusOK, engine.httpapi(db, appid, felist))
		util.Loggrs.Info(c.ClientIP(), " http response:actual")
		return
	} else {
		c.JSON(http.StatusOK, engine.strfunc(db, appid, felist, mfe))
		util.Loggrs.Info(c.ClientIP(), " func response:actual")
		return
	}
	// end
}

func sortResult(results []queryResult) []queryResult {
	// 标记慢查询和关键查询
	for i := range results {
		if results[i].Time > 1500 {
			results[i].IsCritical = true
			results[i].IsSlow = true
		} else if results[i].Time > 600 {
			results[i].IsSlow = true
		}
	}
	// 按Time降序排序
	sort.Slice(results, func(i, j int) bool {
		return results[i].Time > results[j].Time
	})
	return results
}

func ctxIp(ip string) []string {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	domains, err := net.DefaultResolver.LookupAddr(ctx, ip)
	if err != nil {
		return nil
	}
	apicache.Set("ctx"+ip, domains, cache.DefaultExpiration)
	return domains
}

// 通过db执行show processlist
func (engine *threadMap) strfunc(db *gorm.DB, app string, felist []string, mfe []map[string]string) dataItem {
	var result []queryResult
	var running, pending, query, sleep, femsg []string

	var resultmsg []util.ProcessTags
	for _, ip := range felist {
		single, err := conn.StarRocksSingle(app, ip)
		if err != nil {
			util.Loggrs.Warn(err.Error())
			continue
		}

		var dbresult []util.Process
		r := single.Raw("show processlist").Scan(&dbresult)
		if r.Error != nil {
			util.Loggrs.Warn(r.Error.Error())
			continue
		}
		resultmsg = append(resultmsg, util.ProcessTags{
			Prolist: dbresult,
			Fe:      ip,
		})

		/*每次使用完，主动关闭连接数*/
		sqlDB, err := single.DB()
		if err != nil {
			util.Loggrs.Error(err.Error())
			continue
		}
		sqlDB.SetMaxOpenConns(10)                 //最大连接数
		sqlDB.SetMaxIdleConns(3)                  //最大空闲连接数
		sqlDB.SetConnMaxLifetime(5 * time.Second) //空闲连接最多存活时间
		sqlDB.Close()
	}

	//进行第一次筛选
	var user_query_process, user_sleep_process []util.Process
	var user_query_limit, user_sleep_limit []string
	for _, api := range resultmsg {
		for _, process := range api.Prolist {
			// 判断睡眠连接
			switch process.Command {
			case "Sleep":
				user_sleep_limit = append(user_sleep_limit, process.User)
				user_sleep_process = append(user_sleep_process, util.Process{
					Id:        process.Id,
					User:      process.User,
					Host:      process.Host,
					Cluster:   process.User,
					Db:        process.Db,
					Command:   process.Command,
					Time:      process.Time,
					State:     process.State,
					Info:      process.Info,
					IsPending: process.IsPending,
					Warehouse: process.Warehouse,
				})
			case "Query":
				user_query_limit = append(user_query_limit, process.User)
				user_query_process = append(user_query_process, util.Process{
					Id:        process.Id,
					User:      process.User,
					Host:      process.Host,
					Cluster:   process.User,
					Db:        process.Db,
					Command:   process.Command,
					Time:      process.Time,
					State:     process.State,
					Info:      process.Info,
					IsPending: process.IsPending,
					Warehouse: process.Warehouse,
				})
			}
		}
	}
	//统计query
	sumTag1 := countUsers(user_query_process)
	for _, user := range tools.RmDuplicaSlice(user_query_limit) {
		limit := sumTag1[user]
		var concurrency_limit []string
		if limit != 0 {
			concurrency_limit = append(concurrency_limit, fmt.Sprintf("concurrency_limit\n  Alive: %d", limit))
			//统计sleep
			sumTag2 := countUsers(user_sleep_process)
			limit := sumTag2[user]
			if limit != 0 {
				concurrency_limit = append(concurrency_limit, fmt.Sprintf("  Sleep: %d", limit))
			}
			cachelimit.Set(user+"concurrency_limit", strings.Join(concurrency_limit, "\n"), cache.DefaultExpiration)
		}
	}
	second30cache.Set(app+"sleepConnect", user_sleep_process, cache.DefaultExpiration)
	for _, m := range resultmsg {
		for _, process := range m.Prolist {
			if process.Command == "Sleep" {
				sleep = append(sleep, "1")
				continue
			}
			edtime, _ := strconv.Atoi(process.Time)
			var isslow, iscritical bool
			if edtime >= 600 && edtime < 1500 {
				isslow = true
			} else if edtime >= 1500 {
				iscritical = true
			}
			ok, _ := strconv.ParseBool(process.IsPending)
			if ok {
				pending = append(pending, "1")
			}
			query = append(query, "1")
			// 检查白名单标记
			whitelist := false
			if util.Read.Schema.Whitelist != "" {
				list := strings.Split(util.Read.Schema.Whitelist, ",")
				if tools.StrInSlice(process.User, list) {
					whitelist = true
				}
			}
			clientip := strings.Split(process.Host, ":")[0]
			var domainname []string
			if v, ok := apicache.Get("ctx" + clientip); ok {
				domainname = v.([]string)
			} else {
				domainname = ctxIp(clientip)
			}
			clientname := getipname(clientip, domainname...)

			shorts := false
			if v, ok := shortcache.Get(app + "short"); ok {
				if v != nil {
					if tools.StrInSlice(process.User, v.([]string)) {
						shorts = true
					}
				}
			}
			labela := fmt.Sprintf(`<a href="javascript:void(0)" onclick="fetchAndRedirect('http://%s:%d/getstmtid?app=%s&id=%s')" class="plain-text-link" style="color: #333; text-decoration: none; font-weight: normal; cursor: pointer">%s</a>`,
				util.H.Ip, util.Read.Server.Port, app, process.Id, process.Id)

			idname := fmt.Sprintf(`<input type="checkbox" class="query-checkbox" data-id="%s">%s`, process.Id, labela)
			// role juct
			adminpoint(db, app, process.User)
			if adminpoint(db, app, process.User) {
				idname = fmt.Sprintf(`<input type="checkbox" class="query-checkbox" data-id="%s">%s👑`, process.Id, labela)
			}
			var operational, operationalMsg string
			if val, ok := errcache.Get(app + process.User + "operational"); ok {
				operational = val.(string)
			} else {
				operational = process.Db
			}
			if val, ok := errcache.Get(app + process.User + "operationalmsg"); ok {
				operationalMsg = val.(string)
			}

			// 分析并发数
			var limitStr string
			if val, ok := cachelimit.Get(process.User + "concurrency_limit"); ok {
				limitStr = "\n" + val.(string)
			}

			userurl := fmt.Sprintf(`<a href="javascript:void(0)" onclick="fetchAndRedirect('http://%s:%d/ugapi?app=%s&user=%s')" class="plain-text-link" style="color: #333; text-decoration: none; font-weight: normal; cursor: pointer">%s</a>`,
				util.H.Ip, util.Read.Server.Port, app, process.User, process.User)

			result = append(result, queryResult{
				Id:             idname,
				User:           process.User,
				Userurl:        userurl,
				Host:           clientip,
				Clientuser:     clientname,
				Db:             process.Db,
				Command:        fmt.Sprintf(`<button id="kill-connectid" class="btn btn-light btn-sm ms-2" data-connectid="%s">❌</button>`, process.Id),
				Time:           edtime,
				State:          process.State,
				IsPending:      ok,
				Info:           process.Info,
				Warehouse:      fmt.Sprintf(`<button id="kill-disconnect" class="btn btn-light btn-sm ms-2" data-disconnect="%s">⛔</button>`, process.User),
				IsSlow:         isslow,
				IsCritical:     iscritical,
				Whitelist:      whitelist,
				Feip:           fmt.Sprintf("fe:(%s)%s \n%s", m.Fe, limitStr, strings.Join(getmaxconnections(app, process.User, db), "")),
				Shortlist:      shorts,
				Ctxip:          domainname,
				Gethour:        tools.GetHour(edtime),
				Getmin:         tools.GetHour(edtime),
				Admin:          adminpoint(db, app, process.User),
				Operational:    operational,
				OperationalMsg: operationalMsg,
			})

			femsg = append(femsg, fmt.Sprintf("%s (%d)(%d)(%d)", m.Fe, len(query), len(pending), len(sleep)))
		}
	}

	// 载入缓存
	go func() {
		// set
		querycache.Set(app+"data", result, cache.DefaultExpiration)
		querycache.Set(app+"pend", len(pending), cache.DefaultExpiration)
		querycache.Set(app+"sleep", len(sleep), cache.DefaultExpiration)
		querycache.Set(app+"fe", running, cache.DefaultExpiration)

		if app == "sr-cdp" || app == "sr-api" || app == "sr-ma" {
			return
		}
		if _, ok := shortcache.Get(app + "short"); !ok {
			shortdata := meta.ShortQuery(db)
			if shortdata != nil {
				shortcache.Set(app+"short", shortdata, cache.DefaultExpiration)
			}
		}
	}()

	// 统计汇报
	return dataItem{
		Data:      sortResult(result),
		Run:       len(query),
		Pend:      len(pending),
		Sleep:     len(sleep),
		Count:     len(result),
		Fe:        femsg,
		Title:     gettitle(db, app),
		Threshold: getfc2key(app, "query_queue_max_queued_queries", db),
	}
}

// 通过api执行show processlist
func (engine *threadMap) httpapi(db *gorm.DB, app string, felist []string) dataItem {
	restys, err := engine.getmapResty(app)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return dataItem{}
	}
	var count int
	var result []queryResult
	var running, pending, query, sleep, femsg []string

	//start
	var resultmsg []util.HttpSqlApi
	for _, fe := range felist {
		uri := fmt.Sprintf("http://%s:8030/api/v1/catalogs/default_catalog/databases/information_schema/sql", fe)
		respones, err := restys.R().
			SetHeader("Content-Type", "application/json").
			SetBody(map[string]interface{}{
				"query": "show processlist",
			}).Post(uri)
		if err != nil {
			util.Loggrs.Error(err.Error())
			continue
		}
		var m util.HttpSqlApi
		err = json.Unmarshal(respones.Body(), &m)
		if err != nil {
			util.Loggrs.Error(err.Error())
			continue
		}
		m.Fe = fe
		resultmsg = append(resultmsg, m)
	}
	//end

	//进行第一次筛选
	var user_query_process, user_sleep_process []util.Process
	var user_query_limit, user_sleep_limit []string
	for _, api := range resultmsg {
		for _, datum := range api.Data {
			// 判断睡眠连接
			switch datum.Command {
			case "Sleep":
				user_sleep_limit = append(user_sleep_limit, datum.User)
				user_sleep_process = append(user_sleep_process, util.Process{
					Id:        datum.ID,
					User:      datum.User,
					Host:      datum.Host,
					Cluster:   datum.User,
					Db:        datum.Db,
					Command:   datum.Command,
					Time:      datum.Time,
					State:     datum.State,
					Info:      datum.Info,
					IsPending: datum.IsPending,
					Warehouse: datum.Warehouse,
				})
			case "Query":
				if datum.Time == "0" {
					continue
				}
				user_query_limit = append(user_query_limit, datum.User)
				user_query_process = append(user_query_process, util.Process{
					Id:        datum.ID,
					User:      datum.User,
					Host:      datum.Host,
					Cluster:   datum.User,
					Db:        datum.Db,
					Command:   datum.Command,
					Time:      datum.Time,
					State:     datum.State,
					Info:      datum.Info,
					IsPending: datum.IsPending,
					Warehouse: datum.Warehouse,
				})
			}
		}
	}
	//统计query
	sumTag1 := countUsers(user_query_process)
	for _, user := range tools.RmDuplicaSlice(user_query_limit) {
		limit := sumTag1[user]
		var concurrency_limit []string
		if limit != 0 {
			concurrency_limit = append(concurrency_limit, fmt.Sprintf("concurrency_limit\n  Alive: %d", limit))
			//统计sleep
			sumTag2 := countUsers(user_sleep_process)
			limit := sumTag2[user]
			if limit != 0 {
				concurrency_limit = append(concurrency_limit, fmt.Sprintf("  Sleep: %d", limit))
			}
			cachelimit.Set(user+"concurrency_limit", strings.Join(concurrency_limit, "\n"), cache.DefaultExpiration)
		}
	}
	second30cache.Set(app+"sleepConnect", user_sleep_process, cache.DefaultExpiration)
	//进行第二次筛选
	for _, api := range resultmsg {
		for _, datum := range api.Data {
			if datum.Time == "0" {
				continue
			}
			// 判断队列堵塞
			switch datum.IsPending {
			case "true":
				pending = append(pending, datum.User)
			case "false":
				running = append(running, datum.User)
			}
			// 判断睡眠连接
			switch datum.Command {
			case "Sleep":
				sleep = append(sleep, datum.User)
				continue
			case "Query":
				query = append(query, datum.User)
			}
			// id 意图判断
			labela := fmt.Sprintf(`<a href="javascript:void(0)" onclick="fetchAndRedirect('http://%s:%d/getstmtid?app=%s&id=%s')" class="plain-text-link" style="color: #333; text-decoration: none; font-weight: normal; cursor: pointer">%s</a>`,
				util.H.Ip, util.Read.Server.Port, app, datum.ID, datum.ID)
			idname := fmt.Sprintf(`<input type="checkbox" class="query-checkbox" data-id="%s">%s`, datum.ID, labela)
			// role juct
			if adminpoint(db, app, datum.User) {
				idname = fmt.Sprintf(`<input type="checkbox" class="query-checkbox" data-id="%s">%s👑`, datum.ID, labela)
			}
			// 类型转换
			Time, _ := strconv.Atoi(datum.Time)
			isPending, _ := strconv.ParseBool(datum.IsPending)

			// 耗时计算
			var isslow, iscritical bool
			if Time >= 600 && Time < 1500 {
				isslow = true
			} else if Time >= 1500 {
				iscritical = true
			}

			// 物理地址追踪
			clientip := strings.Split(datum.Host, ":")[0]
			var domainname []string
			if v, ok := apicache.Get("ctx" + clientip); ok {
				domainname = v.([]string)
			} else {
				domainname = ctxIp(clientip)
			}
			clientname := getipname(clientip, domainname...)

			// 白名单匹配
			whitelist := false
			if util.Read.Schema.Whitelist != "" {
				list := strings.Split(util.Read.Schema.Whitelist, ",")
				if tools.StrInSlice(datum.User, list) {
					whitelist = true
				}
			}
			// 短查询匹配
			shorts := false
			if v, ok := shortcache.Get(app + "short"); ok {
				if v != nil {
					if tools.StrInSlice(datum.User, v.([]string)) {
						shorts = true
					}
				}
			}

			var operational, operationalMsg string
			if val, ok := errcache.Get(app + datum.User + "operational"); ok {
				operational = val.(string)
			} else {
				operational = datum.Db
			}
			if val, ok := errcache.Get(app + datum.User + "operationalmsg"); ok {
				operationalMsg = val.(string)
			}

			// 分析并发数
			var limitStr string
			if val, ok := cachelimit.Get(datum.User + "concurrency_limit"); ok {
				limitStr = "\n" + val.(string)
			}

			userurl := fmt.Sprintf(`<a href="javascript:void(0)" onclick="fetchAndRedirect('http://%s:%d/ugapi?app=%s&user=%s')" class="plain-text-link" style="color: #333; text-decoration: none; font-weight: normal; cursor: pointer">%s</a>`,
				util.H.Ip, util.Read.Server.Port, app, datum.User, datum.User)

			result = append(result, queryResult{
				Id:             idname,
				User:           datum.User,
				Userurl:        userurl,
				Host:           clientip,
				Clientuser:     clientname,
				Db:             datum.Db,
				Command:        fmt.Sprintf(`<button id="kill-connectid" class="btn btn-light btn-sm ms-2" data-connectid="%s">❌</button>`, datum.ID),
				Time:           Time,
				State:          datum.State,
				IsPending:      isPending,
				Info:           datum.Info,
				Warehouse:      fmt.Sprintf(`<button id="kill-disconnect" class="btn btn-light btn-sm ms-2" data-disconnect="%s">⛔</button>`, datum.User),
				IsSlow:         isslow,
				IsCritical:     iscritical,
				Whitelist:      whitelist,
				Feip:           fmt.Sprintf("fe:(%s)%s \n%s", api.Fe, limitStr, strings.Join(getmaxconnections(app, datum.User, db), "")),
				Shortlist:      shorts,
				Ctxip:          domainname,
				Gethour:        fmt.Sprintf("%s(%s)", datum.ConnectionStartTime, tools.GetHour(Time)),
				Getmin:         tools.GetHour(Time),
				Admin:          adminpoint(db, app, datum.User),
				Operational:    operational,
				OperationalMsg: operationalMsg,
			})
		}
		count = +api.Statistics.ReturnRows
		femsg = append(femsg, fmt.Sprintf("%s (%d)(%d)(%d)", api.Fe, len(query), len(pending), len(sleep)))
	}

	// 载入缓存
	go func() {
		// set
		querycache.Set(app+"data", result, cache.DefaultExpiration)
		querycache.Set(app+"pend", len(pending), cache.DefaultExpiration)
		querycache.Set(app+"sleep", len(sleep), cache.DefaultExpiration)
		querycache.Set(app+"fe", femsg, cache.DefaultExpiration)

		if app == "sr-cdp" || app == "sr-api" || app == "sr-ma" {
			return
		}
		if _, ok := shortcache.Get(app + "short"); !ok {
			shortdata := meta.ShortQuery(db)
			if shortdata != nil {
				shortcache.Set(app+"short", shortdata, cache.DefaultExpiration)
			}
		}
	}()

	// 统计汇报
	return dataItem{
		Data:      sortResult(result),
		Run:       len(query),
		Pend:      len(pending),
		Sleep:     len(sleep),
		Count:     count,
		Fe:        femsg,
		Title:     gettitle(db, app),
		Threshold: getfc2key(app, "query_queue_max_queued_queries", db),
	}
}

func getmaxconnections(app, username string, db *gorm.DB) []string {
	if v, ok := apicache.Get(app + username + "property"); ok {
		return v.([]string)
	}
	var m []map[string]interface{}
	r := db.Raw(fmt.Sprintf("SHOW PROPERTY FOR '%s'", username)).Scan(&m)
	if r.Error != nil {
		return nil
	}
	var msg []string
	for i, m2 := range m {
		msg = append(msg, fmt.Sprintf("%d.%s:%s\n", i, m2["Key"], m2["Value"]))
	}
	go apicache.Set(app+username+"property", msg, cache.DefaultExpiration)
	return msg
}

// 分析每个用户的提交量
func countUsers(processes []util.Process) map[string]int {
	userCount := make(map[string]int)
	for _, process := range processes {
		userCount[process.User]++
	}
	return userCount
}

// 分析账号是否是管理员角色
func adminpoint(db *gorm.DB, app, user string) (b bool) {
	if val, ok := apicache.Get(app + user + "adminpoint"); ok {
		b = val.(bool)
		return b
	}
	var m []map[string]interface{}
	r := db.Raw("show grants for " + strings.ToLower(user)).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		b = false
	}
	marshal, err := json.Marshal(m)
	if err != nil {
		util.Loggrs.Error(err.Error())
		b = false
	}
	if strings.Contains(string(marshal), "_admin") || strings.Contains(string(marshal), "root") {
		b = true
	}
	apicache.Set(app+user+"adminpoint", b, cache.DefaultExpiration)
	return b
}
