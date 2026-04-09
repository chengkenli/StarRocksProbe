/*
 *@author  chengkenli
 *@project StarRocksProbe
 *@package conn
 *@file    ConnectPostgreSQL
 *@date    2026/3/30 14:52
 */

package conn

import (
	"StarRocksProbe/util"
	"fmt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/schema"
	"time"
)

func ConnectItemPostgreSQL(item util.ConnectParms) (*gorm.DB, error) {
	newLogger := logger.New(nil,
		logger.Config{
			SlowThreshold: time.Second * 1000, // 控制慢SQL阈值
		},
	)
	dsn := fmt.Sprintf("user=%s password=%s host=%s port=%d dbname=manager_console sslmode=disable TimeZone=Asia/Shanghai",
		item.MetaUser,
		item.MetaPass,
		item.MetaHost,
		5432,
	)
	db, err := gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			SingularTable: true, // 使用单数表名
		},
		Logger: newLogger,
	})
	if err != nil {
		fmt.Println(err)
	}
	return db, err
}
