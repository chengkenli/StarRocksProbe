/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    assistance
 *@date    2026/3/11 10:59
 */

package app

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"encoding/json"
	"fmt"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

func assistance() {
	if util.Read.Server.Loadhtmlglob == "" {
		return
	}
	r := gin.Default()
	// 允许所有域（仅用于开发）
	r.Use(cors.Default())
	// 加载HTML模板
	r.LoadHTMLGlob(strings.NewReplacer("html/*.html", "assistance/*.html").Replace(util.Read.Server.Loadhtmlglob))
	r.Static("/static", strings.NewReplacer("html/*.html", "assistance").Replace(util.Read.Server.Loadhtmlglob))
	// 定义路由
	r.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})
	r.POST("/assis_user", assistanceUser)
	r.POST("/assis_regex", assistanceRegex)

	err := r.Run(fmt.Sprintf(":%d", util.Read.Server.Port+30))
	if err != nil {
		util.Loggrs.Error(err.Error())
		return
	}
}

// assistance_user
func assistanceUser(c *gin.Context) {
	go func() {
		msg := fmt.Sprintf("%s 来自starrocks assistance的权限自查。", c.ClientIP())
		util.Loggrs.Info(msg)
	}()

	var id map[string]string
	data, _ := c.GetRawData()
	json.Unmarshal(data, &id)
	uri := id["uri"]

	decoded, err := url.QueryUnescape(uri)
	if err != nil {
		util.Loggrs.Error(err.Error())
		c.JSON(http.StatusPaymentRequired, gin.H{"url": "nil", "message": err.Error()})
	}
	if keyRegex(decoded) {
		c.JSON(http.StatusPaymentRequired, gin.H{"url": "nil", "message": "检测到输入的信息不是一个标准的用户名！"})
		return
	}
	//发送POST请求并处理响应
	response, err := resty.New().R().
		SetHeader("Content-Type", "application/json;charset=utf-8").
		Get(uri)
	if err != nil {
		util.Loggrs.Errorf("%v %v", err.Error(), string(response.Body()))
		c.JSON(http.StatusPaymentRequired, gin.H{"url": "nil", "message": err.Error()})
		return
	}
	util.Loggrs.Info(string(response.Body()))
	if string(response.Body()) == "" {
		c.JSON(http.StatusPaymentRequired, gin.H{"url": "nil", "message": "账号不存在！"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": strings.NewReplacer(`"`, "").Replace(string(response.Body())), "message": "nil"})
}

func assistanceRegex(c *gin.Context) {
	go func() {
		msg := fmt.Sprintf("%s 来自starrocks assistance的库表提取。", c.ClientIP())
		util.Loggrs.Info(msg)
	}()
	var id map[string]string
	data, _ := c.GetRawData()
	err := json.Unmarshal(data, &id)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return
	}
	//schema := brpc.AgentSchemaItem(id["context"])
	schema, err := schemaRegexp(id["context"])
	if err != nil {
		util.Loggrs.Error(err.Error())
		return
	}
	util.Loggrs.Info(len(schema))
	if len(schema) == 0 {
		c.JSON(http.StatusOK, "只有遵循标准<库名.表名>的语法才可解析成功！")
		return
	}
	util.Loggrs.Info(schema)
	c.JSON(http.StatusOK, schema)
}

func keyRegex(input string) bool {
	// 使用单词边界匹配，避免匹配到包含关键字的其他单词
	pattern := `\b(select|insert|update|delete|alter|create|drop)\b`
	re := regexp.MustCompile(`(?i)` + pattern) // (?i) 表示不区分大小写

	return re.MatchString(input)
}

func schemaRegexp(input string) ([]string, error) {
	database := []string{
		"adhoc", "ads", "ads_dev", "ads_dev_secure", "ads_rt", "ads_rt_dev", "ads_rt_dev_secure", "ads_rt_secure", "ads_secure",
		"algo", "algo_dev", "ap_secure", "audit", "bi_item", "bi_realty", "bi_realty_secure", "bi_sams_secure", "bi_scm", "bi_sc_secure",
		"cdp", "cdp_api", "cloud_fcst_dm", "cn_backup_secure", "cn_chilled_data", "cn_core_dim_vm", "cn_di_data", "cn_ec_bi_secure",
		"cn_ec_wmdj_user_action", "cn_mdse_dm_dl_tables", "cn_po_home_system", "cn_po_home_system_dev", "cn_pricing_dl_tables",
		"cn_sams_dl_secure", "cn_wc_highsecure", "cn_wc_mb_secure", "cn_wc_mb_vm", "cn_wc_repl_vm", "cn_wc_vm", "cn_wid_dl_secure",
		"cn_wm_mb_secure", "cn_wm_mb_vm", "cn_wm_repl_vm", "cn_wm_vm", "data_test", "demo", "dim", "dim_dev", "dim_dev_secure", "dim_rt",
		"dim_rt_dev", "dim_rt_dev_secure", "dim_rt_secure", "dim_secure", "dm", "dm_dev", "dm_dev_secure", "dm_secure", "dw", "dwd", "dwd_dev",
		"dwd_dev_secure", "dw_dev", "dw_dev_secure", "dwd_rt", "dwd_rt_dev", "dwd_rt_dev_secure", "dwd_rt_secure", "dwd_secure", "dw_rt",
		"dws", "dws_dev", "dws_dev_secure", "dw_secure", "dws_rt", "dws_rt_dev", "dws_rt_dev_secure", "dws_rt_secure", "dws_secure",
		"euclid_scn_forecast_prod", "finance_kettle", "fin_sox", "fin_sox_dev", "flash_report", "flash_report_dev", "flash_report_sit",
		"hyper_bi_secure", "hyper_ec_secure", "hyper_mdse_dm_secure", "information_schema", "ma_test", "mbrship_secure", "mcfc_report",
		"mcfc_report_dev", "o2o_datacubes_secure", "ods", "ods_app_dev", "ods_app_dev_secure", "ods_app_test", "ods_app_test_secure",
		"ods_archive", "ods_dev", "ods_dev_secure", "ods_gray", "ods_migration_td_gray", "ods_rt", "ods_rt_dev", "ods_rt_dev_secure",
		"ods_rt_secure", "ods_secure", "ods_secure_rt", "ods_sox", "ods_sox_app_dev", "ods_sox_app_test", "ods_sox_dev", "ods_sox_test",
		"ods_test", "ods_test_secure", "ops", "pro_dgtmkt_data", "pro_scct_dev", "sams_finance", "scct_inv_monitor", "scct_logis", "scct_logis_dev",
		"scm_dcqe_secure", "scm_network_secure", "scm_secure", "scm_uihealth", "starrocks_monitor", "_statistics_", "supply_kettle", "svccn_logis",
		"svccn_logis_query", "svcdordgtmkt", "sys", "wm_ad_hoc", "wm_cn_util", "wm_common_vm", "ww_core_dim_vm"}

	var schema []string

	// 更宽松的正则表达式，支持数据库标识符（字母、数字、下划线）
	re := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\.([a-zA-Z_][a-zA-Z0-9_]*)`)

	// 查找所有匹配的 schema.table
	matches := re.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		if len(match) >= 3 {
			schemaName := match[1]
			tableName := match[2]
			fullName := schemaName + "." + tableName

			// 检查特殊字符
			b := regexp.MustCompile(`[\\/\(\),:|+><~!@#%^&*='";?-]`).FindString(fullName) != ""
			if !b {
				// 检查schema是否在database列表中
				for _, db := range database {
					if schemaName == db {
						schema = append(schema, fullName)
						break
					}
				}
			}
		}
	}

	// catalog.schema.table 模式
	re2 := regexp.MustCompile(`([a-zA-Z_][a-zA-Z0-9_]*)\.([a-zA-Z_][a-zA-Z0-9_]*)\.([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches2 := re2.FindAllStringSubmatch(input, -1)

	for _, match := range matches2 {
		if len(match) >= 4 {
			catalogName := match[1]
			schemaName := match[2]
			tableName := match[3]
			fullName := catalogName + "." + schemaName + "." + tableName

			// 检查特殊字符
			b := regexp.MustCompile(`[\\/\(\),:|+><~!@#%^&*='";?-]`).FindString(fullName) != ""
			if !b {
				// 检查schema是否在database列表中
				for _, db := range database {
					if schemaName == db {
						schema = append(schema, fullName)
						break
					}
				}
			}
		}
	}

	return tools.RmDuplicaSlice(schema), nil
}
