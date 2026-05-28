package ws

// MessageType 定义消息类型
type MessageType string

// 定义所有消息类型常量
const (
	// 注册相关
	MsgRegister         MessageType = "register"          // 客户端注册消息
	MsgRegisterComplete MessageType = "register_complete" // 注册完成回复

	// 密钥生成相关
	MsgKeyGenRequest  MessageType = "keygen_request"  // 发起密钥生成请求
	MsgKeyGenInvite   MessageType = "keygen_invite"   // 邀请参与密钥生成
	MsgKeyGenResponse MessageType = "keygen_response" // 回复密钥生成邀请
	MsgKeyGenParams   MessageType = "keygen_params"   // 密钥生成参数
	MsgKeyGenResult   MessageType = "keygen_result"   // 密钥生成结果
	MsgKeyGenComplete MessageType = "keygen_complete" // 密钥生成完成

	// 签名相关
	MsgSignRequest  MessageType = "sign_request"  // 发起签名请求
	MsgSignInvite   MessageType = "sign_invite"   // 邀请参与签名
	MsgSignResponse MessageType = "sign_response" // 回复签名邀请
	MsgSignParams   MessageType = "sign_params"   // 签名参数
	MsgSignResult   MessageType = "sign_result"   // 签名结果
	MsgSignComplete MessageType = "sign_complete" // 签名完成

	// 销毁相关
	MsgDestroyRequest  MessageType = "destroy_request"  // 发起密钥销毁请求
	MsgDestroyInvite   MessageType = "destroy_invite"   // 邀请参与密钥销毁
	MsgDestroyResponse MessageType = "destroy_response" // 回复密钥销毁邀请
	MsgDestroyParams   MessageType = "destroy_params"   // 密钥销毁参数
	MsgDestroyResult   MessageType = "destroy_result"   // 密钥销毁结果
	MsgDestroyComplete MessageType = "destroy_complete" // 密钥销毁完成

	// 分片移交相关
	MsgTransferRequest  MessageType = "transfer_request"  // 发起分片移交
	MsgTransferInvite   MessageType = "transfer_invite"   // 邀请双方确认移交
	MsgTransferResponse MessageType = "transfer_response" // 双方回复移交邀请
	MsgTransferComplete MessageType = "transfer_complete" // 分片移交完成

	// 错误消息
	MsgError MessageType = "error" // 错误消息
)

// 客户端角色类型
type ClientRole string

const (
	RoleAdmin   ClientRole = "admin"   // 管理员角色
	RoleOfficer ClientRole = "officer" // 警员角色
	RoleAuditor ClientRole = "auditor" // 审计员角色
)

// 通用消息接口，所有消息类型都应实现此接口
type Message interface {
	GetType() MessageType
}

// 基础消息结构，所有消息都包含类型
type BaseMessage struct {
	Type MessageType `json:"type"` // 消息类型
}

func (m BaseMessage) GetType() MessageType {
	return m.Type
}

// 注册消息 - 包含用户凭证信息
type RegisterMessage struct {
	BaseMessage
	Username string     `json:"username"` // 用户名
	Role     ClientRole `json:"role"`     // 用户角色
	Token    string     `json:"token"`    // JWT令牌
}

// 注册完成消息
type RegisterCompleteMessage struct {
	BaseMessage
	Success bool   `json:"success"` // 注册是否成功
	Message string `json:"message"` // 成功或失败的消息
}

// 错误消息
type ErrorMessage struct {
	BaseMessage
	Message string `json:"message"`           // 错误消息
	Details string `json:"details,omitempty"` // 错误详情
}

// 密钥生成消息

// 密钥生成请求消息 - 协调方发起
type KeyGenRequestMessage struct {
	BaseMessage
	SessionKey      string   `json:"session_key"`              // 会话唯一标识
	TaskNo          string   `json:"task_no,omitempty"`        // 在线任务编号
	CaseNo          string   `json:"case_no,omitempty"`        // 案件编号
	OfflineKeyID    string   `json:"offline_key_id,omitempty"` // 离线密钥编号
	CoinType        string   `json:"coin_type,omitempty"`      // 币种
	RequiredSigners int      `json:"required_signers"`         // 业务门限人数
	TotalParties    int      `json:"total_parties"`            // 总分片数n
	Participants    []string `json:"participants"`             // 参与者用户名列表
}

