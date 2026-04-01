/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    api_userpro
 *@date    2026/1/23 9:27
 */

package app

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"
)

var onces sync.Once

func pluginuserApi(c *gin.Context) {
	app, _ := c.GetQuery("app")
	user, _ := c.GetQuery("user")
	if app == "" || user == "" {
		c.JSON(http.StatusAccepted, gin.H{"message": "集群名称或用户名不允许空值！"})
		return
	}
	url := pluginGrants(app, user)
	c.JSON(http.StatusOK, url)
}

func pluginGrants(app, user string) string {
	db, err := engine.getmapConnect(app)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return ""
	}
	d, fail := userapi(db, app, user)
	if fail != nil {
		util.Loggrs.Error(fail.Error())
		return ""
	}
	marshal, _ := json.Marshal(d)
	data := modelData{
		JsonData:   string(marshal),
		App:        app,
		User:       user,
		Authplugin: d.Authplugin,
		Ts:         trbody(d.Schema),
		Css:        globalcss(),
	}
	dir := strings.NewReplacer("*.html", "properties").Replace(util.Read.Server.Loadhtmlglob)
	onces.Do(func() {
		os.MkdirAll(dir, 0755)
	})
	openfile := fmt.Sprintf("%s/%s.html", dir, user)
	tools.WriteFile(openfile, mode0(data))

	return fmt.Sprintf("http://%s:%d/html%s", util.H.Ip, util.Read.Server.Port, openfile)
}

func trbody(schema apiSchema) []trs {
	// 获取tr数据
	var ts []trs
	if schema.Database != nil {
		for _, item := range schema.Database {
			tag := strings.Split(item, "|")
			ts = append(ts, trs{
				Level:       "Database",
				Catalog:     tag[0],
				Tablename:   tag[1],
				Permissions: strings.Split(tag[2], ","),
			})
		}
	}
	if schema.Table != nil {
		for _, item := range schema.Table {
			tag := strings.Split(item, "|")
			ts = append(ts, trs{
				Level:       "Table",
				Catalog:     tag[0],
				Tablename:   tag[1],
				Permissions: strings.Split(tag[2], ","),
			})
		}
	}
	if schema.Materialized != nil {
		for _, item := range schema.Materialized {
			tag := strings.Split(item, "|")
			ts = append(ts, trs{
				Level:       "Materialized View",
				Catalog:     tag[0],
				Tablename:   tag[1],
				Permissions: strings.Split(tag[2], ","),
			})
		}
	}
	if schema.View != nil {
		for _, item := range schema.View {
			tag := strings.Split(item, "|")
			ts = append(ts, trs{
				Level:       "View",
				Catalog:     tag[0],
				Tablename:   tag[1],
				Permissions: strings.Split(tag[2], ","),
			})
		}
	}

	return ts
}

type modelData struct {
	JsonData   string
	App        string
	User       string
	Authplugin string
	Ts         []trs
	Css        string
}

