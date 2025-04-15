// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

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
func (s *ShareStorage) SaveUserShare(userID, keyID, shareJSON string) error {
	if userID == "" || keyID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	// 检查是否已存在
	var count int64
	if err := database.Model(&model.UserShare{}).Where("user_id = ? AND key_id = ?", userID, keyID).Count(&count).Error; err != nil {
		log.Printf("查询用户分享失败: %v", err)
		return err
	}

	if count > 0 {
		// 更新现有记录
		return database.Model(&model.UserShare{}).Where("user_id = ? AND key_id = ?", userID, keyID).Update("share_json", shareJSON).Error
	}

	// 创建新记录
	share := model.UserShare{
		UserID:    userID,
		KeyID:     keyID,
		ShareJSON: shareJSON,
	}
	return database.Create(&share).Error
}

// GetUserShare 获取指定用户和密钥ID的分享数据
func (s *ShareStorage) GetUserShare(userID, keyID string) (string, error) {
	if userID == "" || keyID == "" {
		return "", ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return "", ErrDatabaseNotInitialized
	}

	var share model.UserShare
	if err := database.Where("user_id = ? AND key_id = ?", userID, keyID).First(&share).Error; err != nil {
		return "", err
	}
	return share.ShareJSON, nil
}

// GetUserShares 获取指定用户的所有密钥分享数据
func (s *ShareStorage) GetUserShares(userID string) (map[string]string, error) {
	if userID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var shares []model.UserShare
	if err := database.Where("user_id = ?", userID).Find(&shares).Error; err != nil {
		log.Printf("获取用户分享失败: %v", err)
		return nil, err
	}

	result := make(map[string]string)
	for _, share := range shares {
		result[share.KeyID] = share.ShareJSON
	}
	return result, nil
}

// DeleteUserShare 删除指定用户和密钥ID的分享数据
func (s *ShareStorage) DeleteUserShare(userID, keyID string) error {
	if userID == "" || keyID == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Where("user_id = ? AND key_id = ?", userID, keyID).Delete(&model.UserShare{})
	if result.Error != nil {
		log.Printf("删除用户分享失败: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}

	return nil
}
