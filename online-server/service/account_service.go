package service

import (
	"errors"
	"fmt"
	"online-server/model"
	"online-server/utils"
	"sync"
)

var (
	accountServiceInstance     *AccountService
	accountServiceInstanceOnce sync.Once
)

// AccountService 提供账户管理相关的服务
type AccountService struct {
	db *utils.Database
}

// GetAccountServiceInstance 获取账户服务实例
func GetAccountServiceInstance() (*AccountService, error) {
	var initErr error

	accountServiceInstanceOnce.Do(func() {
		db, err := utils.GetDBInstance()
		if err != nil {
			initErr = fmt.Errorf("初始化数据库失败: %w", err)
			return
		}

		accountServiceInstance = &AccountService{
			db: db,
		}
	})

	if initErr != nil {
		return nil, initErr
	}

	return accountServiceInstance, nil
}

// GetAccounts 获取所有账户列表
func (s *AccountService) GetAccounts() ([]model.Account, error) {
	var accounts []model.Account
	result := s.db.Find(&accounts)
	if result.Error != nil {
		return nil, fmt.Errorf("获取账户列表失败: %w", result.Error)
	}
	return accounts, nil
}

// GetAccountsByUserID 获取指定用户ID的账户列表
func (s *AccountService) GetAccountsByUserID(userID uint) ([]model.Account, error) {
	var accounts []model.Account
	result := s.db.Where("user_id = ?", userID).Find(&accounts)
	if result.Error != nil {
		return nil, fmt.Errorf("获取用户账户列表失败: %w", result.Error)
	}
	return accounts, nil
}

// GetAccountByID 通过ID获取账户
func (s *AccountService) GetAccountByID(id uint) (*model.Account, error) {
	var account model.Account
	result := s.db.First(&account, id)
	if result.Error != nil {
		return nil, fmt.Errorf("获取账户失败: %w", result.Error)
	}
	return &account, nil
}

// GetAccountByAddress 通过地址获取账户
func (s *AccountService) GetAccountByAddress(address string) (*model.Account, error) {
	var account model.Account
	result := s.db.Where("address = ?", address).First(&account)
	if result.Error != nil {
		return nil, fmt.Errorf("获取账户失败: %w", result.Error)
	}
	return &account, nil
}

// CreateAccount 创建新账户
func (s *AccountService) CreateAccount(account *model.Account) error {
	// 检查地址是否已存在
	var existingAccount model.Account
	result := s.db.Where("address = ?", account.Address).First(&existingAccount)
	if result.Error == nil {
		return errors.New("账户地址已存在")
	}

	// 创建账户
	result = s.db.Create(account)
	if result.Error != nil {
		return fmt.Errorf("创建账户失败: %w", result.Error)
	}
	return nil
}

// UpdateAccount 更新账户信息
func (s *AccountService) UpdateAccount(account *model.Account) error {
	result := s.db.Save(account)
	if result.Error != nil {
		return fmt.Errorf("更新账户失败: %w", result.Error)
	}
	return nil
}

// UpdateAccountBalance 更新账户余额
func (s *AccountService) UpdateAccountBalance(id uint, newBalance string) error {
	result := s.db.Model(&model.Account{}).Where("id = ?", id).Update("balance", newBalance)
	if result.Error != nil {
		return fmt.Errorf("更新账户余额失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("未找到指定的账户")
	}
	return nil
}

// DeleteAccount 删除账户
func (s *AccountService) DeleteAccount(id uint) error {
	result := s.db.Delete(&model.Account{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除账户失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("未找到指定的账户")
	}
	return nil
}

// BatchCreateAccounts 批量创建账户
func (s *AccountService) BatchCreateAccounts(accounts []model.Account) error {
	// 使用事务确保批量操作的原子性
	tx := s.db.Begin()

	for _, account := range accounts {
		// 检查地址是否已存在
		var existingAccount model.Account
		result := tx.Where("address = ?", account.Address).First(&existingAccount)
		if result.Error == nil {
			tx.Rollback()
			return fmt.Errorf("账户地址已存在: %s", account.Address)
		}

		// 创建账户
		if err := tx.Create(&account).Error; err != nil {
			tx.Rollback()
			return fmt.Errorf("批量创建账户失败: %w", err)
		}
	}

	return tx.Commit().Error
}