// 密钥生成邀请消息 - 服务器发送给参与方
type KeyGenInviteMessage struct {
	BaseMessage
	SessionKey      string   `json:"session_key"`         // 会话唯一标识
	TaskNo          string   `json:"task_no,omitempty"`   // 在线任务编号
	CaseNo          string   `json:"case_no,omitempty"`   // 案件编号
	Initiator       string   `json:"initiator"`           // 发起人用户名
	CoinType        string   `json:"coin_type,omitempty"` // 币种
	RequiredSigners int      `json:"required_signers"`    // 业务门限人数
	TotalParties    int      `json:"total_parties"`       // 总分片数n
	PartyIndex      int      `json:"party_index"`         // keygen 原始 party index
	SeID            string   `json:"se_id"`               // 安全芯片标识符
	Participants    []string `json:"participants"`        // 所有参与者用户名列表
	Summary         string   `json:"summary,omitempty"`   // 展示摘要
}

// 密钥生成响应消息 - 参与方回应邀请
type KeyGenResponseMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"`      // 会话唯一标识
	PartyIndex int    `json:"party_index"`      // 参与者索引i
	CPLC       string `json:"cplc"`             // 安全芯片唯一标识符
	Accept     bool   `json:"accept"`           // 是否接受参与
	Reason     string `json:"reason,omitempty"` // 拒绝原因(如果拒绝)
}

// 密钥生成参数消息 - 服务器发送给参与方
type KeyGenParamsMessage struct {
	BaseMessage
	SessionKey   string `json:"session_key"`   // 会话唯一标识
	ManagerAddr  string `json:"manager_addr"`  // 本会话manager地址
	Room         string `json:"room"`          // GG20 room
	Threshold    int    `json:"threshold"`     // GG20 threshold
	TotalParties int    `json:"total_parties"` // 总分片数
	PartyIndex   int    `json:"party_index"`   // keygen 原始 party index
	RecordID     string `json:"record_id"`     // SE 记录编号
	FileName     string `json:"filename"`      // 密钥生成配置文件名
}

// 密钥生成结果消息 - 参与方发送给服务器
type KeyGenResultMessage struct {
	BaseMessage
	SessionKey     string `json:"session_key"`     // 会话唯一标识
	PartyIndex     int    `json:"party_index"`     // keygen 原始 party index
	Address        string `json:"address"`         // 生成的账户地址
	PublicKey      string `json:"public_key"`      // 公钥
	CPLC           string `json:"cplc"`            // 安全芯片唯一标识符
	RecordID       string `json:"record_id"`       // SE 记录编号
	EncryptedShard string `json:"encrypted_shard"` // Base64编码的加密密钥分片
	Success        bool   `json:"success"`         // 密钥生成是否成功
	Message        string `json:"message"`         // 成功或失败的消息
}

// 密钥生成完成消息 - 服务器发送给协调方
type KeyGenCompleteMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"` // 会话唯一标识
	Address    string `json:"address"`     // 生成的账户地址
	Success    bool   `json:"success"`     // 密钥生成是否成功
	Message    string `json:"message"`     // 成功或失败的消息
}

// 签名消息

// 签名请求消息 - 协调方发起
type SignRequestMessage struct {
	BaseMessage
	SessionKey    string            `json:"session_key"`              // 会话唯一标识
	TaskNo        string            `json:"task_no,omitempty"`        // 在线任务编号
	CaseNo        string            `json:"case_no,omitempty"`        // 案件编号
	OfflineKeyID  string            `json:"offline_key_id,omitempty"` // 离线密钥编号
	TransactionNo string            `json:"transaction_no,omitempty"` // 交易编号
	MessageHash   string            `json:"message_hash"`             // 要签名的数据(32字节的哈希值)
	Address       string            `json:"address"`                  // 账户地址
	Participants  []string          `json:"participants"`             // 选定的参与者用户名列表
	Display       map[string]string `json:"display,omitempty"`        // 交易展示字段
}

