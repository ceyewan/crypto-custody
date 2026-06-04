// Package db 提供数据库连接和操作的核心功能
package db

import (
	"fmt"
	"log"
	"offline-server/storage/model"
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

// Init 初始化数据库连接并进行配置
// 创建数据目录、连接数据库、配置连接池参数、迁移数据模型并确保管理员用户存在
//
// 返回：
//   - 如果初始化过程中发生错误，则返回相应的错误信息
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

	if err := migrateLegacyUserRoles(); err != nil {
		return fmt.Errorf("迁移用户角色失败: %w", err)
	}

	// 检查并创建管理员用户
	if err := ensureAdminUser(); err != nil {
		return fmt.Errorf("创建管理员用户失败: %w", err)
	}

	return nil
}

func migrateLegacyUserRoles() error {
	if instance == nil {
		return fmt.Errorf("数据库未初始化")
	}

	updates := map[string]model.Role{
		"coordinator": model.RoleAdmin,
		"participant": model.RoleOfficer,
		"guest":       model.RoleOfficer,
	}
	for oldRole, newRole := range updates {
		if err := instance.Model(&model.User{}).
			Where("role = ?", oldRole).
			Update("role", newRole).Error; err != nil {
			return err
		}
	}
	return nil
}

// autoMigrateModels 自动迁移所有数据库模型
// 确保数据库表结构与定义的模型结构一致
//
// 返回：
//   - 如果迁移过程中发生错误，则返回相应的错误信息
func autoMigrateModels() error {
	if instance == nil {
		return fmt.Errorf("数据库未初始化")
	}

	// 迁移所有模型
	return instance.AutoMigrate(
		&model.OfflineTask{},
		&model.OfflineKey{},
		&model.KeyShard{},
		&model.KeyGenSession{},
		&model.SignSession{},
		&model.Approval{},
		&model.AuditLog{},
		&model.BackupRecord{},
		&model.User{},
		&model.Case{},
		&model.Se{},
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
	if instance == nil {
		return fmt.Errorf("数据库未初始化")
	}
	return instance.AutoMigrate(models...)
}

// Close 关闭当前数据库连接，恢复数据库文件前调用。
func Close() {
	if instance == nil {
		return
	}
	sqlDB, err := instance.DB()
	if err == nil {
		_ = sqlDB.Close()
	}
	instance = nil
}

// GetDB 获取数据库连接实例
// 如果实例不存在，则自动初始化数据库
//
// 返回：
//   - 数据库连接的GORM实例
func GetDB() *gorm.DB {
	if instance == nil {
		log.Println("数据库未初始化，现在正在初始化...")
		if err := Init(); err != nil {
			log.Fatalf("数据库初始化失败: %v", err)
			return nil
		}
		log.Println("数据库初始化成功")
	}
	return instance
}

// ensureAdminUser 确保管理员用户存在
// 如果不存在则创建默认的管理员用户
//
// 返回：
//   - 如果创建过程中发生错误，则返回相应的错误信息
func ensureAdminUser() error {
	if instance == nil {
		return fmt.Errorf("数据库未初始化")
	}

	if err := ensureDefaultUser("admin", "系统管理员", "admin@example.com", "admin123", model.RoleAdmin); err != nil {
		return err
	}

	defaultOfficers := []struct {
		username string
		nickname string
		email    string
	}{
		{"u1", "测试警员1", "u1@offline.local"},
		{"u2", "测试警员2", "u2@offline.local"},
		{"u3", "测试警员3", "u3@offline.local"},
	}
	for _, officer := range defaultOfficers {
		if err := ensureDefaultUser(officer.username, officer.nickname, officer.email, "officer123", model.RoleOfficer); err != nil {
			return err
		}
	}

	if err := ensureDefaultUser("auditor", "测试审计员", "auditor@offline.local", "auditor123", model.RoleAuditor); err != nil {
		return err
	}

	return nil
}

func ensureDefaultUser(username, nickname, email, password string, role model.Role) error {
	var count int64
	instance.Model(&model.User{}).Where("username = ?", username).Count(&count)
	if count > 0 {
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("生成默认用户密码哈希失败: %w", err)
	}

	user := model.User{
		Username: username,
		Nickname: nickname,
		Password: string(hashedPassword),
		Email:    email,
		Role:     role,
		Status:   model.UserStatusActive,
	}

	if err := instance.Create(&user).Error; err != nil {
		return fmt.Errorf("创建默认用户 %s 失败: %w", username, err)
	}

	log.Printf("默认用户创建成功: %s (%s)", username, role)
	return nil
}
