/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_stmtid
 *@date    2025/5/27 15:55
 */

package app

import (
	"StarRocksProbe/conn"
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

func (engine *threadMap) metriStmtId(c *gin.Context) {
	stmtid, _ := c.GetQuery("id")
	appid, _ := c.GetQuery("app")
	beginTime, _ := c.GetQuery("begintime")
	endTime, _ := c.GetQuery("endtime")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}

	db, err := engine.getmapConnect(appid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}

	var result string
	if !regexp.MustCompile(`[a-zA-Z]`).MatchString(stmtid) {
		result = getConnectionId(appid, stmtid, db)
	} else {
		result = getQueryId(appid, stmtid, beginTime, endTime, db)
	}
	c.JSON(http.StatusOK, result)
}

// 从postgresql数据库中获取信息
func metainfoid2(item util.ConnectParms, begin, end, queryid string) map[string]interface{} {
	if item.MetaUser == "" || item.MetaPass == "" || item.MetaHost == "" {
		return nil
	}
	util.Loggrs.Infof("%s 从postgresql元数据中读取审计记录", item.App)
	db, err := conn.ConnectItemPostgreSQL(item)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return nil
	}
	var m map[string]interface{}
	stmt := fmt.Sprintf(`
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
from public.query_record where start_time>='%s' and start_time<'%s' and query_id='%s'`, begin, end, queryid)
	util.Loggrs.Infof(stmt)
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil
	}
	return m
}

// 从mysql数据库中获取信息
func metainfoid(item util.ConnectParms, begin, end, queryid string) map[string]interface{} {
	if item.MetaUser == "" || item.MetaPass == "" || item.MetaHost == "" {
		return nil
	}
	util.Loggrs.Infof("%s 从mysql元数据中读取审计记录", item.App)
	db, err := conn.ConnectItemMySQL(item)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return nil
	}
	/*每次使用完，主动关闭连接数*/
	defer func() {
		sqlDB, err := db.DB()
		if err != nil {
			util.Loggrs.Error(err.Error())
			return
		}
		sqlDB.SetMaxOpenConns(30)                  //最大连接数
		sqlDB.SetMaxIdleConns(30)                  //最大空闲连接数
		sqlDB.SetConnMaxLifetime(30 * time.Second) //空闲连接最多存活时间
		sqlDB.Close()
	}()
	var database string
	switch item.App {
	case "sr-app":
		database = "starrocksemr"
	case "sr-scct":
		database = "manager_console_scct"
	default:
		database = "manager_console"
	}
	var m map[string]interface{}
	stmt := fmt.Sprintf(`
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
from %s.query_record where start_time>='%s' and start_time<'%s' and query_id='%s'`, database, begin, end, queryid)
	util.Loggrs.Infof(stmt)
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil
	}
	return m
}

