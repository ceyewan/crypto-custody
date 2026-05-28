package smoke

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/seclient"
	"offline-client-wails/mpc_core/services"
)

const (
	DefaultAppletAID   = "A000000062CF0101"
	DefaultRecordCount = 5
)

type Options struct {
	ReaderName     string
	AppletAID      string
	PrivateKeyPath string
	RecordCount    int
	Debug          bool
	SkipDirect     bool
	SkipService    bool
	Output         io.Writer
}

type smokeRecord struct {
	RecordID []byte
	Address  []byte
	Message  []byte
	Active   bool
}

func Run(opts Options) error {
	out := opts.Output
	if out == nil {
		out = io.Discard
	}
	if opts.AppletAID == "" {
		opts.AppletAID = DefaultAppletAID
	}
	if opts.RecordCount <= 0 {
		opts.RecordCount = DefaultRecordCount
	}
	if opts.SkipDirect && opts.SkipService {
		return errors.New("at least one smoke suite must be enabled")
	}

	aid, err := parseHexBytes(opts.AppletAID, -1, "applet AID")
	if err != nil {
		return err
	}

	privateKeyPath, err := resolvePrivateKeyPath(opts.PrivateKeyPath)
	if err != nil {
		return err
	}
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return err
	}

	fmt.Fprintf(out, "SE smoke test\n")
	fmt.Fprintf(out, "Applet AID: %s\n", strings.ToUpper(strings.TrimPrefix(opts.AppletAID, "0x")))
	fmt.Fprintf(out, "Reader: %s\n", valueOrDefault(opts.ReaderName, "<first available reader>"))
	fmt.Fprintf(out, "Private key: %s\n", privateKeyPath)

	if !opts.SkipDirect {
		if err := runDirectSuite(out, opts, aid, privateKey); err != nil {
			return err
		}
	}

	if !opts.SkipService {
		if err := runSecurityServiceSuite(out, opts, privateKey); err != nil {
			return err
		}
	}

	fmt.Fprintf(out, "\n[OK] SE smoke test completed\n")
	return nil
}

