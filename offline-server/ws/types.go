package ws

// MessageType 定义了消息的类型
type MessageType string

const (
	// 密钥生成相关消息
	KeyGenRequestMsg  MessageType = "keygen_request"  // 协调方发送的密钥生成请求
	KeyGenInviteMsg   MessageType = "keygen_invite"   // 服务器向参与方发送的密钥生成邀请
	KeyGenResponseMsg MessageType = "keygen_response" // 参与方对密钥生成邀请的响应
	KeyGenParamsMsg   MessageType = "keygen_params"   // 服务器向参与方发送的密钥生成参数
	KeyGenCompleteMsg MessageType = "keygen_complete" // 参与方向服务器发送的密钥生成结果
	KeyGenConfirmMsg  MessageType = "keygen_confirm"  // 服务器向协调方确认密钥生成完成

	// 签名相关消息
	SignRequestMsg  MessageType = "sign_request"  // 协调方发送的签名请求
	SignInviteMsg   MessageType = "sign_invite"   // 服务器向参与方发送的签名邀请
	SignResponseMsg MessageType = "sign_response" // 参与方对签名邀请的响应
	SignParamsMsg   MessageType = "sign_params"   // 服务器向参与方发送的签名参数
	SignResultMsg   MessageType = "sign_result"   // 参与方向服务器发送的签名结果
	SignCompleteMsg MessageType = "sign_complete" // 服务器向协调方发送的签名完成消息

	// 注册与连接相关消息
	RegisterMsg        MessageType = "register"         // 客户端向服务器注册身份
	RegisterConfirmMsg MessageType = "register_confirm" // 服务器确认注册
	// 错误消息
	ErrorMsg MessageType = "error" // 错误消息
)

// Message 定义了基本消息结构
type Message struct {
	Type    MessageType `json:"type"`
	UserID  string      `json:"user_id,omitempty"`
	Token   string      `json:"token,omitempty"` // JWT令牌，用于身份验证
	Payload interface{} `json:"payload"`
}

// RegisterPayload 注册消息的载荷
type RegisterPayload struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"` // "coordinator" 或 "participant"
}

// KeyGenRequestPayload 密钥生成请求的载荷
type KeyGenRequestPayload struct {
	KeyID        string   `json:"key_id"`
	Threshold    int      `json:"threshold"`   // t
	TotalParts   int      `json:"total_parts"` // n
	Participants []string `json:"participants"`
}

// KeyGenInvitePayload 密钥生成邀请的载荷
type KeyGenInvitePayload struct {
	KeyID        string   `json:"key_id"`
	Threshold    int      `json:"threshold"`
	TotalParts   int      `json:"total_parts"`
	PartIndex    int      `json:"part_index"` // i
	Participants []string `json:"participants"`
}

// KeyGenResponsePayload 密钥生成响应的载荷
type KeyGenResponsePayload struct {
	KeyID     string `json:"key_id"`
	PartIndex int    `json:"part_index"` // i
	Response  bool   `json:"response"`   // true表示同意参与，false表示拒绝
}

// KeyGenParamsPayload 密钥生成参数的载荷
type KeyGenParamsPayload struct {
	KeyID       string `json:"key_id"`
	Threshold   int    `json:"threshold"`              // t
	TotalParts  int    `json:"total_parts"`            // n
	PartIndex   int    `json:"part_index"`             // i
	OutputFile  string `json:"output_file"`            // 输出的JSON文件名
	AccountAddr string `json:"account_addr,omitempty"` // 账户地址，可选
}

// KeyGenCompletePayload 密钥生成完成的载荷
type KeyGenCompletePayload struct {
	KeyID       string `json:"key_id"`
	PartIndex   int    `json:"part_index"`   // i
	AccountAddr string `json:"account_addr"` // 账户地址
	ShareJSON   string `json:"share_json"`   // 序列化的密钥分享JSON字符串
}

// SignRequestPayload 签名请求的载荷
type SignRequestPayload struct {
	KeyID        string   `json:"key_id"`
	Data         string   `json:"data"`                   // 要签名的数据
	AccountAddr  string   `json:"account_addr"`           // 账户地址
	Participants []string `json:"participants,omitempty"` // 请求参与签名的用户ID列表，可选
}

// SignInvitePayload 签名邀请的载荷
type SignInvitePayload struct {
	KeyID        string   `json:"key_id"`
	Data         string   `json:"data"`
	AccountAddr  string   `json:"account_addr"`
	PartIndex    int      `json:"part_index"` // i
	Participants []string `json:"participants"`
}

// SignResponsePayload 签名响应的载荷
type SignResponsePayload struct {
	KeyID     string `json:"key_id"`
	PartIndex int    `json:"part_index"` // i
	Response  bool   `json:"response"`   // true表示同意参与，false表示拒绝
}

// SignParamsPayload 签名参数的载荷
type SignParamsPayload struct {
	KeyID        string `json:"key_id"`
	Data         string `json:"data"`
	PartIndex    int    `json:"part_index"`   // i
	Participants string `json:"participants"` // 逗号分隔的参与方索引，如"1,2,3"
	ShareJSON    string `json:"share_json"`   // 密钥分享的JSON字符串
}

// SignResultPayload 签名结果的载荷
type SignResultPayload struct {
	KeyID     string `json:"key_id"`
	PartIndex int    `json:"part_index"` // i
	Signature string `json:"signature"`  // 签名结果
}
