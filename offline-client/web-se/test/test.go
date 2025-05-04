package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"

	"test/utils"
)

const (
	serverBaseURL      = "http://localhost:8080"
	keygenAPIEndpoint  = serverBaseURL + "/api/v1/mpc/keygen"
	signAPIEndpoint    = serverBaseURL + "/api/v1/mpc/sign"
	cplcAPIEndpoint    = serverBaseURL + "/api/v1/mpc/cplc"
	deleteAPIEndpoint  = serverBaseURL + "/api/v1/mpc/delete"
	privateKeyFilePath = "ec_private_key.pem"
	keygenThreshold    = 1
	totalParticipants  = 3
)

// 密钥生成测试
func testKeygen(threshold, parties, index int, username, filename string, wg *sync.WaitGroup) error {
	defer wg.Done()

	payload := utils.KeygenPayload{
		Threshold: threshold,
		Parties:   parties,
		Index:     index,
		Filename:  filename,
		UserName:  username,
	}

	fmt.Printf("\n=== 发起密钥生成请求 ===\n")
	fmt.Printf("用户名: %s\n", username)
	fmt.Printf("门限值: %d\n", threshold)
	fmt.Printf("参与方总数: %d\n", parties)
	fmt.Printf("当前参与方序号: %d\n", index)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("❌ 参与方 %d JSON序列化失败: %v\n", index, err)
		return err
	}

	resp, err := http.Post(keygenAPIEndpoint, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("❌ 参与方 %d 请求失败: %v\n", index, err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("HTTP请求失败, 状态码: %d, 响应: %s", resp.StatusCode, string(body))
		fmt.Printf("❌ 参与方 %d %s\n", index, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	var result utils.KeygenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("❌ 参与方 %d 解析响应JSON失败: %v\n", index, err)
		return err
	}

	if result.Success {
		fmt.Printf("✅ 参与方 %d 密钥生成成功!\n", index)
		fmt.Printf("地址: %s\n", result.Address)
		fmt.Printf("加密密钥长度: %d 字节\n", len(result.EncryptedKey))

		outputFile := fmt.Sprintf("data/keygen_result_%d.json", index)
		// 确保目录存在
		os.MkdirAll("data", os.ModePerm)

		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Printf("❌ 参与方 %d 创建结果文件失败: %v\n", index, err)
			return err
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Printf("❌ 参与方 %d 结果写入文件失败: %v\n", index, err)
			return err
		}
		fmt.Printf("✅ 结果已保存到: %s\n", outputFile)
	} else {
		fmt.Printf("❌ 参与方 %d 密钥生成失败: %s\n", index, result.Message)
	}
	return nil
}

