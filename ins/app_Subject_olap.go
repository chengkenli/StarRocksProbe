/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    app_Subject_olap
 *@date    2026/1/8 23:35
 */

package ins

import (
	"StarRocksProbe/tools"
	"StarRocksProbe/util"
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

// SubjectOlap
// 内表巡检
func SubjectOlap(app, version string, db *gorm.DB, fs, bs []map[string]interface{}, tic map[string]interface{}, fullTic []map[string]interface{}, availdateJson []byte) (string, int, int) {
	var all []string
	// 检查主键索引
	index, diffSum := primaryIndex(db)
	// 检查内表倾斜
	tilt, diffSum2 := olaptilt(db)
	// 副本数不足
	replons, diffSum3 := olapreplicationNum(db)
	// 空表
	null, diffSum4 := olapnil(db)
	// 生成结论
	tableNum, _ := strconv.Atoi(tic["TableNum"].(string))
	partitionNum, _ := strconv.Atoi(tic["PartitionNum"].(string))
	score, grade, concl := olapgeninsp(diffSum, diffSum2, diffSum3, diffSum4, tableNum+partitionNum)
	// 巡检结论
	all = append(all, bodyConclusion(conclusion{
		Score:      score,
		ScoreLevel: grade,
		ScoreConcl: concl,
	}))
	// 巡检对象
	var initconf [][]string
	var msg string
	for _, item := range fullTic {
		if item["DbId"] != "Total" {
			initconf = append(initconf, []string{
				item["DbName"].(string),
				item["TableNum"].(string),
				item["PartitionNum"].(string),
				item["IndexNum"].(string),
				item["TabletNum"].(string),
				item["ReplicaNum"].(string),
			})
		}
		if item["DbId"] == "Total" {
			msg = fmt.Sprintf("自动对集群中 %s 个数据库，共 %s 个表 %s 个分区进行数据倾斜，索引持久化，副本数不足，空存储表等方式进行巡检。", item["DbName"].(string), item["TableNum"].(string), item["PartitionNum"].(string))
		}
	}
	all = append(all, bodySubject(subject{
		Title2:  "内表巡检对象",
		Thead:   []string{"数据库名称", "表数量", "分区数量", "索引数量", "副本数量", "分片数量"},
		Tbody:   initconf,
		Comment: fmt.Sprintf(`<h2 class="section-title"><p class="tooltip" data-tooltip="%s">内表巡检对象</p></h2>`, msg),
	}))

	// 主键持久化巡检
	if diffSum > 0 {
		all = append(all, index)
	}
	// 内表倾斜巡检
	if diffSum2 > 0 {
		all = append(all, tilt)
	}
	// 副本数不足巡检
	if diffSum3 > 0 {
		all = append(all, replons)
	}
	// 空表巡检
	if diffSum4 > 0 {
		all = append(all, null)
	}

	tools.WriteFile(fmt.Sprintf("%s/%s_olap.html", insdir, app),
		bodyTemplates(statictemplates{
			App:           app,
			Title:         "内表巡检",
			Version:       version,
			Body:          strings.Join(all, "\n"),
			Select:        strings.Join(selectids(), "\n"),
			AvaildateJson: string(availdateJson),
			Watermark:     fmt.Sprintf("StarRocks（%s）", strings.ToUpper(app)),
		}))
	util.Loggrs.Info(app, "内表巡检完成。")
	return concl, score, diffSum + diffSum2 + diffSum3 + diffSum4
}

// 智能生成巡检结论
func olapgeninsp(diffvari, diffvari2, diffvari3, diffvari4, itemSum int) (int, string, string) {
	var grade, conclusion string
	sc := float64(itemSum-(diffvari+diffvari2+diffvari3+diffvari4)) / float64(itemSum) * 100
	switch {
	case int(sc) > 99:
		grade = "卓越"
		conclusion = fmt.Sprintf("集群所有内表皆正常。\n\n")
	default:
		grade = "一般"
		var concl []string
		if diffvari > 0 {
			concl = append(concl, fmt.Sprintf("<li >发现【<b>%d</b>】个主键表未开启持久化主键索引 (`enable_persistent_index`=false)。这可能导致重启后主键索引重建时间过长，影响服务可用性。建议开启此选项。</li>", diffvari))
		}
		if diffvari2 > 0 {
			concl = append(concl, fmt.Sprintf("<li>发现【<b>%d</b>】个内表存在底层副本倾斜</li>", diffvari2))
		}
		if diffvari3 > 0 {
			concl = append(concl, fmt.Sprintf("<li>发现【<b>%d</b>】个内表副本数不足3</li>", diffvari3))
		}
		if diffvari4 > 0 {
			concl = append(concl, fmt.Sprintf("<li>发现【<b>%d</b>】个空存储表</li>", diffvari4))
		}
		conclusion = fmt.Sprintf(`共巡检了 %d 个olap配置与分区，<ol style="color: #f5843f;">%v</ol>`, itemSum, strings.Join(concl, "\n"))
	}
	return int(sc), grade, conclusion
}

// 巡检空表
func olapnil(db *gorm.DB) (string, int) {
	stmt := fmt.Sprintf(`SELECT
  pm.DB_NAME,
  pm.TABLE_NAME,
  COUNT(DISTINCT pm.PARTITION_ID) AS partition_count,
  COUNT(tbt.TABLET_ID) AS tablet_count,
  COALESCE(SUM(tbt.DATA_SIZE), 0) AS total_data_size
FROM
  information_schema.partitions_meta pm
  LEFT JOIN information_schema.be_tablets tbt ON pm.PARTITION_ID = tbt.PARTITION_ID
GROUP BY
  pm.DB_NAME,
  pm.TABLE_NAME
HAVING
  COALESCE(SUM(tbt.DATA_SIZE), 0) = 0
ORDER BY
  tablet_count desc`)
	var m []map[string]interface{}
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return "", -1
	}
	var bbodys [][]string
	for _, m2 := range m {
		bbodys = append(bbodys, []string{m2["DB_NAME"].(string), m2["TABLE_NAME"].(string), fmt.Sprintf("%d", m2["partition_count"].(int64)),
			fmt.Sprintf("%d", m2["tablet_count"].(int64)),
			fmt.Sprintf("%d", m2["total_data_size"].(int64)),
		})
	}
	return bodySubject(subject{
		Title2:  "空存储表",
		Thead:   []string{"库名", "表名", "分区数", "分片数", "容量大小"},
		Tbody:   bbodys,
		Comment: `<h2 class="section-title"><p class="tooltip" data-tooltip="空存储表占用一定的元数据">空存储表</p></h2>`,
	}), len(m)
}

