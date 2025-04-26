package main

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type KeygenPayload struct {
	Threshold int    `json:"threshold"`
	Parties   int    `json:"parties"`
	Index     int    `json:"index"`
	Filename  string `json:"filename"`
	UserName  string `json:"userName"`
}

type KeygenResponse struct {
	Success      bool   `json:"success"`
	Address      string `json:"address"`
	EncryptedKey string `json:"encryptedKey"`
}

const (
	serverURL   = "http://localhost:8080"
	apiEndpoint = serverURL + "/api/v1/mpc/keygen"
	threshold   = 2
	parties     = 3
)

func testKeygen(threshold, parties, index int, username string, wg *sync.WaitGroup, results *[]KeygenResponse) {
	defer wg.Done()

	payload := KeygenPayload{
		Threshold: threshold,
		Parties:   parties,
		Index:     index,
		Filename:  fmt.Sprintf("keygen_test_%d.json", index),
		UserName:  username,
	}

	fmt.Printf("\n发起密钥生成请求:\n")
	fmt.Printf("门限值: %d\n", threshold)
	fmt.Printf("参与方总数: %d\n", parties)
	fmt.Printf("当前参与方序号: %d\n", index)
	fmt.Printf("用户名: %s\n", username)

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("\n请求数据序列化失败: %v\n", err)
		return
	}

	resp, err := http.Post(apiEndpoint, "application/json", bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Printf("\n请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("\n请求失败，状态码: %d\n", resp.StatusCode)
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("错误响应: %s\n", string(body))
		return
	}

	var result KeygenResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		fmt.Printf("\n响应解析失败: %v\n", err)
		return
	}

	if result.Success {
		fmt.Println("\n密钥生成成功!")
		fmt.Printf("地址: %s\n", result.Address)
		fmt.Printf("加密密钥长度: %d 字节\n", len(result.EncryptedKey))

		outputFile := fmt.Sprintf("keygen_result_%d.json", index)
		file, err := os.Create(outputFile)
		if err != nil {
			fmt.Printf("结果保存失败: %v\n", err)
			return
		}
		defer file.Close()

		encoder := json.NewEncoder(file)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(result); err != nil {
			fmt.Printf("结果写入文件失败: %v\n", err)
			return
		}
		fmt.Printf("结果已保存到: %s\n", outputFile)
	}

	*results = append(*results, result)
}

const (
	// API 端点
	API_ENDPOINT = "http://localhost:8080/api/v1/mpc/sign" // 请根据实际情况修改
	// 私钥文件路径
	PRIVATE_KEY_FILE = "ec_private_key.pem"
)

var KEYGEN_RESULT_FILES = []string{
	"keygen_result_1.json",
	"keygen_result_2.json",
}

// KeyGenResult 表示密钥生成结果
type KeyGenResult struct {
	Success      bool   `json:"success"`
	Address      string `json:"address"`
	EncryptedKey string `json:"encryptedKey"`
	Username     string `json:"username,omitempty"` // 可能不存在于文件中
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

// 从JSON文件加载密钥生成结果
func loadKeyGenResult(filename string) (*KeyGenResult, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("读取密钥生成结果文件失败: %v", err)
	}

	var result KeyGenResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("解析JSON数据失败: %v", err)
	}

	// 如果文件中没有用户名，则从文件名生成一个
	if result.Username == "" {
		baseName := filepath.Base(filename)
		result.Username = fmt.Sprintf("user_%s", baseName)
	}

	return &result, nil
}

// 对数据进行签名
func signData(privateKey *ecdsa.PrivateKey, data []byte) (string, error) {
	// 计算消息哈希
	hash := sha256.Sum256(data)

	// 签名
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return "", fmt.Errorf("签名失败: %v", err)
	}

	// 将r和s转换为DER格式
	signature, err := marshalECDSASignature(r, s)
	if err != nil {
		return "", err
	}

	// 转为base64
	return base64.StdEncoding.EncodeToString(signature), nil
}

// 将ECDSA签名转换为DER格式
func marshalECDSASignature(r, s *big.Int) ([]byte, error) {
	// ASN.1格式的签名结构
	type ecdsaSignature struct {
		R, S *big.Int
	}
	return asn1.Marshal(ecdsaSignature{R: r, S: s})
}

