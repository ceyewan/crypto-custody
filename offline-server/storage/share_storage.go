// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// ShareStorage 提供对用户密钥分片的存储和访问
type ShareStorage struct {
	mu sync.RWMutex // 使用读写锁提高并发效率
}

var (
	shareInstance *ShareStorage
	shareOnce     sync.Once
)

// GetShareStorage 返回 ShareStorage 的单例实例
// 通过单例模式确保整个应用程序中只有一个存储实例
func GetShareStorage() IShareStorage {
	shareOnce.Do(func() {
		shareInstance = &ShareStorage{}
	})
	return shareInstance
}

// CreateEthereumKeyShard 创建以太坊私钥分片记录
// 参数：
//   - username: 用户名，标识私钥分片的所有者
//   - address: 以太坊地址，标识私钥对应的账户
//   - pcic: 安全芯片标识，用于标识存储加密密钥的安全芯片
//   - privateShard: 加密的私钥分片，Base64编码
//   - shardIndex: 分片索引，表示这是第几个私钥分片
//
// 返回：
//   - 如果创建失败则返回错误信息
func (s *ShareStorage) CreateEthereumKeyShard(username, address, pcic, privateShard string, shardIndex int) error {
	if username == "" || address == "" || pcic == "" || privateShard == "" || shardIndex < 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	// 检查是否已存在相同用户名、地址和索引的分片
	var count int64
	if err := database.Model(&model.EthereumKeyShard{}).
		Where("username = ? AND address = ? AND index = ?", username, address, shardIndex).
		Count(&count).Error; err != nil {
		log.Printf("查询以太坊私钥分片失败: %v", err)
		return ErrOperationFailed
	}

	if count > 0 {
		log.Printf("已存在用户 %s 的地址 %s 索引为 %d 的分片", username, address, shardIndex)
		return ErrRecordNotFound
	}

	// 创建新记录
	keyShard := model.EthereumKeyShard{
		Username:     username,
		Address:      address,
		ShardIndex:   shardIndex,
		PCIC:         pcic,
		PrivateShard: privateShard,
	}

	if err := database.Create(&keyShard).Error; err != nil {
		log.Printf("创建以太坊私钥分片失败: %v", err)
		return ErrOperationFailed
	}

	return nil
}

// GetEthereumKeyShard 根据用户名和以太坊地址获取密钥分片
// 参数：
//   - username: 用户名
//   - address: 以太坊地址
//
// 返回：
//   - 以太坊私钥分片记录指针
//   - 如果找不到记录或查询失败则返回错误信息
func (s *ShareStorage) GetEthereumKeyShard(username, address string) (*model.EthereumKeyShard, error) {
	if username == "" || address == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var keyShard model.EthereumKeyShard
	if err := database.Where("username = ? AND address = ?", username, address).First(&keyShard).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查找以太坊私钥分片失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &keyShard, nil
}
