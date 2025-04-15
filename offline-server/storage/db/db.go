package db

import (
	"fmt"
	"log"
	"offline-server/storage/model"
	"os"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// instance 是数据库连接的单例
	instance *gorm.DB
)

// Init 初始化数据库连接
func Init() error {
	// 确保数据目录存在
	if err := os.MkdirAll("data", 0755); err != nil {
		return fmt.Errorf("创建数据目录失败: %w", err)
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

	// 连接SQLite数据库
	db, err := gorm.Open(sqlite.Open("data/crypto-custody.db"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 设置连接池
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 保存实例
	instance = db

	// 自动迁移数据库模型
	if err := autoMigrateModels(); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	return nil
}

// autoMigrateModels 自动迁移所有数据库模型
func autoMigrateModels() error {
	if instance == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 迁移所有模型
	return instance.AutoMigrate(
		&model.UserShare{},
		&model.KeyGenSession{},
		&model.SignSession{},
		&model.User{},
	)
}

// AutoMigrate 自动迁移数据库模型
func AutoMigrate(models ...interface{}) error {
	if instance == nil {
		return fmt.Errorf("数据库未初始化")
	}
	return instance.AutoMigrate(models...)
}

// GetDB 获取数据库连接实例
func GetDB() *gorm.DB {
	if instance == nil {
		log.Fatal("数据库未初始化")
	}
	return instance
}
