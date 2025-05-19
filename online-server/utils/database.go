// Package utils 提供数据库连接和操作的核心功能
package utils

import (
	"fmt"
	"os"
	"time"

	"online-server/model"

	"github.com/ceyewan/clog"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var (
	// DB 是数据库连接的全局变量
	DB *gorm.DB
	// instance 是数据库连接的单例
	instance *gorm.DB
)

// InitDB 初始化数据库连接并进行配置
// 创建数据目录、连接数据库、配置连接池参数、迁移数据模型并确保管理员用户存在
//
// 返回：
//   - 如果初始化过程中发生错误，则返回相应的错误信息
func InitDB() error {
	dbLogger := clog.Module("database", clog.Config{
		EnableCaller: true,
	})
	dbLogger.Info("开始初始化数据库")

	// 确保数据目录存在
	if err := os.MkdirAll("database", 0755); err != nil {
		dbLogger.Error("创建数据目录失败", clog.Err(err))
		return fmt.Errorf("创建数据目录失败: %w", err)
	}

	dbLogger.Info("数据目录创建成功")

	// 配置GORM日志
	newLogger := logger.New(
		NewGormLogWriter(), // 使用封装的clog写入器
		logger.Config{
			SlowThreshold: time.Second,
			LogLevel:      logger.Warn,
			Colorful:      true,
		},
	)

	// 连接SQLite数据库
	dbLogger.Info("连接SQLite数据库", clog.String("path", "database/crypto-custody.db"))
	db, err := gorm.Open(sqlite.Open("database/crypto-custody.db"), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		dbLogger.Error("数据库连接失败", clog.Err(err))
		return fmt.Errorf("数据库连接失败: %w", err)
	}

	// 设置连接池
	dbLogger.Info("配置数据库连接池")
	sqlDB, err := db.DB()
	if err != nil {
		dbLogger.Error("获取数据库连接失败", clog.Err(err))
		return fmt.Errorf("获取数据库连接失败: %w", err)
	}

	// 设置连接池参数
	sqlDB.SetMaxIdleConns(10)
	sqlDB.SetMaxOpenConns(100)
	sqlDB.SetConnMaxLifetime(time.Hour)
	dbLogger.Info("数据库连接池配置完成",
		clog.Int("max_idle", 10),
		clog.Int("max_open", 100),
		clog.Duration("lifetime", time.Hour))

	// 保存实例
	instance = db
	DB = db

	// 自动迁移数据库模型
	dbLogger.Info("开始自动迁移数据库模型")
	if err := autoMigrateModels(); err != nil {
		dbLogger.Error("数据库迁移失败", clog.Err(err))
		return fmt.Errorf("数据库迁移失败: %w", err)
	}
	dbLogger.Info("数据库模型迁移成功")

	// 检查并创建管理员用户
	dbLogger.Info("检查管理员用户")
	if err := ensureAdminUser(); err != nil {
		dbLogger.Error("创建管理员用户失败", clog.Err(err))
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	dbLogger.Info("数据库初始化完成")
	return nil
}

// CloseDB 关闭数据库连接
func CloseDB() {
	dbLogger := clog.Module("database")
	if instance != nil {
		dbLogger.Info("关闭数据库连接")
		sqlDB, err := instance.DB()
		if err != nil {
			dbLogger.Error("获取SQL DB实例失败", clog.Err(err))
			return
		}
		if err := sqlDB.Close(); err != nil {
			dbLogger.Error("关闭数据库连接失败", clog.Err(err))
		} else {
			dbLogger.Info("数据库连接已关闭")
		}
	}
}

// autoMigrateModels 自动迁移所有数据库模型
// 确保数据库表结构与定义的模型结构一致
//
// 返回：
//   - 如果迁移过程中发生错误，则返回相应的错误信息
func autoMigrateModels() error {
	dbLogger := clog.Module("database")
	if instance == nil {
		dbLogger.Error("数据库未初始化")
		return fmt.Errorf("数据库未初始化")
	}

	// 迁移所有模型
	dbLogger.Info("迁移数据模型",
		clog.String("model", "User"),
		clog.String("model", "Account"),
		clog.String("model", "Transaction"))

	return instance.AutoMigrate(
		&model.User{},
		&model.Account{},
		&model.Transaction{},
	)
}

// AutoMigrate 自动迁移指定的数据库模型
// 用于在需要添加新模型时调用，不需要重新迁移所有已有模型
//
// 参数：
//   - models: 需要迁移的模型列表
//
// 返回：
//   - 如果迁移过程中发生错误，则返回相应的错误信息
func AutoMigrate(models ...interface{}) error {
	dbLogger := clog.Module("database")
	if instance == nil {
		dbLogger.Error("数据库未初始化")
		return fmt.Errorf("数据库未初始化")
	}
	dbLogger.Info("自动迁移指定模型", clog.Int("model_count", len(models)))
	return instance.AutoMigrate(models...)
}

// GetDB 获取数据库连接实例
// 如果实例不存在，则自动初始化数据库
//
// 返回：
//   - 数据库连接的GORM实例
func GetDB() *gorm.DB {
	dbLogger := clog.Module("database")
	if instance == nil {
		dbLogger.Warn("数据库未初始化，现在正在初始化...")
		if err := InitDB(); err != nil {
			dbLogger.Fatal("数据库初始化失败", clog.Err(err))
			return nil
		}
		dbLogger.Info("数据库初始化成功")
	}
	return instance
}

// SetDB 设置数据库连接实例
// 主要用于测试环境中设置测试数据库
//
// 参数：
//   - db: 要设置的数据库连接实例
func SetDB(db *gorm.DB) {
	dbLogger := clog.Module("database")
	dbLogger.Info("手动设置数据库连接")
	instance = db
	DB = db
}

// ensureAdminUser 确保管理员用户存在
// 如果不存在则创建默认的管理员用户
//
// 返回：
//   - 如果创建过程中发生错误，则返回相应的错误信息
func ensureAdminUser() error {
	dbLogger := clog.Module("database")

	if instance == nil {
		dbLogger.Error("数据库未初始化")
		return fmt.Errorf("数据库未初始化")
	}

	// 检查是否已存在admin用户
	var count int64
	instance.Model(&model.User{}).Where("username = ?", "admin").Count(&count)
	dbLogger.Info("检查管理员用户", clog.Int64("found", count))

	// 如果不存在admin用户，则创建
	if count == 0 {
		dbLogger.Info("未检测到管理员用户，正在创建默认管理员...")

		// 设置默认管理员密码
		defaultPassword := os.Getenv("DEFAULT_ADMIN_PASSWORD")

		// 使用bcrypt生成密码哈希
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(defaultPassword), bcrypt.DefaultCost)
		if err != nil {
			dbLogger.Error("生成密码哈希失败", clog.Err(err))
			return fmt.Errorf("生成密码哈希失败: %w", err)
		}

		// 创建默认admin用户
		admin := model.User{
			Username: "admin",
			Password: string(hashedPassword),
			Email:    "admin@example.com",
			Role:     model.RoleAdmin,
		}

		if err := instance.Create(&admin).Error; err != nil {
			dbLogger.Error("创建管理员用户失败", clog.Err(err))
			return fmt.Errorf("创建管理员用户失败: %w", err)
		}

		dbLogger.Info("默认管理员用户创建成功",
			clog.String("username", "admin"),
			clog.String("password", "admin123"),
			clog.String("role", string(model.RoleAdmin)))
	} else {
		dbLogger.Info("管理员用户已存在，无需创建")
	}

	return nil
}