// 签名测试
func testSign(parties, data, filename, username, address, signature, encryptedKey string, index int, wg *sync.WaitGroup) error {
	defer wg.Done()

	payload := utils.SignRequest{
		Parties:      parties,
		Data:         data,
		Filename:     filename,
		EncryptedKey: encryptedKey,
		UserName:     username,
		Address:      address,
		Signature:    signature,
	}

	fmt.Printf("\n=== 发起签名请求 ===\n")
	fmt.Printf("用户名: %s\n", username)
	fmt.Printf("地址: %s\n", address)
	fmt.Printf("待签名数据: %s\n", data)
	fmt.Printf("参与方数量: %s\n", parties)

	// 转换为JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("❌ 参与方 %d JSON序列化失败: %v\n", index, err)
		return err
	}

	// 创建HTTP请求
	resp, err := http.Post(signAPIEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ 参与方 %d 创建HTTP请求失败: %v\n", index, err)
		return err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 参与方 %d 读取响应失败: %v\n", index, err)
		return err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTP请求失败, 状态码: %d, 响应: %s", resp.StatusCode, string(body))
		fmt.Printf("❌ 参与方 %d %s\n", index, errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// 解析响应
	var signResponse utils.SignResponse
	if err := json.Unmarshal(body, &signResponse); err != nil {
		fmt.Printf("❌ 参与方 %d 解析响应JSON失败: %v\n", index, err)
		return err
	}

	// 打印签名结果
	if signResponse.Success {
		fmt.Printf("✅ 参与方 %d 签名成功\n", index)
		fmt.Printf("签名结果: %s\n", signResponse.Signature)
	} else {
		fmt.Printf("❌ 参与方 %d 签名失败: %s\n", index, signResponse.Message)
	}
	return nil
}

// 测试获取CPLC信息
func testGetCPLC() error {
	fmt.Println("\n=== 发起获取CPLC信息请求 ===")

	// 发送GET请求
	resp, err := http.Get(cplcAPIEndpoint)
	if err != nil {
		fmt.Printf("❌ 请求失败: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		errMsg := fmt.Sprintf("HTTP请求失败, 状态码: %d, 响应: %s", resp.StatusCode, string(body))
		fmt.Printf("❌ %s\n", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	var result utils.GetCPLCResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("❌ 解析响应JSON失败: %v\n", err)
		return err
	}

	if result.Success {
		fmt.Println("✅ 获取CPLC信息成功!")
		fmt.Printf("CPLC信息: %s\n", result.CPIC)
	} else {
		fmt.Printf("❌ 获取CPLC信息失败: %s\n", result.Message)
	}
	return nil
}

// 测试删除用户数据
func testDeleteMessage(username, address, signature string) error {
	payload := utils.DeleteRequest{
		UserName:  username,
		Address:   address,
		Signature: signature,
	}

	fmt.Printf("\n=== 发起删除用户数据请求 ===\n")
	fmt.Printf("用户名: %s\n", username)
	fmt.Printf("地址: %s\n", address)
	fmt.Printf("签名长度: %d 字节\n", len(signature))

	// 转换为JSON
	jsonData, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("❌ JSON序列化失败: %v\n", err)
		return err
	}

	// 创建HTTP请求
	resp, err := http.Post(deleteAPIEndpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ 创建HTTP请求失败: %v\n", err)
		return err
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 读取响应失败: %v\n", err)
		return err
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		errMsg := fmt.Sprintf("HTTP请求失败, 状态码: %d, 响应: %s", resp.StatusCode, string(body))
		fmt.Printf("❌ %s\n", errMsg)
		return fmt.Errorf("%s", errMsg)
	}

	// 解析响应
	var deleteResponse utils.DeleteResponse
	if err := json.Unmarshal(body, &deleteResponse); err != nil {
		fmt.Printf("❌ 解析响应JSON失败: %v\n", err)
		return err
	}

	// 打印删除结果
	if deleteResponse.Success {
		fmt.Println("✅ 删除用户数据成功")
		fmt.Printf("删除的地址: %s\n", deleteResponse.Address)
	}
	return nil
}

func main() {
	// 测试获取CPLC信息
	runGetCPLCTest()

	// 测试密钥生成
	runKeygenTest()

	// 测试签名
	runSignTest()

	// 测试删除用户数据
	runDeleteMessageTest()
}

// 运行密钥生成测试
func runKeygenTest() {
	fmt.Println("\n===== 开始密钥生成测试 =====")
	fmt.Printf("服务器地址: %s\n", serverBaseURL)

	var wg sync.WaitGroup

	// 使用不同用户名进行密钥生成
	for i := 1; i <= totalParticipants; i++ {
		username := fmt.Sprintf("test_user_%d", i)
		filename := fmt.Sprintf("keygen_data_%d.json", i)
		fmt.Printf("\n=== 启动参与方 %d 进程 (用户名: %s) ===\n", i, username)
		wg.Add(1)
		go func(idx int, uname, fname string) {
			_ = testKeygen(keygenThreshold, totalParticipants, idx, uname, fname, &wg)
		}(i, username, filename)
		time.Sleep(1 * time.Second) // 等待1秒再发起下一个
	}

	wg.Wait()

	fmt.Println("\n===== 密钥生成测试完成 =====")
}

// 运行获取CPLC信息测试
func runGetCPLCTest() {
	fmt.Println("\n===== 开始获取CPLC信息测试 =====")
	fmt.Printf("服务器地址: %s\n", serverBaseURL)

	if err := testGetCPLC(); err != nil {
		fmt.Printf("❌ 获取CPLC信息测试失败: %v\n", err)
	}

	fmt.Println("\n===== 获取CPLC信息测试完成 =====")
}

// 运行签名测试
func runSignTest() {
	fmt.Println("\n===== 开始签名测试 =====")

	// 获取共享地址（从第一个成功的结果）
	var sharedAddress string
	for i := 1; i <= totalParticipants; i++ {
		keygenFile := fmt.Sprintf("data/keygen_result_%d.json", i)
		if _, err := os.Stat(keygenFile); err == nil {
			keygenResult, err := utils.LoadKeyGenResult(keygenFile)
			if err == nil && keygenResult.Success {
				sharedAddress = keygenResult.Address
				break
			}
		}
	}

	if sharedAddress == "" {
		fmt.Println("❌ 没有找到成功生成的密钥，无法进行签名测试")
		return
	}

	fmt.Printf("使用参与方1和2进行%d/%d门限签名\n", keygenThreshold, totalParticipants)
	fmt.Printf("共享地址: %s\n", sharedAddress)

	// 加载私钥
	privateKey, err := utils.LoadPrivateKey(privateKeyFilePath)
	if err != nil {
		fmt.Printf("❌ 加载私钥失败: %v\n", err)
		return
	}
	fmt.Println("✅ 私钥加载成功")

	// 准备测试数据
	testData := "\"hello\""
	fmt.Printf("测试数据: %s\n", testData)

	// 签名请求处理
	var signWg sync.WaitGroup

	// 只使用参与方1和2
	for i := 1; i <= 2; i++ {
		keygenFile := fmt.Sprintf("data/keygen_result_%d.json", i)
		if _, err := os.Stat(keygenFile); err != nil {
			fmt.Printf("❌ 找不到密钥生成结果文件: %s\n", keygenFile)
			continue
		}

		username := fmt.Sprintf("test_user_%d", i)

		// 加载密钥生成结果
		keygenResult, err := utils.LoadKeyGenResult(keygenFile)
		if err != nil {
			fmt.Printf("❌ 加载密钥生成结果失败 (参与方 %d): %v\n", i, err)
			continue
		}

		// 对地址和用户名进行签名
		hash := sha256.Sum256([]byte(username))
		userBytes := hash[:]
		addrBytes, _ := hex.DecodeString(sharedAddress[2:]) // 使用共享地址
		signatureData := append(userBytes, addrBytes...)
		signature, err := utils.SignData(privateKey, signatureData)
		if err != nil {
			fmt.Printf("❌ 生成签名失败 (参与方 %d): %v\n", i, err)
			continue
		}

		// 发送签名请求
		signWg.Add(1)
		go func(idx int) {
			filename := fmt.Sprintf("sign_data_%d.json", idx)
			_ = testSign("1,2", testData, filename, keygenResult.UserName, sharedAddress, signature, keygenResult.EncryptedKey, idx, &signWg)
		}(i)

		time.Sleep(1 * time.Second) // 等待1秒再发起下一个
	}

	// 等待所有签名请求完成
	signWg.Wait()

	fmt.Println("\n===== 签名测试完成 =====")
}

// 运行删除用户数据测试
func runDeleteMessageTest() {
	fmt.Println("\n===== 开始删除用户数据测试 =====")
	fmt.Printf("服务器地址: %s\n", serverBaseURL)

	// 加载私钥
	privateKey, err := utils.LoadPrivateKey(privateKeyFilePath)
	if err != nil {
		fmt.Printf("❌ 加载私钥失败: %v\n", err)
		return
	}
	fmt.Println("✅ 私钥加载成功")

	// 删除所有参与方的数据
	for i := 1; i <= totalParticipants; i++ {
		keygenFile := fmt.Sprintf("data/keygen_result_%d.json", i)
		if _, err := os.Stat(keygenFile); err != nil {
			fmt.Printf("❌ 找不到密钥生成结果文件: %s\n", keygenFile)
			continue
		}

		// 加载密钥生成结果
		keygenResult, err := utils.LoadKeyGenResult(keygenFile)
		if err != nil {
			fmt.Printf("❌ 加载密钥生成结果失败 (参与方 %d): %v\n", i, err)
			continue
		}

		username := keygenResult.UserName
		address := keygenResult.Address

		fmt.Printf("\n测试删除参与方 %d 数据\n", i)
		fmt.Printf("用户名: %s\n", username)
		fmt.Printf("地址: %s\n", address)

		// 对地址和用户名进行签名
		hash := sha256.Sum256([]byte(username))
		userBytes := hash[:]
		addrBytes, _ := hex.DecodeString(address[2:]) // 去掉0x前缀
		signatureData := append(userBytes, addrBytes...)
		signature, err := utils.SignData(privateKey, signatureData)
		if err != nil {
			fmt.Printf("❌ 生成签名失败 (参与方 %d): %v\n", i, err)
			continue
		}

		// 发送删除请求
		if err := testDeleteMessage(username, address, signature); err != nil {
			fmt.Printf("❌ 删除参与方 %d 数据失败: %v\n", i, err)
		} else {
			fmt.Printf("✅ 删除参与方 %d 数据成功\n", i)
		}

		// 等待一秒，防止请求过快
		time.Sleep(1 * time.Second)
	}

	fmt.Println("\n===== 删除用户数据测试完成 =====")
}
