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
		shard.Status = model.KeyShardStatusPending
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

// GetKeyShardByID 根据分片编号获取分片。
func (s *ShareStorage) GetKeyShardByID(shardID string) (*model.KeyShard, error) {
	if shardID == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var shard model.KeyShard
	if err := database.Where("shard_id = ?", shardID).First(&shard).Error; err != nil {
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

// ListKeyShardsByUsername 获取某个用户持有的全部分片。
func (s *ShareStorage) ListKeyShardsByUsername(username string) ([]model.KeyShard, error) {
	if username == "" {
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
		Where("username = ?", username).
		Order("updated_at DESC").
		Find(&shards).Error; err != nil {
		log.Printf("查询用户全部分片失败: %v", err)
		return nil, ErrOperationFailed
	}
	return shards, nil
}

// ListKeyShards 获取全部分片。
func (s *ShareStorage) ListKeyShards() ([]model.KeyShard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var shards []model.KeyShard
	if err := database.
		Order("updated_at DESC").
		Find(&shards).Error; err != nil {
		log.Printf("查询全部分片失败: %v", err)
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

// UpdateKeyShardsStatusByOfflineKey 批量更新指定离线密钥下特定状态的分片。
func (s *ShareStorage) UpdateKeyShardsStatusByOfflineKey(offlineKeyID string, from, to model.KeyShardStatus) error {
	if offlineKeyID == "" || from == "" || to == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.KeyShard{}).
		Where("offline_key_id = ? AND status = ?", offlineKeyID, from).
		Update("status", to)
	if result.Error != nil {
		log.Printf("批量更新分片状态失败: %v", result.Error)
		return ErrOperationFailed
	}
	return nil
}

// CreateOfflineKeyAndActivatePendingShards 创建离线密钥元数据并原子激活待提交分片。
func (s *ShareStorage) CreateOfflineKeyAndActivatePendingShards(key model.OfflineKey, expectedShardCount int) (*model.OfflineKey, error) {
	if key.OfflineKeyID == "" || key.Address == "" || key.CoinType == "" || key.Algorithm == "" || expectedShardCount <= 0 {
		return nil, ErrInvalidParameter
	}
	if key.Status == "" {
		key.Status = model.OfflineKeyStatusActive
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	if err := database.Transaction(func(tx *gorm.DB) error {
		var pendingCount int64
		if err := tx.Model(&model.KeyShard{}).
			Where("offline_key_id = ? AND address = ? AND status = ?", key.OfflineKeyID, key.Address, model.KeyShardStatusPending).
			Count(&pendingCount).Error; err != nil {
			return err
		}
		if int(pendingCount) != expectedShardCount {
			return ErrInvalidParameter
		}
		if err := tx.Create(&key).Error; err != nil {
			return err
		}
		result := tx.Model(&model.KeyShard{}).
			Where("offline_key_id = ? AND address = ? AND status = ?", key.OfflineKeyID, key.Address, model.KeyShardStatusPending).
			Update("status", model.KeyShardStatusActive)
		if result.Error != nil {
			return result.Error
		}
		if int(result.RowsAffected) != expectedShardCount {
			return ErrInvalidParameter
		}
		return nil
	}); err != nil {
		log.Printf("创建离线密钥并激活分片失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &key, nil
}

// TransferKeyShard 调整单个分片持有人，不改变安全芯片、record_id 或密文。
func (s *ShareStorage) TransferKeyShard(shardID, newUsername string) (*model.KeyShard, error) {
	if shardID == "" || newUsername == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var shard model.KeyShard
	if err := database.Where("shard_id = ?", shardID).First(&shard).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查找待移交分片失败: %v", err)
		return nil, ErrOperationFailed
	}
	if shard.Status != model.KeyShardStatusActive {
		return nil, ErrInvalidParameter
	}

	var existing int64
	if err := database.Model(&model.KeyShard{}).
		Where("address = ? AND username = ? AND status = ? AND shard_id <> ?", shard.Address, newUsername, model.KeyShardStatusActive, shardID).
		Count(&existing).Error; err != nil {
		log.Printf("检查目标用户分片失败: %v", err)
		return nil, ErrOperationFailed
	}
	if existing > 0 {
		return nil, ErrInvalidParameter
	}

	shard.Username = newUsername
	if err := database.Save(&shard).Error; err != nil {
		log.Printf("移交分片失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &shard, nil
}
