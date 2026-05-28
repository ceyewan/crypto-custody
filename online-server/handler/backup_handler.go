package handler

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"online-server/dto"
	"online-server/model"
	"online-server/service"
	"online-server/utils"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

const dbPath = "database/crypto-custody.db"

type encryptedBackupFile struct {
	Version    int    `json:"version"`
	KDF        string `json:"kdf"`
	Salt       string `json:"salt"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	PlainHash  string `json:"plainHash"`
}

func CreateHotBackup(c *gin.Context) {
	record, err := createHotBackupRecord(c)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	service.AuditAction(c, "backup.hot.create", "backup", strconv.FormatUint(uint64(record.ID), 10), "", "success", "", record)
	utils.ResponseWithData(c, "热备份创建成功", record)
}

func CreateColdBackup(c *gin.Context) {
	var req dto.ColdBackupRequest
	if !utils.BindJSON(c, &req) {
		return
	}
	record, err := createColdBackupRecord(c, req.Password)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, err.Error())
		return
	}
	service.AuditAction(c, "backup.cold.export", "backup", strconv.FormatUint(uint64(record.ID), 10), "", "success", "", record)
	utils.ResponseWithData(c, "冷备份创建成功", record)
}

func ListBackups(c *gin.Context) {
	var backups []model.BackupRecord
	if err := utils.GetDB().Order("created_at DESC").Find(&backups).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "查询备份记录失败: "+err.Error())
		return
	}
	utils.ResponseWithData(c, "查询备份记录成功", backups)
}

func DownloadBackup(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var record model.BackupRecord
	if err := utils.GetDB().First(&record, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "备份不存在")
		return
	}
	c.FileAttachment(record.FilePath, record.FileName)
}

func RestoreBackup(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var req dto.RestoreBackupRequest
	_ = c.ShouldBindJSON(&req)
	var record model.BackupRecord
	if err := utils.GetDB().First(&record, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "备份不存在")
		return
	}
	if record.Encrypted && req.Password == "" {
		utils.ResponseWithError(c, http.StatusBadRequest, "冷备份恢复需要密码")
		return
	}
	preRestore, err := createPreRestoreSnapshot(c.GetString("Username"))
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "创建恢复前快照失败: "+err.Error())
		return
	}
	restoreData, err := readBackupPayload(record, req.Password)
	if err != nil {
		utils.ResponseWithError(c, http.StatusBadRequest, "读取备份失败: "+err.Error())
		return
	}
	if err := replaceDatabaseFile(restoreData); err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "恢复数据库失败: "+err.Error())
		return
	}
	now := time.Now().Unix()
	record.RestoredBy = c.GetString("Username")
	record.RestoredAt = &now
	record.Status = "restored"
	if err := utils.GetDB().Save(&record).Error; err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "更新恢复记录失败: "+err.Error())
		return
	}
	service.AuditAction(c, "backup.restore", "backup", strconv.FormatUint(uint64(record.ID), 10), "", "success", "", gin.H{"backup": record, "preRestoreSnapshot": preRestore})
	utils.ResponseWithData(c, "备份恢复成功", gin.H{"backup": record, "preRestoreSnapshot": preRestore})
}

func VerifyBackup(c *gin.Context) {
	id, ok := utils.ParseUintParam(c, "id")
	if !ok {
		return
	}
	var record model.BackupRecord
	if err := utils.GetDB().First(&record, id).Error; err != nil {
		utils.ResponseWithError(c, http.StatusNotFound, "备份不存在")
		return
	}
	hash, err := fileHash(record.FilePath)
	if err != nil {
		utils.ResponseWithError(c, http.StatusInternalServerError, "校验备份失败: "+err.Error())
		return
	}
	utils.ResponseWithData(c, "备份校验完成", gin.H{"valid": hash == record.FileHash, "fileHash": hash})
}

func createHotBackupRecord(c *gin.Context) (*model.BackupRecord, error) {
	if err := os.MkdirAll("backups", 0755); err != nil {
		return nil, err
	}
	no := service.NewBusinessNo("BACKUP")
	name := no + ".db"
	dst := filepath.Join("backups", name)
	if err := copyFile(dbPath, dst); err != nil {
		return nil, err
	}
	hash, err := fileHash(dst)
	if err != nil {
		return nil, err
	}
	record := &model.BackupRecord{
		BackupNo: no, BackupType: "hot", FileName: name, FilePath: dst,
		FileHash: hash, Encrypted: false, CreatedBy: c.GetString("Username"), Status: "created",
	}
	if err := utils.GetDB().Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func createColdBackupRecord(c *gin.Context, password string) (*model.BackupRecord, error) {
	if password == "" {
		return nil, errors.New("冷备份密码不能为空")
	}
	if err := os.MkdirAll("backups", 0755); err != nil {
		return nil, err
	}
	plain, err := os.ReadFile(dbPath)
	if err != nil {
		return nil, err
	}
	no := service.NewBusinessNo("BACKUP")
	name := no + ".cold.enc"
	dst := filepath.Join("backups", name)
	if err := writeEncryptedBackup(dst, plain, password); err != nil {
		return nil, err
	}
	hash, err := fileHash(dst)
	if err != nil {
		return nil, err
	}
	record := &model.BackupRecord{
		BackupNo: no, BackupType: "cold", FileName: name, FilePath: dst,
		FileHash: hash, Encrypted: true, CreatedBy: c.GetString("Username"), Status: "created",
	}
	if err := utils.GetDB().Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func createPreRestoreSnapshot(username string) (*model.BackupRecord, error) {
	if err := os.MkdirAll("backups", 0755); err != nil {
		return nil, err
	}
	no := service.NewBusinessNo("PRERESTORE")
	name := no + ".db"
	dst := filepath.Join("backups", name)
	if err := copyFile(dbPath, dst); err != nil {
		return nil, err
	}
	hash, err := fileHash(dst)
	if err != nil {
		return nil, err
	}
	record := &model.BackupRecord{
		BackupNo: no, BackupType: "hot", FileName: name, FilePath: dst,
		FileHash: hash, Encrypted: false, CreatedBy: username, Status: "created",
	}
	if err := utils.GetDB().Create(record).Error; err != nil {
		return nil, err
	}
	return record, nil
}

func readBackupPayload(record model.BackupRecord, password string) ([]byte, error) {
	if record.Encrypted {
		return readEncryptedBackup(record.FilePath, password)
	}
	return os.ReadFile(record.FilePath)
}

func replaceDatabaseFile(data []byte) error {
	tmpPath := dbPath + ".restore.tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return err
	}
	utils.CloseDB()
	if err := os.Rename(tmpPath, dbPath); err != nil {
		_ = os.Remove(tmpPath)
		_ = utils.InitDB()
		return err
	}
	return utils.InitDB()
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func writeEncryptedBackup(path string, plain []byte, password string) error {
	salt := make([]byte, 16)
	nonce := make([]byte, 12)
	if _, err := rand.Read(salt); err != nil {
		return err
	}
	if _, err := rand.Read(nonce); err != nil {
		return err
	}
	key := deriveBackupKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}
	plainHash := sha256.Sum256(plain)
	envelope := encryptedBackupFile{
		Version:    1,
		KDF:        "sha256-100000",
		Salt:       base64.StdEncoding.EncodeToString(salt),
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(gcm.Seal(nil, nonce, plain, nil)),
		PlainHash:  hex.EncodeToString(plainHash[:]),
	}
	data, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func readEncryptedBackup(path string, password string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var envelope encryptedBackupFile
	if err := json.Unmarshal(data, &envelope); err != nil {
		return nil, err
	}
	salt, err := base64.StdEncoding.DecodeString(envelope.Salt)
	if err != nil {
		return nil, err
	}
	nonce, err := base64.StdEncoding.DecodeString(envelope.Nonce)
	if err != nil {
		return nil, err
	}
	ciphertext, err := base64.StdEncoding.DecodeString(envelope.Ciphertext)
	if err != nil {
		return nil, err
	}
	key := deriveBackupKey(password, salt)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("备份密码错误或文件已损坏")
	}
	sum := sha256.Sum256(plain)
	if hex.EncodeToString(sum[:]) != envelope.PlainHash {
		return nil, errors.New("备份明文哈希校验失败")
	}
	return plain, nil
}

func deriveBackupKey(password string, salt []byte) []byte {
	data := append([]byte(password), salt...)
	sum := sha256.Sum256(data)
	for i := 0; i < 100000; i++ {
		next := sha256.Sum256(sum[:])
		sum = next
	}
	return sum[:]
}

func fileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
