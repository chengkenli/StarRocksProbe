/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    app_Subject_avgs
 *@date    2026/1/8 12:31
 */

package ins

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"github.com/antchfx/htmlquery"
	"github.com/go-resty/resty/v2"
	"gorm.io/gorm"
	"strings"
)

type variable struct {
	IP       string
	Variable []string
}
type resultb struct {
	IP       string `json:"IP"`
	Variable string `json:"Variable"`
}

// SubjectAvgs
// 参数巡检
func SubjectAvgs(app, version string, db *gorm.DB, fs, bs []map[string]interface{}, availdateJson []byte) (string, int, int) {
	var all []string
	// FE/BE参数检查
	b, avgssum, feavgs, beavgs := avgsinsp(app, fs, bs)
	// 汇总差异
	fbody := findInconsistentVariables(feavgs)
	bbody := findInconsistentVariables(beavgs)
	var appendBody []resultb
	appendBody = append(appendBody, fbody...)
	appendBody = append(appendBody, bbody...)
	diffSum := len(rmResultVariables(appendBody))
	// 生成结论
	grade, concl, score := avgsgeninsp(diffSum, len(feavgs)+len(beavgs), avgssum, fbody, bbody)
	// 巡检结论
	all = append(all, bodyConclusion(conclusion{
		Score:      score,
		ScoreLevel: grade,
		ScoreConcl: concl,
	}))

	// 汇总关键全局变量
	all = append(all, bodySubject(subject{
		Title2:  "关键全局变量",
		Thead:   []string{"参数名称", "值"},
		Tbody:   findCoreavgs(db, []string{"enable_pipeline_engine", "pipeline_dop", "parallel_fragment_exec_instance_num", "enable_profile"}),
		Comment: `<h2 class="section-title"><p class="tooltip" data-tooltip="影响集群核心性能的关键全局变量的当前值">关键全局变量</p></h2>`,
	}))

	// 汇总巡检明细
	all = append(all, fmt.Sprintf(`
<div class="section">
    <h2 class="section-title"><p class="tooltip" data-tooltip="静态+动态 所有参数">全局变量</p></h2>
    %v
</div>`, strings.Join(b, "\n")))

	// 汇总差异参数
	var fbodys [][]string
	for _, item := range fbody {
		tag := strings.Split(item.Variable, "=")
		fbodys = append(fbodys, []string{tag[0], item.IP, tag[1]})
	}

	all = append(all, bodySubject(subject{
		Title2:  "FE差异参数",
		Thead:   []string{"参数名称", "节点", "值"},
		Tbody:   diagnMark(fbodys),
		Comment: `<h2 class="section-title"><p class="tooltip" data-tooltip="跨节点不一致的FE参数">FE差异参数</p></h2>`,
	}))

	var bbodys [][]string
	for _, item := range bbody {
		tag := strings.Split(item.Variable, "=")
		bbodys = append(bbodys, []string{tag[0], item.IP, tag[1]})
	}
	all = append(all, bodySubject(subject{
		Title2:  "BE差异参数",
		Thead:   []string{"参数名称", "节点", "值"},
		Tbody:   diagnMark(bbodys),
		Comment: `<h2 class="section-title"><p class="tooltip" data-tooltip="跨节点不一致的BE参数">BE差异参数</p></h2>`,
	}))

	tools.WriteFile(fmt.Sprintf("%s/%s_avgs.html", insdir, app),
		bodyTemplates(statictemplates{
			App:           app,
			Title:         "参数巡检",
			Version:       version,
			Body:          strings.Join(all, "\n"),
			Select:        strings.Join(selectids(), "\n"),
			AvaildateJson: string(availdateJson),
			Watermark:     fmt.Sprintf("StarRocks（%s）", strings.ToUpper(app)),
			Script: `
	<script>
        document.addEventListener('DOMContentLoaded', function() {
            const tabButtons = document.querySelectorAll('.tab-button');
            const tabContents = document.querySelectorAll('.tab-content');
            
            tabButtons.forEach(button => {
                button.addEventListener('click', () => {
                    // 移除所有按钮和内容的active类
                    tabButtons.forEach(btn => btn.classList.remove('active'));
                    tabContents.forEach(content => content.classList.remove('active'));
                    
                    // 为当前按钮和对应内容添加active类
                    button.classList.add('active');
                    const tabId = button.getAttribute('data-tab');
                    document.getElementById(tabId).classList.add('active');
                });
            });
        });
    </script>`}))

	util.Loggrs.Info(app, "参数巡检完成。")
	return concl, score, diffSum
}

// 智能生成巡检结论
func avgsgeninsp(diffSum, count, avgssum int, fbody, bbody []resultb) (string, string, int) {
	var grade, conclusion string
	score := float64(avgssum-diffSum) / float64(avgssum) * 100
	switch {
	case int(score) > 99:
		grade = "卓越"
		conclusion = fmt.Sprintf("本次巡检集群参数共【<b>%d</b>】个，涉及巡检节点有【<b>%d</b>】个，集群所有参数一致。集群参数无异常。\n\n", avgssum, count)
	default:
		grade = "一般"
		var msg []string
		if fbody != nil {
			msg = append(msg, fmt.Sprintf("<li>发现FE有跨节点不一致的参数，可能存在风险</li>"))
		}
		if bbody != nil {
			msg = append(msg, fmt.Sprintf("<li>发现BE有跨节点不一致的参数，可能存在风险</li>"))
		}
		conclusion = fmt.Sprintf("本次巡检集群参数共【<b>%d</b>】个，涉及巡检节点有【<b>%d</b>】个，集群有【<b>%d</b>】处参数出现异常，参数在不同节点间的运行时值不一致，这可能导致集群行为异常。请检查并统一这些参数的配置。！\n\n%s", avgssum, count, diffSum, strings.Join(msg, "\n"))
	}
	return grade, conclusion, int(score)
}

