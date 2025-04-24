package seclient

// APDU指令常量
const (
	CLA             = 0x80 // 命令类
	INS_STORE_DATA  = 0x10 // 存储数据命令
	INS_READ_DATA   = 0x20 // 读取数据命令
	INS_DELETE_DATA = 0x30 // 删除数据命令

	// 状态码
	SW_SUCCESS           = 0x9000 // 成功
	SW_RECORD_NOT_FOUND  = 0x6A83 // 记录未找到
	SW_FILE_FULL         = 0x6A84 // 文件已满
	SW_WRONG_LENGTH      = 0x6700 // 长度错误
	SW_SIGNATURE_INVALID = 0x6982 // 签名无效

	// 固定长度常量
	USERNAME_LENGTH      = 32 // 用户名长度
	ADDR_LENGTH          = 64 // 地址长度
	MESSAGE_LENGTH       = 32 // 消息长度
	MAX_SIGNATURE_LENGTH = 72 // DER格式签名最大长度
	MIN_SIGNATURE_LENGTH = 70 // DER格式签名最小长度
)
