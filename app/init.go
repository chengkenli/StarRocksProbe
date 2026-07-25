/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    init
 *@date    2025/5/22 15:54
 */

package app

import (
	"StarRocksProbe/ins"
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
	"strconv"
	"strings"
	"sync"
	"time"
)

var escache = cache.New(60*time.Second, 60*time.Second)
var querycache = cache.New(5*time.Second, 10*time.Second)
var cachelimit = cache.New(5*time.Second, 10*time.Second)
var shortcache = cache.New(5*time.Minute, 10*time.Minute)
var apicache = cache.New(12*time.Hour, 24*time.Hour)
var errcache = cache.New(1*time.Hour, 1*time.Hour)
var lastcache = cache.New(30*time.Minute, 30*time.Minute)
var second30cache = cache.New(30*time.Second, 30*time.Second)
var second20cache = cache.New(20*time.Second, 20*time.Second)
var engine *threadMap

func init() {
	// 初始化对象引擎
	engine = newPool()
	engine._init()  //初始化连接对象
	engine._init2() //初始化加载mapreduce_ip缓存
	engine._init4() //初始化加载副本元数据面板
	engine._init6() //初始化加载副本元数据面板2
	engine._init7() //初始化加载information_schema.tables
	engine._init8() //初始化定期加载巡检报告
	engine._init9() //初始化加载资源组
}

func (engine *threadMap) _init9() {
	for _, m := range util.MetaLink {
		app := m["app"].(string)
		db, err := engine.getmapConnect(app)
		if err != nil {
			util.Loggrs.Error(err.Error())
			return
		}
		go ResourceClassifiers(app, db)
	}
}

func (engine *threadMap) _init8() {
	go func() {
		// 主架构方式
		crontab := cron.New()
		// 添加定时任务, * * * * * 是 crontab,表示每分钟执行一次
		_, err := crontab.AddFunc("10 0 * * *", func() {
			for _, m := range util.MetaLink {
				app := m["app"].(string)
				db, err := engine.getmapConnect(app)
				if err != nil {
					util.Loggrs.Error(err.Error())
					return
				}
				go ins.Ins(app, db)
				go ResourceClassifiers(app, db)
			}
		})
		if err != nil {
			util.Loggrs.Error(err.Error())
			return
		}
		// 启动定时器
		crontab.Start()
		// 定时任务是另起协程执行的,这里使用 select 简答阻塞.实际开发中需要
		// 根据实际情况进行控制
		select {}
	}()
}

func (engine *threadMap) _init7() {
	go func() {
		time.Sleep(time.Second * 5)
		meta_data()
		ticker := time.NewTicker(time.Hour * 2)
		for {
			select {
			case <-ticker.C:
				meta_data()
			}
		}
	}()
}

func (engine *threadMap) _init6() {
	go func() {
		time.Sleep(time.Second * 5)
		meta_load()
		ticker := time.NewTicker(time.Minute * 30)
		for {
			select {
			case <-ticker.C:
				meta_load()
			}
		}
	}()
}

func (engine *threadMap) _init2() {
	if util.Read.Schema.Ipsystem == "" {
		return
	}
	app := _app()
	if app == "" {
		return
	}
	go func() {
		if util.ClientIPDec == nil {
			db, err := engine.getmapConnect(app)
			if err != nil {
				util.Loggrs.Error(err.Error())
				util.Loggrs.Fatalf(fmt.Sprintf("校验失败！配置文件中的主集群 ipapp:%s 在元数据表%s 中匹配不到连接信息，请确保集群名称一致！", app, util.Read.Metadb.Base))
				return
			}
			getsign(db)
		}

		ticker := time.NewTicker(time.Hour * 5)
		for {
			select {
			case <-ticker.C:
				db, err := engine.getmapConnect(app)
				if err != nil {
					util.Loggrs.Error(err.Error())
					return
				}
				getsign(db)
			}
		}
	}()
}

func getsign(db *gorm.DB) {
	if util.Read.Schema.Ipsystem == "" {
		return
	}
	r := db.Raw(fmt.Sprintf("select * from %s where ts >= DATE(DATE_SUB(NOW(), INTERVAL 5 DAY))", util.Read.Schema.Ipsystem)).Scan(&util.ClientIPDec)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return
	}
	util.Loggrs.Info("[ok] 初始化加载ipsystem缓存", len(util.ClientIPDec))

	if util.Read.Schema.Emrwedat == "" {
		return
	}
	r = db.Raw("select * from " + util.Read.Schema.Emrwedat).Scan(&util.ClientMapReduce)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return
	}
	util.Loggrs.Info("[ok] 初始化加载mapreduce_ip缓存", len(util.ClientMapReduce))
}