// 查找副本数不是3的表
func olapreplicationNum(db *gorm.DB) (string, int) {
	stmt := fmt.Sprintf(`SELECT
  DISTINCT pm.DB_NAME,
  pm.TABLE_NAME,
  pm.REPLICATION_NUM
FROM
  information_schema.partitions_meta pm
WHERE
  pm.REPLICATION_NUM != 3
  AND pm.REPLICATION_NUM IS NOT NULL
  AND pm.PARTITION_NAME != '$shadow_automatic_partition'
ORDER BY
  pm.DB_NAME,
  pm.TABLE_NAME`)

	var m []map[string]interface{}
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return "", -1
	}
	var bbodys [][]string
	for _, m2 := range m {
		bbodys = append(bbodys, []string{m2["DB_NAME"].(string), m2["TABLE_NAME"].(string), fmt.Sprintf("%d", m2["REPLICATION_NUM"].(int64))})
	}

	return bodySubject(subject{
		Title2:  "副本数不足",
		Thead:   []string{"库名", "表名", "副本数"},
		Tbody:   bbodys,
		Comment: `<h2 class="section-title"><p class="tooltip" data-tooltip="副本数不足3的表">副本数不足3的表</p></h2>`,
	}), len(m)
}

// 内表倾斜巡检
func olaptilt(db *gorm.DB) (string, int) {
	stmt := fmt.Sprintf(`WITH BECOUNT AS (
  SELECT
    COUNT(DISTINCT BE_ID) AS be_node_count
  FROM
    information_schema.be_tablets
),
PartitionData AS (
  SELECT
    pm.DB_NAME,
    pm.TABLE_NAME,
    pm.PARTITION_NAME,
    COALESCE(SUM(tbt.DATA_SIZE), 0) / (1024 * 1024 * 1024) AS partition_data_size_gb
  FROM
    information_schema.partitions_meta pm
    LEFT JOIN information_schema.be_tablets tbt ON pm.PARTITION_ID = tbt.PARTITION_ID
  GROUP BY
    pm.DB_NAME,
    pm.TABLE_NAME,
    pm.PARTITION_NAME
),
MaxPartitionData AS (
  SELECT
    DB_NAME,
    TABLE_NAME,
    MAX(partition_data_size_gb) AS max_partition_data_size_gb
  FROM
    PartitionData
  GROUP BY
    DB_NAME,
    TABLE_NAME
),
TableStats AS (
  SELECT
    pd.DB_NAME,
    pd.TABLE_NAME,
    pd.max_partition_data_size_gb AS partition_data_size_gb,
    CASE
      WHEN tc.DISTRIBUTE_BUCKET = 0 THEN (
        SELECT
          be_node_count * 2
        FROM
          BECOUNT
      )
      ELSE tc.DISTRIBUTE_BUCKET
    END AS buckets
  FROM
    MaxPartitionData pd
    JOIN information_schema.tables_config tc ON pd.DB_NAME = tc.TABLE_SCHEMA
    AND pd.TABLE_NAME = tc.TABLE_NAME
)
SELECT
  ts.DB_NAME,
  ts.TABLE_NAME,
  ts.partition_data_size_gb,
  ts.buckets,
  ts.partition_data_size_gb / ts.buckets AS capacity_per_bucket_gb
FROM
  TableStats ts
WHERE
  ts.partition_data_size_gb / ts.buckets > 1
ORDER BY
  capacity_per_bucket_gb DESC`)
	var m []map[string]interface{}
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return "", -1
	}
	var bbodys [][]string
	for _, m2 := range m {
		bbodys = append(bbodys, []string{m2["DB_NAME"].(string), m2["TABLE_NAME"].(string),
			fmt.Sprintf("%.2f", m2["partition_data_size_gb"].(float64)), fmt.Sprintf("%d", m2["buckets"].(int64)),
			fmt.Sprintf("%.2f", m2["capacity_per_bucket_gb"].(float64)),
		})
	}

	return bodySubject(subject{
		Title2:  "数据倾斜",
		Thead:   []string{"库名", "表名", "分区大小(max-gb)", "分桶大小", "每个桶容量(gb)"},
		Tbody:   bbodys,
		Comment: `<h2 class="section-title"><p class="tooltip" data-tooltip="底层分片倾斜">数据倾斜</p></h2>`,
	}), len(m)
}