func mode0(t modelData) string {
	return fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>星辰守望 - StarRocks小助手</title>
    %v
</head>
<body>
    <div class="container">
        <header>
            <h1>星辰守望 - StarRocks小助手</h1>
            <p class="description">此页面仅展示%s在StarRocks集群详细的权限信息</p>
        </header>
		<div class="watermark-container" id="watermark"></div>
        
        <div class="json-container">
            <div class="json-viewer">
                <h2 class="json-title">原始数据</h2>
                <pre id="json-data"></pre>
            </div>
            
            <div class="data-breakdown">
                <h2 class="breakdown-title">验证模块</h2>
                <div id="data-breakdown-content">
                    <!-- 数据解析内容将通过JavaScript动态生成 -->
                </div>
            </div>
        </div>
        
        %v
        
        <footer>
            <p>InfraData StarRocks&copy;2026</p>
        </footer>
    </div>

	<script>
        // 动态创建重复水印
        function createWatermark() {
            const container = document.getElementById('watermark');
            const text = '%s';
            const spacing = 200; // 水印间距
            
            for (let x = 0; x < window.innerWidth; x += spacing) {
                for (let y = 0; y < window.innerHeight; y += spacing) {
                    const watermark = document.createElement('div');
                    watermark.className = 'watermark';
                    watermark.textContent = text;
                    watermark.style.left = x + 'px';
                    watermark.style.top = y + 'px';
                    container.appendChild(watermark);
                }
            }
        }
        createWatermark();
    </script>

    <script>
    %v

		// 搜索功能
    	document.querySelector('.search-box').addEventListener('input', function(e) {
    	    const searchTerm = e.target.value.toLowerCase();
    	    const rows = document.querySelectorAll('.permissions-table tbody tr');
    	    
    	    rows.forEach(row => {
				const catalogName = row.cells[1].textContent.toLowerCase();
    	        const tableName = row.cells[2].textContent.toLowerCase();
    	        const permission = row.cells[3].textContent.toLowerCase();
    	        
    	        if (tableName.includes(searchTerm) || permission.includes(searchTerm) || catalogName.includes(searchTerm)) {
    	            row.style.display = '';
    	        } else {
    	            row.style.display = 'none';
    	        }
    	    });
    	});
    </script>
</body>
</html>`, t.Css, t.User, modeTable(tdbody(t.Ts)), t.User, mode1(t))
}

func mode1(t modelData) string {
	return fmt.Sprintf(`        // JSON数据
        const jsonData = %v;

        // 显示原始JSON数据
        document.getElementById('json-data').textContent = JSON.stringify(jsonData, null, 2);

        // 生成数据解析内容
        const breakdownContent = document.getElementById('data-breakdown-content');
        
        // 添加App信息
        const appItem = document.createElement('div');
        appItem.className = 'data-item';
        appItem.innerHTML = `+"`"+`
            <div class="data-key">StarRocks</div>
            <div class="data-value">%s</div>
            <div class="data-description"></div>
        `+"`"+`;
        breakdownContent.appendChild(appItem);
        
        // 添加用户信息
        const userItem = document.createElement('div');
        userItem.className = 'data-item';
        userItem.innerHTML = `+"`"+`
            <div class="data-key">访问账号</div>
            <div class="data-value">%s</div>
            <div class="data-description"></div>
        `+"`"+`;
        breakdownContent.appendChild(userItem);
        
        // 添加认证插件信息
        const authItem = document.createElement('div');
        authItem.className = 'data-item';
        authItem.innerHTML = `+"`"+`
            <div class="data-key">认证插件</div>
            <div class="data-value">%s</div>
            <div class="data-description"></div>
        `+"`"+`;
        breakdownContent.appendChild(authItem);`, t.JsonData, t.App, t.User, t.Authplugin)
}

func tdbody(ts []trs) string {
	var tb []string
	for _, t := range ts {
		var p []string
		for _, permission := range t.Permissions {
			var colorl string
			switch strings.ToLower(permission) {
			case "create table":
				colorl = `style="background-color: #e3f2fd; color: #1565c0;"`
			case "create view":
				colorl = `style="background-color: #e8f5e9; color: #2e7d32;"`
			case "create function":
				colorl = `style="background-color: #fff3e0; color: #ef6c00;"`
			case "create materialized view":
				colorl = `style="background-color: #f3e5f5; color: #7b1fa2;"`
			case "alter":
				colorl = `style="background-color: #e0f2f1; color: #00897b;"`
			case "drop":
				colorl = `style="background-color: #ffebee; color: #c62828;"`
			case "select":
				colorl = `style="background-color: #fff8e1; color: #ff8f00;"`
			case "insert":
				colorl = `style="background-color: #e8eaf6; color: #303f9f;"`
			case "export":
				colorl = `style="background-color: #fce4ec; color: #ad1457;"`
			case "update":
				colorl = `style="background-color: #f1f8e9; color: #689f38;"`
			case "delete":
				colorl = `style="background-color: #fffde7; color: #f57f17;"`
			case "refresh":
				colorl = `style="background-color: #e0f7fa; color: #0097a7;"`
			default:
				colorl = `style="background-color: #f1f8e9; color: #0097a7;"`
			}
			p = append(p, fmt.Sprintf(`<span class="permission-badge" %v>%s</span>`, colorl, permission))
		}

		tb = append(tb, fmt.Sprintf(`
<tr>
	<td>%s</td>
    <td>%s</td>
    <td>%s</td>
    <td>%s</td>
</tr>`, t.Level, t.Catalog, t.Tablename, strings.Join(p, ",")))
	}
	return strings.Join(tb, "\n")
}

func modeTable(trbody string) string {
	return fmt.Sprintf(`
<div class="schema-section">
            <h2 class="schema-title">权限信息</h2>
            <div class="table-header">
                <h3>库、表、视图权限详情</h3>
                <input type="text" class="search-box" placeholder="搜索表名或权限...">
            </div>
            <div id="schema-details">
				<div class="table-container">
					<table class="permissions-table">
						<thead>
							<tr>
								<th>Level</th>
								<th>Catalog</th>
								<th>Table</th>
								<th>Permission</th>
							</tr>
						</thead>
						<tbody>
							%v
						</tbody>
					</table>
				</div>
            </div>
        </div>`, trbody)
}

func globalcss() string {
	return fmt.Sprintf(`
<style>
* {
    margin: 0;
    padding: 0;
    box-sizing: border-box;
    font-family: 'Segoe UI', Tahoma, Geneva, Verdana, sans-serif;
}

body {
    background: linear-gradient(135deg, #f5f7fa 0%%, #c3cfe2 100%%);
    min-height: 100vh;
    padding: 20px;
    display: flex;
    flex-direction: column;
    align-items: center;
}

.container {
    width: 90%%;
    /*max-width: 1200px;*/
    margin: 0 auto;
}

header {
    text-align: center;
    margin-bottom: 30px;
    padding: 20px;
    background: white;
    border-radius: 10px;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
}

.watermark-container {
    position: fixed;
    top: 0;
    left: 0;
    width: 100%%;
    height: 100%%;
    pointer-events: none;
    z-index: 9999;
    overflow: hidden;
}

.watermark {
    position: absolute;
    color: rgba(0, 0, 0, 0.10);
    transform: rotate(-45deg);
    white-space: nowrap;
}

h1 {
    color: #2c3e50;
    margin-bottom: 10px;
}

.description {
    color: #7f8c8d;
    font-size: 1.1rem;
    max-width: 800px;
    margin: 0 auto;
}

.json-container {
    display: flex;
    flex-wrap: wrap;
    gap: 20px;
    margin-bottom: 30px;
}

.json-viewer {
    flex: 1;
    min-width: 300px;
    background: white;
    border-radius: 10px;
    padding: 20px;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
    overflow: auto;
}

.json-title {
    font-size: 1.3rem;
    color: #2c3e50;
    margin-bottom: 15px;
    padding-bottom: 10px;
    border-bottom: 2px solid #ecf0f1;
}

pre {
    background: #f8f9fa;
    padding: 15px;
    border-radius: 5px;
    overflow-x: auto;
    font-size: 14px;
    line-height: 1.5;
    border-left: 4px solid #3498db;
    max-height: 270px; /* 限制最大高度 */
    overflow-y: auto; /* 添加垂直滚动条 */
}

.data-breakdown {
    flex: 1;
    min-width: 300px;
    background: white;
    border-radius: 10px;
    padding: 20px;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
}

.breakdown-title {
    font-size: 1.3rem;
    color: #2c3e50;
    margin-bottom: 15px;
    padding-bottom: 10px;
    border-bottom: 2px solid #ecf0f1;
}

.data-item {
    margin-bottom: 15px;
    padding: 15px;
    background: #f8f9fa;
    border-radius: 5px;
    border-left: 4px solid #2ecc71;
}

.data-key {
    font-weight: bold;
    color: #2c3e50;
    margin-bottom: 5px;
}

.data-value {
    color: #34495e;
    word-break: break-word;
}

.schema-section {
    background: white;
    border-radius: 10px;
    padding: 20px;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
    margin-bottom: 30px;
}

.schema-title {
    font-size: 1.3rem;
    color: #2c3e50;
    margin-bottom: 15px;
    padding-bottom: 10px;
    border-bottom: 2px solid #ecf0f1;
}

.schema-table {
    width: 100%%;
    border-collapse: collapse;
    margin-top: 10px;
}

.schema-table th {
    background: #3498db;
    color: white;
    padding: 12px 15px;
    text-align: left;
}

.schema-table td {
    padding: 10px 15px;
    border-bottom: 1px solid #ecf0f1;
}

.schema-table tr:nth-child(even) {
    background: #f8f9fa;
}

.schema-table tr:hover {
    background: #e8f4fc;
}

footer {
    text-align: center;
    margin-top: 20px;
    color: #7f8c8d;
    font-size: 0.9rem;
}

@media (max-width: 768px) {
    .json-container {
        flex-direction: column;
    }
}



/* 表格样式 */
/* 表格容器样式 */
.table-container {
    max-height: 500px; /* 必须有固定高度 */
    overflow-y: auto;  /* 必须有overflow */
    border: 1px solid #e0e0e0;
    border-radius: 8px;
    margin-top: 15px;
}

/* 表格样式 */
.permissions-table {
    width: 100%%;
    border-collapse: collapse;
}

.permissions-table th {
    /*background: linear-gradient(135deg, #6a11cb 0%%, #2575fc 100%%);*/
    background: linear-gradient(135deg, #6a11cb 0%%, #2575fc 100%%);
    color: white;
    padding: 15px;
    text-align: left;
    font-weight: 600;
    font-size: 1rem;
    position: sticky;
    top: 0;
    z-index: 100; /* 提高z-index确保在最上层 */
}

.permissions-table td {
    padding: 12px 15px;
    border-bottom: 1px solid #f0f0f0;
}

.permissions-table tr:nth-child(even) {
    background-color: #f8f9fa;
}

.permissions-table tr:hover {
    background-color: #eef5ff;
    transition: background-color 0.2s;
}

.permission-badge {
    display: inline-block;
    padding: 4px 10px;
    border-radius: 20px;
    font-size: 0.85rem;
    font-weight: 600;
}

.permission-select {
    background-color: #e8f5e9;
    color: #2e7d32;
}

.permission-insert {
    background-color: #e3f2fd;
    color: #1565c0;
}

.permission-update {
    background-color: #fff3e0;
    color: #ef6c00;
}

.permission-delete {
    background-color: #ffebee;
    color: #c62828;
}

.table-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 15px;
}

.search-box {
    padding: 8px 15px;
    border: 1px solid #ddd;
    border-radius: 5px;
    width: 250px;
    font-size: 0.9rem;
}

footer {
    text-align: center;
    padding: 20px;
    color: #7f8c8d;
    font-size: 0.9rem;
}

@media (max-width: 768px) {
    .json-container {
        flex-direction: column;
    }
    
    .table-header {
        flex-direction: column;
        align-items: flex-start;
    }
    
    .search-box {
        width: 100%%;
        margin-top: 10px;
    }
    
    .permissions-table {
        display: block;
        overflow-x: auto;
    }
}
</style>
`)
}

type apiSchema struct {
	Database     []string
	Table        []string
	Materialized []string
	View         []string
	Role         []string
}
type apiData struct {
	App        string
	User       string
	Authplugin string
	Schema     apiSchema
}

type trs struct {
	Level       string
	Catalog     string
	Tablename   string
	Permissions []string
}

func userapi(db *gorm.DB, app, user string) (apiData, error) {
	var m []map[string]interface{}
	r := db.Raw("show grants for " + strings.ToLower(user)).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return apiData{}, r.Error
	}

	var auth map[string]interface{}
	r = db.Raw("SHOW AUTHENTICATION FOR " + strings.ToLower(user)).Scan(&auth)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return apiData{}, r.Error
	}
	var database, table, materialized, view, role, masking, rowaccess []string
	for _, m2 := range m {
		var catalog string
		if m2["Catalog"] == nil {
			catalog = "default_catalog"
		} else {
			catalog = m2["Catalog"].(string)
		}
		td := strings.Split(m2["Grants"].(string), " TO ")[0]
		if strings.Contains(td, "'") {
			role = append(role, strings.NewReplacer("'", "", "GRANT ", "").Replace(td))
			continue
		}
		data := strings.Split(strings.Split(m2["Grants"].(string), " TO ")[0], " ON ")
		grant := strings.NewReplacer("GRANT ", "", ", ", ",").Replace(data[0])

		/*库表归类*/
		if strings.Contains(data[1], "DATABASE") && !strings.Contains(data[1], "MATERIALIZED") && !strings.Contains(data[1], "VIEWS") {
			d := strings.Split(data[1], " ")
			database = append(database, fmt.Sprintf("%s|%s|%s", catalog, d[len(d)-1], grant))
			continue
		}
		if strings.Contains(data[1], "TABLE") {
			d := strings.Split(data[1], " ")
			table = append(table, fmt.Sprintf("%s|%s|%s", catalog, d[len(d)-1], grant))
			continue
		}
		if strings.Contains(data[1], "MATERIALIZED") {
			d := strings.Split(data[1], " ")
			materialized = append(materialized, fmt.Sprintf("%s|%s|%s", catalog, d[len(d)-1], grant))
			continue
		}
		if strings.Contains(data[1], "VIEW") || strings.Contains(data[1], "VIEWS") {
			d := strings.Split(data[1], " ")
			view = append(view, fmt.Sprintf("%s|%s|%s", catalog, d[len(d)-1], grant))
			continue
		}
		if strings.Contains(data[1], "ROW ACCESS POLICY") || strings.Contains(data[1], "ROW ACCESS POLICY") {
			d := strings.Split(data[1], " ")
			rowaccess = append(rowaccess, fmt.Sprintf("%s|%s|%s", catalog, d[len(d)-1], grant))
			continue
		}
		if strings.Contains(data[1], "MASKING") || strings.Contains(data[1], "MASKING") {
			d := strings.Split(data[1], " ")
			masking = append(masking, fmt.Sprintf("%s|%s|%s", catalog, d[len(d)-1], grant))
			continue
		}

	}
	sort.Strings(database)
	sort.Strings(materialized)
	sort.Strings(view)
	sort.Strings(table)
	sort.Strings(rowaccess)
	sort.Strings(masking)

	aa := apiData{
		App:        app,
		User:       user,
		Authplugin: auth["AuthPlugin"].(string),
		Schema: apiSchema{
			Database:     database,
			Table:        table,
			Materialized: materialized,
			View:         view,
			Role:         role,
		},
	}
	return aa, nil
}
