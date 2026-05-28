package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"offline-client-wails/mpc_core/clog"
	"offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/seclient"
	"offline-client-wails/mpc_core/services"
)

type partyState struct {
	Index          int
	RecordID       string
	Address        string
	PublicKey      string
	EncryptedShard []byte
}

type managerProcess struct {
	url     string
	room    string
	cmd     *exec.Cmd
	logFile *os.File
}

func main() {
	var managerBin string
	var privateKeyPath string
	var readerName string
	var appletAID string
	var tempDir string
	var keepRecords bool
	var debug bool

	flag.StringVar(&managerBin, "manager-bin", "", "gg20 manager binary path")
	flag.StringVar(&privateKeyPath, "private-key", "", "server ECDSA private key PEM path")
	flag.StringVar(&readerName, "reader", "", "reader name substring; empty uses the first available reader")
	flag.StringVar(&appletAID, "aid", "A000000062CF0101", "SE applet AID hex")
	flag.StringVar(&tempDir, "temp-dir", "", "temporary directory for local share files")
	flag.BoolVar(&keepRecords, "keep-records", false, "keep smoke records in SE for debugging")
	flag.BoolVar(&debug, "debug", false, "enable production mpc_core debug logs")
	flag.Parse()

	if debug {
		_ = clog.Init(clog.Config{
			Level:         clog.DebugLevel,
			Format:        clog.FormatConsole,
			Filename:      "logs/mpc-e2e-smoke.log",
			Name:          "default",
			ConsoleOutput: true,
			EnableCaller:  false,
			EnableColor:   false,
		})
		defer clog.Sync()
	}

	if err := run(managerBin, privateKeyPath, readerName, appletAID, tempDir, keepRecords); err != nil {
		fmt.Fprintf(os.Stderr, "\n[FAIL] %v\n", err)
		os.Exit(1)
	}
}

func run(managerBin, privateKeyPath, readerName, appletAID, tempDir string, keepRecords bool) error {
	managerBin, err := resolvePath(managerBin, []string{
		filepath.Join("../../offline-server/bin", defaultManagerBinaryName()),
		filepath.Join("offline-server/bin", defaultManagerBinaryName()),
		filepath.Join("../offline-server/bin", defaultManagerBinaryName()),
		"../../offline-server/bin/gg20_sm_manager",
		"offline-server/bin/gg20_sm_manager",
		"../offline-server/bin/gg20_sm_manager",
	})
	if err != nil {
		return fmt.Errorf("resolve manager binary: %w", err)
	}
	privateKeyPath, err = resolvePath(privateKeyPath, []string{
		"../../offline-server/private_keys/ec_private_key.pem",
		"../secured/genkey/ec_private_key.pem",
		"offline-server/private_keys/ec_private_key.pem",
		"offline-client/secured/genkey/ec_private_key.pem",
	})
	if err != nil {
		return fmt.Errorf("resolve private key: %w", err)
	}
	privateKey, err := loadPrivateKey(privateKeyPath)
	if err != nil {
		return err
	}
	if tempDir == "" {
		tempDir, err = os.MkdirTemp("", "mpc-e2e-smoke-")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
		defer os.RemoveAll(tempDir)
	}

	fmt.Println("MPC 2-of-3 E2E smoke")
	fmt.Printf("Manager: %s\n", managerBin)
	fmt.Printf("Private key: %s\n", privateKeyPath)
	fmt.Printf("Reader: %s\n", valueOrDefault(readerName, "<first available reader>"))
	fmt.Printf("Temp dir: %s\n", tempDir)

	cplc, err := probeSE(readerName, appletAID, tempDir)
	if err != nil {
		return err
	}
	fmt.Printf("[OK] SE reachable, CPLC=%s\n", strings.ToUpper(hex.EncodeToString(cplc)))

	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	parties := []partyState{
		{Index: 1, RecordID: deriveRecordID("mpc-smoke-"+nonce, 1)},
		{Index: 2, RecordID: deriveRecordID("mpc-smoke-"+nonce, 2)},
		{Index: 3, RecordID: deriveRecordID("mpc-smoke-"+nonce, 3)},
	}

	cleanup := true
	defer func() {
		if cleanup && !keepRecords {
			cleanupRecords(parties, privateKey, readerName, appletAID, tempDir)
		}
	}()

	if err := runKeygen(managerBin, readerName, appletAID, tempDir, parties); err != nil {
		return err
	}
	fmt.Println("[OK] keygen completed for 3 parties")

	if err := ensureSameAddress(parties); err != nil {
		return err
	}
	fmt.Printf("[OK] shared address=%s\n", parties[0].Address)

	combos := [][]int{{1, 2}, {1, 3}, {2, 3}}
	for i, combo := range combos {
		messageHash := fmt.Sprintf("%064x", i+1)
		signature, err := runSignCombo(managerBin, readerName, appletAID, tempDir, parties, combo, messageHash, privateKey)
		if err != nil {
			return err
		}
		fmt.Printf("[OK] sign combo %s signature=%s\n", joinInts(combo), shortHex(signature))
	}

	if !keepRecords {
		if err := cleanupRecords(parties, privateKey, readerName, appletAID, tempDir); err != nil {
			return err
		}
		cleanup = false
		fmt.Println("[OK] SE smoke records cleaned")
	}

	fmt.Println("\n[OK] MPC 2-of-3 E2E smoke completed")
	return nil
}