func getipname(clientip string, deviceName ...string) string {
	for _, data := range util.ClientIPDec {
		var devicename string
		if deviceName != nil {
			sign := strings.Split(deviceName[0], ".")
			if len(sign) >= 1 {
				devicename = sign[0]
			}
		}
		if data.ComputerName == devicename {
			//return fmt.Sprintf("%s (%s)(%s) %s(%s)", data.UserName, data.ComputerType, data.ComputerStatus, data.ComputerVersion, data.Brand)
			return data.UserName
		} else if clientip == util.H.Ip {
			return fmt.Sprintf("infra")
		} else if data.IpAddress == clientip {
			//return fmt.Sprintf("%s (%s)(%s) %s(%s)", data.UserName, data.ComputerType, data.ComputerStatus, data.ComputerVersion, data.Brand)
			return data.UserName
		} else if tools.StrInSlice(clientip, util.Read.Schema.Gybi) {
			return fmt.Sprintf("观远bi")
		} else if tools.StrInSlice(devicename, util.Read.Schema.Gybi) {
			return fmt.Sprintf("观远bi")
		}
	}

	for _, data := range util.ClientMapReduce {
		if data.Ip == clientip {
			return fmt.Sprintf("%s(%s)", strings.ToLower(data.Comment), data.Dept)
		}
	}

	return ""
}

func leader(db *gorm.DB) string {
	var m []map[string]interface{}
	r := db.Raw("show frontends").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Warn(r.Error.Error())
		return ""
	}
	var leaderip string
	for _, item := range m {
		if item["Alive"].(string) != "true" {
			continue
		}
		if item["Role"].(string) == "LEADER" {
			leaderip = item["IP"].(string)
			break
		}
	}
	return leaderip
}

// 假如前端传过来一个默认值，那么给他返回第一个集群
func setdefault(app string) string {
	var reals string
	var applist []string
	if app == "sr-default" {
		for _, m := range util.MetaLink {
			applist = append(applist, m["app"].(string))
		}
		reals = applist[0]
	} else {
		reals = app
	}
	return reals
}

func gettitle(db *gorm.DB, app string) string {
	if val, ok := apicache.Get(app + "title"); ok {
		return val.(string)
	}
	var nick string
	for _, m := range util.MetaLink {
		if m["app"].(string) == app {
			nick = m["nickname"].(string)
		}
	}
	var (
		f util.Fronends
		b util.Backends
		e util.Computes
	)
	db.Raw("show frontends").Scan(&f)

	var backends int
	db.Raw("show backends").Scan(&b)
	backends = len(b)
	if backends == 0 {
		db.Raw("show compute nodes").Scan(&e)
		backends = len(e)
	}
	lastcache.Set(app+"numRun", b, cache.DefaultExpiration)
	title := fmt.Sprintf("%s FE:(%d) BE:(%d) VERSION:(%v)", nick, len(f), backends, tools.Version(app, db))
	go apicache.Set(app+"title", title, cache.DefaultExpiration)
	return title
}

func storage(sr string, infos util.Backends) string {
	var f, g, h, k float64
	for _, info := range infos {
		f = f + flos(info.TotalCapacity)
		k = k + flos(info.DataUsedCapacity)
		g = g + flos(info.AvailCapacity)
	}
	h = f - g
	return fmt.Sprintf("%s总存储:%0.2ftb, 目前存储:%0.2ftb(数据实际:%0.2ftb), 空闲:%0.2ftb, 百分比:%0.2f%%", sr, f, h, k, f-h, h/f*100)
}

func flos(s string) float64 {
	var maxDiskUsedPct float64
	if strings.Contains(strings.ToLower(strings.ReplaceAll(s, " ", "")), "gb") {
		m := strings.Split(s, " ")[0]
		maxDiskUsedPct, _ = strconv.ParseFloat(m, 64)
		maxDiskUsedPct = maxDiskUsedPct / 1024
		return maxDiskUsedPct
	}
	m := strings.Split(s, " ")[0]
	maxDiskUsedPct, _ = strconv.ParseFloat(m, 64)
	return maxDiskUsedPct
}

func fooTime(timeStr string, hour int) string {
	// 定义时间字符串的格式
	layout := "2006-01-02 15:04:05"
	// 解析时间字符串
	t, err := time.Parse(layout, timeStr)
	if err != nil {
		util.Loggrs.Error("解析时间出错:", err)
		return ""
	}
	// 计算8小时前的时间
	eightHoursBefore := t.Add(time.Duration(hour) * time.Hour)
	// 输出结果
	return eightHoursBefore.Format(layout)
}

func (engine *threadMap) _init4() {
	go func() {
		ticker := time.NewTicker(time.Hour * 3)
		for {
			select {
			case <-ticker.C:
				grafana()
			}
		}
	}()
}

