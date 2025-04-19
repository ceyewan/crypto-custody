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
	// StatusWaitingInviteResponse 等待邀请响应
	StatusWaitingInviteResponse SessionStatus = "waiting_invite_response"
	// StatusAccepted 已接受
	StatusAccepted SessionStatus = "accepted"
	// StatusRejected 已拒绝
	StatusRejected SessionStatus = "rejected"
	// StatusProcessing 处理中
	StatusProcessing SessionStatus = "processing"
	// StatusWaitingProcessResponse 等待处理响应
	StatusWaitingProcessResponse SessionStatus = "waiting_process_response"
	// StatusCompleted 已完成
	StatusCompleted SessionStatus = "completed"
	// StatusFailed 失败
	StatusFailed SessionStatus = "failed"
)

// KeyGenSession 密钥生成会话模型
type KeyGenSession struct {
	gorm.Model
	SessionKey   string        `gorm:"column:session_key;uniqueIndex;size:100;comment:会话密钥"`
	Initiator    string        `gorm:"column:initiator;size:100;comment:发起者用户名"`
	Threshold    int           `gorm:"column:threshold;comment:阈值数量"`
	TotalParts   int           `gorm:"column:total_parts;comment:总分片数量"`
	Participants StringSlice   `gorm:"column:participants;type:text;comment:参与者用户名列表"`
	Responses    StringMap     `gorm:"column:responses;type:text;comment:响应状态映射"`
	Completed    StringMap     `gorm:"column:completed;type:text;comment:完成状态映射"`
	AccountAddr  string        `gorm:"column:account_addr;size:100;comment:账户地址"`
	Status       SessionStatus `gorm:"column:status;size:20;default:'created';comment:会话状态"`
}

// SignSession 签名会话模型
type SignSession struct {
	gorm.Model
	SessionKey   string          `gorm:"column:session_key;uniqueIndex;size:100;comment:会话密钥"`
	Initiator    string          `gorm:"column:initiator;size:100;comment:发起者用户名"`
	AccountAddr  string          `gorm:"column:account_addr;size:100;comment:账户地址"`
	Data         string          `gorm:"column:data;type:text;comment:签名数据"`
	Participants StringSlice     `gorm:"column:participants;type:text;comment:参与者用户名列表"`
	Responses    StringMap       `gorm:"column:responses;type:text;comment:响应状态映射"`
	Results      StringStringMap `gorm:"column:results;type:text;comment:签名结果映射"`
	Signature    string          `gorm:"column:signature;type:text;comment:最终签名数据"`
	Status       SessionStatus   `gorm:"column:status;size:20;default:'created';comment:会话状态"`
}

// UserShare 用户密钥分享模型
type UserShare struct {
	gorm.Model
	UserName   string `gorm:"column:user_name;size:100;index:idx_user_key,priority:1;comment:用户名"`
	SessionKey string `gorm:"column:session_key;size:100;index:idx_user_key,priority:2;comment:会话密钥"`
	ShareJSON  string `gorm:"column:share_json;type:text;comment:分享JSON数据"`
}
