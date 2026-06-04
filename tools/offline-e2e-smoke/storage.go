package main

import (
	"sort"
	"sync"

	"offline-server/storage"
	"offline-server/storage/model"
)

type memoryShareStorage struct {
	mu     sync.RWMutex
	shards map[string]model.KeyShard
}

func newMemoryShareStorage() *memoryShareStorage {
	return &memoryShareStorage{shards: make(map[string]model.KeyShard)}
}

func (s *memoryShareStorage) CreateKeyShard(shard model.KeyShard) (*model.KeyShard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.shards[shardKey(shard.Username, shard.Address)] = shard
	return &shard, nil
}

func (s *memoryShareStorage) GetKeyShardForParticipant(username, address string) (*model.KeyShard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	shard, ok := s.shards[shardKey(username, address)]
	if !ok || shard.Status != model.KeyShardStatusActive {
		return nil, storage.ErrRecordNotFound
	}
	return &shard, nil
}

func (s *memoryShareStorage) GetKeyShardByID(shardID string) (*model.KeyShard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, shard := range s.shards {
		if shard.ShardID == shardID {
			return &shard, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (s *memoryShareStorage) ListActiveKeyShardsByAddress(address string) ([]model.KeyShard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var shards []model.KeyShard
	for _, shard := range s.shards {
		if shard.Address == address && shard.Status == model.KeyShardStatusActive {
			shards = append(shards, shard)
		}
	}
	sort.Slice(shards, func(i, j int) bool { return shards[i].ShardIndex < shards[j].ShardIndex })
	return shards, nil
}

func (s *memoryShareStorage) ListKeyShardsByAddress(address string) ([]model.KeyShard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var shards []model.KeyShard
	for _, shard := range s.shards {
		if shard.Address == address {
			shards = append(shards, shard)
		}
	}
	sort.Slice(shards, func(i, j int) bool { return shards[i].ShardIndex < shards[j].ShardIndex })
	return shards, nil
}

func (s *memoryShareStorage) ListKeyShardsByUsername(username string) ([]model.KeyShard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var shards []model.KeyShard
	for _, shard := range s.shards {
		if shard.Username == username {
			shards = append(shards, shard)
		}
	}
	sort.Slice(shards, func(i, j int) bool { return shards[i].ShardIndex < shards[j].ShardIndex })
	return shards, nil
}

func (s *memoryShareStorage) ListKeyShards() ([]model.KeyShard, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var shards []model.KeyShard
	for _, shard := range s.shards {
		shards = append(shards, shard)
	}
	sort.Slice(shards, func(i, j int) bool { return shards[i].ShardID < shards[j].ShardID })
	return shards, nil
}

func (s *memoryShareStorage) UpdateKeyShardStatus(shardID string, status model.KeyShardStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, shard := range s.shards {
		if shard.ShardID == shardID {
			shard.Status = status
			s.shards[key] = shard
			return nil
		}
	}
	return storage.ErrRecordNotFound
}

func (s *memoryShareStorage) TransferKeyShard(shardID, newUsername string) (*model.KeyShard, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for key, shard := range s.shards {
		if shard.ShardID == shardID {
			delete(s.shards, key)
			shard.Username = newUsername
			s.shards[shardKey(newUsername, shard.Address)] = shard
			return &shard, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

type memorySeStorage struct {
	mu     sync.RWMutex
	byID   map[string]model.Se
	byCPLC map[string]model.Se
}

func newMemorySeStorage() *memorySeStorage {
	return &memorySeStorage{
		byID:   make(map[string]model.Se),
		byCPLC: make(map[string]model.Se),
	}
}

func (s *memorySeStorage) add(se model.Se) {
	s.byID[se.SeID] = se
	s.byCPLC[se.CPLC] = se
}

func (s *memorySeStorage) CreateSe(seID, cplc, custodyLocation, registeredBy string) (*model.Se, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	se := model.Se{
		SeID:            seID,
		CPLC:            cplc,
		CustodyLocation: custodyLocation,
		RegisteredBy:    registeredBy,
		Status:          model.SeStatusActive,
	}
	s.add(se)
	return &se, nil
}

func (s *memorySeStorage) GetSeBySeId(seID string) (*model.Se, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	se, ok := s.byID[seID]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &se, nil
}

func (s *memorySeStorage) GetSeByCPLC(cplc string) (*model.Se, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	se, ok := s.byCPLC[cplc]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &se, nil
}

func (s *memorySeStorage) GetAllSe() ([]model.Se, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ses := make([]model.Se, 0, len(s.byID))
	for _, se := range s.byID {
		ses = append(ses, se)
	}
	sort.Slice(ses, func(i, j int) bool { return ses[i].SeID < ses[j].SeID })
	return ses, nil
}

func (s *memorySeStorage) GetActiveSeIds(count int) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0, len(s.byID))
	for seID, se := range s.byID {
		if se.Status == model.SeStatusActive {
			ids = append(ids, seID)
		}
	}
	sort.Strings(ids)
	if len(ids) < count {
		return nil, storage.ErrRecordNotFound
	}
	return append([]string(nil), ids[:count]...), nil
}

func (s *memorySeStorage) UpdateSeStatus(seID string, status model.SeStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	se, ok := s.byID[seID]
	if !ok {
		return storage.ErrRecordNotFound
	}
	se.Status = status
	s.add(se)
	return nil
}

type memoryOfflineKeyStorage struct {
	mu        sync.RWMutex
	byAddress map[string]model.OfflineKey
}

func newMemoryOfflineKeyStorage() *memoryOfflineKeyStorage {
	return &memoryOfflineKeyStorage{byAddress: make(map[string]model.OfflineKey)}
}

func (s *memoryOfflineKeyStorage) CreateOfflineKey(key model.OfflineKey) (*model.OfflineKey, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byAddress[key.Address] = key
	return &key, nil
}

func (s *memoryOfflineKeyStorage) GetOfflineKeyByID(offlineKeyID string) (*model.OfflineKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, key := range s.byAddress {
		if key.OfflineKeyID == offlineKeyID {
			return &key, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (s *memoryOfflineKeyStorage) GetOfflineKeyByAddress(address string) (*model.OfflineKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	key, ok := s.byAddress[address]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &key, nil
}

func (s *memoryOfflineKeyStorage) ListOfflineKeys() ([]model.OfflineKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys := make([]model.OfflineKey, 0, len(s.byAddress))
	for _, key := range s.byAddress {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i].OfflineKeyID < keys[j].OfflineKeyID })
	return keys, nil
}

func (s *memoryOfflineKeyStorage) GetOfflineKeyByTaskNo(taskNo string) (*model.OfflineKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, key := range s.byAddress {
		if key.TaskNo == taskNo {
			return &key, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (s *memoryOfflineKeyStorage) UpdateOfflineKeyOwner(offlineKeyID, logicalOwner string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for address, key := range s.byAddress {
		if key.OfflineKeyID == offlineKeyID {
			key.LogicalOwner = logicalOwner
			s.byAddress[address] = key
			return nil
		}
	}
	return storage.ErrRecordNotFound
}

func (s *memoryOfflineKeyStorage) UpdateOfflineKeyStatus(offlineKeyID string, status model.OfflineKeyStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for address, key := range s.byAddress {
		if key.OfflineKeyID == offlineKeyID {
			key.Status = status
			s.byAddress[address] = key
			return nil
		}
	}
	return storage.ErrRecordNotFound
}

type memoryKeyGenStorage struct {
	mu       sync.RWMutex
	sessions map[string]model.KeyGenSession
}

func newMemoryKeyGenStorage() *memoryKeyGenStorage {
	return &memoryKeyGenStorage{sessions: make(map[string]model.KeyGenSession)}
}

func (s *memoryKeyGenStorage) CreateSession(session model.KeyGenSession) (*model.KeyGenSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.SessionKey] = session
	return &session, nil
}

func (s *memoryKeyGenStorage) GetSession(sessionKey string) (*model.KeyGenSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionKey]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &session, nil
}

func (s *memoryKeyGenStorage) GetSessionByAccountAddr(accountAddr string) (*model.KeyGenSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, session := range s.sessions {
		if session.AccountAddr == accountAddr {
			return &session, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (s *memoryKeyGenStorage) UpdateStatus(sessionKey string, status model.SessionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[sessionKey]
	session.Status = status
	s.sessions[sessionKey] = session
	return nil
}

func (s *memoryKeyGenStorage) UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[sessionKey]
	if len(session.Responses) > index {
		session.Responses[index] = string(status)
	}
	s.sessions[sessionKey] = session
	return nil
}

func (s *memoryKeyGenStorage) UpdateAccountAddr(sessionKey, accountAddr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[sessionKey]
	session.AccountAddr = accountAddr
	s.sessions[sessionKey] = session
	return nil
}

func (s *memoryKeyGenStorage) DeleteSession(sessionKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionKey)
	return nil
}

func (s *memoryKeyGenStorage) UpdateSeIDs(sessionKey string, seIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[sessionKey]
	session.SeIDs = model.StringSlice(seIDs)
	s.sessions[sessionKey] = session
	return nil
}

func (s *memoryKeyGenStorage) AllKeyGenInvitationsAccepted(sessionKey string) bool {
	return true
}

func (s *memoryKeyGenStorage) AllKeyGenPartsCompleted(sessionKey string) bool {
	return true
}

type memorySignStorage struct {
	mu       sync.RWMutex
	sessions map[string]model.SignSession
}

func newMemorySignStorage() *memorySignStorage {
	return &memorySignStorage{sessions: make(map[string]model.SignSession)}
}

func (s *memorySignStorage) CreateSession(session model.SignSession) (*model.SignSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessions[session.SessionKey] = session
	return &session, nil
}

func (s *memorySignStorage) GetSession(sessionKey string) (*model.SignSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	session, ok := s.sessions[sessionKey]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &session, nil
}

func (s *memorySignStorage) GetSessionByTaskNo(taskNo string) (*model.SignSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, session := range s.sessions {
		if session.TaskNo == taskNo {
			return &session, nil
		}
	}
	return nil, storage.ErrSessionNotFound
}

func (s *memorySignStorage) UpdateStatus(sessionKey string, status model.SessionStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[sessionKey]
	session.Status = status
	s.sessions[sessionKey] = session
	return nil
}

func (s *memorySignStorage) UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[sessionKey]
	if len(session.Responses) > index {
		session.Responses[index] = string(status)
	}
	s.sessions[sessionKey] = session
	return nil
}

func (s *memorySignStorage) UpdateSignature(sessionKey, signature string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[sessionKey]
	session.Signature = signature
	session.Status = model.StatusCompleted
	s.sessions[sessionKey] = session
	return nil
}

func (s *memorySignStorage) DeleteSession(sessionKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, sessionKey)
	return nil
}

func (s *memorySignStorage) UpdateSeIDs(sessionKey string, seIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.sessions[sessionKey]
	session.SeIDs = model.StringSlice(seIDs)
	s.sessions[sessionKey] = session
	return nil
}

func (s *memorySignStorage) AllKeyGenInvitationsAccepted(sessionKey string) bool {
	return true
}

func (s *memorySignStorage) AllKeyGenPartsCompleted(sessionKey string) bool {
	return true
}

type memoryAuditStorage struct{}

func (memoryAuditStorage) CreateAuditLog(log model.AuditLog) error {
	return nil
}

func (memoryAuditStorage) ListAuditLogs(limit int) ([]model.AuditLog, error) {
	return nil, nil
}

func (memoryAuditStorage) SearchAuditLogs(filter storage.AuditLogFilter) ([]model.AuditLog, int64, error) {
	return nil, 0, nil
}

type memoryApprovalStorage struct{}

func (memoryApprovalStorage) CreateApproval(approval model.Approval) (*model.Approval, error) {
	return &approval, nil
}

func (memoryApprovalStorage) ListApprovals(limit int) ([]model.Approval, error) {
	return nil, nil
}

func (memoryApprovalStorage) ListApprovalsPage(page, pageSize int) ([]model.Approval, int64, error) {
	return nil, 0, nil
}

func shardKey(username, address string) string {
	return username + "|" + address
}
