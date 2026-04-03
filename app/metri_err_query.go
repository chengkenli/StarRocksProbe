/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_showerr
 *@date    2025/6/5 16:15
 */

package app

import (
	"StarRocksProbe/conn"
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
	"time"
)

type showErr struct {
	Starttime string `json:"starttime"`
	User      string `json:"user"`
	Queryid   string `json:"queryid"`
	Errmsg    string `json:"errmsg"`
	Errinfo   string `json:"errinfo"`
	Stmt      string `json:"stmt"`
	Category  int    `json:"category"`
}

func (engine *threadMap) ShowErr(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}

	beginTime := c.GetHeader("Begintime")
	endTime := c.GetHeader("Endtime")

	appid = setdefault(appid)
	db, err := engine.getmapConnect(appid)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return
	}

	errs := showErrors(appid, db, beginTime, endTime)
	c.JSON(http.StatusOK, sortByStartTimeDesc(errs))
}

func init_meta(app string) util.ConnectParms {
	var avg util.ConnectParms
	// connect
	if len(util.MetaLink) == 0 {
		return util.ConnectParms{}
	}
	for _, m := range util.MetaLink {
		if m["app"].(string) == app {
			var manager_access_key, manager_secret_key, muser, mpass, mhost string
			if m["manager_access_key"] != nil {
				manager_access_key = m["manager_access_key"].(string)
			}
			if m["manager_secret_key"] != nil {
				manager_secret_key = m["manager_secret_key"].(string)
			}
			if m["meta_user"] != nil {
				muser = m["meta_user"].(string)
			}
			if m["meta_pass"] != nil {
				mpass = m["meta_pass"].(string)
			}
			if m["meta_host"] != nil {
				mhost = m["meta_host"].(string)
			}
			avg = util.ConnectParms{
				App:        app,
				Host:       m["feip"].(string),
				Port:       int(m["feport"].(int32)),
				User:       m["user"].(string),
				Pass:       m["password"].(string),
				Uri:        m["address"].(string),
				AaccessKey: manager_access_key,
				SecretKey:  manager_secret_key,
				MetaUser:   muser,
				MetaPass:   mpass,
				MetaHost:   mhost,
			}
			break
		}
	}
	return avg
}

// 从postgresql中获取信息
func metainfo2(item util.ConnectParms, begin, end string) []map[string]interface{} {
	if item.MetaUser == "" || item.MetaPass == "" || item.MetaHost == "" {
		return nil
	}
	util.Loggrs.Infof("%s 从postgresql元数据中读取审计记录", item.App)
	db, err := conn.ConnectItemPostgreSQL(item)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return nil
	}
	var m []map[string]interface{}
	var stmt string
	if util.Read.Schema.FilterErrmsg == "" {
		stmt = fmt.Sprintf(`
select
query_id       as queryid       ,
conn_id        as connid        ,
remote_ip      as clientIp      ,
fe_host        as feIp          ,
user           as user          ,
start_time     as starttime     ,
end_time       as endtime       ,
time_used      as queryTime     ,
state          as state         ,
error_message  as errormessage  ,
"sql"          as stmt          ,
cpu_cost_ns    as cpuCostNs     ,
mem_cost_bytes as memCostBytes  ,
scan_rows      as scanRows      ,
scan_bytes     as scanBytes     ,
digest         as digest        ,
"database"     as db            ,
profile        as profile       ,
plan           as plan
from public.query_record where start_time>='%s' and start_time<'%s' and state = 'FAILED' order by start_time desc`, begin, end)
	} else {
		whor := fmt.Sprintf("and NOT (lower(error_message) REGEXP '%s')", util.Read.Schema.FilterErrmsg)
		stmt = fmt.Sprintf(`
select
query_id       as queryid       ,
conn_id        as connid        ,
remote_ip      as clientIp      ,
fe_host        as feIp          ,
user           as user          ,
start_time     as starttime     ,
end_time       as endtime       ,
time_used      as queryTime     ,
state          as state         ,
error_message  as errormessage  ,
"sql"          as stmt          ,
cpu_cost_ns    as cpuCostNs     ,
mem_cost_bytes as memCostBytes  ,
scan_rows      as scanRows      ,
scan_bytes     as scanBytes     ,
digest         as digest        ,
"database"     as db            ,
profile        as profile       ,
plan           as plan
from public.query_record where start_time>='%s' and start_time<'%s' and state = 'FAILED' %v order by start_time desc`, begin, end, whor)
	}
	util.Loggrs.Infof(stmt)
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil
	}
	return m
}

