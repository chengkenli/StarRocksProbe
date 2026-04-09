/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    metri_get_security
 *@date    2026/4/8 15:27
 */

package app

import (
	"StarRocksProbe/util"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"net/http"
	"regexp"
	"strings"
)

type sepolicy struct {
	Id       int         `json:"id"`
	Database interface{} `json:"database"`
	Name     interface{} `json:"name"`
	Object   interface{} `json:"object"`
	User     interface{} `json:"user"`
	Tag      interface{} `json:"tag"`
	Command  string      `json:"command"`
}

// security
func (engine *threadMap) getsecurity(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	if val, ok := lastcache.Get(appid + "securities"); ok {
		securities := val.([]sepolicy)
		c.JSON(http.StatusOK, gin.H{"data": securities, "running": len(securities), "pending": 0, "total": len(securities)})
		return
	}
	db, err := engine.getmapConnect(appid)
	if err != nil {
		util.Loggrs.Error(err.Error())
		c.JSON(http.StatusInternalServerError, err.Error())
		return
	}
	var m []map[string]interface{}
	r := db.Raw("select * from default_catalog.sys.policy_references").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		c.JSON(http.StatusInternalServerError, r.Error)
		return
	}
	var securities []sepolicy
	for i, m2 := range m {
		var mask []map[string]interface{}
		r := db.Raw(fmt.Sprintf("USE %s;SHOW CREATE MASKING POLICY %s", m2["POLICY_DATABASE"], m2["POLICY_NAME"])).Scan(&mask)
		if r.Error != nil {
			util.Loggrs.Error(r.Error.Error())
			continue
		}
		var createPolicy string
		var usernames []string
		for _, m3 := range mask {
			createPolicy = m3["Create Policy"].(string)
			matches := regexp.MustCompile(`\\'([^\\']+)\\'@\\'%\\'`).FindAllStringSubmatch(m3["Create Policy"].(string), -1)
			for _, match := range matches {
				if len(match) > 1 {
					usernames = append(usernames, match[1])
				}
			}
			// 方法2：精确匹配IN列表（更健壮）
			match2 := regexp.MustCompile(`IN\s*\(([^)]+)\)`).FindStringSubmatch(m3["Create Policy"].(string))
			if len(match2) > 1 {
				// 提取括号内的内容并分割
				for _, user := range strings.Split(match2[1], ",") {
					// 去除空格和单引号
					cleanUser := strings.Trim(strings.TrimSpace(user), "'")
					usernames = append(usernames, cleanUser)
				}
			}
		}

		var tag string
		if strings.Contains(createPolicy, "NOT IN") {
			tag = "non-restrictive"
		} else {
			tag = "restrictive"
		}
		object := fmt.Sprintf("%s.%s.%s.%s", m2["REF_CATALOG"], m2["REF_DATABASE"], m2["REF_OBJECT_NAME"], m2["REF_COLUMN"])

		securities = append(securities,
			sepolicy{
				Id:       i,
				Database: m2["POLICY_DATABASE"],
				Name:     m2["POLICY_NAME"],
				Object:   object,
				User:     usernames,
				Tag:      tag,
				Command:  fmt.Sprintf(`<button id="drop-security" class="btn btn-light btn-sm ms-2" data-security="%s">❌</button>`, fmt.Sprintf("%s+%s+%s", object, m2["POLICY_NAME"], m2["POLICY_DATABASE"])),
			})
	}
	lastcache.Set(appid+"securities", securities, cache.DefaultExpiration)
	c.JSON(http.StatusOK, gin.H{"data": securities, "running": len(securities), "pending": 0, "total": len(securities)})
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
	policydatabase := strings.Split(strings.Join(o.Objectname, ""), "+")[2]
	policyname := strings.Split(strings.Join(o.Objectname, ""), "+")[1]
	ojname := strings.Split(strings.Split(strings.Join(o.Objectname, ""), "+")[0], ".")
	if len(ojname) == 4 {
		catalog := ojname[0]
		database := ojname[1]
		table := ojname[2]
		column := ojname[3]

		err := operationPolicyUnset(db, fmt.Sprintf("%s.%s.%s", catalog, database, table), column)
		if err != nil {
			util.Loggrs.Error(err.Error())
			c.JSON(http.StatusNotImplemented, gin.H{
				"message": fmt.Sprintf("success:%d,failed:%d %v", 0, 1, err.Error()),
			})
			return
		}
		err = operationPolicyDrop(db, policydatabase, policyname)
		if err != nil {
			util.Loggrs.Error(err.Error())
			c.JSON(http.StatusNotImplemented, gin.H{
				"message": fmt.Sprintf("success:%d,failed:%d %v", 0, 1, err.Error()),
			})
			return
		}
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
	stmt := fmt.Sprintf("ALTER TABLE %s MODIFY COLUMN %s UNSET MASKING POLICY", table, column)
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
	stmt := fmt.Sprintf("USE %s;DROP MASKING POLICY %s", database, policyname)
	r := db.Exec(stmt)
	if r.Error != nil {
		util.Loggrs.Warn(stmt)
		return r.Error
	}
	util.Loggrs.Info(fmt.Sprintf("drop [%s]", policyname))
	return nil
}
