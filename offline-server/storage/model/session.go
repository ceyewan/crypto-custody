package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

// StringSlice 是一个string切片，实现了GORM自定义类型接口
type StringSlice []string

// Value 将StringSlice转换为数据库值
func (s StringSlice) Value() (driver.Value, error) {
	if len(s) == 0 {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan 从数据库值扫描到StringSlice
func (s *StringSlice) Scan(value interface{}) error {
	if value == nil {
		*s = StringSlice{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型断言为[]byte失败")
	}
	return json.Unmarshal(bytes, s)
}

// StringMap 是一个string到bool的映射，实现了GORM自定义类型接口
type StringMap map[string]bool

// Value 将StringMap转换为数据库值
func (m StringMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan 从数据库值扫描到StringMap
func (m *StringMap) Scan(value interface{}) error {
	if value == nil {
		*m = StringMap{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型断言为[]byte失败")
	}
	return json.Unmarshal(bytes, m)
}

// StringStringMap 是一个string到string的映射，实现了GORM自定义类型接口
type StringStringMap map[string]string

// Value 将StringStringMap转换为数据库值
func (m StringStringMap) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan 从数据库值扫描到StringStringMap
func (m *StringStringMap) Scan(value interface{}) error {
	if value == nil {
		*m = StringStringMap{}
		return nil
	}
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("类型断言为[]byte失败")
	}
	return json.Unmarshal(bytes, m)
}

// SessionStatus 表示会话状态
type SessionStatus string

const (
	// StatusCreated 已创建
	StatusCreated SessionStatus = "created"
	// StatusInvited 已发送邀请
	StatusInvited SessionStatus = "invited"
	// StatusAccepted 已接受
	StatusAccepted SessionStatus = "accepted"
	// StatusRejected 已拒绝
	StatusRejected SessionStatus = "rejected"
	// StatusProcessing 处理中
	StatusProcessing SessionStatus = "processing"
	// StatusCompleted 已完成
	StatusCompleted SessionStatus = "completed"
	// StatusFailed 失败
	StatusFailed SessionStatus = "failed"
)

// KeyGenSession 密钥生成会话模型
type KeyGenSession struct {
	gorm.Model
	KeyID        string        `gorm:"uniqueIndex;size:100"` // 密钥ID，唯一标识
	InitiatorID  string        `gorm:"size:100"`             // 发起人ID
	Threshold    int           // 阈值 t
	TotalParts   int           // 总分片数 n
	Participants StringSlice   `gorm:"type:text"`                 // 参与者ID列表
	Responses    StringMap     `gorm:"type:text"`                 // 参与者响应状态
	Completed    StringMap     `gorm:"type:text"`                 // 参与者完成状态
	AccountAddr  string        `gorm:"size:100"`                  // 账户地址
	Status       SessionStatus `gorm:"size:20;default:'created'"` // 会话状态
}

// SignSession 签名会话模型
type SignSession struct {
	gorm.Model
	KeyID        string          `gorm:"uniqueIndex;size:100"`      // 密钥ID，唯一标识
	InitiatorID  string          `gorm:"size:100"`                  // 发起人ID
	AccountAddr  string          `gorm:"size:100"`                  // 账户地址
	Data         string          `gorm:"type:text"`                 // 要签名的数据
	Participants StringSlice     `gorm:"type:text"`                 // 参与者ID列表
	Responses    StringMap       `gorm:"type:text"`                 // 参与者响应状态
	Results      StringStringMap `gorm:"type:text"`                 // 参与者签名结果
	Signature    string          `gorm:"type:text"`                 // 最终签名结果
	Status       SessionStatus   `gorm:"size:20;default:'created'"` // 会话状态
}

// UserShare 用户密钥分享模型
type UserShare struct {
	gorm.Model
	UserID    string `gorm:"size:100;index:idx_user_key,priority:1"` // 用户ID
	KeyID     string `gorm:"size:100;index:idx_user_key,priority:2"` // 密钥ID
	ShareJSON string `gorm:"type:text"`                              // 序列化的密钥分享JSON字符串
}