func getQueryId(appid, stmtid, begin, end string, db *gorm.DB) string {
	// 优先从mysql中取
	avg := init_meta(appid)
	item := metainfoid(avg, begin, end, stmtid)
	if item["queryid"] != nil {
		msg := fmt.Sprintf(`
💬query_id       :  %v
💬conn_id        :  %v
💬remote_ip      :  %v
💬fe_host        :  %v
💬user           :  %v
💬start_time     :  %v
💬end_time       :  %v
💬time_used      :  %v
💬state          :  %v
💬error_message  :  %v
💬sql            :  %v
💬cpu_cost_ns    :  %v
💬mem_cost_bytes :  %v
💬scan_rows      :  %v
💬scan_bytes     :  %v
💬digest         :  %v
💬database       :  %v


💬profile        :  %v


💬plan           :  %v`, item["queryid"],
			item["connid"],
			item["clientIp"],
			item["feIp"],
			item["user"],
			item["starttime"],
			item["endtime"],
			item["queryTime"],
			item["state"],
			item["errormessage"],
			item["stmt"],
			item["cpuCostNs"],
			item["memCostBytes"],
			item["scanRows"],
			item["scanBytes"],
			item["digest"],
			item["db"],
			item["profile"],
			item["plan"],
		)
		filename := fmt.Sprintf("%s/%s.%s.log", util.Read.Log.Path, appid, stmtid)
		tools.WriteFile(filename, msg)
		session := fmt.Sprintf("http://%s:%d/log/%s", util.H.Ip, util.Read.Server.Port, filename)
		return session
	}

	if util.Read.Schema.Auditops == "" {
		return ""
	}
	var m map[string]interface{}
	r := db.Raw(fmt.Sprintf("select * from %s where queryId='%s'", util.Read.Schema.Auditops, stmtid)).Scan(&m)
	if r.Error != nil {
		return ""
	}

	if m == nil {
		return ""
	}
	var pendingTimeMs int64
	v, ok := m["pendingTimeMs"]
	if ok {
		pendingTimeMs = v.(int64)
	}

	msg := fmt.Sprintf(`
💬QueryId          :         %s
💬Timestamp        :         %s
💬QueryType        :         %s
💬ClientIp         :         %s
💬User             :         %s
💬AuthorizedUser   :         %s
💬ResourceGroup    :         %s
💬Catalog          :         %s
💬Db               :         %s
💬State            :         %s
💬ErrorCode        :         %s
💬QueryTime        :         %d
💬ScanBytes        :         %d
💬ScanRows         :         %d
💬ReturnRows       :         %d
💬CpuCostNs        :         %d
💬MemCostBytes     :         %d
💬StmtId           :         %d
💬IsQuery          :         %b
💬FeIp             :         %s
💬Digest           :         %s
💬PlanCpuCosts     :         %f
💬PlanMemCosts     :         %f
💬PendingTimeMs    :         %d
💬Stmt             :         
%s
`,
		m["queryId"],
		m["timestamp"],
		m["queryType"],
		m["clientIp"],
		m["user"],
		m["authorizedUser"],
		m["resourceGroup"],
		m["catalog"],
		m["db"],
		m["state"],
		m["errorCode"],
		m["queryTime"].(int64),
		m["scanBytes"].(int64),
		m["scanRows"].(int64),
		m["returnRows"].(int64),
		m["cpuCostNs"].(int64),
		m["memCostBytes"].(int64),
		m["stmtId"].(int64),
		m["isQuery"].(int64),
		m["feIp"],
		m["digest"],
		m["planCpuCosts"].(float64),
		m["planMemCosts"].(float64),
		pendingTimeMs,
		m["stmt"],
	)

	filename := fmt.Sprintf("%s/%s.%s.log", util.Read.Log.Path, appid, stmtid)
	tools.WriteFile(filename, msg)
	session := fmt.Sprintf("http://%s:%d/log/%s", util.H.Ip, util.Read.Server.Port, filename)
	return session
}

func getConnectionId(appid, stmtid string, db *gorm.DB) string {

	id, _ := strconv.Atoi(stmtid)

	var felist []string
	var m []map[string]interface{}
	r := db.Raw("show frontends").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Warn(r.Error.Error())
		return ""
	}
	for _, item := range m {
		if item["Alive"].(string) != "true" {
			continue
		}
		felist = append(felist, item["IP"].(string))
	}

	var session string
	var global sync.WaitGroup
	for _, ip := range felist {
		global.Add(1)
		ip := ip
		go func() {
			defer global.Done()

			single, err := conn.StarRocksSingle(appid, ip)
			if err != nil {
				util.Loggrs.Warn(err.Error())
				return
			}

			var dbresult []util.Process
			r = single.Raw("show full processlist").Scan(&dbresult)
			if r.Error != nil {
				util.Loggrs.Warn(r.Error.Error())
				return
			}
			defer func() {
				/*每次使用完，主动关闭连接数*/
				sqlDB, err := single.DB()
				if err != nil {
					util.Loggrs.Error(err.Error())
					return
				}
				sqlDB.SetMaxOpenConns(10)                 //最大连接数
				sqlDB.SetMaxIdleConns(3)                  //最大空闲连接数
				sqlDB.SetConnMaxLifetime(5 * time.Second) //空闲连接最多存活时间
				sqlDB.Close()
			}()

			done := make(chan struct{}, 10)
			var wg sync.WaitGroup
			for _, m := range dbresult {
				wg.Add(1)
				m := m
				go func() {
					defer func() {
						<-done
						wg.Done()
					}()
					done <- struct{}{}

					connectionId, _ := strconv.Atoi(m.Id)
					if connectionId == id {
						filename := fmt.Sprintf("%s/%s.%s.log", util.Read.Log.Path, appid, m.Id)
						tools.WriteFile(filename, fmt.Sprintf(`
💬App:            %s
💬Fe:             %s
💬ClientIP:       %s
💬Type:           %s
💬ConnectionId:   %s
💬Database:       %s
💬User:           %s
💬ExecTime:       %s
💬Stmt:
%s`, appid, ip, m.Host, m.State, m.Id, m.Db, m.User, m.Time, m.Info))
						session = fmt.Sprintf("http://%s:%d/log/%s", util.H.Ip, util.Read.Server.Port, filename)
						return
					}
				}()
			}
			wg.Wait()
		}()
	}
	global.Wait()
	return session
}
