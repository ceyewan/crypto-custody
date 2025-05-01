// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// CaseStorage 提供对案件的存储和访问
type CaseStorage struct {
	mu sync.RWMutex // 使用读写锁提高并发效率
}

var (
	caseInstance *CaseStorage
	caseOnce     sync.Once
)

// GetCaseStorage 返回 CaseStorage 的单例实例
// 通过单例模式确保整个应用程序中只有一个存储实例
func GetCaseStorage() ICaseStorage {
	caseOnce.Do(func() {
		caseInstance = &CaseStorage{}
	})
	return caseInstance
}

// CreateCase 创建新案件
// 参数：
//   - name: 案件名称，唯一标识一个案件
//   - description: 案件描述信息
//   - threshold: 门限值，表示签名所需的最小分片数量
//   - totalShards: 总分片数，表示私钥分片的总数量
//
// 返回：
//   - 创建的案件对象指针
//   - 如果创建失败则返回错误信息
func (s *CaseStorage) CreateCase(name, description string, threshold, totalShards int) (*model.Case, error) {
	if name == "" || threshold <= 0 || totalShards <= 0 || threshold > totalShards {
		return nil, ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	// 检查案件名称是否已存在
	var count int64
	if err := database.Model(&model.Case{}).Where("name = ?", name).Count(&count).Error; err != nil {
		log.Printf("查询案件失败: %v", err)
		return nil, ErrOperationFailed
	}
	if count > 0 {
		return nil, ErrRecordNotFound
	}

	// 创建案件
	caseObj := model.Case{
		Name:        name,
		Description: description,
		Status:      model.CaseInProgressing, // 初始状态为进行中
		Threshold:   threshold,
		TotalShards: totalShards,
	}

	if err := database.Create(&caseObj).Error; err != nil {
		log.Printf("创建案件失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &caseObj, nil
}

// GetCaseByID 根据ID获取案件信息
// 参数：
//   - id: 案件ID
//
// 返回：
//   - 案件对象指针
//   - 如果案件不存在或查询失败则返回错误信息
func (s *CaseStorage) GetCaseByID(id uint) (*model.Case, error) {
	if id == 0 {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var caseObj model.Case
	if err := database.First(&caseObj, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查询案件失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &caseObj, nil
}

// GetCaseByName 根据名称获取案件信息
// 参数：
//   - name: 案件名称
//
// 返回：
//   - 案件对象指针
//   - 如果案件不存在或查询失败则返回错误信息
func (s *CaseStorage) GetCaseByName(name string) (*model.Case, error) {
	if name == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var caseObj model.Case
	if err := database.Where("name = ?", name).First(&caseObj).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查询案件失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &caseObj, nil
}

// GetCaseByAddress 根据以太坊地址获取案件信息
// 参数：
//   - address: 以太坊账户地址
//
// 返回：
//   - 案件对象指针
//   - 如果案件不存在或查询失败则返回错误信息
func (s *CaseStorage) GetCaseByAddress(address string) (*model.Case, error) {
	if address == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var caseObj model.Case
	if err := database.Where("address = ?", address).First(&caseObj).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查询案件失败: %v", err)
		return nil, ErrOperationFailed
	}

	return &caseObj, nil
}

// GetAllCases 获取所有案件列表
// 返回：
//   - 案件列表数组
//   - 如果查询失败则返回错误信息
func (s *CaseStorage) GetAllCases() ([]model.Case, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var cases []model.Case
	if err := database.Find(&cases).Error; err != nil {
		log.Printf("查询所有案件失败: %v", err)
		return nil, ErrOperationFailed
	}

	return cases, nil
}

// UpdateCase 更新案件信息
// 参数：
//   - id: 案件ID
//   - updates: 需要更新的字段map
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *CaseStorage) UpdateCase(id uint, updates map[string]interface{}) error {
	if id == 0 || updates == nil || len(updates) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.Case{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		log.Printf("更新案件失败: %v", result.Error)
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// UpdateCaseStatus 更新案件状态
// 参数：
//   - id: 案件ID
//   - status: 新的案件状态
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *CaseStorage) UpdateCaseStatus(id uint, status model.CaseStatus) error {
	if id == 0 {
		return ErrInvalidParameter
	}

	// 验证状态是否有效
	validStatus := false
	switch status {
	case model.CaseInProgressing, model.CaseCompleted, model.CaseClosed:
		validStatus = true
	}

	if !validStatus {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.Case{}).Where("id = ?", id).Update("status", status)
	if result.Error != nil {
		log.Printf("更新案件状态失败: %v", result.Error)
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// UpdateCaseAddress 更新案件关联的账户地址
// 参数：
//   - id: 案件ID
//   - address: 以太坊账户地址
//
// 返回：
//   - 如果更新失败则返回错误信息
func (s *CaseStorage) UpdateCaseAddress(id uint, address string) error {
	if id == 0 || address == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Model(&model.Case{}).Where("id = ?", id).Update("address", address)
	if result.Error != nil {
		log.Printf("更新案件地址失败: %v", result.Error)
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}

// DeleteCase 删除案件
// 参数：
//   - id: 案件ID
//
// 返回：
//   - 如果删除失败则返回错误信息
func (s *CaseStorage) DeleteCase(id uint) error {
	if id == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	result := database.Delete(&model.Case{}, id)
	if result.Error != nil {
		log.Printf("删除案件失败: %v", result.Error)
		return ErrOperationFailed
	}

	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}

	return nil
}
