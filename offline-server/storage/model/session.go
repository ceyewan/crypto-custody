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

// SessionStatus 表示会话状态
type SessionStatus string

const (
	// StatusCreated 已创建
	StatusCreated SessionStatus = "created"
	// StatusInvited 已发送邀请
	StatusInvited SessionStatus = "invited"
	// StatusProcessing 处理中
	StatusProcessing SessionStatus = "processing"
	// StatusCompleted 已完成
	StatusCompleted SessionStatus = "completed"
	// StatusFailed 失败
	StatusFailed SessionStatus = "failed"
	// StatusCancelled 已取消
	StatusCancelled SessionStatus = "cancelled"
)

// ParticipantStatus 表示参与者的状态
type ParticipantStatus string

const (
	// ParticipantInit 参与者初始化
	ParticipantInit ParticipantStatus = "init"
	// ParticipantAccepted 参与者已接受
	ParticipantAccepted ParticipantStatus = "accepted"
	// ParticipantRejected 参与者已拒绝
	ParticipantRejected ParticipantStatus = "rejected"
	// ParticipantCompleted 参与者已完成
	ParticipantCompleted ParticipantStatus = "completed"
	// ParticipantFailed 参与者失败
	ParticipantFailed ParticipantStatus = "failed"
)

// KeyGenSession 密钥生成会话模型
type KeyGenSession struct {
	gorm.Model
	SessionKey      string        `gorm:"column:session_key;uniqueIndex;size:100;comment:会话密钥"`
	TaskNo          string        `gorm:"column:task_no;index;size:100;comment:在线任务编号"`
	OfflineKeyID    string        `gorm:"column:offline_key_id;index;size:100;comment:离线密钥编号"`
	Initiator       string        `gorm:"column:initiator;size:100;comment:发起者用户名"`
	RequiredSigners int           `gorm:"column:required_signers;comment:业务门限人数"`
	TotalParties    int           `gorm:"column:total_parties;comment:总分片数量"`
	GG20Threshold   int           `gorm:"column:gg20_threshold;comment:GG20 threshold 参数"`
	ManagerAddr     string        `gorm:"column:manager_addr;size:200;comment:本会话 manager 地址"`
	Room            string        `gorm:"column:room;size:100;comment:GG20 room"`
	Participants    StringSlice   `gorm:"column:participants;type:text;comment:参与者用户名列表"`
	Responses       StringSlice   `gorm:"column:responses;type:text;comment:响应状态列表，与参与者一一对应"`
	SeIDs           StringSlice   `gorm:"column:se_ids;type:text;comment:安全芯片列表"`
	AccountAddr     string        `gorm:"column:account_addr;size:100;comment:账户地址"`
	PublicKey       string        `gorm:"column:public_key;type:text;comment:公钥"`
	Status          SessionStatus `gorm:"column:status;size:20;default:'created';comment:会话状态"`
}

// SignSession 签名会话模型
type SignSession struct {
	gorm.Model
	SessionKey    string        `gorm:"column:session_key;uniqueIndex;size:100;comment:会话密钥"`
	TaskNo        string        `gorm:"column:task_no;index;size:100;comment:在线任务编号"`
	OfflineKeyID  string        `gorm:"column:offline_key_id;index;size:100;comment:离线密钥编号"`
	TransactionNo string        `gorm:"column:transaction_no;size:100;comment:交易编号"`
	Initiator     string        `gorm:"column:initiator;size:100;comment:发起者用户名"`
	Address       string        `gorm:"column:address;size:100;comment:账户地址"`
	MessageHash   string        `gorm:"column:message_hash;type:text;comment:待签名哈希"`
	ManagerAddr   string        `gorm:"column:manager_addr;size:200;comment:本会话 manager 地址"`
	Room          string        `gorm:"column:room;size:100;comment:GG20 room"`
	Participants  StringSlice   `gorm:"column:participants;type:text;comment:参与者用户名列表"`
	Parties       string        `gorm:"column:parties;size:100;comment:原始 shard index 列表"`
	Responses     StringSlice   `gorm:"column:responses;type:text;comment:响应状态列表，与参与者一一对应"`
	SeIDs         StringSlice   `gorm:"column:se_ids;type:text;comment:安全芯片列表"`
	Signature     string        `gorm:"column:signature;type:text;comment:最终签名数据"`
	Status        SessionStatus `gorm:"column:status;size:20;default:'created';comment:会话状态"`
}
