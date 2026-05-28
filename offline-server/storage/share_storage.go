// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// ShareStorage 提供对离线密钥分片的存储和访问。
type ShareStorage struct {
	mu sync.RWMutex
}

var (
	shareInstance *ShareStorage
	shareOnce     sync.Once
)

// GetShareStorage 返回 ShareStorage 的单例实例。
func GetShareStorage() IShareStorage {
	shareOnce.Do(func() {
		shareInstance = &ShareStorage{}
	})
	return shareInstance
}

// CreateKeyShard 创建密钥分片记录。
func (s *ShareStorage) CreateKeyShard(shard model.KeyShard) (*model.KeyShard, error) {
	if shard.ShardID == "" || shard.OfflineKeyID == "" || shard.Username == "" ||
		shard.Address == "" || shard.RecordID == "" || shard.SeCPLC == "" ||
		shard.EncryptedBlob == "" || shard.ShardIndex <= 0 {
		return nil, ErrInvalidParameter
	}
	if shard.BlobType == "" {
		shard.BlobType = model.BlobTypeMPCShare
	}
	if shard.KeyVersion == 0 {
		shard.KeyVersion = 1
	}
	if shard.Status == "" {
		shard.Status = model.KeyShardStatusActive
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	if err := database.Create(&shard).Error; err != nil {
		log.Printf("创建密钥分片失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &shard, nil
}

// GetKeyShardForParticipant 根据用户名和地址获取可用分片。
func (s *ShareStorage) GetKeyShardForParticipant(username, address string) (*model.KeyShard, error) {
	if username == "" || address == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var shard model.KeyShard
	if err := database.
		Where("username = ? AND address = ? AND status = ?", username, address, model.KeyShardStatusActive).
		First(&shard).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查找密钥分片失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &shard, nil
}

// ListActiveKeyShardsByAddress 获取地址下所有可用分片。
func (s *ShareStorage) ListActiveKeyShardsByAddress(address string) ([]model.KeyShard, error) {
	if address == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var shards []model.KeyShard
	if err := database.
		Where("address = ? AND status = ?", address, model.KeyShardStatusActive).
		Order("shard_index ASC").
		Find(&shards).Error; err != nil {
		log.Printf("查询地址分片失败: %v", err)
		return nil, ErrOperationFailed
	}
	return shards, nil
}

// ListKeyShardsByAddress 获取地址下所有分片。
func (s *ShareStorage) ListKeyShardsByAddress(address string) ([]model.KeyShard, error) {
	if address == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var shards []model.KeyShard
	if err := database.
		Where("address = ?", address).
		Order("shard_index ASC").
		Find(&shards).Error; err != nil {
		log.Printf("查询地址全部分片失败: %v", err)
		return nil, ErrOperationFailed
	}
	return shards, nil
}

// UpdateKeyShardStatus 更新分片状态。
func (s *ShareStorage) UpdateKeyShardStatus(shardID string, status model.KeyShardStatus) error {
	if shardID == "" || status == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.KeyShard{}).
		Where("shard_id = ?", shardID).
		Update("status", status)
	if result.Error != nil {
		log.Printf("更新分片状态失败: %v", result.Error)
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}
