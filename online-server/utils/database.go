// Package utils 提供数据库连接和操作的核心功能
package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	dbPath := databasePath()
	dbDir := filepath.Dir(dbPath)

	// 确保数据目录存在
	if err := os.MkdirAll(dbDir, 0755); err != nil {
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
	dbLogger.Info("连接SQLite数据库", clog.String("path", dbPath))
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
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
		instance = nil
		DB = nil
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
		clog.String("model", "Transaction"),
		clog.String("model", "Case"),
		clog.String("model", "OfflineTask"),
		clog.String("model", "AuditLog"),
		clog.String("model", "BackupRecord"),
		clog.String("model", "Job"))

	if err := instance.AutoMigrate(
		&model.User{},
		&model.Account{},
		&model.Transaction{},
		&model.Case{},
		&model.OfflineTask{},
		&model.AuditLog{},
		&model.BackupRecord{},
		&model.Job{},
	); err != nil {
		return err
	}

	return migrateTransactionHashIndexes()
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

	if err := migrateLegacyUserRoles(); err != nil {
		return err
	}

	adminPassword := strings.TrimSpace(os.Getenv("DEFAULT_ADMIN_PASSWORD"))
	if adminPassword == "" {
		adminPassword = "admin123"
	}
	if err := ensureDefaultUser("admin", "系统管理员", "admin@example.com", adminPassword, model.RoleAdmin); err != nil {
		return err
	}

	defaultOfficers := []struct {
		username string
		nickname string
		email    string
	}{
		{"u1", "测试警员 1", "u1@example.com"},
		{"u2", "测试警员 2", "u2@example.com"},
		{"u3", "测试警员 3", "u3@example.com"},
	}
	for _, officer := range defaultOfficers {
		if err := ensureDefaultUser(officer.username, officer.nickname, officer.email, "officer123", model.RoleOfficer); err != nil {
			return err
		}
	}

	if err := ensureDefaultUser("auditor", "审计员", "auditor@example.com", "auditor123", model.RoleAuditor); err != nil {
		return err
	}

	return nil
}

func migrateLegacyUserRoles() error {
	if err := instance.Model(&model.User{}).
		Where("role = ?", "guest").
		Update("role", model.RoleOfficer).Error; err != nil {
		return fmt.Errorf("迁移历史用户角色失败: %w", err)
	}
	return nil
}

