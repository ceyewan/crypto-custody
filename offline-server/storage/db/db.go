package db

import (
	"fmt"
	"log"
	"offline-server/storage/model"
	"offline-server/tools"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
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

	// 检查并创建管理员用户
	if err := ensureAdminUser(); err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
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
		log.Fatal("数据库未初始化，现在正在初始化...")
		if err := Init(); err != nil {
			log.Fatalf("数据库初始化失败: %v", err)
			return nil
		}
		log.Println("数据库初始化成功")
	}
	return instance
}

// ensureAdminUser 确保管理员用户存在
func ensureAdminUser() error {
	if instance == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 检查是否已存在admin用户
	var count int64
	instance.Model(&model.User{}).Where("username = ?", "admin").Count(&count)

	// 如果不存在admin用户，则创建
	if count == 0 {
		log.Println("未检测到管理员用户，正在创建默认管理员...")

		// 设置默认管理员密码
		defaultPassword := "admin123"

		// 使用bcrypt生成密码哈希
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			return fmt.Errorf("生成密码哈希失败: %w", err)
		}

		// 创建默认admin用户
		admin := model.User{
			Username: "admin",
			Password: string(hashedPassword),
			Email:    "admin@example.com",
			Role:     string(tools.Admin),
		}

		if err := instance.Create(&admin).Error; err != nil {
			return fmt.Errorf("创建管理员用户失败: %w", err)
		}

		log.Println("默认管理员用户创建成功: admin (默认密码: admin123)")
	}

	return nil
}
