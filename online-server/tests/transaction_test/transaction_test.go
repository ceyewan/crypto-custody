package transaction_test

import (
	"fmt"
	"os"
	"testing"
)

// 测试前的全局设置
func TestMain(m *testing.M) {
	// 验证测试环境设置
	validateTestEnvironment()

	// 运行测试
	exitCode := m.Run()

	// 退出
	os.Exit(exitCode)
}

// 验证测试环境设置
func validateTestEnvironment() {
	// 检查是否已设置测试账户地址
	if TestAccountA == "0x" || TestAccountB == "0x" {
		fmt.Println("警告: 测试账户地址尚未设置，请在common_test.go文件中设置TestAccountA和TestAccountB")
	}

	// 检查是否已设置交易签名
	if TestSignature == "" {
		fmt.Println("警告: 交易签名尚未设置，请在common_test.go文件中设置TestSignature")
	}

	// 检查服务器是否正在运行
	fmt.Println("提示: 确保在运行测试前，服务器已在localhost:8080上启动")
}
