package main

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	clientcfg "offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/services"
	"offline-server/manager"
	"offline-server/storage/model"
	"offline-server/tools"
	serverws "offline-server/ws"
	memstorage "offline-server/ws/storage"

	"github.com/gorilla/websocket"
)

const (
	defaultAppletAID = "A000000062CF0101"
	offlineKeyID     = "offline-e2e-smoke-key"
)

type smokeOptions struct {
	readerName     string
	appletAID      string
	managerBin     string
	tempDir        string
	keepTemp       bool
	debug          bool
	repoRoot       string
	serverWorkDir  string
	serverKeyPath  string
	managerLogDir  string
	clientTempRoot string
}

type desktopClient struct {
	username string
	role     serverws.ClientRole
	conn     *websocket.Conn
	security *services.SecurityService
	mpc      *services.MPCService
	cplc     string
}

type keyShare struct {
	Username       string
	PartyIndex     int
	Address        string
	PublicKey      string
	CPLC           string
	RecordID       string
	EncryptedShard string
}

func main() {
	var opts smokeOptions
	flag.StringVar(&opts.readerName, "reader", "", "SE reader name substring; empty uses the first available reader")
	flag.StringVar(&opts.appletAID, "aid", defaultAppletAID, "SE applet AID hex")
	flag.StringVar(&opts.managerBin, "manager-bin", "", "gg20 manager binary path")
	flag.StringVar(&opts.tempDir, "temp-dir", "", "temporary working directory")
	flag.BoolVar(&opts.keepTemp, "keep-temp", false, "keep temporary files for debugging")
	flag.BoolVar(&opts.debug, "debug", false, "enable SE debug logs in desktop clients")
	flag.Parse()

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "\n[FAIL] %v\n", err)
		os.Exit(1)
	}
}

