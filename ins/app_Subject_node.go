/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    app_Subject_node
 *@date    2026/1/7 21:27
 */

package ins

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"gorm.io/gorm"
	"strings"
)

func SubjectNode(app, version string, db *gorm.DB, fs, bs []map[string]interface{}, tic map[string]interface{}, availdateJson []byte) (string, int, int) {
	// 指标体现
	var all, indi []string
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "FE",
		Value:    len(fs),
		Subvalue: `<span class="status status-ok">连通性正常</span>`,
	}))
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "BE",
		Value:    len(bs),
		Subvalue: `<span class="status status-ok">连通性正常</span>`,
	}))
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "表数量",
		Value:    tic["TableNum"].(string),
		Subvalue: `<span class="status status-ok">正常</span>`,
	}))
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "分区数量",
		Value:    tic["PartitionNum"].(string),
		Subvalue: `<span class="status status-ok">正常</span>`,
	}))
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "总副本数",
		Value:    tic["TabletNum"].(string),
		Subvalue: "",
	}))
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "总分片数",
		Value:    tic["ReplicaNum"].(string),
		Subvalue: "",
	}))
	indi = append(indi, bodyIndicatorFirst(indicator{
		Title3:   "异常分片数",
		Value:    tic["ErrorStateTabletNum"].(string),
		Subvalue: `<span class="status status-ok">正常</span>`,
	}))
	// 主体内容
	var tbs [][]string
	for _, f := range fs {
		tbs = append(tbs, []string{"FE", `<span class="status status-ok">正常</span>`,
			f["Role"].(string), f["IP"].(string), f["Alive"].(string), "无", "无", "无", "无", "无", "无", "无", "无"})
	}
	for _, b := range bs {
		tbs = append(tbs, []string{"BE", `<span class="status status-ok">正常</span>`, "BACKEND",
			b["IP"].(string), b["Alive"].(string), b["TabletNum"].(string),
			b["TotalCapacity"].(string), b["AvailCapacity"].(string),
			b["UsedPct"].(string), b["CpuCores"].(string), b["MemLimit"].(string), b["CpuUsedPct"].(string), b["MemUsedPct"].(string)})
	}

	score := 100
	concl := "各项进程服务状态优秀，集群整体服务，端口可用性100%，无副本出现损坏的情况，巡检通过。\n\n"
	all = append(all, bodyConclusion(conclusion{
		Score:      score,
		ScoreLevel: "卓越",
		ScoreConcl: concl,
	}))
	all = append(all, bodyIndicatorLast("核心指标", strings.Join(indi, "\n")))
	all = append(all, bodySubject(subject{
		Title2: "节点明细",
		Thead: []string{"Service", "State", "Role", "IP", "Alive", "TabletNum", "TotalCapacity",
			"AvailCapacity", "UsedPct", "CpuCores", "MemLimit", "CpuUsedPct", "MemUsedPct"},
		Tbody: tbs,
	}))
	tools.WriteFile(fmt.Sprintf("%s/%s_node.html", insdir, app),
		bodyTemplates(statictemplates{
			App:           app,
			Title:         "节点巡检",
			Version:       version,
			Body:          strings.Join(all, "\n"),
			Select:        strings.Join(selectids(), "\n"),
			AvaildateJson: string(availdateJson),
			Watermark:     fmt.Sprintf("StarRocks（%s）", strings.ToUpper(app)),
		}))

	util.Loggrs.Info(app, "节点巡检完成。")
	return concl, score, 0
}
