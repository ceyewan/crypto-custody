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

// AccountService 提供账户相关的服务，包括账户创建、查询、更新等功能
type AccountService struct {
	mu sync.RWMutex // 读写锁，用于并发操作保护
}

// GetAccountServiceInstance 获取AccountService的单例实例
//
// 确保整个应用程序中只有一个AccountService实例存在
//
// 返回：
// - *AccountService：AccountService的单例实例
// - error：实例化过程中发生的错误（如有）
func GetAccountServiceInstance() (*AccountService, error) {
	accountServiceInstanceOnce.Do(func() {
		accountServiceInstance = &AccountService{}
	})
	return accountServiceInstance, nil
}

// CreateAccount 创建新账户
//
// 使用提供的信息创建新的账户记录
//
// 参数：
// - account：包含所需信息的账户对象
//
// 返回：
// - error：账户创建过程中的错误（如有）
func (s *AccountService) CreateAccount(account *model.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查地址是否已存在
	var count int64
	if err := utils.GetDB().Model(&model.Account{}).Where("address = ?", account.Address).Count(&count).Error; err != nil {
		return fmt.Errorf("检查账户地址失败: %w", err)
	}

	if count > 0 {
		return errors.New("该地址已存在")
	}

	// 创建账户
	if err := utils.GetDB().Create(account).Error; err != nil {
		return fmt.Errorf("创建账户失败: %w", err)
	}

	return nil
}

// BatchCreateAccounts 批量创建账户
//
// 批量插入多个账户记录
//
// 参数：
// - accounts：包含所需信息的账户对象切片
//
// 返回：
// - error：账户批量创建过程中的错误（如有）
func (s *AccountService) BatchCreateAccounts(accounts []model.Account) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 批量创建账户
	if err := utils.GetDB().Create(&accounts).Error; err != nil {
		return fmt.Errorf("批量创建账户失败: %w", err)
	}

	return nil
}

// UpdateAccountBalance 更新账户余额
//
// 根据账户ID更新账户的余额
//
// 参数：
// - accountID：要更新的账户ID
// - balance：新的余额值
//
// 返回：
// - error：余额更新过程中的错误（如有）
func (s *AccountService) UpdateAccountBalance(accountID uint, balance string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查账户是否存在
	var account model.Account
	if err := utils.GetDB().First(&account, accountID).Error; err != nil {
		return fmt.Errorf("查询账户失败: %w", err)
	}

	// 更新余额
	if err := utils.GetDB().Model(&account).Update("balance", balance).Error; err != nil {
		return fmt.Errorf("更新余额失败: %w", err)
	}

	return nil
}

// GetAccountByAddress 根据地址获取账户
//
// 通过唯一的账户地址检索账户信息
//
// 参数：
// - address：要查询的账户地址
//
// 返回：
// - *model.Account：如果找到，则返回账户记录
// - error：数据库查询过程中发生的错误（如有）
func (s *AccountService) GetAccountByAddress(address string) (*model.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var account model.Account
	if err := utils.GetDB().Where("address = ?", address).First(&account).Error; err != nil {
		return nil, fmt.Errorf("查询账户失败: %w", err)
	}

	return &account, nil
}

// GetAccountByID 根据ID获取账户
//
// 通过账户ID检索账户信息
//
// 参数：
// - id：要查询的账户ID
//
// 返回：
// - *model.Account：如果找到，则返回账户记录
// - error：数据库查询过程中发生的错误（如有）
func (s *AccountService) GetAccountByID(id uint) (*model.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var account model.Account
	if err := utils.GetDB().First(&account, id).Error; err != nil {
		return nil, fmt.Errorf("查询账户失败: %w", err)
	}

	return &account, nil
}

// GetAccounts 获取所有账户
//
// 从数据库中检索所有账户记录
//
// 返回：
// - []model.Account：包含所有账户记录的切片
// - error：数据库查询过程中发生的错误（如有）
func (s *AccountService) GetAccounts() ([]model.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var accounts []model.Account
	if err := utils.GetDB().Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("查询账户列表失败: %w", err)
	}

	return accounts, nil
}

// GetAccountsByImportedBy 获取由指定用户导入的所有账户
//
// 通过导入者用户名检索所有账户
//
// 参数：
// - username：要查询账户的用户名（导入者）
//
// 返回：
// - []model.Account：包含该用户导入的所有账户的切片
// - error：数据库查询过程中发生的错误（如有）
func (s *AccountService) GetAccountsByImportedBy(username string) ([]model.Account, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var accounts []model.Account
	if err := utils.GetDB().Where("imported_by = ?", username).Find(&accounts).Error; err != nil {
		return nil, fmt.Errorf("查询用户导入的账户列表失败: %w", err)
	}

	return accounts, nil
}

// DeleteAccount 删除账户
//
// 通过ID从数据库中删除账户
//
// 参数：
// - accountID：要删除的账户ID
//
// 返回：
// - error：账户删除过程中发生的错误（如有）
func (s *AccountService) DeleteAccount(accountID uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := utils.GetDB().Delete(&model.Account{}, accountID).Error; err != nil {
		return fmt.Errorf("删除账户失败: %w", err)
	}

	return nil
}