// 从mysql数据库中获取信息
func metainfo(item util.ConnectParms, begin, end string) []map[string]interface{} {
	if item.MetaUser == "" || item.MetaPass == "" || item.MetaHost == "" {
		return nil
	}
	util.Loggrs.Infof("%s 从mysql元数据中读取审计记录", item.App)
	db, err := conn.ConnectItemMySQL(item)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return nil
	}
	var database string
	switch item.App {
	case "sr-app":
		database = "starrocksemr"
	case "sr-scct":
		database = "manager_console_scct"
	default:
		database = "manager_console"
	}
	var m []map[string]interface{}
	var stmt string
	if util.Read.Schema.FilterErrmsg == "" {
		stmt = fmt.Sprintf(`
select
query_id       as queryid       ,
conn_id        as connid        ,
remote_ip      as clientIp      ,
fe_host        as feIp          ,
user           as user          ,
start_time     as starttime     ,
end_time       as endtime       ,
time_used      as queryTime     ,
state          as state         ,
error_message  as errormessage  ,
`+"`"+`sql`+"`"+`            as stmt          ,
cpu_cost_ns    as cpuCostNs     ,
mem_cost_bytes as memCostBytes  ,
scan_rows      as scanRows      ,
scan_bytes     as scanBytes     ,
digest         as digest        ,
`+"`"+`database`+"`"+`       as db            ,
profile        as profile       ,
plan           as plan
from %s.query_record where start_time>='%s' and start_time<'%s' and state = 'FAILED' order by start_time desc`, database, begin, end)
	} else {
		whor := fmt.Sprintf("and NOT (lower(error_message) REGEXP '%s')", util.Read.Schema.FilterErrmsg)
		stmt = fmt.Sprintf(`
select
query_id       as queryid       ,
conn_id        as connid        ,
remote_ip      as clientIp      ,
fe_host        as feIp          ,
user           as user          ,
start_time     as starttime     ,
end_time       as endtime       ,
time_used      as queryTime     ,
state          as state         ,
error_message  as errormessage  ,
`+"`"+`sql`+"`"+`            as stmt          ,
cpu_cost_ns    as cpuCostNs     ,
mem_cost_bytes as memCostBytes  ,
scan_rows      as scanRows      ,
scan_bytes     as scanBytes     ,
digest         as digest        ,
`+"`"+`database`+"`"+`       as db            ,
profile        as profile       ,
plan           as plan
from %s.query_record where start_time>='%s' and start_time<'%s' and state = 'FAILED' %v order by start_time desc`, database, begin, end, whor)
	}
	util.Loggrs.Infof(stmt)
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil
	}
	return m
}

// 从starrocks的审计表中获取信息
func srinfo(item util.ConnectParms, db *gorm.DB, app, begin, end string) []map[string]interface{} {
	if util.Read.Schema.Auditops == "" {
		return nil
	}
	util.Loggrs.Infof("%s 从sr审计表中读取审计记录", item.App)
	val, _ := util.Lastcache.Get(app + "version")
	version := tools.VToFloat(val.(string))

	var queryid, starttime string
	if version >= 2.5 {
		queryid = "queryId"
		starttime = "timestamp"
	} else {
		queryid = "query_id"
		starttime = "time"
	}
	os.Setenv("TZ", "Asia/Shanghai")
	var m []map[string]interface{}

	columns := fmt.Sprintf("%s as queryid,%s as starttime,queryType,clientIp,user,authorizedUser,resourceGroup,catalog,db,state,errorCode,queryTime,scanBytes,scanRows,returnRows,cpuCostNs,memCostBytes,stmtId,isQuery,feIp,stmt,digest,planCpuCosts,planMemCosts,pendingTimeMs", queryid, starttime)
	util.Loggrs.Info(begin, ",", end)
	stmt := fmt.Sprintf("select %s from %s where %s >= '%s' and %s<'%s' and user != 'root' and state='ERR' order by starttime desc", columns, util.Read.Schema.Auditops, starttime, begin, starttime, end)
	util.Loggrs.Info(stmt)
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil
	}
	return m
}

