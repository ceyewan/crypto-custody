package utils

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"io"
	"os"
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
