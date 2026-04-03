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

type metadata struct {
	Table               string `json:"table"`
	PartitionName       string `json:"partition_name"`
	PartitionDataSizeGb string `json:"partition_data_size_gb"`
	Buckets             int64  `json:"buckets"`
	TabletDataSizeGb    string `json:"tablet_data_size_gb"`
	Comment             string `json:"comment"`
	//information_schema.tables
	CreateTime string `json:"create_time"`
}

func (engine *threadMap) metadata(c *gin.Context) {

	appid := c.GetHeader("AppID")
	if appid == "" {
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "app is nil."})
		return
	}
	appid = setdefault(appid)
	if v, ok := apicache.Get(appid + "metadata"); ok {
		c.JSON(http.StatusOK, gin.H{"data": v.([]metadata), "running": 0, "pending": 0, "total": len(v.([]metadata))})
		util.Loggrs.Info(c.ClientIP(), " meta response:cache")
		return
	}

	db, err := engine.getmapConnect(appid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, nil)
		return
	}
	meta, err := meta_tablet(appid, db)
	if err != nil {
		util.Loggrs.Error(err.Error())
		c.JSON(http.StatusInternalServerError, nil)
		return
	}
	apicache.Set(appid+"metadata", meta, cache.DefaultExpiration)
	util.Loggrs.Info(c.ClientIP(), " meta response:actual")
	c.JSON(http.StatusOK, gin.H{"data": meta, "running": 0, "pending": 0, "total": len(meta)})

}

func meta_tablet(app string, db *gorm.DB) ([]metadata, error) {
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
    COALESCE(SUM(tbt.DATA_SIZE), 0) / (1024 * 1024 * 1024) AS partition_data_size_gb
  FROM
    information_schema.partitions_meta pm
    LEFT JOIN information_schema.be_tablets tbt ON pm.PARTITION_ID = tbt.PARTITION_ID
  GROUP BY
    pm.DB_NAME,
    pm.TABLE_NAME,
    pm.PARTITION_NAME
),
TableStats AS (
  SELECT
    pd.DB_NAME,
    pd.TABLE_NAME,
    pd.PARTITION_NAME,
    pd.partition_data_size_gb,
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
    PartitionData pd
    JOIN information_schema.tables_config tc ON pd.DB_NAME = tc.TABLE_SCHEMA
    AND pd.TABLE_NAME = tc.TABLE_NAME
    CROSS JOIN BECOUNT
)
SELECT
  ts.DB_NAME,
  ts.TABLE_NAME,
  ts.PARTITION_NAME,
  ts.partition_data_size_gb,
  ts.buckets,
  ts.partition_data_size_gb / ts.buckets AS capacity_per_bucket_gb
FROM
  TableStats ts
ORDER BY
  capacity_per_bucket_gb DESC
LIMIT
  100
`
	var m []map[string]interface{}
	r := db.Raw(stmt).Scan(&m)
	if r.Error != nil {
		util.Loggrs.Error(r.Error.Error())
		return nil, r.Error
	}
	var meta []metadata
	for _, item := range m {
		tablet_size := item["partition_data_size_gb"].(float64) / (float64(item["buckets"].(int64)) * 3)
		if tablet_size < 3 {
			continue
		}

		if val, ok := apicache.Get(app + "information_schema_tables"); ok {
			pool := find_meta(val.([]map[string]interface{}), fmt.Sprintf("%s.%s", item["DB_NAME"].(string), item["TABLE_NAME"].(string)))
			if pool["TABLE_NAME"] != nil {
				meta = append(meta, metadata{
					Table:               fmt.Sprintf("%s.%s", item["DB_NAME"].(string), item["TABLE_NAME"].(string)),
					PartitionName:       item["PARTITION_NAME"].(string),
					PartitionDataSizeGb: fmt.Sprintf("%.2f", item["partition_data_size_gb"].(float64)),
					Buckets:             item["buckets"].(int64),
					TabletDataSizeGb:    fmt.Sprintf("%.2f", tablet_size),
					CreateTime:          pool["CREATE_TIME"].(time.Time).Format("2006-01-02 15:04:05"),
					Comment:             pool["TABLE_COMMENT"].(string),
				})
			} else {
				meta = append(meta, metadata{
					Table:               fmt.Sprintf("%s.%s", item["DB_NAME"].(string), item["TABLE_NAME"].(string)),
					PartitionName:       item["PARTITION_NAME"].(string),
					PartitionDataSizeGb: fmt.Sprintf("%.2f", item["partition_data_size_gb"].(float64)),
					Buckets:             item["buckets"].(int64),
					TabletDataSizeGb:    fmt.Sprintf("%.2f", tablet_size),
				})
			}
		} else {
			meta = append(meta, metadata{
				Table:               fmt.Sprintf("%s.%s", item["DB_NAME"].(string), item["TABLE_NAME"].(string)),
				PartitionName:       item["PARTITION_NAME"].(string),
				PartitionDataSizeGb: fmt.Sprintf("%.2f", item["partition_data_size_gb"].(float64)),
				Buckets:             item["buckets"].(int64),
				TabletDataSizeGb:    fmt.Sprintf("%.2f", tablet_size),
			})
		}
	}
	return meta, nil
}