func grafana() {
	go func() {
		var wg sync.WaitGroup
		for _, m := range util.MetaLink {
			wg.Add(1)
			m := m
			go func() {
				defer wg.Done()

				app := m["app"].(string)

				_, t_ok := apicache.Get(app + "TabletNum")
				_, r_ok := apicache.Get(app + "ReplicaNum")

				if !t_ok || !r_ok {
					db, err := engine.getmapConnect(app)
					if err != nil {
						util.Loggrs.Error(err.Error())
						return
					}
					var staic []map[string]interface{}
					r := db.Raw("show proc '/statistic'").Scan(&staic)
					if r.Error != nil {
						util.Loggrs.Error(r.Error.Error())
						return
					}
					util.Lastcache.Set(app+"statistic", staic, cache.DefaultExpiration)
					for _, m2 := range staic {
						dbid := m2["DbId"].(string)
						if dbid != "Total" {
							continue
						}
						apicache.Set(app+"DbName", m2["DbName"].(string), cache.DefaultExpiration)
						apicache.Set(app+"TableNum", m2["TableNum"].(string), cache.DefaultExpiration)
						apicache.Set(app+"PartitionNum", m2["PartitionNum"].(string), cache.DefaultExpiration)
						apicache.Set(app+"TabletNum", m2["TabletNum"].(string), cache.DefaultExpiration)
						apicache.Set(app+"ReplicaNum", m2["ReplicaNum"].(string), cache.DefaultExpiration)
						apicache.Set(app+"UnhealthyTabletNum", m2["UnhealthyTabletNum"].(string), cache.DefaultExpiration)
						break
					}
					util.Loggrs.Info("reload statistic ", app)
				}
			}()
		}
		wg.Wait()
	}()
}

// 判断是不是首次访问
// @result true=首次访问
// @result false=不是首次访问
func firstVisitHandler(c *gin.Context) bool {
	session := sessions.Default(c)
	if session.Get("visited") == nil {
		session.Set("visited", true)
		// 设置 Session 持久化（例如 1 天有效期）
		session.Options(sessions.Options{
			MaxAge:   86400 * 1, // 单位：秒
			HttpOnly: true,      // 防止 XSS
			Secure:   false,     // HTTPS 下启用
		})
		if err := session.Save(); err != nil { // 必须调用 Save
			util.Loggrs.Error("保存Session失败:", err)
			return false
		}
		return true
	}
	return false
}

// 提前计算倾斜数据和变异数据
// @parm nil
// @result nil
// 判断好直接交给cache管理
func meta_load() {
	for _, m := range util.MetaLink {
		app := m["app"].(string)
		db, err := engine.getmapConnect(app)
		if err != nil {
			util.Loggrs.Error(err.Error())
			return
		}
		r, _ := meta_replica(app, db)
		apicache.Set(app+"replicadata", r, cache.DefaultExpiration)

		t, _ := meta_tablet(app, db)
		apicache.Set(app+"metadata", t, cache.DefaultExpiration)
	}
}

// 获取information_schema.tables的数据保存落地
// @parm nil
// @result nil
// 判断好直接交给cache管理
func meta_data() {
	var wg sync.WaitGroup
	for _, m := range util.MetaLink {
		wg.Add(1)
		m := m
		go func() {
			defer wg.Done()

			app := m["app"].(string)
			db, err := engine.getmapConnect(app)
			if err != nil {
				util.Loggrs.Error(err.Error())
				return
			}
			var m []map[string]interface{}
			r := db.Raw("SELECT TABLE_CATALOG,TABLE_SCHEMA,TABLE_NAME,TABLE_TYPE,ENGINE,VERSION,ROW_FORMAT,TABLE_ROWS,AVG_ROW_LENGTH,DATA_LENGTH,MAX_DATA_LENGTH,INDEX_LENGTH,DATA_FREE,AUTO_INCREMENT,CREATE_TIME,UPDATE_TIME,CHECK_TIME,TABLE_COLLATION,CHECKSUM,CREATE_OPTIONS,TABLE_COMMENT FROM information_schema.tables").Scan(&m)
			if r.Error != nil {
				util.Loggrs.Error(r.Error.Error())
			}
			util.Loggrs.Info(fmt.Sprintf("reload information_schema %s:%d", app, len(m)))
			apicache.Set(app+"information_schema_tables", m, cache.DefaultExpiration)
		}()
	}
	wg.Wait()
}

// 根据表名从结果集中，拿到对应的表信息
// @parm []map[string]interface{}
// @parm 表名
// @result map[string]interface{}
func find_meta(data []map[string]interface{}, tablename string) map[string]interface{} {
	tag := strings.Split(tablename, ".")
	for _, item := range data {
		if item["TABLE_SCHEMA"] == tag[0] && item["TABLE_NAME"] == tag[1] {
			return item
		}
	}
	return nil
}
