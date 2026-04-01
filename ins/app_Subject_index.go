/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    app_Subject_index
 *@date    2026/1/9 10:31
 */

package ins

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"gorm.io/gorm"
	"strings"
	"sync"
	"time"
)

// SubjectIndex
// 巡检首页
func SubjectIndex(app, version, felogpath, belogpath string, db *gorm.DB, fs, bs []map[string]interface{}, availdateJson []byte) {
	//tic, fullTic := statistic(db)
	var tic map[string]interface{}
	var fullTic []map[string]interface{}
	ticker := time.NewTicker(time.Minute * 1)
loop:
	for {
		select {
		case <-ticker.C:
			if val, ok := util.Lastcache.Get(app + "statistic"); ok {
				for _, i2 := range val.([]map[string]interface{}) {
					if i2["DbId"].(string) == "Total" {
						tic = i2
						fullTic = val.([]map[string]interface{})
						//跳出
						break loop
					}
				}
			}
			util.Loggrs.Info("等待statistic生效！")
		}
	}
	var scoreConf, diffsumConf, scoreNode, diffsumNode, scoreOlap, diffsumOlap, scoreLog, diffsumLog, scoreAvgs, diffsumAvgs int
	var wg sync.WaitGroup
	wg.Add(5)
	go func() {
		defer wg.Done()
		// 配置巡检【完成】
		util.Loggrs.Info(app + "配置巡检...")
		_, scoreConf, diffsumConf = SubjectConfig(app, version, felogpath, belogpath, db, fs, bs, availdateJson)
	}()
	go func() {
		defer wg.Done()
		// 节点巡检【完成】
		util.Loggrs.Info(app + "节点巡检...")
		_, scoreNode, diffsumNode = SubjectNode(app, version, db, fs, bs, tic, availdateJson)
	}()
	go func() {
		defer wg.Done()
		// 内表巡检【完成】
		util.Loggrs.Info(app + "内表巡检...")
		_, scoreOlap, diffsumOlap = SubjectOlap(app, version, db, fs, bs, tic, fullTic, availdateJson)
	}()
	go func() {
		defer wg.Done()
		// 日志巡检【完成】
		util.Loggrs.Info(app + "日志巡检...")
		_, scoreLog, diffsumLog = SubjectLog(app, version, felogpath, belogpath, db, fs, bs, availdateJson)
	}()
	go func() {
		defer wg.Done()
		// 参数巡检【完成】
		util.Loggrs.Info(app + "参数巡检...")
		_, scoreAvgs, diffsumAvgs = SubjectAvgs(app, version, db, fs, bs, availdateJson)
	}()
	wg.Wait()
	////////////////////////////////////////////////////////
	score := (scoreNode + scoreLog + scoreAvgs + scoreConf + scoreOlap) / 5
	var conclmsg []string
	if scoreLog < 100 {
		conclmsg = append(conclmsg, fmt.Sprintf("<b>日志巡检</b>"))
	}
	if scoreNode < 100 {
		conclmsg = append(conclmsg, fmt.Sprintf("<b>机器巡检</b>"))
	}
	if scoreAvgs < 100 {
		conclmsg = append(conclmsg, fmt.Sprintf("<b>参数巡检</b>"))
	}
	if scoreConf < 100 {
		conclmsg = append(conclmsg, fmt.Sprintf("<b>配置巡检</b>"))
	}
	if scoreOlap < 100 {
		conclmsg = append(conclmsg, fmt.Sprintf("<b>内表巡检</b>"))
	}
	ctxt := fmt.Sprintf("本次对StarRocks %s集群的巡检已完成，主要检查了集群节点状态、数据分片分布、查询性能指标、系统资源使用情况以及日志错误记录等关键内容。巡检结果显示，集群整体运行稳定，但其中在 %v 中发现潜在风险，需要进行核实处理。\n\n", app, strings.Join(conclmsg, ","))
	////////////////////////////////////////////////////////
	var all, indi []string
	// 获取集群明细信息
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "集群总节点数",
		Value:    len(fs) + len(bs),
		Subvalue: fmt.Sprintf("FE数：%d，BE数：%d", len(fs), len(bs)),
	}))
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "集群表数量",
		Value:    tic["TableNum"].(string),
		Subvalue: fmt.Sprintf("分区数量：%s，副本数：%s，分片数：%s", tic["PartitionNum"].(string), tic["TabletNum"].(string), tic["ReplicaNum"].(string)),
	}))
	u := statisuser(db)
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "活跃用户数",
		Value:    u,
		Subvalue: "",
	}))
	sumrequest := statisrequest(db)
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "24小时访问量",
		Value:    sumrequest,
		Subvalue: "",
	}))
	all = append(all, bodyIndicatorLast("集群规模", strings.Join(indi, "\n")))
	// 巡检结论
	var level string
	switch {
	case score > 99:
		level = "卓越"
	case score >= 90:
		level = "优秀"
	default:
		level = "一般"
	}
	all = append(all, bodyConclusion(conclusion{
		Score:      score,
		ScoreLevel: level,
		ScoreConcl: ctxt,
	}))
	// 巡检选项

	tb := [][]string{
		organize(organizes{
			app:     app,
			name:    "日志巡检",
			label:   "log",
			diffSum: diffsumLog,
			score:   scoreLog,
		}),
		organize(organizes{
			app:     app,
			name:    "机器巡检",
			label:   "node",
			diffSum: diffsumNode,
			score:   scoreNode,
		}),
		organize(organizes{
			app:     app,
			name:    "参数巡检",
			label:   "avgs",
			diffSum: diffsumAvgs,
			score:   scoreAvgs,
		}),
		organize(organizes{
			app:     app,
			name:    "配置巡检",
			label:   "conf",
			diffSum: diffsumConf,
			score:   scoreConf,
		}),
		organize(organizes{
			app:     app,
			name:    "内表巡检",
			label:   "olap",
			diffSum: diffsumOlap,
			score:   scoreOlap,
		}),
	}
	all = append(all, bodySubject(subject{
		Title2:  "各模块巡检概览",
		Thead:   []string{"巡检模块", "摘要", "评分", "详情"},
		Tbody:   tb,
		Comment: `<h2 class="section-title"><p class="tooltip" data-tooltip="巡检总概览">各模块巡检概览</p></h2>`,
	}))
	// 写入文件
	tools.WriteFile(fmt.Sprintf("%s/%s.html", insdir, app),
		bodyTemplates(statictemplates{
			App:           app,
			Title:         "星辰守望 - StarRocks小助手 巡检",
			Version:       version,
			Body:          strings.Join(all, "\n"),
			Select:        strings.Join(selectids(), "\n"),
			AvaildateJson: string(availdateJson),
			Watermark:     fmt.Sprintf("StarRocks（%s）", strings.ToUpper(app)),
		}))
	util.Loggrs.Info(app, "总结完成。")
}

