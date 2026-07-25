/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_get_security
 *@date    2026/4/8 15:27
 */

package app

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"net/http"
	"regexp"
	"strings"
	"sync"
)

type sepolicy struct {
	Id       int         `json:"id"`
	Database interface{} `json:"database"`
	Name     interface{} `json:"name"`
	Object   interface{} `json:"object"`
	Command  string      `json:"command"`
	Users    []string    `json:"users"`
	Roles    []string    `json:"roles"`
}

// security
func (engine *threadMap) getsecurity(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	if val, ok := apicache.Get(appid + "securities"); ok {
		securities := val.([]sepolicy)
		c.JSON(http.StatusOK, gin.H{"data": securities, "running": len(securities), "pending": 0, "total": len(securities)})
		return
	}
	securities, err := actualSecure(appid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": securities, "running": len(securities), "pending": 0, "total": len(securities)})
}

// 实时刷新
func (engine *threadMap) refreshSecure(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	securities, err := actualSecure(appid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": securities, "running": len(securities), "pending": 0, "total": len(securities)})
}

// 真实获取数据
func actualSecure(app string) (securities []sepolicy, err error) {

	app = setdefault(app)
	db, err := engine.getmapConnect(app)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return
	}
	var m []map[string]interface{}
	stl := fmt.Sprintf(`SELECT * FROM default_catalog.sys.policy_references
WHERE REF_CATALOG     NOT LIKE '%%[NULL]%%'
   AND POLICY_DATABASE NOT LIKE '%%:%%'
   AND POLICY_NAME     NOT LIKE '%%:%%'
   AND REF_DATABASE    NOT LIKE '%%[NULL]%%'
   AND REF_OBJECT_NAME NOT LIKE '%%[NULL]%%'
   AND REF_COLUMN      NOT LIKE '%%[NULL]%%'`)
	r := db.Raw(stl).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return
	}
	if m == nil {
		return nil, nil
	}
	done := make(chan struct{}, 3)
	var wg sync.WaitGroup
	for i, m2 := range m {
		wg.Add(1)

		go func(i int, m2 map[string]interface{}) {
			defer func() {
				<-done
				wg.Done()
			}()

			done <- struct{}{}

			var mask map[string]interface{}
			r := db.Raw(fmt.Sprintf("USE %s;SHOW CREATE MASKING POLICY %s", m2["POLICY_DATABASE"], m2["POLICY_NAME"])).Scan(&mask)
			if r.Error != nil {
				util.Loggrs.Error(r.Error.Error())
				return
			}
			stmt := mask["Create Policy"].(string)
			roles, users := ext2usersAndroles(stmt)
			object := fmt.Sprintf("%s.%s.%s.%s", m2["REF_CATALOG"], m2["REF_DATABASE"], m2["REF_OBJECT_NAME"], m2["REF_COLUMN"])

			securities = append(securities,
				sepolicy{
					Id:       i,
					Database: m2["POLICY_DATABASE"],
					Name:     object,
					Object:   m2["POLICY_NAME"],
					Roles:    roles,
					Users:    users,
					Command:  fmt.Sprintf(`<button id="drop-security" style="border: none; outline: none;background: transparent;" class="bi bi-power text-danger" data-security="%s"></button>`, fmt.Sprintf("%s+%s+%s", object, m2["POLICY_NAME"], m2["POLICY_DATABASE"])),
				})
		}(i, m2)
	}
	wg.Wait()

	apicache.Set(app+"securities", securities, cache.DefaultExpiration)
	return
}

func (engine *threadMap) dropsecurity(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	type objects struct {
		Objectname []string `json:"object"`
	}
	var o objects
	data, _ := c.GetRawData()
	json.Unmarshal(data, &o)

	appid = setdefault(appid)
	db, err := engine.getmapConnect(appid)
	if err != nil {
		util.Loggrs.Error(err.Error())
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	util.Loggrs.Info(o.Objectname)
	ojn := strings.Split(strings.Join(o.Objectname, ""), "+")
	policydatabase := ojn[2]
	policyname := ojn[1]
	ojname := strings.Split(ojn[0], ".")

	err = operationPolicyUnset(db, fmt.Sprintf("%s.%s.%s", ojname[0], ojname[1], ojname[2]), ojname[3])
	if err != nil {
		util.Loggrs.Error(err.Error())
	}
	err = operationPolicyDrop(db, policydatabase, policyname)
	if err != nil {
		util.Loggrs.Error(err.Error())
		c.JSON(http.StatusNotImplemented, gin.H{
			"message": fmt.Sprintf("success:%d,failed:%d %v", 0, 1, err.Error()),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": fmt.Sprintf("success:%d,failed:%d", 1, 0),
	})
}

// 对policy关联的字段进行unset释放
// @parm db数据库连接对象
// @parm schema
// @parm column
func operationPolicyUnset(db *gorm.DB, table, column string) error {
	stmt := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN `%s` UNSET MASKING POLICY", table, column)
	r := db.Exec(stmt)
	if r.Error != nil {
		util.Loggrs.Warn(stmt)
		return r.Error
	}
	util.Loggrs.Info(fmt.Sprintf("unset [%s].[%s]", table, column))
	return nil
}

// 清理某个policy
// @parm db数据库连接对象
// @parm schema
// @parm column
func operationPolicyDrop(db *gorm.DB, database, policyname string) error {
	stmt := fmt.Sprintf("USE %s;DROP MASKING POLICY %s FORCE", database, policyname)
	r := db.Exec(stmt)
	if r.Error != nil {
		util.Loggrs.Warn(stmt)
		return r.Error
	}
	util.Loggrs.Info(fmt.Sprintf("drop [%s]", policyname))
	return nil
}

// 使用正则表达式，从语句中分析出哪些是已经解密的用户和角色
// @avgs stmt sql语句
// result [][]slice 切片
func ext2usersAndroles(sql string) (roles []string, users []string) {
	// 正则表达式匹配 CURRENT_ROLE() 中的角色列表
	roleMatches := regexp.MustCompile(`(?i)find_in_set\s*\(\s*'([^']+)'\s*,\s*replace\s*\(\s*current_role\s*\(\s*\)\s*,\s*'[^']+'\s*,\s*'[^']*'\s*\)\s*\)`).FindAllStringSubmatch(sql, -1)
	for _, match := range roleMatches {
		if len(match) > 1 {
			roles = append(roles, match[1])
		}
	}
	// 正则表达式匹配 CURRENT_USER() 中的用户列表
	userMatches := regexp.MustCompile(`NOT IN\s*\(\s*((?:'[^']+'(?:\s*,\s*'[^']+')*))\s*\)`).FindAllStringSubmatch(sql, -1)
	for _, match := range userMatches {
		if len(match) > 1 {
			// 提取整个用户列表字符串，然后按逗号分割
			userListStr := match[1]
			// 移除单引号并按逗号分割
			userList := strings.Split(userListStr, ",")
			for _, user := range userList {
				user = strings.TrimSpace(user)
				user = strings.Trim(user, "'") // 移除单引号
				if user != "" {
					users = append(users, user)
				}
			}
		}
	}
	// 去重
	roles = tools.RmDuplicaSlice(roles)
	users = tools.RmDuplicaSlice(users)
	return
}
