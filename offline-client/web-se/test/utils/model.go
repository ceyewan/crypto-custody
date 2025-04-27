package utils

type KeygenPayload struct {
	Threshold int    `json:"threshold"`
	Parties   int    `json:"parties"`
	Index     int    `json:"index"`
	Filename  string `json:"filename"`
	UserName  string `json:"userName"`
}

type KeygenResponse struct {
	Success      bool   `json:"success"`                // 是否成功
	UserName     string `json:"userName,omitempty"`     // 用户名
	Message      string `json:"message,omitempty"`      // 消息
	Address      string `json:"address,omitempty"`      // 地址
	EncryptedKey string `json:"encryptedKey,omitempty"` // 加密后的密钥 (base64编码)
}

// SignRequest 表示签名请求
type SignRequest struct {
	Parties      string `json:"parties"`      // 参与方信息
	Data         string `json:"data"`         // 待签名数据
	Filename     string `json:"filename"`     // 相关文件名
	UserName     string `json:"userName"`     // 用户名
	Address      string `json:"address"`      // 地址
	EncryptedKey string `json:"encryptedKey"` // 加密密钥
	Signature    string `json:"signature"`    // 签名
}

// SignResponse 表示签名响应
type SignResponse struct {
	Success   bool   `json:"success"`
	Signature string `json:"signature"`
	Message   string `json:"message,omitempty"`
}

// GetCPLCResponse 表示CPLC信息响应
type GetCPLCResponse struct {
	Success bool   `json:"success"`           // 是否成功
	Message string `json:"message,omitempty"` // 消息
	CPIC    string `json:"cpic,omitempty"`    // CPLC信息
}

// DeleteRequest 表示删除请求
type DeleteRequest struct {
	UserName  string `json:"username"`  // 用户名
	Address   string `json:"address"`   // 地址
	Signature string `json:"signature"` // 签名
}

// DeleteResponse 表示删除响应
type DeleteResponse struct {
	Success bool   `json:"success"`           // 是否成功
	Message string `json:"message,omitempty"` // 消息
	Address string `json:"address,omitempty"` // 地址
}