// 签名邀请消息 - 服务器发送给参与方
type SignInviteMessage struct {
	BaseMessage
	SessionKey      string            `json:"session_key"` // 会话唯一标识
	TaskNo          string            `json:"task_no,omitempty"`
	CaseNo          string            `json:"case_no,omitempty"`
	TransactionNo   string            `json:"transaction_no,omitempty"`
	OfflineKeyID    string            `json:"offline_key_id,omitempty"`
	Initiator       string            `json:"initiator"`
	MessageHash     string            `json:"message_hash"` // 要签名的数据(32字节的哈希值)
	Address         string            `json:"address"`      // 账户地址
	RequiredSigners int               `json:"required_signers"`
	TotalParties    int               `json:"total_parties"`
	PartyIndex      int               `json:"party_index"`  // keygen 原始 party index
	SeID            string            `json:"se_id"`        // 安全芯片标识符
	Participants    []string          `json:"participants"` // 参与签名的所有用户名
	Summary         string            `json:"summary,omitempty"`
	Display         map[string]string `json:"display,omitempty"`
}

// 签名响应消息 - 参与方回应邀请
type SignResponseMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"`      // 会话唯一标识
	PartyIndex int    `json:"party_index"`      // keygen 原始 party index
	CPLC       string `json:"cplc"`             // 安全芯片唯一标识符
	Accept     bool   `json:"accept"`           // 是否接受参与
	Reason     string `json:"reason,omitempty"` // 拒绝原因(如果拒绝)
}

// 签名参数消息 - 服务器发送给参与方
type SignParamsMessage struct {
	BaseMessage
	SessionKey     string            `json:"session_key"`     // 会话唯一标识
	ManagerAddr    string            `json:"manager_addr"`    // 本会话manager地址
	Room           string            `json:"room"`            // GG20 room
	MessageHash    string            `json:"message_hash"`    // 要签名的数据
	Address        string            `json:"address"`         // 账户地址
	Signature      string            `json:"signature"`       // 用于从安全芯片中获取私钥分片的签名
	Parties        string            `json:"parties"`         // 参与者列表(逗号分隔的索引)
	PartyIndex     int               `json:"party_index"`     // keygen 原始 party index
	SigningIndex   int               `json:"signing_index"`   // 当前参与者在 parties 中的 1-based 位置
	RecordID       string            `json:"record_id"`       // SE 记录编号
	FileName       string            `json:"filename"`        // 签名配置文件名
	EncryptedShard string            `json:"encrypted_shard"` // Base64编码的加密密钥分片
	Display        map[string]string `json:"display,omitempty"`
}

// 签名结果消息 - 参与方发送给服务器
type SignResultMessage struct {
	BaseMessage
	SessionKey   string `json:"session_key"`   // 会话唯一标识
	SigningIndex int    `json:"signing_index"` // 当前签名索引
	Success      bool   `json:"success"`       // 签名是否成功
	Signature    string `json:"signature"`     // 签名结果
	Message      string `json:"message"`       // 成功或失败的消息
}

// 签名完成消息 - 服务器发送给协调方
type SignCompleteMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"` // 会话唯一标识
	Signature  string `json:"signature"`   // 最终签名结果
	Success    bool   `json:"success"`     // 签名是否成功
	Message    string `json:"message"`     // 成功或失败的消息
}

// 密钥销毁请求消息 - 管理方发起
type DestroyRequestMessage struct {
	BaseMessage
	SessionKey   string   `json:"session_key"`              // 会话唯一标识
	OfflineKeyID string   `json:"offline_key_id,omitempty"` // 离线密钥编号
	Address      string   `json:"address,omitempty"`        // 账户地址
	Participants []string `json:"participants,omitempty"`   // 指定销毁参与者，留空表示该密钥全部 active shard
	Reason       string   `json:"reason,omitempty"`         // 销毁原因
}