func runKeygen(managerBin, readerName, appletAID, tempRoot string, parties []partyState) error {
	manager, err := startManager(managerBin, "mpc-smoke-keygen-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	if err != nil {
		return err
	}
	defer manager.stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	errs := make(chan error, len(parties))
	var wg sync.WaitGroup
	for i := range parties {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := smokeConfig(readerName, appletAID, filepath.Join(tempRoot, fmt.Sprintf("keygen-party-%d", parties[i].Index)))
			securityService, err := services.NewSecurityService(cfg)
			if err != nil {
				errs <- err
				return
			}
			mpcService := services.NewMPCService(cfg, securityService)
			filename := fmt.Sprintf("keygen_%d.json", parties[i].Index)
			address, publicKey, encryptedShard, err := mpcService.KeyGeneration(
				ctx,
				manager.url,
				manager.room,
				1,
				3,
				parties[i].Index,
				filename,
				parties[i].RecordID,
			)
			if err != nil {
				errs <- fmt.Errorf("party %d keygen: %w", parties[i].Index, err)
				return
			}
			parties[i].Address = address
			parties[i].PublicKey = publicKey
			parties[i].EncryptedShard = encryptedShard
			errs <- nil
		}()
	}
	wg.Wait()
	close(errs)
	return firstErr(errs)
}

func runSignCombo(managerBin, readerName, appletAID, tempRoot string, parties []partyState, combo []int, messageHash string, privateKey *ecdsa.PrivateKey) (string, error) {
	manager, err := startManager(managerBin, "mpc-smoke-sign-"+joinInts(combo)+"-"+strconv.FormatInt(time.Now().UnixNano(), 10))
	if err != nil {
		return "", err
	}
	defer manager.stop()

	partyByIndex := make(map[int]partyState, len(parties))
	for _, party := range parties {
		partyByIndex[party.Index] = party
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	partiesArg := joinInts(combo)
	results := make(chan string, len(combo))
	errs := make(chan error, len(combo))
	var wg sync.WaitGroup
	for signingIndex, partyIndex := range combo {
		signingIndex := signingIndex + 1
		party := partyByIndex[partyIndex]
		wg.Add(1)
		go func() {
			defer wg.Done()
			cfg := smokeConfig(readerName, appletAID, filepath.Join(tempRoot, fmt.Sprintf("sign-%s-party-%d", joinInts(combo), party.Index)))
			securityService, err := services.NewSecurityService(cfg)
			if err != nil {
				errs <- err
				return
			}
			mpcService := services.NewMPCService(cfg, securityService)
			authSignature, err := signSEAuthorization(privateKey, party.RecordID, party.Address)
			if err != nil {
				errs <- err
				return
			}
			result, err := mpcService.SignMessage(
				ctx,
				manager.url,
				manager.room,
				signingIndex,
				partiesArg,
				messageHash,
				fmt.Sprintf("sign_%s_%d.json", joinInts(combo), signingIndex),
				party.RecordID,
				party.Address,
				party.EncryptedShard,
				authSignature,
			)
			if err != nil {
				errs <- fmt.Errorf("party %d sign combo %s: %w", party.Index, partiesArg, err)
				return
			}
			results <- result
			errs <- nil
		}()
	}
	wg.Wait()
	close(errs)
	close(results)
	if err := firstErr(errs); err != nil {
		return "", err
	}

	var signature string
	for result := range results {
		if signature == "" {
			signature = result
			continue
		}
		if result != signature {
			return "", fmt.Errorf("signature mismatch for combo %s", partiesArg)
		}
	}
	if signature == "" {
		return "", fmt.Errorf("empty signature for combo %s", partiesArg)
	}
	return signature, nil
}

func startManager(managerBin, room string) (*managerProcess, error) {
	port, err := freePort()
	if err != nil {
		return nil, err
	}
	logPath := filepath.Join(os.TempDir(), room+".manager.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, fmt.Errorf("create manager log: %w", err)
	}
	cmd := exec.Command(managerBin, "--address", "127.0.0.1", "--port", strconv.Itoa(port))
	cmd.Env = append(os.Environ(),
		"ROCKET_ADDRESS=127.0.0.1",
		"ROCKET_PORT="+strconv.Itoa(port),
	)
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, fmt.Errorf("start manager: %w", err)
	}
	process := &managerProcess{
		url:     "http://127.0.0.1:" + strconv.Itoa(port),
		room:    room,
		cmd:     cmd,
		logFile: logFile,
	}
	if err := waitForPort(port, 5*time.Second); err != nil {
		process.stop()
		return nil, fmt.Errorf("manager did not become ready; log=%s: %w", logPath, err)
	}
	return process, nil
}

