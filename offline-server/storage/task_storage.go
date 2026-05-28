package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

type OfflineTaskStorage struct {
	mu sync.RWMutex
}

var (
	taskInstance *OfflineTaskStorage
	taskOnce     sync.Once
)

func GetOfflineTaskStorage() IOfflineTaskStorage {
	taskOnce.Do(func() {
		taskInstance = &OfflineTaskStorage{}
	})
	return taskInstance
}

func (s *OfflineTaskStorage) CreateTask(task model.OfflineTask) (*model.OfflineTask, error) {
	if task.TaskNo == "" || task.TaskType == "" {
		return nil, ErrInvalidParameter
	}
	if task.Status == "" {
		task.Status = model.OfflineTaskImported
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}
	if err := database.Create(&task).Error; err != nil {
		log.Printf("创建离线任务失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &task, nil
}

func (s *OfflineTaskStorage) GetTask(taskNo string) (*model.OfflineTask, error) {
	if taskNo == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}
	var task model.OfflineTask
	if err := database.Where("task_no = ?", taskNo).First(&task).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrRecordNotFound
		}
		log.Printf("查询离线任务失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &task, nil
}

func (s *OfflineTaskStorage) UpdateTaskStatus(taskNo string, status model.OfflineTaskStatus) error {
	if taskNo == "" || status == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	result := database.Model(&model.OfflineTask{}).
		Where("task_no = ?", taskNo).
		Update("status", status)
	if result.Error != nil {
		log.Printf("更新离线任务状态失败: %v", result.Error)
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}

func (s *OfflineTaskStorage) UpdateTaskResultHash(taskNo, resultHash string) error {
	if taskNo == "" || resultHash == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	result := database.Model(&model.OfflineTask{}).
		Where("task_no = ?", taskNo).
		Update("result_hash", resultHash)
	if result.Error != nil {
		log.Printf("更新离线任务结果哈希失败: %v", result.Error)
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrRecordNotFound
	}
	return nil
}