func migrateTransactionHashIndexes() error {
	if instance == nil {
		return fmt.Errorf("数据库未初始化")
	}
	if instance.Dialector.Name() != "sqlite" {
		return nil
	}

	var schema string
	if err := instance.Raw("SELECT sql FROM sqlite_master WHERE type = 'table' AND name = 'transactions'").Scan(&schema).Error; err != nil {
		return fmt.Errorf("读取交易表结构失败: %w", err)
	}
	if schema == "" || (!strings.Contains(schema, "uni_transactions_message_hash") && !strings.Contains(schema, "uni_transactions_tx_hash")) {
		return nil
	}

	dbLogger := clog.Module("database")
	dbLogger.Info("迁移交易表哈希索引，移除空哈希不应触发的唯一约束")

	return instance.Transaction(func(tx *gorm.DB) error {
		statements := []string{
			"DROP TABLE IF EXISTS `transactions_without_hash_unique`",
			"ALTER TABLE `transactions` RENAME TO `transactions_without_hash_unique`",
			`CREATE TABLE "transactions" (
				"id" integer PRIMARY KEY AUTOINCREMENT,
				"created_at" datetime,
				"updated_at" datetime,
				"deleted_at" datetime,
				"tx_no" text,
				"case_id" integer,
				"case_no" text,
				"tx_type" text DEFAULT "withdraw",
				"from_account_id" integer,
				"from_address" text NOT NULL,
				"to_address" text NOT NULL,
				"value" text NOT NULL,
				"coin_type" text DEFAULT "ETH",
				"reason" text,
				"unsigned_payload" blob,
				"message_hash" text,
				"tx_hash" text,
				"signature" blob,
				"receipt" blob,
				"status" integer NOT NULL,
				"created_by" text,
				"approved_by" text,
				"exported_at" integer,
				"signed_at" integer,
				"broadcasted_at" integer,
				"confirmed_at" integer
			)`,
			`INSERT INTO "transactions" (
				"id", "created_at", "updated_at", "deleted_at", "tx_no", "case_id", "case_no", "tx_type",
				"from_account_id", "from_address", "to_address", "value", "coin_type", "reason",
				"unsigned_payload", "message_hash", "tx_hash", "signature", "receipt", "status",
				"created_by", "approved_by", "exported_at", "signed_at", "broadcasted_at", "confirmed_at"
			)
			SELECT
				"id", "created_at", "updated_at", "deleted_at", "tx_no", "case_id", "case_no", "tx_type",
				"from_account_id", "from_address", "to_address", "value", "coin_type", "reason",
				"unsigned_payload", "message_hash", "tx_hash", "signature", "receipt", "status",
				"created_by", "approved_by", "exported_at", "signed_at", "broadcasted_at", "confirmed_at"
			FROM "transactions_without_hash_unique"`,
			"DROP TABLE `transactions_without_hash_unique`",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_deleted_at` ON `transactions`(`deleted_at`)",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_tx_no` ON `transactions`(`tx_no`)",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_case_id` ON `transactions`(`case_id`)",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_case_no` ON `transactions`(`case_no`)",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_tx_type` ON `transactions`(`tx_type`)",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_from_account_id` ON `transactions`(`from_account_id`)",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_from_address` ON `transactions`(`from_address`)",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_to_address` ON `transactions`(`to_address`)",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_message_hash` ON `transactions`(`message_hash`)",
			"CREATE INDEX IF NOT EXISTS `idx_transactions_tx_hash` ON `transactions`(`tx_hash`)",
		}
		for _, statement := range statements {
			if err := tx.Exec(statement).Error; err != nil {
				return fmt.Errorf("迁移交易表哈希索引失败: %w", err)
			}
		}
		return nil
	})
}

func ensureDefaultUser(username, nickname, email, password string, role model.Role) error {
	dbLogger := clog.Module("database")

	var user model.User
	if err := instance.Where("username = ?", username).First(&user).Error; err == nil {
		updates := map[string]interface{}{
			"role":   role,
			"status": model.UserStatusActive,
		}
		if user.Nickname == "" && nickname != "" {
			updates["nickname"] = nickname
		}
		if user.Email == "" {
			updates["email"] = email
		}
		if err := instance.Model(&user).Updates(updates).Error; err != nil {
			return fmt.Errorf("更新默认用户 %s 失败: %w", username, err)
		}
		dbLogger.Info("默认用户已存在，已校正基础属性", clog.String("username", username), clog.String("role", string(role)))
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		dbLogger.Error("生成密码哈希失败", clog.String("username", username), clog.Err(err))
		return fmt.Errorf("生成默认用户密码哈希失败: %w", err)
	}

	newUser := model.User{
		Username: username,
		Nickname: nickname,
		Password: string(hashedPassword),
		Email:    email,
		Role:     role,
		Status:   model.UserStatusActive,
	}

	if err := instance.Create(&newUser).Error; err != nil {
		dbLogger.Error("创建默认用户失败", clog.String("username", username), clog.Err(err))
		return fmt.Errorf("创建默认用户 %s 失败: %w", username, err)
	}

	dbLogger.Info("默认用户创建成功", clog.String("username", username), clog.String("role", string(role)))
	return nil
}

func databasePath() string {
	value := strings.TrimSpace(os.Getenv("DATABASE_PATH"))
	if value == "" {
		return "database/crypto-custody.db"
	}
	return value
}
