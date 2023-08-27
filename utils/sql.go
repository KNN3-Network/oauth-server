package utils

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

var db *gorm.DB

type OauthBind struct {
	Addr         string `json:"addr" gorm:"column:addr;primaryKey"`
	Github       string `json:"github"`
	Gmail        string `json:"gmail"`
	Discord      string `json:"discord"`
	DiscordName  string `json:"discord_name"`
	Exchange     string `json:"exchange"`
	ExchangeName string `json:"exchange_name"`
}

func (OauthBind) TableName() string {
	return "oauth_bind"
}

func init() {
	var err error
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	// 连接到 MySQL 服务器
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/lens?charset=utf8mb4&parseTime=True&loc=Local",
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOST"),
		os.Getenv("DB_PORT"),
	)
	db, err = gorm.Open(mysql.New(mysql.Config{
		DSN: dsn,
	}), &gorm.Config{})

	if err != nil {
		log.Fatal(err)
	}
	// 获取通用数据库对象 sql.DB ，然后使用其提供的功能
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("failed to get underlying sql.DB: %v", err)
	}

	// SetMaxIdleConns 用于设置连接池中空闲连接的最大数量。
	sqlDB.SetMaxIdleConns(10)

	// SetMaxOpenConns 设置打开数据库连接的最大数量。
	sqlDB.SetMaxOpenConns(100)
}

func GetDB() *gorm.DB {
	return db
}
