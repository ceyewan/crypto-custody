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
	Threshold    int      `json:"threshold"`
	Participants []string `json:"participants"`
}

// KeyGenInvitePayload 密钥生成邀请的载荷
type KeyGenInvitePayload struct {
	KeyID        string   `json:"key_id"`
	Threshold    int      `json:"threshold"`
	Participants []string `json:"participants"`
}

// KeyGenResponsePayload 密钥生成响应的载荷
type KeyGenResponsePayload struct {
	KeyID    string `json:"key_id"`
	Response bool   `json:"response"` // true表示同意参与，false表示拒绝
}

// KeyGenParamsPayload 密钥生成参数的载荷
type KeyGenParamsPayload struct {
	KeyID      string `json:"key_id"`
	Threshold  int    `json:"threshold"`   // t
	TotalParts int    `json:"total_parts"` // n
	PartIndex  int    `json:"part_index"`  // i
	OutputFile string `json:"output_file"` // 输出的JSON文件名
}

// KeyGenCompletePayload 密钥生成完成的载荷
type KeyGenCompletePayload struct {
	KeyID     string `json:"key_id"`
	ShareJSON string `json:"share_json"` // 序列化的密钥分享JSON字符串
}

// SignRequestPayload 签名请求的载荷
type SignRequestPayload struct {
	KeyID        string   `json:"key_id"`
	Data         string   `json:"data"`         // 要签名的数据
	Participants []string `json:"participants"` // 请求参与签名的用户ID列表
}

// SignInvitePayload 签名邀请的载荷
type SignInvitePayload struct {
	KeyID        string   `json:"key_id"`
	Data         string   `json:"data"`
	Participants []string `json:"participants"`
}

// SignResponsePayload 签名响应的载荷
type SignResponsePayload struct {
	KeyID    string `json:"key_id"`
	Response bool   `json:"response"` // true表示同意参与，false表示拒绝
}

// SignParamsPayload 签名参数的载荷
type SignParamsPayload struct {
	KeyID        string `json:"key_id"`
	Data         string `json:"data"`
	Participants string `json:"participants"` // 逗号分隔的参与方索引
	ShareJSON    string `json:"share_json"`   // 密钥分享的JSON字符串
}

// SignResultPayload 签名结果的载荷
type SignResultPayload struct {
	KeyID     string `json:"key_id"`
	Signature string `json:"signature"` // 签名结果
}

// // KeyGenSession 表示一个密钥生成会话
// type KeyGenSession struct {
// 	KeyID        string
// 	Threshold    int
// 	Participants []string
// 	Responses    map[string]bool // 用户ID -> 同意/拒绝
// 	Completed    map[string]bool // 已完成密钥生成的用户
// }

// // SignSession 表示一个签名会话
// type SignSession struct {
// 	KeyID        string
// 	Data         string
// 	Participants []string
// 	Responses    map[string]bool   // 用户ID -> 同意/拒绝
// 	Results      map[string]string // 用户ID -> 签名结果
// }