func runDirectSuite(out io.Writer, opts Options, aid []byte, privateKey *ecdsa.PrivateKey) (err error) {
	fmt.Fprintf(out, "\n== Direct mpc_core/seclient suite ==\n")

	reader, err := seclient.NewCardReader(seclient.WithDebug(opts.Debug))
	if err != nil {
		return fmt.Errorf("create card reader: %w", err)
	}
	defer reader.Close()

	if err := reader.Connect(opts.ReaderName); err != nil {
		return fmt.Errorf("connect reader: %w", err)
	}
	cplc, err := reader.GetCPLC()
	if err != nil {
		return fmt.Errorf("get CPLC: %w", err)
	}
	fmt.Fprintf(out, "[OK] Connected, CPLC=%s\n", strings.ToUpper(hex.EncodeToString(cplc)))
	if err := reader.SelectApplet(aid); err != nil {
		return fmt.Errorf("select applet: %w", err)
	}
	fmt.Fprintf(out, "[OK] selected applet\n")

	records, err := generateRecords(opts.RecordCount)
	if err != nil {
		return err
	}
	cleaned := false
	defer func() {
		if cleaned {
			return
		}
		if cleanupErr := cleanupDirectRecords(out, reader, privateKey, records); cleanupErr != nil && err == nil {
			err = cleanupErr
		}
	}()

	fmt.Fprintf(out, "\nStore records\n")
	for i := range records {
		index, count, err := reader.StoreData(records[i].RecordID, records[i].Address, records[i].Message)
		if err != nil {
			return fmt.Errorf("store record %d: %w", i, err)
		}
		records[i].Active = true
		fmt.Fprintf(out, "[OK] store #%d index=%d count=%d\n", i, index, count)
	}

	fmt.Fprintf(out, "\nRead records\n")
	for i := range records {
		message, err := reader.ReadData(records[i].RecordID, records[i].Address, mustSignRecord(privateKey, records[i]))
		if err != nil {
			return fmt.Errorf("read record %d: %w", i, err)
		}
		if !bytes.Equal(message, records[i].Message) {
			return fmt.Errorf("read record %d: message mismatch", i)
		}
		fmt.Fprintf(out, "[OK] read #%d\n", i)
	}

	deleteCount := opts.RecordCount - 2
	if deleteCount < 1 {
		deleteCount = 1
	}
	fmt.Fprintf(out, "\nDelete first %d records\n", deleteCount)
	for i := 0; i < deleteCount; i++ {
		index, remaining, err := reader.DeleteData(records[i].RecordID, records[i].Address, mustSignRecord(privateKey, records[i]))
		if err != nil {
			return fmt.Errorf("delete record %d: %w", i, err)
		}
		records[i].Active = false
		fmt.Fprintf(out, "[OK] delete #%d index=%d remaining=%d\n", i, index, remaining)
	}

	fmt.Fprintf(out, "\nVerify deleted and active records\n")
	for i := range records {
		message, err := reader.ReadData(records[i].RecordID, records[i].Address, mustSignRecord(privateKey, records[i]))
		if records[i].Active {
			if err != nil {
				return fmt.Errorf("read active record %d: %w", i, err)
			}
			if !bytes.Equal(message, records[i].Message) {
				return fmt.Errorf("read active record %d: message mismatch", i)
			}
			fmt.Fprintf(out, "[OK] active #%d still readable\n", i)
			continue
		}
		if err == nil {
			return fmt.Errorf("deleted record %d is still readable", i)
		}
		fmt.Fprintf(out, "[OK] deleted #%d rejected\n", i)
	}

	activeIndex := firstActiveRecord(records)
	if activeIndex < 0 {
		return errors.New("no active record left for invalid-signature and update tests")
	}

	fmt.Fprintf(out, "\nInvalid signature check\n")
	invalidSignature := mustSignRecord(privateKey, records[activeIndex])
	if len(invalidSignature) > 1 {
		invalidSignature[0] ^= 0xFF
		invalidSignature[1] ^= 0xFF
	}
	if _, err := reader.ReadData(records[activeIndex].RecordID, records[activeIndex].Address, invalidSignature); err == nil {
		return errors.New("invalid signature was accepted")
	}
	fmt.Fprintf(out, "[OK] invalid signature rejected\n")

	fmt.Fprintf(out, "\nUpdate active record\n")
	updatedMessage := padASCII(fmt.Sprintf("updated-record-%02d", activeIndex), seclient.MESSAGE_LENGTH)
	if _, _, err := reader.StoreData(records[activeIndex].RecordID, records[activeIndex].Address, updatedMessage); err != nil {
		return fmt.Errorf("update record %d: %w", activeIndex, err)
	}
	records[activeIndex].Message = updatedMessage
	message, err := reader.ReadData(records[activeIndex].RecordID, records[activeIndex].Address, mustSignRecord(privateKey, records[activeIndex]))
	if err != nil {
		return fmt.Errorf("read updated record %d: %w", activeIndex, err)
	}
	if !bytes.Equal(message, updatedMessage) {
		return fmt.Errorf("updated record %d: message mismatch", activeIndex)
	}
	fmt.Fprintf(out, "[OK] update verified\n")

	fmt.Fprintf(out, "\nInput and not-found checks\n")
	if _, _, err := reader.StoreData(make([]byte, seclient.RECORD_ID_LENGTH+1), records[activeIndex].Address, records[activeIndex].Message); err == nil {
		return errors.New("invalid record_id length accepted")
	}
	fmt.Fprintf(out, "[OK] invalid record_id length rejected\n")
	if _, _, err := reader.StoreData(records[activeIndex].RecordID, make([]byte, seclient.ADDR_LENGTH-1), records[activeIndex].Message); err == nil {
		return errors.New("invalid address length accepted")
	}
	fmt.Fprintf(out, "[OK] invalid address length rejected\n")
	if _, _, err := reader.StoreData(records[activeIndex].RecordID, records[activeIndex].Address, make([]byte, seclient.MESSAGE_LENGTH+1)); err == nil {
		return errors.New("invalid message length accepted")
	}
	fmt.Fprintf(out, "[OK] invalid message length rejected\n")

	missing := smokeRecord{
		RecordID: mustRandomBytes(seclient.RECORD_ID_LENGTH),
		Address:  mustRandomBytes(seclient.ADDR_LENGTH),
		Message:  mustRandomBytes(seclient.MESSAGE_LENGTH),
	}
	if _, err := reader.ReadData(missing.RecordID, missing.Address, mustSignRecord(privateKey, missing)); err == nil {
		return errors.New("missing record was readable")
	}
	fmt.Fprintf(out, "[OK] missing record rejected\n")

	fmt.Fprintf(out, "\nCleanup direct records\n")
	if err := cleanupDirectRecords(out, reader, privateKey, records); err != nil {
		return err
	}
	cleaned = true
	fmt.Fprintf(out, "[OK] direct suite cleanup complete\n")
	return nil
}

