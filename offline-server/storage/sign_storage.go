// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// SignStorage 提供对签名会话的存储和访问。
type SignStorage struct {
	mu sync.RWMutex
}

var (
	signInstance *SignStorage
	signOnce     sync.Once
)

// GetSignStorage 返回 SignStorage 的单例实例。
func GetSignStorage() ISignStorage {
	signOnce.Do(func() {
		signInstance = &SignStorage{}
	})
	return signInstance
}

func (s *SignStorage) CreateSession(session model.SignSession) (*model.SignSession, error) {
	if session.SessionKey == "" || session.Initiator == "" || session.MessageHash == "" ||
		session.Address == "" || len(session.Participants) == 0 {
		return nil, ErrInvalidParameter
	}
	if session.Status == "" {
		session.Status = model.StatusCreated
	}
	if len(session.Responses) == 0 {
		session.Responses = makeWaitingResponses(session.Participants)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}
	if err := database.Create(&session).Error; err != nil {
		log.Printf("创建签名会话失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &session, nil
}

func (s *SignStorage) GetSession(sessionKey string) (*model.SignSession, error) {
	if sessionKey == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var session model.SignSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSessionNotFound
		}
		return nil, ErrOperationFailed
	}
	return &session, nil
}

func (s *SignStorage) GetSessionByTaskNo(taskNo string) (*model.SignSession, error) {
	if taskNo == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var session model.SignSession
	if err := database.Where("task_no = ?", taskNo).Order("created_at DESC").First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSessionNotFound
		}
		return nil, ErrOperationFailed
	}
	return &session, nil
}

func (s *SignStorage) UpdateStatus(sessionKey string, status model.SessionStatus) error {
	if sessionKey == "" || status == "" {
		return ErrInvalidParameter
	}
	return updateSessionField(&model.SignSession{}, sessionKey, "status", status)
}

func (s *SignStorage) UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error {
	if sessionKey == "" || index < 0 || status == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	var session model.SignSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSessionNotFound
		}
		return ErrOperationFailed
	}
	if index >= len(session.Participants) {
		return ErrInvalidParameter
	}
	if len(session.Responses) < len(session.Participants) {
		responses := make(model.StringSlice, len(session.Participants))
		copy(responses, session.Responses)
		session.Responses = responses
	}
	session.Responses[index] = string(status)
	if err := database.Save(&session).Error; err != nil {
		log.Printf("更新签名参与者状态失败: %v", err)
		return ErrOperationFailed
	}
	return nil
}

func (s *SignStorage) UpdateSignature(sessionKey, signature string) error {
	if sessionKey == "" || signature == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	result := database.Model(&model.SignSession{}).Where("session_key = ?", sessionKey).Updates(map[string]interface{}{
		"signature": signature,
		"status":    model.StatusCompleted,
	})
	if result.Error != nil {
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *SignStorage) DeleteSession(sessionKey string) error {
	if sessionKey == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	result := database.Where("session_key = ?", sessionKey).Delete(&model.SignSession{})
	if result.Error != nil {
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *SignStorage) UpdateSeIDs(sessionKey string, seIDs []string) error {
	if sessionKey == "" || len(seIDs) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	var session model.SignSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return ErrSessionNotFound
		}
		return ErrOperationFailed
	}
	if len(seIDs) != len(session.Participants) {
		return ErrInvalidParameter
	}
	session.SeIDs = model.StringSlice(seIDs)
	if err := database.Save(&session).Error; err != nil {
		return ErrOperationFailed
	}
	return nil
}

func (s *SignStorage) AllKeyGenInvitationsAccepted(sessionKey string) bool {
	session, err := s.GetSession(sessionKey)
	if err != nil {
		return false
	}
	for _, status := range session.Responses {
		if status != string(model.ParticipantAccepted) {
			return false
		}
	}
	return len(session.Responses) > 0
}

func (s *SignStorage) AllKeyGenPartsCompleted(sessionKey string) bool {
	session, err := s.GetSession(sessionKey)
	if err != nil {
		return false
	}
	for _, status := range session.Responses {
		if status != string(model.ParticipantCompleted) {
			return false
		}
	}
	return len(session.Responses) > 0
}
