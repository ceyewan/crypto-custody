package utils

import (
	"backend/models"
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/ceyewan/clog"
)

var DB *gorm.DB

func ConnectDatabase() {
	// 确保数据目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		fmt.Println("创建数据目录失败:", err)
		panic("无法创建数据目录")
	}

	// 配置GORM日志
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Warn,
			Colorful:      true,
		},
	)

	var err error
	DB, err = gorm.Open(sqlite.Open("data/wallet_demo.db"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		fmt.Println("数据库连接失败:", err)
		panic("无法连接到数据库")
	}

	// 设置连接池
	sqlDB, err := DB.DB()
	if err != nil {
		fmt.Println("获取数据库连接失败:", err)
		panic("无法获取数据库连接")
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	clog.Info("数据库连接成功")

	// 自动迁移
	DB.AutoMigrate(&models.Account{})
	DB.AutoMigrate(&models.User{})
}