// 收集报错信息
func showErrors(app string, db *gorm.DB, begin, end string) []showErr {
	avg := init_meta(app)

	var m []map[string]interface{}
	// 因为这里sr-qa集群属于存算分离+pg的版本，所以这里必须要走pg
	if app == "sr-qa" {
		// 从postgresql元数据中分析数据
		m = metainfo2(avg, begin, end)
	} else {
		// 从mysql元数据中分析数据
		m = metainfo(avg, begin, end)
	}
	if len(m) >= 1 {
		var errs []showErr
		for _, m2 := range m {
			queryId := fmt.Sprintf(`<a href="javascript:void(0)" onclick="fetchAndRedirect('http://%s:%d/getstmtid?app=%s&id=%s&begintime=%s&endtime=%s')" class="plain-text-link" style="color: #333; text-decoration: none; font-weight: normal; cursor: pointer">%s</a>`,
				util.H.Ip, util.Read.Server.Port, app, m2["queryid"].(string),
				m2["starttime"].(time.Time).Add(-30*time.Minute).Format("2006-01-02 15:04:05"),
				m2["endtime"].(time.Time).Add(30*time.Minute).Format("2006-01-02 15:04:05"),
				m2["queryid"].(string))
			var tag, ts string
			if m2["memCostBytes"].(int64) > 0 {
				tag = fmt.Sprintf(`<span style="color: red;">%v</span>`, tools.FormatBytes(m2["memCostBytes"].(int64)))
			} else {
				tag = fmt.Sprintf(`<span style="color: gray;">%v</span>`, tools.FormatBytes(m2["memCostBytes"].(int64)))
			}
			if int(m2["queryTime"].(float64))/1000 > 0 {
				ts = fmt.Sprintf(`<span style="color: blue;">%v</span>`, tools.GetHour(int(m2["queryTime"].(float64))/1000))
			} else {
				ts = fmt.Sprintf(`<span style="color: gray;">%v</span>`, tools.GetHour(int(m2["queryTime"].(float64))/1000))
			}
			errs = append(errs, showErr{
				Starttime: fmt.Sprintf("%v (time:%v)", m2["starttime"].(time.Time).Format("2006-01-02 15:04:05"), ts),
				User:      fmt.Sprintf("%s (memory:%v)", m2["user"].(string), tag),
				Queryid:   queryId,
				Errmsg:    m2["errormessage"].(string),
				Errinfo:   "",
				Stmt:      m2["stmt"].(string),
				Category:  categoryModel(m2["errormessage"].(string)),
			})
		}
		return errs
	}

	// 如果前面拿不到，再从sr审计表中拿
	m = srinfo(avg, db, app, begin, end)
	if len(m) == 0 {
		return nil
	}
	version := tools.Version(app, db)
	var errs []showErr
	switch {
	case version >= 3.3 && version != 3.39:
		msgs := scan1(app, avg, m)
		errs = append(errs, msgs...)
		util.Loggrs.Info(fmt.Sprintf("scan1> source:%d, target:%d", len(m), len(msgs)))
	case version == 3.39:
		msgs := scan2(app, avg, m)
		errs = append(errs, msgs...)
		util.Loggrs.Info(fmt.Sprintf("scan2> source:%d, target:%d", len(m), len(msgs)))
	default:
		msgs := scan0(app, avg, m)
		errs = append(errs, msgs...)
		util.Loggrs.Info(fmt.Sprintf("scan0> source:%d, target:%d", len(m), len(msgs)))
	}
	return errs
}

