// Package storage 提供对系统持久化数据的存储和访问管理
package storage

import (
	"log"
	"sync"

	"gorm.io/gorm"

	"offline-server/storage/db"
	"offline-server/storage/model"
)

// KeyGenStorage 提供对密钥生成会话的存储和访问。
type KeyGenStorage struct {
	mu sync.RWMutex
}

var (
	keyGenInstance *KeyGenStorage
	keyGenOnce     sync.Once
)

// GetKeyGenStorage 返回 KeyGenStorage 的单例实例。
func GetKeyGenStorage() IKeyGenStorage {
	keyGenOnce.Do(func() {
		keyGenInstance = &KeyGenStorage{}
	})
	return keyGenInstance
}

// CreateSession 创建新的密钥生成会话。
func (s *KeyGenStorage) CreateSession(session model.KeyGenSession) (*model.KeyGenSession, error) {
	if session.SessionKey == "" || session.Initiator == "" || session.RequiredSigners <= 0 ||
		session.TotalParties <= 0 || len(session.Participants) == 0 ||
		session.RequiredSigners > session.TotalParties {
		return nil, ErrInvalidParameter
	}
	if session.GG20Threshold == 0 {
		session.GG20Threshold = session.RequiredSigners - 1
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
		log.Printf("创建密钥生成会话失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &session, nil
}

func makeWaitingResponses(participants []string) model.StringSlice {
	responses := make(model.StringSlice, len(participants))
	for i := range responses {
		responses[i] = string(model.ParticipantInit)
	}
	return responses
}

func (s *KeyGenStorage) GetSession(sessionKey string) (*model.KeyGenSession, error) {
	if sessionKey == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var session model.KeyGenSession
	if err := database.Where("session_key = ?", sessionKey).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSessionNotFound
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &session, nil
}

func (s *KeyGenStorage) GetSessionByAccountAddr(accountAddr string) (*model.KeyGenSession, error) {
	if accountAddr == "" {
		return nil, ErrInvalidParameter
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	database := db.GetDB()
	if database == nil {
		return nil, ErrDatabaseNotInitialized
	}

	var session model.KeyGenSession
	if err := database.Where("account_addr = ?", accountAddr).First(&session).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrSessionNotFound
		}
		log.Printf("获取密钥生成会话失败: %v", err)
		return nil, ErrOperationFailed
	}
	return &session, nil
}

func (s *KeyGenStorage) UpdateStatus(sessionKey string, status model.SessionStatus) error {
	if sessionKey == "" || status == "" {
		return ErrInvalidParameter
	}
	return updateSessionField(&model.KeyGenSession{}, sessionKey, "status", status)
}

func (s *KeyGenStorage) UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error {
	if sessionKey == "" || index < 0 || status == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}

	var session model.KeyGenSession
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
		log.Printf("更新密钥生成参与者状态失败: %v", err)
		return ErrOperationFailed
	}
	return nil
}

func (s *KeyGenStorage) UpdateAccountAddr(sessionKey, accountAddr string) error {
	if sessionKey == "" || accountAddr == "" {
		return ErrInvalidParameter
	}
	return updateSessionField(&model.KeyGenSession{}, sessionKey, "account_addr", accountAddr)
}

func (s *KeyGenStorage) DeleteSession(sessionKey string) error {
	if sessionKey == "" {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	result := database.Where("session_key = ?", sessionKey).Delete(&model.KeyGenSession{})
	if result.Error != nil {
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}
	return nil
}

func (s *KeyGenStorage) UpdateSeIDs(sessionKey string, seIDs []string) error {
	if sessionKey == "" || len(seIDs) == 0 {
		return ErrInvalidParameter
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	var session model.KeyGenSession
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

func (s *KeyGenStorage) AllKeyGenInvitationsAccepted(sessionKey string) bool {
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

func (s *KeyGenStorage) AllKeyGenPartsCompleted(sessionKey string) bool {
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

func updateSessionField(modelValue interface{}, sessionKey, field string, value interface{}) error {
	database := db.GetDB()
	if database == nil {
		return ErrDatabaseNotInitialized
	}
	result := database.Model(modelValue).Where("session_key = ?", sessionKey).Update(field, value)
	if result.Error != nil {
		return ErrOperationFailed
	}
	if result.RowsAffected == 0 {
		return ErrSessionNotFound
	}
	return nil
}