func runSecurityServiceSuite(out io.Writer, opts Options, privateKey *ecdsa.PrivateKey) (err error) {
	fmt.Fprintf(out, "\n== SecurityService suite ==\n")

	service, err := services.NewSecurityService(&config.Config{
		Debug:          opts.Debug,
		CardReaderName: opts.ReaderName,
		AppletAID:      opts.AppletAID,
	})
	if err != nil {
		return fmt.Errorf("create security service: %w", err)
	}
	defer service.Close()

	cplc, err := service.GetCPLC()
	if err != nil {
		return fmt.Errorf("service get CPLC: %w", err)
	}
	fmt.Fprintf(out, "[OK] service get CPLC=%s\n", strings.ToUpper(hex.EncodeToString(cplc)))

	record := smokeRecord{
		RecordID: mustRandomBytes(seclient.RECORD_ID_LENGTH),
		Address:  mustRandomBytes(seclient.ADDR_LENGTH),
		Message:  mustRandomBytes(seclient.MESSAGE_LENGTH),
	}
	recordIDHex := hex.EncodeToString(record.RecordID)
	addressHex := "0x" + hex.EncodeToString(record.Address)
	stored := false
	defer func() {
		if !stored {
			return
		}
		cleanupErr := service.DeleteData(recordIDHex, addressHex, mustSignRecord(privateKey, record))
		if cleanupErr != nil && err == nil {
			err = fmt.Errorf("cleanup security service record: %w", cleanupErr)
		}
	}()

	if err := service.StoreData(recordIDHex, addressHex, record.Message); err != nil {
		return fmt.Errorf("service store: %w", err)
	}
	stored = true
	fmt.Fprintf(out, "[OK] service store\n")

	message, err := service.ReadData(recordIDHex, addressHex, mustSignRecord(privateKey, record))
	if err != nil {
		return fmt.Errorf("service read: %w", err)
	}
	if !bytes.Equal(message, record.Message) {
		return errors.New("service read message mismatch")
	}
	fmt.Fprintf(out, "[OK] service read\n")

	invalidSignature := mustSignRecord(privateKey, record)
	if len(invalidSignature) > 1 {
		invalidSignature[0] ^= 0xFF
		invalidSignature[1] ^= 0xFF
	}
	if _, err := service.ReadData(recordIDHex, addressHex, invalidSignature); err == nil {
		return errors.New("service accepted invalid signature")
	}
	fmt.Fprintf(out, "[OK] service invalid signature rejected\n")

	if err := service.DeleteData(recordIDHex, addressHex, mustSignRecord(privateKey, record)); err != nil {
		return fmt.Errorf("service delete: %w", err)
	}
	stored = false
	fmt.Fprintf(out, "[OK] service delete\n")

	if _, err := service.ReadData(recordIDHex, addressHex, mustSignRecord(privateKey, record)); err == nil {
		return errors.New("service read after delete succeeded")
	}
	fmt.Fprintf(out, "[OK] service read after delete rejected\n")
	return nil
}