func run(opts smokeOptions) error {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	opts.repoRoot = repoRoot

	managerBin, err := resolveManagerBin(repoRoot, opts.managerBin)
	if err != nil {
		return err
	}
	opts.managerBin = managerBin

	if opts.tempDir == "" {
		opts.tempDir, err = os.MkdirTemp("", "offline-e2e-smoke-")
		if err != nil {
			return fmt.Errorf("create temp dir: %w", err)
		}
	} else if err := os.MkdirAll(opts.tempDir, 0755); err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	opts.tempDir, err = filepath.Abs(opts.tempDir)
	if err != nil {
		return err
	}
	if !opts.keepTemp {
		defer os.RemoveAll(opts.tempDir)
	}

	opts.serverWorkDir = filepath.Join(opts.tempDir, "server")
	opts.managerLogDir = filepath.Join(opts.tempDir, "manager-logs")
	opts.clientTempRoot = filepath.Join(opts.tempDir, "clients")
	if err := os.MkdirAll(opts.serverWorkDir, 0755); err != nil {
		return err
	}
	if err := prepareServerPrivateKey(repoRoot, opts.serverWorkDir); err != nil {
		return err
	}
	opts.serverKeyPath = filepath.Join(opts.serverWorkDir, "private_keys", "ec_private_key.pem")

	oldWD, err := os.Getwd()
	if err != nil {
		return err
	}
	if err := os.Chdir(opts.serverWorkDir); err != nil {
		return fmt.Errorf("switch to server work dir: %w", err)
	}
	defer func() { _ = os.Chdir(oldWD) }()

	fmt.Println("Offline WebSocket + MPC + SE E2E smoke")
	fmt.Printf("Repo: %s\n", opts.repoRoot)
	fmt.Printf("Manager: %s\n", opts.managerBin)
	fmt.Printf("Server work dir: %s\n", opts.serverWorkDir)
	fmt.Printf("Reader: %s\n", valueOrDefault(opts.readerName, "<first available reader>"))

	probeCfg := clientConfig(opts, "probe")
	probeSecurity, err := services.NewSecurityService(probeCfg)
	if err != nil {
		return err
	}
	cplcBytes, err := probeSecurity.GetCPLC()
	if err != nil {
		return fmt.Errorf("probe SE: %w", err)
	}
	cplc := strings.ToUpper(hex.EncodeToString(cplcBytes))
	fmt.Printf("[OK] SE reachable, CPLC=%s\n", cplc)

	server, wsURL, shareStore, err := startServer(opts, cplc)
	if err != nil {
		return err
	}
	defer server.Stop()
	fmt.Printf("[OK] offline WebSocket server started, url=%s\n", wsURL)

	coordinator, err := dialClient(wsURL, "admin", serverws.RoleAdmin, nil)
	if err != nil {
		return err
	}
	defer coordinator.close()

	participants := []string{"u1", "u2", "u3"}
	clients := make(map[string]*desktopClient, len(participants))
	for _, username := range participants {
		client, err := newDesktopParticipant(opts, wsURL, username, cplc)
		if err != nil {
			return err
		}
		defer client.close()
		clients[username] = client
	}
	fmt.Println("[OK] coordinator and 3 desktop clients connected")

	shares, address, err := runKeygen(coordinator, clients, participants)
	if err != nil {
		return err
	}
	fmt.Printf("[OK] keygen completed, address=%s\n", address)

	cleanupSecurity := clients["u1"].security
	recordsCleaned := false
	defer func() {
		if !recordsCleaned {
			if err := cleanupSERecords(cleanupSecurity, shares); err != nil {
				fmt.Fprintf(os.Stderr, "[WARN] cleanup SE records: %v\n", err)
			}
		}
	}()

	markRecordsCleaned := func() error {
		if recordsCleaned {
			return nil
		}
		if err := cleanupSERecords(cleanupSecurity, shares); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] cleanup SE records: %v\n", err)
			return err
		}
		recordsCleaned = true
		return nil
	}

	combos := [][]string{
		{"u1", "u2"},
		{"u1", "u3"},
		{"u2", "u3"},
	}
	for i, combo := range combos {
		signature, err := runSignCombo(coordinator, clients, address, combo, i+1)
		if err != nil {
			return err
		}
		fmt.Printf("[OK] sign combo %s signature=%s\n", strings.Join(combo, ","), shortHex(signature))
	}

	storedShards, err := shareStore.ListActiveKeyShardsByAddress(address)
	if err != nil {
		return err
	}
	if len(storedShards) != 3 {
		return fmt.Errorf("server stored shard count=%d, want 3", len(storedShards))
	}
	fmt.Println("[OK] server storage has 3 active key shards")

	if err := markRecordsCleaned(); err != nil {
		return err
	}
	fmt.Println("[OK] SE smoke records cleaned")
	fmt.Println("\n[OK] offline WebSocket + MPC + SE E2E smoke completed")
	return nil
}

func startServer(opts smokeOptions, cplc string) (*serverws.Server, string, *memoryShareStorage, error) {
	shareStore := newMemoryShareStorage()
	seStore := newMemorySeStorage()
	for i := 1; i <= 3; i++ {
		seStore.add(model.Se{
			SeID:   fmt.Sprintf("SE%02d", i),
			CPLC:   cplc,
			Status: model.SeStatusActive,
		})
	}

	runtime := manager.NewSessionRuntime(manager.SessionRuntimeConfig{
		BinaryPath:      opts.managerBin,
		BindAddress:     "127.0.0.1",
		PublicHost:      "127.0.0.1",
		LogDir:          opts.managerLogDir,
		GracefulTimeout: 5 * time.Second,
		Environment:     os.Environ(),
	})
	handler := serverws.NewMessageHandlerWithDependencies(
		shareStore,
		seStore,
		newMemoryOfflineKeyStorage(),
		newMemoryKeyGenStorage(),
		newMemorySignStorage(),
		memoryAuditStorage{},
		memoryApprovalStorage{},
		memstorage.NewSessionManager(),
		runtime,
	)

	addr, err := freeLocalAddr()
	if err != nil {
		return nil, "", nil, err
	}
	server := serverws.NewServerWithHandler(addr, serverws.ServerConfig{
		PingInterval:     200 * time.Millisecond,
		ReadTimeout:      10 * time.Second,
		WriteTimeout:     10 * time.Second,
		MessageSizeLimit: 1024 * 1024,
	}, handler)
	if err := server.Start(); err != nil {
		return nil, "", nil, err
	}
	return server, "ws://" + addr + "/ws", shareStore, nil
}

