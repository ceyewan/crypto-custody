package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("=== 安全芯片Applet测试工具 ===")
	fmt.Println("本工具将测试JavaCard安全芯片的数据存储和读取功能")
	fmt.Println("详细执行记录将保存在logs目录下")

	err := RunAppletTests()
	if err != nil {
		fmt.Printf("测试出错: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("\n所有测试已完成!")
}