// 主键持久化巡检
func primaryIndex(db *gorm.DB) (string, int) {
	stmt := fmt.Sprintf(`
-- 查找没有开启主键索引落盘的表
WITH TableDataSize AS (
    -- 计算每个表的数据大小
    SELECT
        pm.DB_NAME,
        pm.TABLE_NAME,
        COALESCE(SUM(tbt.DATA_SIZE), 0) / (1024 * 1024 * 1024) AS total_data_size_gb  -- 转换为 GB
    FROM
        information_schema.partitions_meta pm
    LEFT JOIN
        information_schema.be_tablets tbt ON pm.PARTITION_ID = tbt.PARTITION_ID
    GROUP BY
        pm.DB_NAME,
        pm.TABLE_NAME
)
SELECT
    tc.TABLE_SCHEMA AS "database_name",
    tc.TABLE_NAME AS "table_name",
    tc.properties,
    td.total_data_size_gb AS "data_size",  -- 数据大小
    td.total_data_size_gb * 0.15 AS "estimated_memory_usage"  -- 估算内存占用（假设为数据大小的 15%%）
FROM
    information_schema.tables_config tc
LEFT JOIN
    TableDataSize td ON tc.TABLE_SCHEMA = td.DB_NAME AND tc.TABLE_NAME = td.TABLE_NAME
WHERE
    tc.table_model = "PRIMARY_KEYS"
    AND tc.properties LIKE '%%enable_persistent_index":"false"%%'
ORDER BY
    td.total_data_size_gb DESC`)

	var m []map[string]interface{}
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return "", -1
	}
	if m == nil {
		return "", 0
	}
	var bbodys [][]string
	for _, m2 := range m {
		bbodys = append(bbodys, []string{m2["database_name"].(string), m2["table_name"].(string),
			fmt.Sprintf("%.2f", m2["data_size"].(float64)),
			fmt.Sprintf("%.2f", m2["estimated_memory_usage"].(float64)),
		})
	}
	return bodySubject(subject{
		Title2:  "主键索引未开启持久化",
		Thead:   []string{"库名", "表名", "表大小(GB)", "估计内存使用量(GB)"},
		Tbody:   bbodys,
		Comment: `<h2 class="section-title"><p class="tooltip" data-tooltip="未开启持久化主键索引 (enable_persistent_index=false)。这可能导致重启后主键索引重建时间过长，影响服务可用性。建议开启此选项。">主键索引未开启持久化</p></h2>`,
	}), len(m)
}
