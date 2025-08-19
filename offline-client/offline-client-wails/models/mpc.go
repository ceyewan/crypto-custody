package models

// KeyGenRequest 密钥生成请求
type KeyGenRequest struct {
	Threshold int    `json:"threshold" binding:"required,min=1"` // t: 门限值
	Parties   int    `json:"parties" binding:"required,min=2"`   // n: 参与方总数
	Index     int    `json:"index" binding:"required,min=1"`     // i: 当前参与方序号
	Filename  string `json:"filename" binding:"required"`        // 输出文件名
	UserName  string `json:"username" binding:"required"`        // 用户名
}

// KeyGenResponse 密钥生成响应
type KeyGenResponse struct {
	Success      bool   `json:"success"`                // 是否成功
	UserName     string `json:"userName,omitempty"`     // 用户名
	Message      string `json:"message,omitempty"`      // 消息
	Address      string `json:"address,omitempty"`      // 地址
	EncryptedKey string `json:"encryptedKey,omitempty"` // 加密后的密钥 (base64编码)
}

// SignRequest 签名请求
type SignRequest struct {
	Parties      string `json:"parties" binding:"required"`      // p: 参与方
	Data         string `json:"data" binding:"required"`         // d: 签名数据
	Filename     string `json:"filename" binding:"required"`     // l: 本地密钥文件
	EncryptedKey string `json:"encryptedKey" binding:"required"` // 加密后的密钥 (base64编码)
	UserName     string `json:"userName" binding:"required"`     // 用户名
	Address      string `json:"address" binding:"required"`      // 地址 (0x前缀)
	Signature    string `json:"signature" binding:"required"`    // 签名 (base64编码)
}

// SignResponse 签名响应
type SignResponse struct {
	Success   bool   `json:"success"`             // 是否成功
	Message   string `json:"message,omitempty"`   // 消息
	Signature string `json:"signature,omitempty"` // 签名结果 (0x前缀)
}

// DeleteRequest 删除请求
type DeleteRequest struct {
	UserName  string `json:"username" binding:"required"`  // 用户名
	Address   string `json:"address" binding:"required"`   // 地址 (0x前缀)
	Signature string `json:"signature" binding:"required"` // 签名 (base64编码)
}

// DeleteResponse 删除响应
type DeleteResponse struct {
	Success bool   `json:"success"`           // 是否成功
	Message string `json:"message,omitempty"` // 消息
	Address string `json:"address,omitempty"` // 地址 (0x前缀)
}

// GetCPLCResponse CPLC信息响应
type GetCPLCResponse struct {
	Success bool   `json:"success"`           // 是否成功
	Message string `json:"message,omitempty"` // 消息
	CPIC    string `json:"cpic,omitempty"`    // CPLC信息
}
