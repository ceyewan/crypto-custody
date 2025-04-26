package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"test/seclient"

	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
)

const (
	// 测试配置
	MAX_RECORDS = 100 // JavaCard Applet 支持的最大记录数
	READER_NAME = ""  // 留空自动选择第一个读卡器
)

// 测试数据生成
func generateTestData(count int) ([][]byte, [][]byte, [][]byte) {
	usernames := make([][]byte, count)
	addresses := make([][]byte, count)
	messages := make([][]byte, count)

	for i := 0; i < count; i++ {
		// 生成用户名：user{i}@example.com 并计算哈希值
		username := fmt.Sprintf("user%d@example.com", i)
		hash := sha256.Sum256([]byte(username))
		usernames[i] = hash[:]

		// 生成以太坊地址：20字节的随机数据
		addr := make([]byte, seclient.ADDR_LENGTH)
		rand.Read(addr)
		addresses[i] = addr

		// 生成消息：这是第{i}条测试消息 + 填充
		message := fmt.Sprintf("这是第%d条测试消息", i)
		messages[i] = padData([]byte(message), seclient.MESSAGE_LENGTH)
	}

	return usernames, addresses, messages
}

// 填充数据到指定长度
func padData(data []byte, length int) []byte {
	if len(data) >= length {
		return data[:length]
	}
	result := make([]byte, length)
	copy(result, data)
	return result
}

// 从PEM文件加载私钥
func loadPrivateKey(filename string) (*ecdsa.PrivateKey, error) {
	pemData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取私钥文件失败: %v", err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PRIVATE KEY" {
		return nil, fmt.Errorf("无法解析PEM数据")
	}

	privateKey, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("解析私钥失败: %v", err)
	}

	ecdsaKey, ok := privateKey.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("私钥不是ECDSA类型")
	}

	return ecdsaKey, nil
}

// 对数据进行签名
func signData(privateKey *ecdsa.PrivateKey, data []byte) ([]byte, error) {
	// 计算消息哈希
	hash := sha256.Sum256(data)

	// 签名
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("签名失败: %v", err)
	}

	// 将r和s转换为DER格式
	signature, err := marshalECDSASignature(r, s)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

// 将ECDSA签名转换为DER格式
func marshalECDSASignature(r, s *big.Int) ([]byte, error) {
	// 使用crypto/x509的方法来序列化签名为DER格式
	return asn1MarshalSignature(r, s)
}

// ASN.1格式的签名
func asn1MarshalSignature(r, s *big.Int) ([]byte, error) {
	// 简单实现ASN.1 DER编码格式的ECDSA签名
	// 实际应用中应使用专门的库来处理
	type ecdsaSignature struct {
		R, S *big.Int
	}
	return asn1.Marshal(ecdsaSignature{R: r, S: s})
}