func newDesktopParticipant(opts smokeOptions, wsURL, username, cplc string) (*desktopClient, error) {
	cfg := clientConfig(opts, username)
	securityService, err := services.NewSecurityService(cfg)
	if err != nil {
		return nil, err
	}
	mpcService := services.NewMPCService(cfg, securityService)
	client, err := dialClient(wsURL, username, serverws.RoleOfficer, func(c *desktopClient) {
		c.security = securityService
		c.mpc = mpcService
		c.cplc = cplc
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}

func dialClient(wsURL, username string, role serverws.ClientRole, configure func(*desktopClient)) (*desktopClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", username, err)
	}
	client := &desktopClient{username: username, role: role, conn: conn}
	if configure != nil {
		configure(client)
	}

	token, err := tools.GenerateToken(username, string(role), time.Hour)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := client.send(serverws.RegisterMessage{
		BaseMessage: serverws.BaseMessage{Type: serverws.MsgRegister},
		Username:    username,
		Role:        role,
		Token:       token,
	}); err != nil {
		_ = conn.Close()
		return nil, err
	}
	ack, err := readJSON[serverws.RegisterCompleteMessage](client, 5*time.Second)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if ack.Type != serverws.MsgRegisterComplete || !ack.Success {
		_ = conn.Close()
		return nil, fmt.Errorf("register %s failed: %+v", username, ack)
	}
	return client, nil
}

func runKeygen(coordinator *desktopClient, clients map[string]*desktopClient, participants []string) ([]keyShare, string, error) {
	sessionKey := "offline-e2e-keygen-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := coordinator.send(serverws.KeyGenRequestMessage{
		BaseMessage:     serverws.BaseMessage{Type: serverws.MsgKeyGenRequest},
		SessionKey:      sessionKey,
		TaskNo:          "task-" + sessionKey,
		OfflineKeyID:    offlineKeyID,
		RequiredSigners: 2,
		TotalParties:    3,
		Participants:    participants,
	}); err != nil {
		return nil, "", err
	}

	invites := make(map[string]serverws.KeyGenInviteMessage, len(participants))
	for _, username := range participants {
		invite, err := readJSON[serverws.KeyGenInviteMessage](clients[username], 5*time.Second)
		if err != nil {
			return nil, "", err
		}
		if invite.Type != serverws.MsgKeyGenInvite {
			return nil, "", fmt.Errorf("unexpected keygen invite for %s: %+v", username, invite)
		}
		invites[username] = invite
	}

	for _, username := range participants {
		invite := invites[username]
		if err := clients[username].send(serverws.KeyGenResponseMessage{
			BaseMessage: serverws.BaseMessage{Type: serverws.MsgKeyGenResponse},
			SessionKey:  invite.SessionKey,
			PartyIndex:  invite.PartyIndex,
			CPLC:        clients[username].cplc,
			Accept:      true,
		}); err != nil {
			return nil, "", err
		}
	}

	paramsByUser := make(map[string]serverws.KeyGenParamsMessage, len(participants))
	for _, username := range participants {
		params, err := readJSON[serverws.KeyGenParamsMessage](clients[username], 5*time.Second)
		if err != nil {
			return nil, "", err
		}
		if params.Type != serverws.MsgKeyGenParams || params.ManagerAddr == "" || params.Room == "" {
			return nil, "", fmt.Errorf("bad keygen params for %s: %+v", username, params)
		}
		paramsByUser[username] = params
	}

	results, err := performKeygenForAll(clients, paramsByUser, participants)
	if err != nil {
		return nil, "", err
	}
	for _, result := range results {
		if err := clients[result.Username].send(serverws.KeyGenResultMessage{
			BaseMessage:    serverws.BaseMessage{Type: serverws.MsgKeyGenResult},
			SessionKey:     sessionKey,
			PartyIndex:     result.PartyIndex,
			Address:        result.Address,
			PublicKey:      result.PublicKey,
			CPLC:           result.CPLC,
			RecordID:       result.RecordID,
			EncryptedShard: result.EncryptedShard,
			Success:        true,
			Message:        "ok",
		}); err != nil {
			return nil, "", err
		}
	}

	complete, err := readJSON[serverws.KeyGenCompleteMessage](coordinator, 5*time.Second)
	if err != nil {
		return nil, "", err
	}
	if complete.Type != serverws.MsgKeyGenComplete || !complete.Success || complete.Address == "" {
		return nil, "", fmt.Errorf("bad keygen completion: %+v", complete)
	}
	return results, complete.Address, nil
}