func (p *managerProcess) stop() {
	if p == nil || p.cmd == nil || p.cmd.Process == nil {
		return
	}
	defer func() {
		if p.logFile != nil {
			_ = p.logFile.Close()
		}
	}()
	_ = p.cmd.Process.Signal(os.Interrupt)
	done := make(chan struct{})
	go func() {
		_ = p.cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		_ = p.cmd.Process.Kill()
	}
}

func probeSE(readerName, appletAID, tempRoot string) ([]byte, error) {
	cfg := smokeConfig(readerName, appletAID, filepath.Join(tempRoot, "probe"))
	securityService, err := services.NewSecurityService(cfg)
	if err != nil {
		return nil, err
	}
	return securityService.GetCPLC()
}

func cleanupRecords(parties []partyState, privateKey *ecdsa.PrivateKey, readerName, appletAID, tempRoot string) error {
	cfg := smokeConfig(readerName, appletAID, filepath.Join(tempRoot, "cleanup"))
	securityService, err := services.NewSecurityService(cfg)
	if err != nil {
		return err
	}
	var errs []string
	for _, party := range parties {
		if party.RecordID == "" || party.Address == "" {
			continue
		}
		signature, err := signSEAuthorization(privateKey, party.RecordID, party.Address)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if err := securityService.DeleteData(party.RecordID, party.Address, signature); err != nil {
			errs = append(errs, fmt.Sprintf("party %d: %v", party.Index, err))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("cleanup failed: %s", strings.Join(errs, "; "))
	}
	return nil
}

func smokeConfig(readerName, appletAID, tempDir string) *config.Config {
	return &config.Config{
		Debug:          false,
		CardReaderName: readerName,
		AppletAID:      appletAID,
		TempDir:        tempDir,
		ManagerAddr:    "http://127.0.0.1:8000",
	}
}

func signSEAuthorization(privateKey *ecdsa.PrivateKey, recordIDHex, addressHex string) ([]byte, error) {
	recordID, err := decodeFixedHex(recordIDHex, seclient.RECORD_ID_LENGTH, "record_id")
	if err != nil {
		return nil, err
	}
	address, err := decodeFixedHex(addressHex, seclient.ADDR_LENGTH, "address")
	if err != nil {
		return nil, err
	}
	data := make([]byte, 0, len(recordID)+len(address))
	data = append(data, recordID...)
	data = append(data, address...)
	digest := sha256.Sum256(data)
	return ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
}

func decodeFixedHex(value string, length int, label string) ([]byte, error) {
	data, err := hex.DecodeString(strings.TrimPrefix(value, "0x"))
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
		return nil, errors.New("private key is not PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err == nil {
		ecdsaKey, ok := key.(*ecdsa.PrivateKey)
		if !ok {
			return nil, errors.New("private key is not ECDSA")
		}
		return ecdsaKey, nil
	}
	ecdsaKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse private key: %w", err)
	}
	return ecdsaKey, nil
}

func deriveRecordID(seed string, partyIndex int) string {
	digest := sha256.Sum256([]byte(fmt.Sprintf("%s|party|%d", seed, partyIndex)))
	return hex.EncodeToString(digest[:])
}

func ensureSameAddress(parties []partyState) error {
	if len(parties) == 0 {
		return errors.New("no parties")
	}
	address := parties[0].Address
	publicKey := parties[0].PublicKey
	for _, party := range parties {
		if party.Address == "" || party.Address != address {
			return fmt.Errorf("address mismatch: party %d address=%s expected=%s", party.Index, party.Address, address)
		}
		if party.PublicKey == "" || party.PublicKey != publicKey {
			return fmt.Errorf("public key mismatch: party %d", party.Index)
		}
		if len(party.EncryptedShard) == 0 {
			return fmt.Errorf("party %d encrypted shard is empty", party.Index)
		}
	}
	return nil
}

func firstErr(errs <-chan error) error {
	for err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

func joinInts(values []int) string {
	sorted := append([]int(nil), values...)
	sort.Ints(sorted)
	parts := make([]string, len(sorted))
	for i, value := range sorted {
		parts[i] = strconv.Itoa(value)
	}
	return strings.Join(parts, ",")
}

func shortHex(value string) string {
	if len(value) <= 18 {
		return value
	}
	return value[:18] + "..."
}

func valueOrDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func resolvePath(value string, candidates []string) (string, error) {
	if value != "" {
		if _, err := os.Stat(value); err != nil {
			return "", err
		}
		return value, nil
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("not found in candidates: %s", strings.Join(candidates, ", "))
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("allocate port: %w", err)
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func waitForPort(port int, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	address := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 100*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for %s", address)
}

func defaultManagerBinaryName() string {
	name := fmt.Sprintf("gg20_sm_manager_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}
