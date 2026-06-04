package main

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/pem"
	"flag"
	"fmt"
	"os"
	"strings"

	clientcfg "offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/services"

	_ "github.com/mattn/go-sqlite3"
)

const defaultAppletAID = "A000000062CF0101"

type shard struct {
	Username string
	RecordID string
	Address  string
}

func main() {
	var dbPath, keyPath, address, readerName, appletAID, mode string
	flag.StringVar(&dbPath, "db", "../../offline-server-handoff/data/crypto-custody.db", "offline SQLite DB path")
	flag.StringVar(&keyPath, "private-key", "../../offline-server-handoff/private_keys/ec_private_key.pem", "SE authorization private key")
	flag.StringVar(&address, "address", "", "address to check; default first active offline key")
	flag.StringVar(&readerName, "reader", "", "SE reader name substring")
	flag.StringVar(&appletAID, "aid", defaultAppletAID, "SE applet AID hex")
	flag.StringVar(&mode, "mode", "sha256", "signature mode: sha256, double-sha256, raw, record-id")
	flag.Parse()

	if err := run(dbPath, keyPath, address, readerName, appletAID, mode); err != nil {
		fmt.Fprintf(os.Stderr, "[FAIL] %v\n", err)
		os.Exit(1)
	}
}

func run(dbPath, keyPath, address, readerName, appletAID, mode string) error {
	privateKey, err := loadPrivateKey(keyPath)
	if err != nil {
		return err
	}
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	if address == "" {
		if err := db.QueryRow("select address from offline_keys where status='active' order by id limit 1").Scan(&address); err != nil {
			return fmt.Errorf("load first active address: %w", err)
		}
	}
	rows, err := db.Query("select username, record_id, address from key_shards where address = ? and status='active' order by shard_index", address)
	if err != nil {
		return err
	}
	defer rows.Close()

	var shards []shard
	for rows.Next() {
		var item shard
		if err := rows.Scan(&item.Username, &item.RecordID, &item.Address); err != nil {
			return err
		}
		shards = append(shards, item)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(shards) == 0 {
		return fmt.Errorf("no active shards for address %s", address)
	}

	security, err := services.NewSecurityService(&clientcfg.Config{
		CardReaderName: readerName,
		AppletAID:      appletAID,
		TempDir:        "runs/se-read-check",
		LogDir:         "runs/se-read-check-logs",
	})
	if err != nil {
		return err
	}
	for _, item := range shards {
		signature, err := signRecord(privateKey, item.RecordID, item.Address, mode)
		if err != nil {
			return fmt.Errorf("%s sign auth: %w", item.Username, err)
		}
		data, err := security.ReadData(item.RecordID, item.Address, signature)
		if err != nil {
			fmt.Printf("[FAIL] %s record=%s address=%s: %v\n", item.Username, item.RecordID, item.Address, err)
			continue
		}
		fmt.Printf("[OK] %s record=%s address=%s bytes=%d\n", item.Username, item.RecordID, item.Address, len(data))
	}
	return nil
}

func signRecord(privateKey *ecdsa.PrivateKey, recordIDHex, addressHex, mode string) ([]byte, error) {
	recordID, err := decodeFixedHex(recordIDHex, 32, "record_id")
	if err != nil {
		return nil, err
	}
	address, err := decodeFixedHex(addressHex, 20, "address")
	if err != nil {
		return nil, err
	}
	data := make([]byte, 0, len(recordID)+len(address))
	data = append(data, recordID...)
	data = append(data, address...)
	switch mode {
	case "sha256":
		digest := sha256.Sum256(data)
		return ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	case "double-sha256":
		first := sha256.Sum256(data)
		second := sha256.Sum256(first[:])
		return ecdsa.SignASN1(rand.Reader, privateKey, second[:])
	case "raw":
		return ecdsa.SignASN1(rand.Reader, privateKey, data)
	case "record-id":
		return ecdsa.SignASN1(rand.Reader, privateKey, recordID)
	default:
		return nil, fmt.Errorf("unknown mode %s", mode)
	}
}

func decodeFixedHex(value string, length int, label string) ([]byte, error) {
	data, err := hex.DecodeString(strings.TrimPrefix(strings.TrimSpace(value), "0x"))
	if err != nil {
		return nil, fmt.Errorf("invalid %s hex: %w", label, err)
	}
	if len(data) != length {
		return nil, fmt.Errorf("%s must be %d bytes", label, length)
	}
	return data, nil
}

func loadPrivateKey(path string) (*ecdsa.PrivateKey, error) {
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}
	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, fmt.Errorf("private key is not PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	ecdsaKey, ok := key.(*ecdsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("private key is %T, not ECDSA", key)
	}
	return ecdsaKey, nil
}