func performKeygenForAll(clients map[string]*desktopClient, paramsByUser map[string]serverws.KeyGenParamsMessage, participants []string) ([]keyShare, error) {
	type outcome struct {
		share keyShare
		err   error
	}
	outcomes := make(chan outcome, len(participants))
	var wg sync.WaitGroup
	for _, username := range participants {
		username := username
		params := paramsByUser[username]
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			address, publicKey, encryptedShard, err := clients[username].mpc.KeyGeneration(
				ctx,
				params.ManagerAddr,
				params.Room,
				params.Threshold,
				params.TotalParties,
				params.PartyIndex,
				params.FileName,
				params.RecordID,
			)
			if err != nil {
				outcomes <- outcome{err: fmt.Errorf("%s keygen: %w", username, err)}
				return
			}
			outcomes <- outcome{share: keyShare{
				Username:       username,
				PartyIndex:     params.PartyIndex,
				Address:        address,
				PublicKey:      publicKey,
				CPLC:           clients[username].cplc,
				RecordID:       params.RecordID,
				EncryptedShard: base64.StdEncoding.EncodeToString(encryptedShard),
			}}
		}()
	}
	wg.Wait()
	close(outcomes)

	shares := make([]keyShare, 0, len(participants))
	for result := range outcomes {
		if result.err != nil {
			return nil, result.err
		}
		shares = append(shares, result.share)
	}
	sort.Slice(shares, func(i, j int) bool { return shares[i].PartyIndex < shares[j].PartyIndex })
	if err := ensureSameKey(shares); err != nil {
		return nil, err
	}
	return shares, nil
}

