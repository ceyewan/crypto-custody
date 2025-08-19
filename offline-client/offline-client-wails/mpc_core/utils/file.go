package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

// EnsureDir 确保目录存在
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// WriteFile 将数据写入文件
func WriteFile(path string, data []byte) error {
	return os.WriteFile(path, data, 0644)
}

// ReadFile 从文件读取数据
func ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

// DeleteFile 删除文件
func DeleteFile(path string) error {
	return os.Remove(path)
}

// CompressData 压缩数据
func CompressData(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w, err := gzip.NewWriterLevel(&b, gzip.BestCompression)
	if err != nil {
		return nil, err
	}
	_, err = w.Write(data)
	if err != nil {
		return nil, err
	}
	err = w.Close()
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// DecompressData 解压数据
func DecompressData(data []byte) ([]byte, error) {
	b := bytes.NewBuffer(data)
	r, err := gzip.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	var resB bytes.Buffer
	_, err = io.Copy(&resB, r)
	if err != nil {
		return nil, err
	}

	return resB.Bytes(), nil
}

// EncodeBase64 将字节数组编码为Base64字符串
func EncodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DecodeBase64 将Base64字符串解码为字节数组
func DecodeBase64(s string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(s)
}

// ParseJSONFile 解析JSON文件并返回解析结果
func ParseJSONFile(filePath string) (map[string]interface{}, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return result, nil
}

// ExtractPublicKey 从JSON数据中提取公钥
func ExtractPublicKey(jsonData map[string]interface{}) (string, error) {
	// 获取y_sum_s字段
	ySumS, ok := jsonData["y_sum_s"]
	if !ok {
		return "", errors.New("找不到y_sum_s字段")
	}

	ySumSMap, ok := ySumS.(map[string]interface{})
	if !ok {
		return "", errors.New("y_sum_s字段格式不正确")
	}

	// 获取point字段
	point, ok := ySumSMap["point"]
	if !ok {
		return "", errors.New("找不到point字段")
	}

	pointArray, ok := point.([]interface{})
	if !ok {
		return "", errors.New("point字段不是数组")
	}

	// 构建公钥
	return formatPublicKeyFromPoints(pointArray)
}

// 格式化点数组为公钥字符串
func formatPublicKeyFromPoints(pointArray []interface{}) (string, error) {
	var publicKeyParts []string
	for _, value := range pointArray {
		intValue, ok := value.(float64)
		if !ok {
			return "", errors.New("point数组包含非数字元素")
		}
		publicKeyParts = append(publicKeyParts, fmt.Sprintf("%02x", int(intValue)))
	}

	return strings.Join(publicKeyParts, ""), nil
}