// 这是最原始的方式，支持2.x-3.x
func scan0(app string, avg util.ConnectParms, m []map[string]interface{}) []showErr {
	/*-------------------------登录manager web-------------------------------*/
	//创建Resty客户端
	Client := resty.New()
	//发送POST请求并处理响应
	result, err := Client.R().SetBody(map[string]string{
		"name":     avg.User,
		"password": avg.Pass,
	}).Post(avg.Uri + "/api/user/login")
	if err != nil {
		util.Loggrs.Warn("err:", err.Error())
		return nil
	}
	if strings.Contains(string(result.Body()), "Failed") {
		util.Loggrs.Warn("login is failed.")
		return nil
	}
	/*-------------------------登录manager web end----------------------------*/
	var errs []showErr

	ch := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for _, m2 := range m {
		wg.Add(1)
		go func(m2 map[string]interface{}) {
			defer func() {
				<-ch
				wg.Done()
			}()

			ch <- struct{}{}

			// cache get
			if v, ok := errcache.Get(app + m2["queryid"].(string)); ok {
				errs = append(errs, v.(showErr))
				return
			}

			/*新的合并模块---------重新利用Cookies请求license*/
			respones3, err := Client.R().Get(avg.Uri + "/api/query/detail/" + m2["queryid"].(string))
			if err != nil {
				util.Loggrs.Warn("请求失败:", err.Error())
				return
			}
			type mm struct {
				Code int `json:"code"`
				Data struct {
					QueryDetail struct {
						QueryID      string   `json:"queryId"`
						User         string   `json:"user"`
						Status       string   `json:"status"`
						ErrorMessage string   `json:"errorMessage"`
						StartTime    int      `json:"startTime"`
						EndTime      int      `json:"endTime"`
						TimeUsed     int      `json:"timeUsed"`
						SQL          string   `json:"sql"`
						ExplainText  []string `json:"explainText"`
						ProfileText  []string `json:"profileText"`
					} `json:"queryDetail"`
				} `json:"data"`
			}

			var m mm
			err = json.Unmarshal(respones3.Body(), &m)
			if err != nil {
				util.Loggrs.Warn(err.Error())
				util.Loggrs.Warn(string(respones3.Body()))
				return
			}

			queryId := fmt.Sprintf(`<a href="javascript:void(0)" onclick="fetchAndRedirect('http://%s:%d/getstmtid?app=%s&id=%s')" class="plain-text-link" style="color: #333; text-decoration: none; font-weight: normal; cursor: pointer">%s</a>`,
				util.H.Ip, util.Read.Server.Port, app, m2["queryid"].(string), m2["queryid"].(string))

			if m.Data.QueryDetail.ErrorMessage == "" {
				errs = append(errs, showErr{
					Starttime: m2["starttime"].(time.Time).Format("2006-01-02 15:04:05"),
					User:      strings.ReplaceAll(m2["user"].(string), "default_cluster:", ""),
					Queryid:   queryId,
					Errmsg:    "manager openapi 找不到报错信息",
					Errinfo:   "",
					Stmt:      m2["stmt"].(string),
					Category:  categoryModel(""),
				})
				return
			}

			sign := showErr{
				Starttime: m2["starttime"].(time.Time).Format("2006-01-02 15:04:05"),
				User:      strings.ReplaceAll(m2["user"].(string), "default_cluster:", ""),
				Queryid:   queryId,
				Errmsg:    m.Data.QueryDetail.ErrorMessage,
				Errinfo:   "",
				Stmt:      m.Data.QueryDetail.SQL,
				Category:  categoryModel(m.Data.QueryDetail.ErrorMessage),
			}
			if sign.Queryid == "" {
				return
			}
			// cache set
			if _, ok := errcache.Get(app + m2["queryid"].(string)); !ok {
				errcache.Set(app+m2["queryid"].(string), sign, cache.DefaultExpiration)
			}
			errs = append(errs, sign)

		}(m2)
	}
	wg.Wait()

	return errs
}