// 发送签名请求
func sendSignRequest(request SignRequest, index int, wg *sync.WaitGroup, results *sync.Map) {
	defer wg.Done()

	fmt.Printf("\n=== 参与方 %d 发起签名请求 ===\n", index)
	fmt.Printf("用户名: %s\n", request.UserName)
	fmt.Printf("地址: %s\n", request.Address)

	// 转换为JSON
	jsonData, err := json.Marshal(request)
	if err != nil {
		fmt.Printf("❌ 参与方 %d JSON序列化失败: %v\n", index, err)
		results.Store(index, &SignResponse{Success: false, Message: err.Error()})
		return
	}

	// 创建HTTP请求
	req, err := http.NewRequest("POST", API_ENDPOINT, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("❌ 参与方 %d 创建HTTP请求失败: %v\n", index, err)
		results.Store(index, &SignResponse{Success: false, Message: err.Error()})
		return
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")

	// 发送请求
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("❌ 参与方 %d 发送HTTP请求失败: %v\n", index, err)
		results.Store(index, &SignResponse{Success: false, Message: err.Error()})
		return
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("❌ 参与方 %d 读取响应失败: %v\n", index, err)
		results.Store(index, &SignResponse{Success: false, Message: err.Error()})
		return
	}

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ 参与方 %d HTTP请求失败, 状态码: %d, 响应: %s\n", index, resp.StatusCode, string(body))
		results.Store(index, &SignResponse{Success: false, Message: string(body)})
		return
	}

	// 解析响应
	var signResponse SignResponse
	if err := json.Unmarshal(body, &signResponse); err != nil {
		fmt.Printf("❌ 参与方 %d 解析响应JSON失败: %v\n", index, err)
		results.Store(index, &SignResponse{Success: false, Message: err.Error()})
		return
	}

	fmt.Printf("✅ 参与方 %d 签名%s\n", index, map[bool]string{true: "成功", false: "失败"}[signResponse.Success])
	if signResponse.Success {
		fmt.Printf("签名结果: %s\n", signResponse.Signature)
	} else {
		fmt.Printf("错误消息: %s\n", signResponse.Message)
	}

	results.Store(index, &signResponse)
}

func main() {
	fmt.Println("开始测试密钥生成服务...")
	fmt.Printf("服务器地址: %s\n", serverURL)

	var wg sync.WaitGroup
	// var keygenResults sync.Mutex
	results := make([]KeygenResponse, 0)

	// 使用不同用户名进行密钥生成
	for i := 1; i <= parties; i++ {
		username := fmt.Sprintf("test_user_%d", i)
		fmt.Printf("\n=== 启动参与方 %d 进程 (用户名: %s) ===\n", i, username)
		wg.Add(1)
		go testKeygen(threshold, parties, i, username, &wg, &results)
		time.Sleep(1 * time.Second) // 等待1秒再发起下一个
	}

	wg.Wait()

	fmt.Println("\n=== 密钥生成测试完成 ===")
	fmt.Printf("成功生成密钥的参与方数量: %d/%d\n", len(results), parties)
	if len(results) > 0 {
		fmt.Println("\n所有生成的地址:")
		for i, result := range results {
			fmt.Printf("参与方 %d: %s\n", i+1, result.Address)
		}
	}

	// 获取共享的地址（应该一致）
	var sharedAddress string
	if len(results) > 0 {
		sharedAddress = results[0].Address
	} else {
		fmt.Println("❌ 没有成功生成的密钥，无法进行签名测试")
		return
	}

	fmt.Println("\n=== 开始并发签名测试 ===")
	fmt.Printf("使用参与方1和2进行2/3门限签名\n")

	// 加载私钥
	privateKey, err := loadPrivateKey(PRIVATE_KEY_FILE)
	if err != nil {
		fmt.Printf("❌ 加载私钥失败: %v\n", err)
		return
	}
	fmt.Println("✅ 私钥加载成功")

	// 准备测试数据
	testData := fmt.Sprintf("这是一条测试消息，时间戳: %d", time.Now().Unix())
	fmt.Printf("测试数据: %s\n", testData)

	// 签名请求并发处理
	var signWg sync.WaitGroup
	var signResults sync.Map

	for i := 1; i <= 2; i++ { // 只使用参与方1和2
		keygenFile := fmt.Sprintf("keygen_result_%d.json", i)
		username := fmt.Sprintf("test_user_%d", i)

		// 加载密钥生成结果
		keygenResult, err := loadKeyGenResult(keygenFile)
		if err != nil {
			fmt.Printf("❌ 加载密钥生成结果失败 (参与方 %d): %v\n", i, err)
			continue
		}

		// 对地址和用户名进行签名
		hash := sha256.Sum256([]byte(username))
		userBytes := hash[:]
		addrBytes, _ := hex.DecodeString(sharedAddress[2:]) // 使用共享地址
		signatureData := append(userBytes, addrBytes...)
		signature, err := signData(privateKey, signatureData)
		if err != nil {
			fmt.Printf("❌ 生成签名失败 (参与方 %d): %v\n", i, err)
			continue
		}

		// 构建签名请求
		signRequest := SignRequest{
			Parties:      "1,2",
			Data:         testData,
			Filename:     keygenFile,
			UserName:     username,
			Address:      sharedAddress,
			EncryptedKey: keygenResult.EncryptedKey,
			Signature:    signature,
		}

		// 并发发送签名请求
		signWg.Add(1)
		go sendSignRequest(signRequest, i, &signWg, &signResults)
		time.Sleep(1 * time.Second) // 等待1秒再发起下一个
	}

	// 等待所有签名请求完成
	signWg.Wait()

	// 检查签名结果
	fmt.Println("\n=== 签名测试结果汇总 ===")
	successCount := 0

	for i := 1; i <= 2; i++ {
		if resultAny, ok := signResults.Load(i); ok {
			if result, ok := resultAny.(*SignResponse); ok {
				if result.Success {
					successCount++
					fmt.Printf("参与方 %d: ✅ 签名成功\n", i)
				} else {
					fmt.Printf("参与方 %d: ❌ 签名失败: %s\n", i, result.Message)
				}
			}
		} else {
			fmt.Printf("参与方 %d: ❓ 没有返回结果\n", i)
		}
	}

	fmt.Printf("\n成功签名的参与方数量: %d/2\n", successCount)
	fmt.Println("=== 签名测试完成 ===")
}