func cleanupDirectRecords(out io.Writer, reader *seclient.CardReader, privateKey *ecdsa.PrivateKey, records []smokeRecord) error {
	for i := range records {
		if !records[i].Active {
			continue
		}
		_, remaining, err := reader.DeleteData(records[i].RecordID, records[i].Address, mustSignRecord(privateKey, records[i]))
		if err != nil {
			return fmt.Errorf("cleanup record %d: %w", i, err)
		}
		records[i].Active = false
		fmt.Fprintf(out, "[OK] cleanup #%d remaining=%d\n", i, remaining)
	}
	return nil
}

func generateRecords(count int) ([]smokeRecord, error) {
	records := make([]smokeRecord, count)
	for i := range records {
		recordID, err := randomBytes(seclient.RECORD_ID_LENGTH)
		if err != nil {
			return nil, err
		}
		address, err := randomBytes(seclient.ADDR_LENGTH)
		if err != nil {
			return nil, err
		}
		records[i] = smokeRecord{
			RecordID: recordID,
			Address:  address,
			Message:  padASCII(fmt.Sprintf("smoke-record-%02d", i), seclient.MESSAGE_LENGTH),
		}
	}
	return records, nil
}

func loadPrivateKey(filename string) (*ecdsa.PrivateKey, error) {
	pemData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("read private key: %w", err)
	}

	block, _ := pem.Decode(pemData)
	if block == nil {
		return nil, errors.New("private key is not PEM")
	}

	if key, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		ecdsaKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not ECDSA")
		}
		return ecdsaKey, nil
	}

	ecdsaKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse ECDSA private key: %w", err)
	}
	return ecdsaKey, nil
}

func resolvePrivateKeyPath(path string) (string, error) {
	if path != "" {
		if _, err := os.Stat(path); err != nil {
			return "", fmt.Errorf("private key %q is not readable: %w", path, err)
		}
		return path, nil
	}

	candidates := []string{
		"../secured/genkey/ec_private_key.pem",
		"offline-client/secured/genkey/ec_private_key.pem",
		"../../../secured/genkey/ec_private_key.pem",
		"../../genkey/ec_private_key.pem",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Clean(candidate), nil
		}
	}

	return "", errors.New("private key not found; pass -private-key or run from offline-client/offline-client-wails")
}

func signRecord(privateKey *ecdsa.PrivateKey, recordID, address []byte) ([]byte, error) {
	data := make([]byte, 0, len(recordID)+len(address))
	data = append(data, recordID...)
	data = append(data, address...)
	hash := sha256.Sum256(data)

	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("sign record: %w", err)
	}
	return signature, nil
}

func mustSignRecord(privateKey *ecdsa.PrivateKey, record smokeRecord) []byte {
	signature, err := signRecord(privateKey, record.RecordID, record.Address)
	if err != nil {
		panic(err)
	}
	return signature
}

func parseHexBytes(value string, expectedLen int, name string) ([]byte, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "0x")
	data, err := hex.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("invalid %s hex: %w", name, err)
	}
	if expectedLen >= 0 && len(data) != expectedLen {
		return nil, fmt.Errorf("%s must be %d bytes", name, expectedLen)
	}
	return data, nil
}

func randomBytes(length int) ([]byte, error) {
	data := make([]byte, length)
	if _, err := rand.Read(data); err != nil {
		return nil, fmt.Errorf("generate random bytes: %w", err)
	}
	return data, nil
}

func mustRandomBytes(length int) []byte {
	data, err := randomBytes(length)
	if err != nil {
		panic(err)
	}
	return data
}

func padASCII(value string, length int) []byte {
	data := []byte(value)
	result := make([]byte, length)
	copy(result, data)
	return result
}

func firstActiveRecord(records []smokeRecord) int {
	for i := range records {
		if records[i].Active {
			return i
		}
	}
	return -1
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
