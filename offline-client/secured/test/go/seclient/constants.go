package seclient

// APDU指令常量
const (
	CLA                 = 0x80 // 命令类
	INS_STORE_DATA      = 0x10 // 存储数据命令
	INS_READ_DATA       = 0x20 // 读取数据命令
	INS_DELETE_DATA     = 0x30 // 删除数据命令
	SW_SUCCESS          = 0x9000
	SW_RECORD_NOT_FOUND = 0x6A83
	SW_FILE_FULL        = 0x6A84
	SW_WRONG_LENGTH     = 0x6700

	// 固定长度常量
	USERNAME_LENGTH = 32
	ADDR_LENGTH     = 64
	MESSAGE_LENGTH  = 32
)
