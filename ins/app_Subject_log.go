/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    app_Subject_log
 *@date    2026/1/8 9:13
 */

package ins

import (
	"StarRocksProbe/conn"
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"gorm.io/gorm"
	"regexp"
	"strings"
	"time"
)

func SubjectLog(app, version, felogpath, belogpath string, db *gorm.DB, fs, bs []map[string]interface{}, availdateJson []byte) (string, int, int) {
	var all []string

	// 日志巡检汇总
	b, diffSum := loginsp(0, felogpath, belogpath, fs, bs)
	grade, concl, score := loggeninsp(diffSum, len(fs)+len(bs))
	// 巡检结论
	all = append(all, bodyConclusion(conclusion{
		Score:      score,
		ScoreLevel: grade,
		ScoreConcl: concl,
	}))
	// 汇总巡检明细
	all = append(all, fmt.Sprintf(`
<div class="section">
    <h2 class="section-title"><p class="tooltip" data-tooltip="以下是检测到包含 'error', 'fail', 'exception' 等关键字的日志内容（倒序显示）。点击节点切换日志。">日志巡检</p></h2>
    %v
</div>`, strings.Join(b, "\n")))

	tools.WriteFile(fmt.Sprintf("%s/%s_log.html", insdir, app),
		bodyTemplates(statictemplates{
			App:     app,
			Title:   "日志巡检",
			Version: version,
			Body:    strings.Join(all, "\n"),

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

	util.Loggrs.Info(app, "日志巡检完成。")
	return concl, score, diffSum
}

// 智能生成巡检结论
func loggeninsp(diffSum, count int) (string, string, int) {
	var grade, conclusion string
	score := float64(count-diffSum) / float64(count) * 100
	switch {
	case score > 99:
		grade = "卓越"
		conclusion = fmt.Sprintf("日志检查通过，未发现问题。核心服务（FE、BE、Broker）运行平稳，资源使用无异常。检测到包含 'error', 'fail', 'exception' 等关键字的日志内容（倒序显示）。点击节点切换日志。\n\n")
	case score >= 90:
		grade = "良好"
		conclusion = fmt.Sprintf("发现 %d 类日志相关风险，包含未知错误记录，但未影响核心服务可用性。\n\n", diffSum)
	default:
		grade = "一般"
		conclusion = fmt.Sprintf("发现 %d 类日志相关风险，可能影响查询性能或数据一致性。建议立即介入排查并处理。\n\n", diffSum)
	}
	return grade, conclusion, int(score)
}

// 日志诊断
func logdiagnosis(logmsg string) bool {
	// 获取今天和昨天的日期（两种格式）
	now := time.Now()

	// 格式1: yyyy-mm-dd
	today1 := now.Format("2006-01-02")
	yesterday1 := now.AddDate(0, 0, -1).Format("2006-01-02")

	// 格式2: yyyymmdd
	today2 := now.Format("20060102")
	yesterday2 := now.AddDate(0, 0, -1).Format("20060102")

	// 定义匹配两种日期格式的正则表达式
	datePattern1 := `\d{4}-\d{2}-\d{2}` // yyyy-mm-dd
	datePattern2 := `\d{8}`             // yyyymmdd

	dateRegex1 := regexp.MustCompile(datePattern1)
	dateRegex2 := regexp.MustCompile(datePattern2)

	// 查找日志中的所有日期（两种格式）
	dates1 := dateRegex1.FindAllString(logmsg, -1)
	dates2 := dateRegex2.FindAllString(logmsg, -1)

	// 检查格式1的日期（yyyy-mm-dd）
	for _, date := range dates1 {
		if date == today1 || date == yesterday1 {
			return true
		}
	}

	// 检查格式2的日期（yyyymmdd），并验证是否为有效日期
	for _, date := range dates2 {
		if date == today2 || date == yesterday2 {
			return true
		}
		// 额外的验证：检查是否是合理的日期（可选）
		if len(date) == 8 {
			_, err := time.Parse("20060102", date)
			if err == nil && (date == today2 || date == yesterday2) {
				return true
			}
		}
	}
	return false
}

// 日志巡检
func loginsp(diffSum int, felogpath, belogpath string, fs, bs []map[string]interface{}) ([]string, int) {
	if curritem() == "" {
		return nil, -1
	}
	var button, tab_button, fecontent, becontent, felog, belog, body []string
	// FE日志巡检
	for i, f := range fs {
		// 设置日志按钮
		if i == 0 {
			button = append(button, fmt.Sprintf(`<button class="tab-button active" data-tab="tab%d">%s</button>`, i+1, f["IP"].(string)))
		} else {
			button = append(button, fmt.Sprintf(`<button class="tab-button" data-tab="tab%d">%s</button>`, i+1, f["IP"].(string)))
		}
		// 登录机器
		ot := conn.ConnectSSH(
			&util.ConnSSH{
				User:           curritem(),
				Host:           f["IP"].(string),
				Port:           22,
				PrivateKeyFile: fmt.Sprintf("~/.ssh/id_rsa"),
				Command:        fmt.Sprintf("tail -n %d %s/fe.log", 200, felogpath),
			})
		if ot == nil {
			continue
		}
		// 设置日志内容
		b := strings.NewReplacer("\n", "<br>").Replace(ot.(string))
		var tag string
		if logdiagnosis(b) {
			tag = `<span class="status status-warning">异常</span>`
			diffSum = diffSum + 1
		} else {
			tag = `<span class="status status-ok">正常</span>`
		}
		felog = append(felog, fmt.Sprintf(`<pre style="max-height: 600px; overflow-y: auto;">%s</pre><p> %s</p>`, b, tag))
	}

	// BE日志巡检
	for i, b := range bs {
		// 设置日志按钮
		button = append(button, fmt.Sprintf(`<button class="tab-button" data-tab="tab%d">%s</button>`, i+1+len(fs), b["IP"].(string)))
		// 登录机器
		ot := conn.ConnectSSH(
			&util.ConnSSH{
				User:           curritem(),
				Host:           b["IP"].(string),
				Port:           22,
				PrivateKeyFile: fmt.Sprintf("~/.ssh/id_rsa"),
				Command:        fmt.Sprintf("tail -n %d %s/be.INFO", 200, belogpath),
			})
		if ot == nil {
			continue
		}
		// 设置日志内容
		b := strings.NewReplacer("\n", "<br>").Replace(ot.(string))
		var tag string
		if logdiagnosis(b) {
			tag = `<span class="status status-warning">异常</span>`
			diffSum = diffSum + 1
		} else {
			tag = `<span class="status status-ok">正常</span>`
		}
		belog = append(belog, fmt.Sprintf(`<pre style="max-height: 600px; overflow-y: auto;">%s</pre><p> %s</p>`, b, tag))
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
	return body, diffSum
}