// 这里也是最原始的方式，但这里的密码是加了base64的
func scan2(app string, avg util.ConnectParms, m []map[string]interface{}) []showErr {
	/*-------------------------登录manager web-------------------------------*/
	//创建Resty客户端
	Client := resty.New()
	//发送POST请求并处理响应
	result, err := Client.R().SetBody(map[string]string{
		"name":     avg.User,
		"password": base64.StdEncoding.EncodeToString([]byte(avg.Pass)),
	}).Post(avg.Uri + "/api/user/login")
	if err != nil {
		util.Loggrs.Warn("err:", err.Error())
		return nil
	}
	if strings.Contains(string(result.Body()), "Failed") {
		util.Loggrs.Warn("login is failed.")
		return nil
	}
	/*-------------------------登录manager web end----------------------------*/
	var errs []showErr

	ch := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for _, m2 := range m {
		wg.Add(1)
		go func(m2 map[string]interface{}) {
			defer func() {
				<-ch
				wg.Done()
			}()

			ch <- struct{}{}

			// cache get
			if v, ok := errcache.Get(app + m2["queryid"].(string)); ok {
				errs = append(errs, v.(showErr))
				return
			}

			/*新的合并模块---------重新利用Cookies请求license*/
			respones3, err := Client.R().Get(avg.Uri + "/api/query/detail/" + m2["queryid"].(string))
			if err != nil {
				util.Loggrs.Warn("请求失败:", err.Error())
				return
			}
			type mm struct {
				Code int `json:"code"`
				Data struct {
					QueryDetail struct {
						QueryID      string   `json:"queryId"`
						User         string   `json:"user"`
						Status       string   `json:"status"`
						ErrorMessage string   `json:"errorMessage"`
						StartTime    int      `json:"startTime"`
						EndTime      int      `json:"endTime"`
						TimeUsed     int      `json:"timeUsed"`
						SQL          string   `json:"sql"`
						ExplainText  []string `json:"explainText"`
						ProfileText  []string `json:"profileText"`
					} `json:"queryDetail"`
				} `json:"data"`
			}

			var m mm
			err = json.Unmarshal(respones3.Body(), &m)
			if err != nil {
				util.Loggrs.Warn(err.Error())
				return
			}

			queryId := fmt.Sprintf(`<a href="javascript:void(0)" onclick="fetchAndRedirect('http://%s:%d/getstmtid?app=%s&id=%s')" class="plain-text-link" style="color: #333; text-decoration: none; font-weight: normal; cursor: pointer">%s</a>`,
				util.H.Ip, util.Read.Server.Port, app, m2["queryid"].(string), m2["queryid"].(string))

			if m.Data.QueryDetail.ErrorMessage == "" {
				errs = append(errs, showErr{
					Starttime: m2["starttime"].(time.Time).Format("2006-01-02 15:04:05"),
					User:      strings.ReplaceAll(m2["user"].(string), "default_cluster:", ""),
					Queryid:   queryId,
					Errmsg:    "manager openapi 找不到报错信息",
					Errinfo:   "",
					Stmt:      m2["stmt"].(string),
					Category:  categoryModel(""),
				})
				return
			}

			sign := showErr{
				Starttime: m2["starttime"].(time.Time).Format("2006-01-02 15:04:05"),
				User:      strings.ReplaceAll(m2["user"].(string), "default_cluster:", ""),
				Queryid:   queryId,
				Errmsg:    m.Data.QueryDetail.ErrorMessage,
				Errinfo:   "",
				Stmt:      m.Data.QueryDetail.SQL,
				Category:  categoryModel(m.Data.QueryDetail.ErrorMessage),
			}
			if sign.Queryid == "" {
				return
			}
			// cache set
			if _, ok := errcache.Get(app + m2["queryid"].(string)); !ok {
				errcache.Set(app+m2["queryid"].(string), sign, cache.DefaultExpiration)
			}

			errs = append(errs, sign)

		}(m2)
	}
	wg.Wait()
	return errs
}