func runSignCombo(coordinator *desktopClient, clients map[string]*desktopClient, address string, participants []string, sequence int) (string, error) {
	sessionKey := fmt.Sprintf("offline-e2e-sign-%s-%d", strings.Join(participants, "-"), time.Now().UnixNano())
	messageHash := fmt.Sprintf("%064x", sequence)
	if err := coordinator.send(serverws.SignRequestMessage{
		BaseMessage:   serverws.BaseMessage{Type: serverws.MsgSignRequest},
		SessionKey:    sessionKey,
		TaskNo:        "task-" + sessionKey,
		OfflineKeyID:  offlineKeyID,
		TransactionNo: "tx-" + sessionKey,
		MessageHash:   messageHash,
		Address:       address,
		Participants:  participants,
	}); err != nil {
		return "", err
	}

	invites := make(map[string]serverws.SignInviteMessage, len(participants))
	for _, username := range participants {
		invite, err := readJSON[serverws.SignInviteMessage](clients[username], 5*time.Second)
		if err != nil {
			return "", err
		}
		if invite.Type != serverws.MsgSignInvite {
			return "", fmt.Errorf("unexpected sign invite for %s: %+v", username, invite)
		}
		invites[username] = invite
	}

	for _, username := range participants {
		invite := invites[username]
		if err := clients[username].send(serverws.SignResponseMessage{
			BaseMessage: serverws.BaseMessage{Type: serverws.MsgSignResponse},
			SessionKey:  invite.SessionKey,
			PartyIndex:  invite.PartyIndex,
			CPLC:        clients[username].cplc,
			Accept:      true,
		}); err != nil {
			return "", err
		}
	}

	paramsByUser := make(map[string]serverws.SignParamsMessage, len(participants))
	for _, username := range participants {
		params, err := readJSON[serverws.SignParamsMessage](clients[username], 5*time.Second)
		if err != nil {
			return "", err
		}
		if params.Type != serverws.MsgSignParams || params.ManagerAddr == "" || params.Room == "" ||
			params.Parties == "" || params.Signature == "" || params.EncryptedShard == "" {
			return "", fmt.Errorf("bad sign params for %s: %+v", username, params)
		}
		paramsByUser[username] = params
	}

	results, err := performSignForAll(clients, paramsByUser, participants)
	if err != nil {
		return "", err
	}
	for _, result := range results {
		if err := clients[result.username].send(serverws.SignResultMessage{
			BaseMessage:  serverws.BaseMessage{Type: serverws.MsgSignResult},
			SessionKey:   sessionKey,
			SigningIndex: result.signingIndex,
			Success:      true,
			Signature:    result.signature,
			Message:      "ok",
		}); err != nil {
			return "", err
		}
	}

	complete, err := readJSON[serverws.SignCompleteMessage](coordinator, 5*time.Second)
	if err != nil {
		return "", err
	}
	if complete.Type != serverws.MsgSignComplete || !complete.Success || complete.Signature == "" {
		return "", fmt.Errorf("bad sign completion: %+v", complete)
	}
	return complete.Signature, nil
}

type signOutcome struct {
	username     string
	signingIndex int
	signature    string
	err          error
}

func performSignForAll(clients map[string]*desktopClient, paramsByUser map[string]serverws.SignParamsMessage, participants []string) ([]signOutcome, error) {
	outcomes := make(chan signOutcome, len(participants))
	var wg sync.WaitGroup
	for _, username := range participants {
		username := username
		params := paramsByUser[username]
		wg.Add(1)
		go func() {
			defer wg.Done()
			encryptedShard, err := base64.StdEncoding.DecodeString(params.EncryptedShard)
			if err != nil {
				outcomes <- signOutcome{err: fmt.Errorf("%s encrypted shard: %w", username, err)}
				return
			}
			seSignature, err := base64.StdEncoding.DecodeString(params.Signature)
			if err != nil {
				outcomes <- signOutcome{err: fmt.Errorf("%s SE signature: %w", username, err)}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
			defer cancel()
			signature, err := clients[username].mpc.SignMessage(
				ctx,
				params.ManagerAddr,
				params.Room,
				params.SigningIndex,
				params.Parties,
				params.MessageHash,
				params.FileName,
				params.RecordID,
				params.Address,
				encryptedShard,
				seSignature,
			)
			if err != nil {
				outcomes <- signOutcome{err: fmt.Errorf("%s signing: %w", username, err)}
				return
			}
			outcomes <- signOutcome{username: username, signingIndex: params.SigningIndex, signature: signature}
		}()
	}
	wg.Wait()
	close(outcomes)

	results := make([]signOutcome, 0, len(participants))
	var signature string
	for result := range outcomes {
		if result.err != nil {
			return nil, result.err
		}
		if signature == "" {
			signature = result.signature
		} else if signature != result.signature {
			return nil, fmt.Errorf("signature mismatch: %s != %s", signature, result.signature)
		}
		results = append(results, result)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].signingIndex < results[j].signingIndex })
	return results, nil
}

