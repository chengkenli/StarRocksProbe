/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    api_memory_usage_detail
 *@date    2025/12/31 13:24
 */

package app

import (
	"StarRocksProbe/util"
	"context"
	"encoding/json"
	"fmt"
	"github.com/antchfx/htmlquery"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

func (engine *threadMap) memoryUsage(c *gin.Context) {
	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)

	webdir := util.Read.Log.Path + "/html"
	os.Mkdir(webdir, 0755)
	filename := fmt.Sprintf("%s/html/%s.memory_usage.%s.html", util.Read.Log.Path, appid, time.Now().Format("20060102150405"))
	MemoryUsageDetail(appid, filename)
	c.JSON(http.StatusOK, gin.H{"url": fmt.Sprintf("%s:%d/reports%s", util.H.Ip, util.Read.Server.Port, filename)})
}

// MemoryRecord
// 定义结构体来存储每行数据
type memoryRecord struct {
	Level              string `json:"level"`
	Label              string `json:"label"`
	Parent             string `json:"parent"`
	Limit              string `json:"limit"`
	CurrentConsumption string `json:"current_consumption"`
	PeakConsumption    string `json:"peak_consumption"`
}

// globalMemoryRecord
// 这里再包一层，带上be节点
type globalMemoryRecord struct {
	IP           string
	MemoryRecord []memoryRecord
}

//func MemoryUsageDetail(app, filename string) {
//	backendIP := get_backend_nodes(app)
//	var globalMemoryRecords []globalMemoryRecord
//
//	var wg sync.WaitGroup
//	for _, ip := range backendIP {
//		wg.Add(1)
//		ip := ip
//		go func() {
//			defer wg.Done()
//			data := getBackendMemtracker(ip)
//			globalMemoryRecords = append(globalMemoryRecords, data)
//		}()
//	}
//	wg.Wait()
//
//	marshal, _ := json.Marshal(globalMemoryRecords)
//	memory_usage_file(filename, string(marshal))
//	util.Loggrs.Info(app, " ", filename)
//}

func MemoryUsageDetail(app, filename string) {
	backendIP := get_backend_nodes(app)
	var globalMemoryRecords []globalMemoryRecord

	var wg sync.WaitGroup
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	for _, ip := range backendIP {
		wg.Add(1)
		ip := ip
		go func() {
			defer wg.Done()
			data := getBackendMemtracker(ip)
			globalMemoryRecords = append(globalMemoryRecords, data)
		}()
	}
	if TimeoutQueries(ctx, &wg) {
		marshal, _ := json.Marshal(globalMemoryRecords)
		memory_usage_file(filename, string(marshal))
		util.Loggrs.Infof(app, " ", filename)
	} else {
		util.Loggrs.Errorf(app, " MemoryUsageDetail timeout")
	}
}

// 获取backend节点的内存使用情况
// @args ip
// @result []byte
func getBackendMemtracker(ip string) globalMemoryRecord {
	uri := fmt.Sprintf("http://%s:8040/mem_tracker", ip)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	//发送POST请求并处理响应
	response, err := resty.New().R().
		SetHeader("Content-Type", "application/json;charset=utf-8").
		SetContext(ctx).
		Get(uri)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return globalMemoryRecord{}
	}

	// 解析 HTML
	doc, err := htmlquery.Parse(strings.NewReader(string(response.Body())))
	if err != nil {
		util.Loggrs.Error(err.Error())
		return globalMemoryRecord{}
	}
	// 使用 XPath 定位表格的 tbody
	tbody := htmlquery.FindOne(doc, "//table/tbody")
	if tbody == nil {
		util.Loggrs.Error("未找到表格 tbody")
		return globalMemoryRecord{}
	}

	var records []memoryRecord

	// 遍历 tbody 中的每一行（tr 元素）
	rows := htmlquery.Find(tbody, "./tr")
	for _, row := range rows {
		// 提取当前行中的所有单元格（td 元素）
		cells := htmlquery.Find(row, "./td")
		if len(cells) >= 6 {
			record := memoryRecord{
				Level:              strings.TrimSpace(htmlquery.InnerText(cells[0])),
				Label:              strings.TrimSpace(htmlquery.InnerText(cells[1])),
				Parent:             strings.TrimSpace(htmlquery.InnerText(cells[2])),
				Limit:              strings.TrimSpace(htmlquery.InnerText(cells[3])),
				CurrentConsumption: strings.TrimSpace(htmlquery.InnerText(cells[4])),
				PeakConsumption:    strings.TrimSpace(htmlquery.InnerText(cells[5])),
			}
			records = append(records, record)
		}
	}
	global := globalMemoryRecord{
		IP:           ip,
		MemoryRecord: records,
	}
	return global
}

// 获取集群的backend节点ip
// @avgs app 集群名称
// @return []string 数组
func get_backend_nodes(app string) []string {
	db, err := engine.getmapConnect(app)
	if err != nil {
		util.Loggrs.Error(err.Error())
		return nil
	}
	var m []map[string]interface{}
	r := db.Raw("show backends").Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil
	}
	if m == nil {
		r := db.Raw("SHOW COMPUTE NODES").Scan(&m)
		if r.Error != nil {
			util.Loggrs.Error(r.Error.Error())
			return nil
		}
	}
	var ip []string
	for _, m2 := range m {
		ip = append(ip, m2["IP"].(string))
	}
	return ip
}
