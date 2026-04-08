/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package app
 *@file    api_metadata
 *@date    2025/11/5 15:09
 */

package app

import (
	"StarRocksProbe/util"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
	"gorm.io/gorm"
	"net/http"
	"time"
)

type replicadata struct {
	Table              string `json:"table"`
	TabletCount        int64  `json:"tablet_count"`
	TotalDataSizeGb    string `json:"total_data_size_gb"`
	MaxPartitionSizeGb string `json:"max_partition_size_gb"`
	Buckets            int64  `json:"buckets"`
	Comment            string `json:"comment"`
	//information_schema.tables
	CreateTime string `json:"create_time"`
}

func (engine *threadMap) replica(c *gin.Context) {

	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)

	if v, ok := apicache.Get(appid + "replicadata"); ok {
		c.JSON(http.StatusOK, gin.H{"data": v.([]replicadata), "running": 0, "pending": 0, "total": len(v.([]replicadata))})
		util.Loggrs.Info(c.ClientIP(), " replica response:cache")
		return
	}

	db, err := engine.getmapConnect(appid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}
	meta, err := meta_replica(appid, db)
	if err != nil {
		c.JSON(http.StatusInternalServerError, meta)
		return
	}
	apicache.Set(appid+"replicadata", meta, cache.DefaultExpiration)
	util.Loggrs.Info(c.ClientIP(), " replica response:actual")
	c.JSON(http.StatusOK, gin.H{"data": meta, "running": 0, "pending": 0, "total": len(meta)})

}

func meta_replica(app string, db *gorm.DB) ([]replicadata, error) {

	stmt := `

WITH BECOUNT AS (
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
COALESCE(SUM(tbt.DATA_SIZE), 0) AS partition_data_size,
COUNT(tbt.TABLET_ID) AS tablet_count
FROM
information_schema.partitions_meta pm
LEFT JOIN
information_schema.be_tablets tbt ON pm.PARTITION_ID = tbt.PARTITION_ID
GROUP BY
pm.DB_NAME,
pm.TABLE_NAME,
pm.PARTITION_NAME
),
TableStats AS (
SELECT
pd.DB_NAME,
pd.TABLE_NAME,
SUM(pd.tablet_count) AS tablet_count,
SUM(pd.partition_data_size) / (1024 * 1024 * 1024) AS
total_data_size_gb, 
CASE
WHEN SUM(pd.tablet_count) > 0 THEN SUM(pd.partition_data_size) /
SUM(pd.tablet_count) / (1024 * 1024 * 1024)
ELSE 0
END AS avg_data_size_per_tablet_gb,
MAX(pd.partition_data_size) / (1024 * 1024 * 1024) AS
max_partition_size_gb, 
COUNT(DISTINCT pd.PARTITION_NAME) AS partition_count,
CASE
WHEN tc.DISTRIBUTE_BUCKET = 0 THEN (SELECT be_node_count * 2 FROM
BECOUNT) 
ELSE tc.DISTRIBUTE_BUCKET
END AS buckets 
FROM
PartitionData pd
JOIN
information_schema.tables_config tc ON pd.DB_NAME = tc.TABLE_SCHEMA AND
pd.TABLE_NAME = tc.TABLE_NAME
CROSS JOIN
BECOUNT
GROUP BY
pd.DB_NAME,
pd.TABLE_NAME,
tc.DISTRIBUTE_BUCKET
)
SELECT
ts.DB_NAME,
ts.TABLE_NAME,
ts.tablet_count,
ts.total_data_size_gb,
ts.avg_data_size_per_tablet_gb,
CASE
WHEN ts.partition_count > 1 THEN ts.max_partition_size_gb
ELSE NULL
END AS max_partition_size_gb,
ts.buckets,
CASE
WHEN ts.partition_count > 1 THEN ts.max_partition_size_gb / ts.buckets
ELSE NULL
END AS max_partition_size_per_bucket_gb
FROM
TableStats ts
ORDER BY
ts.tablet_count DESC
LIMIT 50
`
	var m []map[string]interface{}
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		return nil, r.Error
	}
	var meta []replicadata
	for _, item := range m {
		tablet_count := item["tablet_count"].(int64)
		if tablet_count < 30000 {
			continue
		}

		if val, ok := apicache.Get(app + "information_schema_tables"); ok {
			pool := find_meta(val.([]map[string]interface{}), fmt.Sprintf("%s.%s", item["DB_NAME"].(string), item["TABLE_NAME"].(string)))
			if pool["TABLE_NAME"] != nil {
				meta = append(meta, replicadata{
					Table:              fmt.Sprintf("%s.%s", item["DB_NAME"].(string), item["TABLE_NAME"].(string)),
					TabletCount:        tablet_count,
					TotalDataSizeGb:    fmt.Sprintf("%.2f", item["total_data_size_gb"].(float64)),
					MaxPartitionSizeGb: fmt.Sprintf("%.2f", item["max_partition_size_gb"].(float64)),
					Buckets:            item["buckets"].(int64),
					Comment:            pool["TABLE_COMMENT"].(string),
					CreateTime:         pool["CREATE_TIME"].(time.Time).Format("2006-01-02 15:04:05"),
				})
			} else {
				meta = append(meta, replicadata{
					Table:              fmt.Sprintf("%s.%s", item["DB_NAME"].(string), item["TABLE_NAME"].(string)),
					TabletCount:        tablet_count,
					TotalDataSizeGb:    fmt.Sprintf("%.2f", item["total_data_size_gb"].(float64)),
					MaxPartitionSizeGb: fmt.Sprintf("%.2f", item["max_partition_size_gb"].(float64)),
					Buckets:            item["buckets"].(int64),
					Comment:            "",
				})
			}
		} else {
			meta = append(meta, replicadata{
				Table:              fmt.Sprintf("%s.%s", item["DB_NAME"].(string), item["TABLE_NAME"].(string)),
				TabletCount:        tablet_count,
				TotalDataSizeGb:    fmt.Sprintf("%.2f", item["total_data_size_gb"].(float64)),
				MaxPartitionSizeGb: fmt.Sprintf("%.2f", item["max_partition_size_gb"].(float64)),
				Buckets:            item["buckets"].(int64),
				Comment:            "",
			})
		}
	}
	return meta, nil
}