// 关键全局变量
func findCoreavgs(db *gorm.DB, key []string) [][]string {
	var vbody [][]string
	for _, item := range key {
		var m map[string]interface{}
		r := db.Raw(fmt.Sprintf("SHOW VARIABLES like '%s'", item)).Scan(&m)
		if r.Error != nil {
			util.Loggrs.Error(r.Error.Error())
			continue
		}
		//m["Value"].(string)
		if m["Variable_name"] == nil {
			r = db.Raw(fmt.Sprintf("ADMIN SHOW FRONTEND CONFIG LIKE '%s'", item)).Scan(&m)
			if r.Error != nil {
				util.Loggrs.Error(r.Error.Error())
				continue
			}
			if m["Value"] != nil {
				vbody = append(vbody, []string{item, m["Value"].(string)})
			}
		} else {
			vbody = append(vbody, []string{item, m["Value"].(string)})
		}
	}
	return vbody
}

// 参数巡检
func avgsinsp(app string, fs, bs []map[string]interface{}) ([]string, int, []variable, []variable) {
	var c util.ConnectParms
	for _, m := range util.MetaLink {
		if m["app"].(string) == app {
			c = util.ConnectParms{
				User: m["user"].(string),
				Pass: m["password"].(string),
			}
		}
	}

	var feavgs, beavgs []variable
	var feavgssum, beavgssum int
	var button, tab_button, fecontent, becontent, felog, belog, body []string
	// FE参数巡检
	for i, f := range fs {
		// 设置日志按钮
		if i == 0 {
			button = append(button, fmt.Sprintf(`<button class="tab-button active" data-tab="tab%d">%s</button>`, i+1, f["IP"].(string)))
		} else {
			button = append(button, fmt.Sprintf(`<button class="tab-button" data-tab="tab%d">%s</button>`, i+1, f["IP"].(string)))
		}
		//创建Resty客户端
		Client := resty.New().SetDisableWarn(true)
		//发送POST请求并处理响应
		response, err := Client.R().SetBasicAuth(c.User, c.Pass).Get(fmt.Sprintf("http://%s:8030/variable", f["IP"].(string)))
		if err != nil {
			util.Loggrs.Warn("报错：", err.Error())
			continue
		}
		// 解析 HTML
		doc, err := htmlquery.Parse(strings.NewReader(string(response.Body())))
		if err != nil {
			util.Loggrs.Error(err.Error())
			continue
		}
		// 使用 XPath 定位表格的 tbody
		tbody := htmlquery.FindOne(doc, "/html/body/div/pre[1]/text()")
		if tbody == nil {
			util.Loggrs.Error("未找到表格 tbody")
			continue
		}
		feavgs = append(feavgs, variable{
			IP:       f["IP"].(string),
			Variable: strings.Split(tbody.Data, "\n"),
		})
		feavgssum = len(strings.Split(tbody.Data, "\n"))

		// 设置日志内容
		b := strings.NewReplacer("\n", "<br>").Replace(tbody.Data)
		felog = append(felog, fmt.Sprintf(`<pre style="max-height: 600px; overflow-y: auto;">%s</pre><p></p>`, b))
	}

	// BE参数巡检
	for i, b := range bs {
		// 设置日志按钮
		button = append(button, fmt.Sprintf(`<button class="tab-button" data-tab="tab%d">%s</button>`, i+1+len(fs), b["IP"].(string)))
		// 登录机器

		//创建Resty客户端
		Client := resty.New().SetDisableWarn(true)
		//发送POST请求并处理响应
		response, err := Client.R().Get(fmt.Sprintf("http://%s:8040/varz", b["IP"].(string)))
		if err != nil {
			util.Loggrs.Warn("报错：", err.Error())
			continue
		}
		// 解析 HTML
		doc, err := htmlquery.Parse(strings.NewReader(string(response.Body())))
		if err != nil {
			util.Loggrs.Error(err.Error())
			continue
		}
		// 使用 XPath 定位表格的 tbody
		tbody := htmlquery.FindOne(doc, "/html/body/pre/text()")
		if tbody == nil {
			util.Loggrs.Error("未找到表格 tbody")
			continue
		}
		beavgs = append(beavgs, variable{
			IP:       b["IP"].(string),
			Variable: strings.Split(tbody.Data, "\n"),
		})
		beavgssum = len(strings.Split(tbody.Data, "\n"))

		// 设置日志内容
		b := strings.NewReplacer("\n", "<br>").Replace(tbody.Data)
		belog = append(belog, fmt.Sprintf(`<pre style="max-height: 600px; overflow-y: auto;">%s</pre><p></p>`, b))
	}

	// 设置整个标题的tab
	tab_button = append(tab_button, fmt.Sprintf(`<div class="tabs">%s</div>`, strings.Join(button, "\n")))
	// 设置内容
	for i, item := range felog {
		if i == 0 {
			fecontent = append(fecontent, fmt.Sprintf(`<div class="tab-content active" id="tab%d">%s</div>`, i+1, item))
		} else {
			fecontent = append(fecontent, fmt.Sprintf(`<div class="tab-content" id="tab%d">%s</div>`, i+1, item))
		}
	}
	for i, item := range belog {
		becontent = append(becontent, fmt.Sprintf(`<div class="tab-content" id="tab%d">%s</div>`, i+1+len(fs), item))
	}
	body = append(body, tab_button...)
	body = append(body, fecontent...)
	body = append(body, becontent...)
	return body, feavgssum + beavgssum, feavgs, beavgs
}