func scan1(app string, avg util.ConnectParms, m []map[string]interface{}) []showErr {
	if len(m) == 0 {
		return nil
	}
	var errs []showErr
	client := resty.New()
	ch := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for _, m2 := range m {
		wg.Add(1)
		go func(m2 map[string]interface{}) {
			defer func() {
				<-ch
				wg.Done()
			}()

			ch <- struct{}{}

			// cache get
			if v, ok := errcache.Get(app + m2["queryid"].(string)); ok {
				errs = append(errs, v.(showErr))
				return
			}

			sign := getOpenApi(app, client, avg, m2)

			if sign.Queryid == "" {
				util.Loggrs.Info("queryid找不到")
				return
			}

			// cache set
			if _, ok := errcache.Get(app + m2["queryid"].(string)); !ok {
				errcache.Set(app+m2["queryid"].(string), sign, cache.DefaultExpiration)
			}

			errs = append(errs, sign)
		}(m2)
	}
	wg.Wait()
	return errs
}

func getOpenApi(app string, client *resty.Client, avg util.ConnectParms, m2 map[string]interface{}) showErr {
	nonce := tools.RandomPassWord(32)
	unix := time.Now().Unix()

	AuthCredential := fmt.Sprintf("%s/%d/%s", avg.AaccessKey, unix, nonce)
	AuthCredentialEncrypted := util.HexEncrypt(AuthCredential, avg.SecretKey)
	AuthContent := fmt.Sprintf(`HTTPMethod:GET
CanonicalURI:/openapi/v1/query/record/%s
CanonicalQueryString:
CanonicalForm:`, m2["queryid"].(string))
	AuthSignature := util.HexEncrypt(AuthContent, AuthCredentialEncrypted)
	AuthHeader := fmt.Sprintf("MANAGER-HMAC-SHA256 Credential=%s,Signature=%s", AuthCredential, AuthSignature)

	uri := fmt.Sprintf("%s/openapi/v1/query/record/%s", avg.Uri, m2["queryid"].(string))
	//发送POST请求并处理响应
	respones, err := client.
		R().
		SetHeaders(map[string]string{
			"Content-Type":  "application/x-www-form-urlencoded",
			"Authorization": AuthHeader,
		}).
		Get(strings.NewReplacer(" ", "").Replace(uri))
	if err != nil {
		util.Loggrs.Warn("报错：", err.Error())
		return showErr{}
	}

	queryId := fmt.Sprintf(`<a href="javascript:void(0)" onclick="fetchAndRedirect('http://%s:%d/getstmtid?app=%s&id=%s')" class="plain-text-link" style="color: #333; text-decoration: none; font-weight: normal; cursor: pointer">%s</a>`,
		util.H.Ip, util.Read.Server.Port, app, m2["queryid"].(string), m2["queryid"].(string))

	if strings.Contains(string(respones.Body()), "404 page not found") {
		return showErr{
			Starttime: m2["starttime"].(time.Time).Format("2006-01-02 15:04:05"),
			User:      strings.ReplaceAll(m2["user"].(string), "default_cluster:", ""),
			Queryid:   queryId,
			Errmsg:    "manager openapi 找不到报错信息",
			Errinfo:   "",
			Stmt:      m2["stmt"].(string),
			Category:  categoryModel(""),
		}
	}
	var item util.HexData
	err = json.Unmarshal(respones.Body(), &item)
	if err != nil {
		util.Loggrs.Warn(err.Error())
		return showErr{}
	}

	if item.Data.StartTime > 0 {
		return showErr{
			Starttime: time.Unix(int64(item.Data.StartTime), 0).Format("2006-01-02 15:04:05"),
			User:      item.Data.User,
			Queryid:   queryId,
			Errmsg:    item.Data.FailedReason,
			Errinfo:   "",
			Stmt:      item.Data.SQL,
			Category:  categoryModel(item.Data.FailedReason),
		}
	} else {
		return showErr{
			Starttime: m2["starttime"].(time.Time).Format("2006-01-02 15:04:05"),
			User:      strings.ReplaceAll(m2["user"].(string), "default_cluster:", ""),
			Queryid:   queryId,
			Errmsg:    "manager openapi 找不到报错信息",
			Errinfo:   "",
			Stmt:      m2["stmt"].(string),
			Category:  categoryModel(""),
		}
	}
}

