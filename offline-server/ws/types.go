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

	// 错误消息
	MsgError MessageType = "error" // 错误消息
)

// 客户端角色类型
type ClientRole string

const (
	RoleCoordinator ClientRole = "coordinator" // 协调者角色
	RoleParticipant ClientRole = "participant" // 参与者角色
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
	SessionKey   string   `json:"session_key"`  // 会话唯一标识
	Threshold    int      `json:"threshold"`    // 门限值t
	TotalParts   int      `json:"total_parts"`  // 总分片数n
	Participants []string `json:"participants"` // 参与者用户名列表
}

// 密钥生成邀请消息 - 服务器发送给参与方
type KeyGenInviteMessage struct {
	BaseMessage
	SessionKey   string   `json:"session_key"`  // 会话唯一标识
	Coordinator  string   `json:"coordinator"`  // 发起协调者用户名
	Threshold    int      `json:"threshold"`    // 门限值t
	TotalParts   int      `json:"total_parts"`  // 总分片数n
	PartIndex    int      `json:"part_index"`   // 当前参与者索引i
	SeID         string   `json:"se_id"`        // 安全芯片标识符
	Participants []string `json:"participants"` // 所有参与者用户名列表
}

// 密钥生成响应消息 - 参与方回应邀请
type KeyGenResponseMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"`      // 会话唯一标识
	PartIndex  int    `json:"part_index"`       // 参与者索引i
	CPIC       string `json:"cpic"`             // 安全芯片唯一标识符
	Accept     bool   `json:"accept"`           // 是否接受参与
	Reason     string `json:"reason,omitempty"` // 拒绝原因(如果拒绝)
}

// 密钥生成参数消息 - 服务器发送给参与方
type KeyGenParamsMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"` // 会话唯一标识
	Threshold  int    `json:"threshold"`   // 门限值
	TotalParts int    `json:"total_parts"` // 总分片数
	PartIndex  int    `json:"part_index"`  // 参与者索引i
	FileName   string `json:"filename"`    // 密钥生成配置文件名
}

// 密钥生成结果消息 - 参与方发送给服务器
type KeyGenResultMessage struct {
	BaseMessage
	SessionKey     string `json:"session_key"`     // 会话唯一标识
	PartIndex      int    `json:"part_index"`      // 参与者索引i
	Address        string `json:"address"`         // 生成的账户地址
	CPIC           string `json:"cpic"`            // 安全芯片唯一标识符
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
	SessionKey   string   `json:"session_key"`  // 会话唯一标识
	Threshold    int      `json:"threshold"`    // 门限值
	TotalParts   int      `json:"total_parts"`  // 总分片数
	Data         string   `json:"data"`         // 要签名的数据(32字节的哈希值)
	Address      string   `json:"address"`      // 账户地址
	Participants []string `json:"participants"` // 选定的参与者用户名列表
}

// 签名邀请消息 - 服务器发送给参与方
type SignInviteMessage struct {
	BaseMessage
	SessionKey   string   `json:"session_key"`  // 会话唯一标识
	Data         string   `json:"data"`         // 要签名的数据(32字节的哈希值)
	Address      string   `json:"address"`      // 账户地址
	PartIndex    int      `json:"part_index"`   // 参与者索引i
	SeID         string   `json:"se_id"`        // 安全芯片标识符
	Participants []string `json:"participants"` // 参与签名的所有用户名
}

// 签名响应消息 - 参与方回应邀请
type SignResponseMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"`      // 会话唯一标识
	PartIndex  int    `json:"part_index"`       // 参与者索引i
	CPIC       string `json:"cpic"`             // 安全芯片唯一标识符
	Accept     bool   `json:"accept"`           // 是否接受参与
	Reason     string `json:"reason,omitempty"` // 拒绝原因(如果拒绝)
}

// 签名参数消息 - 服务器发送给参与方
type SignParamsMessage struct {
	BaseMessage
	SessionKey     string `json:"session_key"`     // 会话唯一标识
	Data           string `json:"data"`            // 要签名的数据(Base64编码)
	Address        string `json:"address"`         // 账户地址
	Signature      string `json:"signature"`       // 用于从安全芯片中获取私钥分片的签名
	Parties        string `json:"parties"`         // 参与者列表(逗号分隔的索引)
	PartIndex      int    `json:"part_index"`      // 参与者索引i
	FileName       string `json:"filename"`        // 签名配置文件名
	EncryptedShard string `json:"encrypted_shard"` // Base64编码的加密密钥分片
}

// 签名结果消息 - 参与方发送给服务器
type SignResultMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"` // 会话唯一标识
	PartIndex  int    `json:"part_index"`  // 参与者索引i
	Success    bool   `json:"success"`     // 签名是否成功
	Signature  string `json:"signature"`   // 签名结果
	Message    string `json:"message"`     // 成功或失败的消息
}

// 签名完成消息 - 服务器发送给协调方
type SignCompleteMessage struct {
	BaseMessage
	SessionKey string `json:"session_key"` // 会话唯一标识
	Signature  string `json:"signature"`   // 最终签名结果
	Success    bool   `json:"success"`     // 签名是否成功
	Message    string `json:"message"`     // 成功或失败的消息
}
