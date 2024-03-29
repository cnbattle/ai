package ai

import (
	"fmt"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

var DB *gorm.DB

//DB=true
//DB_DSN=root:123456@(127.0.0.1:3306)/aiio?charset=utf8mb4&parseTime=true&loc=Local
//DB_PREFIX=

func init() {
	if GetDefaultEnvToBool("DB", false) {
		LOG.Trace("auto initialization DB")
		DB = InitGorm(GetEnv("DB_DSN"), GetEnv("DB_PREFIX"))
	}
}

func InitGorm(dsn, tablePrefix string) *gorm.DB {
	if len(dsn) == 0 {
		panic("dsn is empty")
	}
	Db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		SkipDefaultTransaction: true,
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   tablePrefix,
			SingularTable: true,
		},
		PrepareStmt:                              true,
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		panic(fmt.Sprintf("InitGorm err:%v,dsn:%+v", err, dsn))
	}
	return Db
}
