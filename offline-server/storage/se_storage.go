// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// SeStorage 提供对安全芯片记录的存储和访问
type SeStorage struct {
	mu sync.RWMutex // 使用读写锁提高并发效率
}

var (
	seInstance *SeStorage
	seOnce     sync.Once
)

// GetSeStorage 返回 SeStorage 的单例实例
// 通过单例模式确保整个应用程序中只有一个存储实例
func GetSeStorage() ISeStorage {
	seOnce.Do(func() {
		seInstance = &SeStorage{}
	})
	return seInstance
}

// CreateSe 创建新的安全芯片记录
// 参数：
//   - seId: 安全芯片的用户可读ID
//   - cpic: 安全芯片的唯一标识符
//
// 返回：
//   - 创建的安全芯片记录指针
//   - 如果创建失败则返回错误信息
func (s *SeStorage) CreateSe(seId, cpic string) (*model.Se, error) {
	if seId == "" || cpic == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	// 检查是否已存在相同ID或CPIC的安全芯片
	var count int64
	if err := database.Model(&model.Se{}).Where("se_id = ? OR cpic = ?", seId, cpic).Count(&count).Error; err != nil {
		log.Printf("查询安全芯片失败: %v", err)
		return nil, ErrOperationFailed
	}

	if count > 0 {
		return nil, ErrSeExists
	}

	// 创建安全芯片记录
	se := model.Se{
		SeId: seId,
		CPIC: cpic,
	}

	if err := database.Create(&se).Error; err != nil {
		log.Printf("创建安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &se, nil
}

// GetSeBySeId 根据安全芯片ID获取记录
// 参数：
//   - seId: 安全芯片的用户可读ID
//
// 返回：
//   - 安全芯片记录指针
//   - 如果找不到记录或查询失败则返回错误信息
func (s *SeStorage) GetSeBySeId(seId string) (*model.Se, error) {
	if seId == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var se model.Se
	if err := database.Where("se_id = ?", seId).First(&se).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查询安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &se, nil
}

// GetSeByCPIC 根据CPIC获取安全芯片记录
// 参数：
//   - cpic: 安全芯片的唯一标识符
//
// 返回：
//   - 安全芯片记录指针
//   - 如果找不到记录或查询失败则返回错误信息
func (s *SeStorage) GetSeByCPIC(cpic string) (*model.Se, error) {
	if cpic == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var se model.Se
	if err := database.Where("cpic = ?", cpic).First(&se).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查询安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &se, nil
}

// GetAllSe 获取所有安全芯片记录
// 返回：
//   - 安全芯片记录数组
//   - 如果查询失败则返回错误信息
func (s *SeStorage) GetAllSe() ([]model.Se, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var ses []model.Se
	if err := database.Find(&ses).Error; err != nil {
		log.Printf("查询所有安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}

	return ses, nil
}