func cleanupSERecords(securityService *services.SecurityService, shares []keyShare) error {
	var errs []string
	for _, share := range shares {
		signatureB64, err := serverws.SignData(share.RecordID, share.Address)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		signature, err := base64.StdEncoding.DecodeString(signatureB64)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if err := securityService.DeleteData(share.RecordID, share.Address, signature); err != nil {
			errs = append(errs, fmt.Sprintf("party %d: %v", share.PartyIndex, err))
		}
	}
	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

func clientConfig(opts smokeOptions, name string) *clientcfg.Config {
	return &clientcfg.Config{
		Debug:          opts.debug,
		CardReaderName: opts.readerName,
		AppletAID:      opts.appletAID,
		TempDir:        filepath.Join(opts.clientTempRoot, name),
		ManagerAddr:    "http://127.0.0.1:8000",
	}
}

func (c *desktopClient) send(msg any) error {
	return c.conn.WriteJSON(msg)
}

func (c *desktopClient) close() {
	if c != nil && c.conn != nil {
		_ = c.conn.Close()
	}
}

func readJSON[T any](client *desktopClient, timeout time.Duration) (T, error) {
	var zero T
	if err := client.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return zero, err
	}
	_, raw, err := client.conn.ReadMessage()
	if err != nil {
		return zero, fmt.Errorf("%s read: %w", client.username, err)
	}
	var base serverws.BaseMessage
	if err := json.Unmarshal(raw, &base); err != nil {
		return zero, fmt.Errorf("%s decode base: %w; raw=%s", client.username, err, raw)
	}
	if base.Type == serverws.MsgError {
		var errMsg serverws.ErrorMessage
		_ = json.Unmarshal(raw, &errMsg)
		return zero, fmt.Errorf("%s received server error: %s %s", client.username, errMsg.Message, errMsg.Details)
	}
	var msg T
	if err := json.Unmarshal(raw, &msg); err != nil {
		return zero, fmt.Errorf("%s decode message: %w; raw=%s", client.username, err, raw)
	}
	return msg, nil
}

func ensureSameKey(shares []keyShare) error {
	if len(shares) == 0 {
		return errors.New("empty keygen results")
	}
	address := shares[0].Address
	publicKey := shares[0].PublicKey
	for _, share := range shares {
		if share.Address != address {
			return fmt.Errorf("address mismatch for %s: %s != %s", share.Username, share.Address, address)
		}
		if share.PublicKey != publicKey {
			return fmt.Errorf("public key mismatch for %s", share.Username)
		}
		if share.RecordID == "" || share.EncryptedShard == "" {
			return fmt.Errorf("incomplete key share for %s", share.Username)
		}
	}
	return nil
}

func prepareServerPrivateKey(repoRoot, workDir string) error {
	source := filepath.Join(repoRoot, "offline-server", "private_keys", "ec_private_key.pem")
	if _, err := os.Stat(source); err != nil {
		return fmt.Errorf("server private key not found: %w", err)
	}
	targetDir := filepath.Join(workDir, "private_keys")
	if err := os.MkdirAll(targetDir, 0700); err != nil {
		return err
	}
	return copyFile(source, filepath.Join(targetDir, "ec_private_key.pem"), 0600)
}

func copyFile(source, target string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func resolveManagerBin(repoRoot, explicit string) (string, error) {
	if explicit != "" {
		return filepath.Abs(explicit)
	}
	name := defaultManagerBinaryName()
	candidates := []string{
		filepath.Join(repoRoot, "offline-server", "bin", name),
		filepath.Join(repoRoot, "offline-server", "bin", "gg20_sm_manager"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("manager binary not found, expected %s under offline-server/bin", name)
}

func defaultManagerBinaryName() string {
	name := fmt.Sprintf("gg20_sm_manager_%s_%s", runtime.GOOS, runtime.GOARCH)
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return name
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if exists(filepath.Join(wd, "offline-server")) && exists(filepath.Join(wd, "offline-client", "offline-client-wails")) {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			return "", errors.New("could not find repo root")
		}
		wd = parent
	}
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func freeLocalAddr() (string, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", err
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		return "", err
	}
	return addr, nil
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
