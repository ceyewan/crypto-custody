package models

// KeyGenRequest 密钥生成请求
type KeyGenRequest struct {
	ManagerAddr string `json:"manager_addr" binding:"required"`      // 本会话 manager 地址
	Room        string `json:"room" binding:"required"`              // 本会话 room
	Threshold   int    `json:"threshold" binding:"required,min=0"`   // GG20 t
	Parties     int    `json:"parties" binding:"required,min=2"`     // n: 参与方总数
	PartyIndex  int    `json:"party_index" binding:"required,min=1"` // keygen 原始 party index
	RecordID    string `json:"record_id" binding:"required"`         // SE 记录编号
	Filename    string `json:"filename" binding:"required"`          // 输出文件名
}

// KeyGenResponse 密钥生成响应
type KeyGenResponse struct {
	Success        bool   `json:"success"`                   // 是否成功
	Message        string `json:"message,omitempty"`         // 消息
	Address        string `json:"address,omitempty"`         // 地址
	PublicKey      string `json:"public_key,omitempty"`      // 公钥
	RecordID       string `json:"record_id,omitempty"`       // SE 记录编号
	EncryptedShard string `json:"encrypted_shard,omitempty"` // 加密后的分片 (base64编码)
}

// SignRequest 签名请求
type SignRequest struct {
	ManagerAddr    string `json:"manager_addr" binding:"required"`        // 本会话 manager 地址
	Room           string `json:"room" binding:"required"`                // 本会话 room
	Parties        string `json:"parties" binding:"required"`             // p: 原始 shard index 列表
	SigningIndex   int    `json:"signing_index" binding:"required,min=1"` // 本方在 parties 中的位置
	MessageHash    string `json:"message_hash" binding:"required"`        // d: 签名数据
	Filename       string `json:"filename" binding:"required"`            // l: 本地密钥文件
	EncryptedShard string `json:"encrypted_shard" binding:"required"`     // 加密后的分片 (base64编码)
	RecordID       string `json:"record_id" binding:"required"`           // SE 记录编号
	Address        string `json:"address" binding:"required"`             // 地址 (0x前缀)
	Signature      string `json:"signature" binding:"required"`           // SE 授权签名 (base64编码)
}

// SignResponse 签名响应
type SignResponse struct {
	Success   bool   `json:"success"`             // 是否成功
	Message   string `json:"message,omitempty"`   // 消息
	Signature string `json:"signature,omitempty"` // 签名结果 (0x前缀)
}

// DeleteRequest 删除请求
type DeleteRequest struct {
	RecordID  string `json:"record_id" binding:"required"` // SE 记录编号
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
	Success  bool   `json:"success"`           // 是否成功
	Message  string `json:"message,omitempty"` // 消息
	CPLCInfo string `json:"cplc_info"`         // CPLC信息
}