// 按时间降序排序
func sortByStartTimeDesc(errs []showErr) []showErr {
	// 创建副本以避免修改原始切片
	sorted := make([]showErr, len(errs))
	copy(sorted, errs)

	// 自定义排序函数
	sort.Slice(sorted, func(i, j int) bool {
		ti, err1 := time.Parse("2006-01-02 15:04:05", sorted[i].Starttime)
		tj, err2 := time.Parse("2006-01-02 15:04:05", sorted[j].Starttime)

		// 如果解析错误，保持原顺序
		if err1 != nil || err2 != nil {
			return false
		}
		// 降序排序（最新时间排前面）
		return ti.After(tj)
	})
	return sorted
}
func sortByStartTimeDesc2(errs []util.BrokerMsg) []util.BrokerMsg {
	// 创建副本以避免修改原始切片
	sorted := make([]util.BrokerMsg, len(errs))
	copy(sorted, errs)

	// 自定义排序函数
	sort.Slice(sorted, func(i, j int) bool {
		ti, err1 := time.Parse("2006-01-02 15:04:05", sorted[i].CreateTime)
		tj, err2 := time.Parse("2006-01-02 15:04:05", sorted[j].CreateTime)

		// 如果解析错误，保持原顺序
		if err1 != nil || err2 != nil {
			return false
		}
		// 降序排序（最新时间排前面）
		return ti.After(tj)
	})
	return sorted
}
func sortByStartTimeDesc3(errs []util.TaskRuns) []util.TaskRuns {
	// 创建副本以避免修改原始切片
	sorted := make([]util.TaskRuns, len(errs))
	copy(sorted, errs)

	// 自定义排序函数
	sort.Slice(sorted, func(i, j int) bool {
		ti, err1 := time.Parse("2006-01-02 15:04:05", sorted[i].CREATE_TIME)
		tj, err2 := time.Parse("2006-01-02 15:04:05", sorted[j].CREATE_TIME)

		// 如果解析错误，保持原顺序
		if err1 != nil || err2 != nil {
			return false
		}
		// 降序排序（最新时间排前面）
		return ti.After(tj)
	})
	return sorted
}
func sortByStartTimeDesc4(errs []util.TaskOptimize) []util.TaskOptimize {
	// 创建副本以避免修改原始切片
	sorted := make([]util.TaskOptimize, len(errs))
	copy(sorted, errs)

	// 自定义排序函数
	sort.Slice(sorted, func(i, j int) bool {
		ti, err1 := time.Parse("2006-01-02 15:04:05", sorted[i].CreateTime.(string))
		tj, err2 := time.Parse("2006-01-02 15:04:05", sorted[j].CreateTime.(string))

		// 如果解析错误，保持原顺序
		if err1 != nil || err2 != nil {
			return false
		}
		// 降序排序（最新时间排前面）
		return ti.After(tj)
	})
	return sorted
}
func sortByStartTimeDesc5(errs []util.TaskOptimizeErr) []util.TaskOptimizeErr {
	// 创建副本以避免修改原始切片
	sorted := make([]util.TaskOptimizeErr, len(errs))
	copy(sorted, errs)

	// 自定义排序函数
	sort.Slice(sorted, func(i, j int) bool {
		ti, err1 := time.Parse("2006-01-02 15:04:05", sorted[i].CreateTime.(string))
		tj, err2 := time.Parse("2006-01-02 15:04:05", sorted[j].CreateTime.(string))

		// 如果解析错误，保持原顺序
		if err1 != nil || err2 != nil {
			return false
		}
		// 降序排序（最新时间排前面）
		return ti.After(tj)
	})
	return sorted
}

// 评估报凑信息属于平台侧还是用户侧
// 0 未知
// 1 平台侧
// 2 用户侧
func categoryModel(errmsg string) int {
	for _, item := range util.Read.Category.Num1 {
		if strings.Contains(strings.ToLower(errmsg), strings.ToLower(item)) {
			return 1
		}
	}
	for _, item := range util.Read.Category.Num2 {
		if strings.Contains(strings.ToLower(errmsg), strings.ToLower(item)) {
			return 2
		}
	}
	return 0
}
