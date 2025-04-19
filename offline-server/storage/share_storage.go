// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// ShareStorage 提供对用户密钥分享的存储和访问
type ShareStorage struct {
	mu sync.RWMutex // 使用读写锁提高并发效率
}

var (
	shareInstance *ShareStorage
	shareOnce     sync.Once
)

// GetShareStorage 返回 ShareStorage 的单例实例
func GetShareStorage() IShareStorage {
	shareOnce.Do(func() {
		shareInstance = &ShareStorage{}
	})
	return shareInstance
}

// SaveUserShare 保存用户密钥分享
func (s *ShareStorage) SaveUserShare(userName, sessionKey, shareJSON string) error {
	if userName == "" || sessionKey == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	// 使用 GORM 的 Upsert 操作
	userShare := model.UserShare{
		UserName:   userName,
		SessionKey: sessionKey,
		ShareJSON:  shareJSON,
	}

	// 查询是否存在记录
	var count int64
	if err := database.Model(&model.UserShare{}).Where("user_name = ? AND session_key = ?", userName, sessionKey).Count(&count).Error; err != nil {
		log.Printf("查询用户分享失败: %v", err)
		return err
	}

	// 根据是否存在记录选择创建或更新
	if count > 0 {
		// 更新已有记录
		result := database.Model(&model.UserShare{}).Where("user_name = ? AND session_key = ?", userName, sessionKey).Update("share_json", shareJSON)
		if result.Error != nil {
			log.Printf("更新用户分享失败: %v", result.Error)
			return result.Error
		}
	} else {
		// 创建新记录
		if err := database.Create(&userShare).Error; err != nil {
			log.Printf("创建用户分享失败: %v", err)
			return err
		}
	}

	return nil
}

// GetUserShare 获取指定用户和密钥ID的分享数据
func (s *ShareStorage) GetUserShare(userName, sessionKey string) (string, error) {
	if userName == "" || sessionKey == "" {
		return "", ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return "", ErrDatabaseNotInitialized
	}

	var userShare model.UserShare
	if err := database.Where("user_name = ? AND session_key = ?", userName, sessionKey).First(&userShare).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", ErrSessionNotFound
		}
		log.Printf("获取用户分享失败: %v", err)
		return "", err
	}

	return userShare.ShareJSON, nil
}

// GetUserShares 获取指定用户的所有密钥分享数据
func (s *ShareStorage) GetUserShares(userName string) (map[string]string, error) {
	if userName == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var userShares []model.UserShare
	if err := database.Where("user_name = ?", userName).Find(&userShares).Error; err != nil {
		log.Printf("获取用户分享列表失败: %v", err)
		return nil, err
	}

	shares := make(map[string]string)
	for _, share := range userShares {
		shares[share.SessionKey] = share.ShareJSON
	}

	return shares, nil
}

// DeleteUserShare 删除指定用户和密钥ID的分享数据
func (s *ShareStorage) DeleteUserShare(userName, sessionKey string) error {
	if userName == "" || sessionKey == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Where("user_name = ? AND session_key = ?", userName, sessionKey).Delete(&model.UserShare{})
	if result.Error != nil {
		log.Printf("删除用户分享失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}
