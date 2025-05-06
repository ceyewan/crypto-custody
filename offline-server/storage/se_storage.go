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

// GetRandomSeIds 随机选取指定数量的安全芯片ID
// 参数：
//   - count: 需要随机获取的安全芯片ID数量
//
// 返回：
//   - 随机选取的安全芯片SeID数组
//   - 如果获取失败则返回错误信息
func (s *SeStorage) GetRandomSeIds(count int) ([]string, error) {
	if count <= 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var ses []model.Se
	// 使用随机排序来确保获取随机的安全芯片记录
	if err := database.Order("RANDOM()").Find(&ses).Error; err != nil {
		log.Printf("随机获取安全芯片记录失败: %v", err)
		return nil, ErrOperationFailed
	}

	// 如果数据库中没有安全芯片记录
	if len(ses) == 0 {
		return nil, ErrRecordNotFound
	}

	// 准备返回结果
	seIds := make([]string, count)

	// 先填充所有可用的不同ID
	availableCount := len(ses)
	for i := 0; i < min(count, availableCount); i++ {
		seIds[i] = ses[i].SeId
	}

	// 如果可用ID不足，重复使用已有ID填充至请求数量
	if availableCount < count {
		log.Printf("警告: 请求%d个安全芯片ID, 但仅找到%d个，将重复使用已有ID", count, availableCount)
		for i := availableCount; i < count; i++ {
			// 循环使用已有ID填充剩余位置
			seIds[i] = ses[i%availableCount].SeId
		}
	}

	return seIds, nil
}