// 整理结论信息
// @result 摘要，分数
type organizes struct {
	app, label, name string
	diffSum, score   int
}

func organize(o2 organizes) []string {
	var summ, score string
	// 生成摘要
	switch {
	case o2.score > 99:
		summ = fmt.Sprintf(`<p style="color: green;">检查通过，未发现明显问题。</p>`)
		score = fmt.Sprintf(`<p style="font-size: 20px; font-weight: bold; color: green;">%d</p>`, o2.score)
	default:
		switch o2.label {
		case "log":
			summ = fmt.Sprintf(`<p style="color: #ff8f00;">发现 <b>%d</b> 类日志相关风险，包括配置问题和错误记录。</p>`, o2.diffSum)
		case "monitor":
			summ = fmt.Sprintf(`<p style="color: #ff8f00;">发现 <b>%d</b> 项监控指标风险，可能影响集群性能与稳定性。</p>`, o2.diffSum)
		case "avgs":
			summ = fmt.Sprintf(`<p style="color: #ff8f00;">发现 <b>%d</b> 类参数配置风险，可能影响集群稳定性和可维护性。</p>`, o2.diffSum)
		case "node":
			summ = fmt.Sprintf(`<p style="color: #ff8f00;">发现 <b>%d</b> 处操作系统参数与最佳实践不一致，可能影响集群性能和稳定性。</p>`, o2.diffSum)
		case "conf":
			summ = fmt.Sprintf(`<p style="color: #ff8f00;">发现 <b>%d</b> 类远程文件相关风险，包括系统配置文件不一致和Hadoop配置问题。</p>`, o2.diffSum)
		case "olap":
			summ = fmt.Sprintf(`<p style="color: #ff8f00;">共发现 <b>%d</b> 类 Tablet 相关风险，可能影响集群稳定性和性能。</p>`, o2.diffSum)
		}
		score = fmt.Sprintf(`<p>%d</p>`, o2.score)
	}
	return []string{o2.name, summ, score, fmt.Sprintf(`<a href="%s_%s.html">查看详情 »</a>`, o2.app, o2.label)}
}