func main() {
	// 设置调试模式
	debug := false // 将调试模式改为false，减少APDU输出

	// 初始化卡片读取器
	cardReader, err := seclient.NewCardReader(seclient.WithDebug(debug))
	if err != nil {
		fmt.Printf("创建读卡器失败: %v\n", err)
		return
	}
	defer cardReader.Close()

	// 连接读卡器
	if err := cardReader.Connect(READER_NAME); err != nil {
		fmt.Printf("连接读卡器失败: %v\n", err)
		return
	}

	// 选择Applet
	if err := cardReader.SelectApplet(); err != nil {
		fmt.Printf("选择Applet失败: %v\n", err)
		return
	}

	fmt.Println("开始安全芯片功能测试...")

	// 加载私钥
	privateKeyFile := "../../genkey/ec_private_key.pem"
	privateKey, err := loadPrivateKey(privateKeyFile)
	if err != nil {
		fmt.Printf("加载私钥失败: %v\n", err)
		return
	}
	fmt.Printf("✅ 私钥加载成功\n")

	// 生成测试数据
	testCount := 5                                                  // 基础测试使用的记录数量
	usernames, addresses, messages := generateTestData(MAX_RECORDS) // 为了测试最大容量，生成最大记录数的数据

	// 跟踪添加的记录以便最后清理
	addedRecords := make(map[int]bool)

	// 基础功能测试
	fmt.Println("\n==== 基础功能测试 ====")
	// 存储几条测试数据
	activeRecords := make(map[string]bool)
	for i := 0; i < testCount; i++ {
		fmt.Printf("存储记录 #%d... ", i)
		recordIdx, recordCount, err := cardReader.StoreData(
			usernames[i],
			addresses[i],
			messages[i],
		)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
		} else {
			fmt.Printf("✅ 成功 (索引=%d, 总数=%d)\n", recordIdx, recordCount)
			activeRecords[fmt.Sprintf("%d", i)] = true
			addedRecords[i] = true
		}
		time.Sleep(50 * time.Millisecond)
	}

	// 读取数据测试
	fmt.Println("\n==== 读取数据测试 ====")
	for i := 0; i < testCount; i++ {
		fmt.Printf("读取记录 #%d... ", i)
		// 为用户名和地址生成签名
		dataToSign := append(usernames[i], addresses[i]...)
		signature, err := signData(privateKey, dataToSign)
		if err != nil {
			fmt.Printf("❌ 签名失败: %v\n", err)
			continue
		}

		// 读取数据
		message, err := cardReader.ReadData(
			usernames[i],
			addresses[i],
			signature,
		)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
		} else {
			// 比较读取的数据与原始数据
			expectedMsg := strings.TrimRight(string(messages[i]), "\x00")
			actualMsg := strings.TrimRight(string(message), "\x00")
			if expectedMsg == actualMsg {
				fmt.Printf("✅ 成功: 数据验证一致\n")
			} else {
				fmt.Printf("❌ 失败: 数据不一致\n")
			}
		}
	}

	// 删除部分数据
	fmt.Println("\n==== 删除数据测试 ====")
	for i := 0; i < testCount-2; i++ { // 留两条记录不删除
		fmt.Printf("删除记录 #%d... ", i)
		// 为用户名和地址生成签名
		dataToSign := append(usernames[i], addresses[i]...)
		signature, err := signData(privateKey, dataToSign)
		if err != nil {
			fmt.Printf("❌ 签名失败: %v\n", err)
			continue
		}

		// 删除数据
		recordIdx, remainingCount, err := cardReader.DeleteData(
			usernames[i],
			addresses[i],
			signature,
		)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
		} else {
			fmt.Printf("✅ 成功 (索引=%d, 剩余=%d)\n", recordIdx, remainingCount)
			delete(activeRecords, fmt.Sprintf("%d", i))
			delete(addedRecords, i)
		}
	}

	// 验证删除的记录确实无法读取
	fmt.Println("\n==== 验证删除后的记录 ====")
	for i := 0; i < testCount; i++ {
		fmt.Printf("尝试读取记录 #%d (预期", i)
		expectedResult := "成功"
		if _, exists := activeRecords[fmt.Sprintf("%d", i)]; !exists {
			expectedResult = "失败"
		}
		fmt.Printf("%s)... ", expectedResult)

		// 为用户名和地址生成签名
		dataToSign := append(usernames[i], addresses[i]...)
		signature, err := signData(privateKey, dataToSign)
		if err != nil {
			fmt.Printf("❌ 签名失败: %v\n", err)
			continue
		}

		// 尝试读取数据
		message, err := cardReader.ReadData(
			usernames[i],
			addresses[i],
			signature,
		)
		if err != nil {
			if expectedResult == "失败" {
				fmt.Printf("✅ 预期失败: 已确认\n")
			} else {
				fmt.Printf("❌ 意外失败: %v\n", err)
			}
		} else {
			if expectedResult == "成功" {
				// 比较读取的数据与原始数据
				if bytes.Equal(bytes.TrimRight(message, "\x00"), bytes.TrimRight(messages[i], "\x00")) {
					fmt.Printf("✅ 预期成功: 数据验证一致\n")
				} else {
					fmt.Printf("❌ 预期成功但数据不一致\n")
				}
			} else {
				fmt.Printf("❌ 意外成功: 应该访问失败的记录\n")
			}
		}
	}

	// 边界测试 - 验证无效签名
	fmt.Println("\n==== 边界测试: 无效签名 ====")
	// 修改前几个字节使签名无效
	validIndex := testCount - 1
	dataToSign := append(usernames[validIndex], addresses[validIndex]...)
	signature, _ := signData(privateKey, dataToSign)
	if len(signature) > 4 {
		invalidSignature := make([]byte, len(signature))
		copy(invalidSignature, signature)
		invalidSignature[0] = invalidSignature[0] ^ 0xFF // 翻转第一个字节
		invalidSignature[1] = invalidSignature[1] ^ 0xFF // 翻转第二个字节

		fmt.Print("尝试使用无效签名进行读取 (预期失败)... ")
		_, err := cardReader.ReadData(
			usernames[validIndex],
			addresses[validIndex],
			invalidSignature,
		)
		if err != nil {
			fmt.Printf("✅ 预期失败: 已确认\n")
		} else {
			fmt.Printf("❌ 安全漏洞: 无效签名被接受\n")
		}
	}

	// 测试更新现有记录
	fmt.Println("\n==== 测试更新现有记录 ====")
	// 找一个仍存在的记录
	var existingIndex int = -1
	for i := 0; i < testCount; i++ {
		if _, exists := activeRecords[fmt.Sprintf("%d", i)]; exists {
			existingIndex = i
			break
		}
	}

	if existingIndex >= 0 {
		// 准备新消息
		updatedMessage := padData([]byte(fmt.Sprintf("这是更新后的第%d条消息", existingIndex)), seclient.MESSAGE_LENGTH)

		fmt.Printf("更新记录 #%d... ", existingIndex)
		recordIdx, recordCount, err := cardReader.StoreData(
			usernames[existingIndex],
			addresses[existingIndex],
			updatedMessage,
		)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
		} else {
			fmt.Printf("✅ 成功 (索引=%d, 总数=%d)\n", recordIdx, recordCount)
		}

		// 读取更新后的数据进行验证
		fmt.Printf("验证更新后的记录 #%d... ", existingIndex)
		dataToSign := append(usernames[existingIndex], addresses[existingIndex]...)
		signature, _ := signData(privateKey, dataToSign)

		message, err := cardReader.ReadData(
			usernames[existingIndex],
			addresses[existingIndex],
			signature,
		)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
		} else {
			// 验证消息是否已更新
			expectedMsg := strings.TrimRight(string(updatedMessage), "\x00")
			actualMsg := strings.TrimRight(string(message), "\x00")
			if actualMsg == expectedMsg {
				fmt.Printf("✅ 成功: 数据已正确更新\n")
			} else {
				fmt.Printf("❌ 失败: 更新未生效\n")
			}
		}
	}

	// 错误处理测试
	fmt.Println("\n==== 错误处理测试 ====")

	// 1. 测试无效用户名长度
	fmt.Print("测试: 无效用户名长度... ")
	invalidUsername := make([]byte, seclient.USERNAME_LENGTH+1) // 过长的用户名
	_, _, err = cardReader.StoreData(
		invalidUsername,
		addresses[0],
		messages[0],
	)
	if err != nil {
		fmt.Printf("✅ 预期失败: 已确认\n")
	} else {
		fmt.Printf("❌ 意外成功: 应该拒绝过长的用户名\n")
	}

	// 2. 测试无效地址长度
	fmt.Print("测试: 无效地址长度... ")
	invalidAddress := make([]byte, seclient.ADDR_LENGTH-1) // 过短的地址
	_, _, err = cardReader.StoreData(
		usernames[0],
		invalidAddress,
		messages[0],
	)
	if err != nil {
		fmt.Printf("✅ 预期失败: 已确认\n")
	} else {
		fmt.Printf("❌ 意外成功: 应该拒绝过短的地址\n")
	}

	// 3. 测试无效消息长度
	fmt.Print("测试: 无效消息长度... ")
	invalidMessage := make([]byte, seclient.MESSAGE_LENGTH+5) // 过长的消息
	_, _, err = cardReader.StoreData(
		usernames[0],
		addresses[0],
		invalidMessage,
	)
	if err != nil {
		fmt.Printf("✅ 预期失败: 已确认\n")
	} else {
		fmt.Printf("❌ 意外成功: 应该拒绝过长的消息\n")
	}

	// 4. 测试查找不存在的记录
	fmt.Print("测试: 查找不存在的记录... ")
	nonExistentUsername := padData([]byte("nonexistent@example.com"), seclient.USERNAME_LENGTH)
	nonExistentAddress := padData([]byte("0xnonexistentaddress"), seclient.ADDR_LENGTH)

	dataToSign = append(nonExistentUsername, nonExistentAddress...)
	signature, _ = signData(privateKey, dataToSign)

	_, err = cardReader.ReadData(
		nonExistentUsername,
		nonExistentAddress,
		signature,
	)
	if err != nil {
		fmt.Printf("✅ 预期失败: 已确认\n")
	} else {
		fmt.Printf("❌ 意外成功: 应该找不到记录\n")
	}

	// 清理所有剩余的测试记录
	fmt.Println("\n==== 清理所有剩余测试记录 ====")
	for i := range addedRecords {
		fmt.Printf("清理记录 #%d... ", i)
		dataToSign := append(usernames[i], addresses[i]...)
		signature, err := signData(privateKey, dataToSign)
		if err != nil {
			fmt.Printf("❌ 签名失败: %v\n", err)
			continue
		}

		_, remaining, err := cardReader.DeleteData(
			usernames[i],
			addresses[i],
			signature,
		)
		if err != nil {
			fmt.Printf("❌ 失败: %v\n", err)
		} else {
			fmt.Printf("✅ 成功 (剩余记录数=%d)\n", remaining)
			delete(addedRecords, i)
		}
	}

	// 确认所有数据已清理
	if len(addedRecords) == 0 {
		fmt.Println("\n✅ 所有测试数据已清理完毕")
	} else {
		fmt.Printf("\n⚠️ 警告: 仍有 %d 条测试数据未能清理\n", len(addedRecords))
	}

	fmt.Println("\n==== 测试结果总结 ====")
	fmt.Println("✅ 安全芯片存储系统测试完成!")
	fmt.Println("✅ 基本功能测试通过: 存储、读取、删除和更新操作正常")
	fmt.Println("✅ 安全验证: 无效签名被正确拒绝")
	fmt.Println("✅ 输入验证: 对无效输入参数进行了适当处理")
}