// 密钥销毁邀请消息 - 服务器发送给参与方
type DestroyInviteMessage struct {
	BaseMessage
	SessionKey   string `json:"session_key"`    // 会话唯一标识
	OfflineKeyID string `json:"offline_key_id"` // 离线密钥编号
	CaseNo       string `json:"case_no,omitempty"`
	Initiator    string `json:"initiator"`
	Address      string `json:"address"`     // 账户地址
	PartyIndex   int    `json:"party_index"` // keygen 原始 party index
	SeID         string `json:"se_id"`       // 安全芯片标识符
	Summary      string `json:"summary,omitempty"`
	Reason       string `json:"reason,omitempty"`
}

// 密钥销毁响应消息 - 参与方回应邀请
type DestroyResponseMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"`      // 会话唯一标识
	PartyIndex int    `json:"party_index"`      // keygen 原始 party index
	CPLC       string `json:"cplc"`             // 安全芯片唯一标识符
	Accept     bool   `json:"accept"`           // 是否接受参与
	Reason     string `json:"reason,omitempty"` // 拒绝原因
}

// 密钥销毁参数消息 - 服务器发送给参与方
type DestroyParamsMessage struct {
	BaseMessage
	SessionKey   string `json:"session_key"`    // 会话唯一标识
	OfflineKeyID string `json:"offline_key_id"` // 离线密钥编号
	Address      string `json:"address"`        // 账户地址
	PartyIndex   int    `json:"party_index"`    // keygen 原始 party index
	RecordID     string `json:"record_id"`      // SE 记录编号
	Signature    string `json:"signature"`      // SE 删除授权签名
}

// 密钥销毁结果消息 - 参与方发送给服务器
type DestroyResultMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"`       // 会话唯一标识
	PartyIndex int    `json:"party_index"`       // keygen 原始 party index
	Success    bool   `json:"success"`           // 删除是否成功
	Message    string `json:"message,omitempty"` // 成功或失败的消息
}

// 密钥销毁完成消息 - 服务器发送给发起方
type DestroyCompleteMessage struct {
	BaseMessage
	SessionKey   string `json:"session_key"`    // 会话唯一标识
	OfflineKeyID string `json:"offline_key_id"` // 离线密钥编号
	Address      string `json:"address"`        // 账户地址
	Destroyed    int    `json:"destroyed"`      // 已销毁分片数
	Success      bool   `json:"success"`        // 销毁是否成功
	Message      string `json:"message"`        // 成功或失败的消息
}

// 分片移交请求消息 - 管理员发起。
type TransferRequestMessage struct {
	BaseMessage
	SessionKey   string `json:"session_key"`
	ShardID      string `json:"shard_id"`
	OfflineKeyID string `json:"offline_key_id,omitempty"`
	Address      string `json:"address,omitempty"`
	CaseNo       string `json:"case_no,omitempty"`
	ShardIndex   int    `json:"shard_index,omitempty"`
	FromUsername string `json:"from_username"`
	ToUsername   string `json:"to_username"`
	Reason       string `json:"reason,omitempty"`
}

// TransferInviteMessage 是发给移出和接收双方的确认邀请。
type TransferInviteMessage struct {
	BaseMessage
	SessionKey   string `json:"session_key"`
	ShardID      string `json:"shard_id"`
	OfflineKeyID string `json:"offline_key_id,omitempty"`
	Address      string `json:"address"`
	CaseNo       string `json:"case_no,omitempty"`
	ShardIndex   int    `json:"shard_index"`
	FromUsername string `json:"from_username"`
	ToUsername   string `json:"to_username"`
	Initiator    string `json:"initiator"`
	Reason       string `json:"reason,omitempty"`
	Summary      string `json:"summary,omitempty"`
}

// TransferResponseMessage 是移出或接收方对分片移交的回复。
type TransferResponseMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"`
	ShardID    string `json:"shard_id"`
	Accept     bool   `json:"accept"`
	Reason     string `json:"reason,omitempty"`
}

// TransferCompleteMessage 表示分片移交已完成或失败。
type TransferCompleteMessage struct {
	BaseMessage
	SessionKey   string `json:"session_key"`
	ShardID      string `json:"shard_id"`
	Address      string `json:"address"`
	FromUsername string `json:"from_username"`
	ToUsername   string `json:"to_username"`
	Success      bool   `json:"success"`
	Message      string `json:"message"`
}
