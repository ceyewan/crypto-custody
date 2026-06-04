package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	clientcfg "offline-client-wails/mpc_core/config"
	"offline-client-wails/mpc_core/services"
	serverws "offline-server/ws"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gorilla/websocket"
)

const defaultAppletAID = "A000000062CF0101"

type options struct {
	onlineURL       string
	offlineURL      string
	offlineWSURL    string
	ganacheRPC      string
	outDir          string
	onlineUser      string
	onlinePassword  string
	offlineAdmin    string
	offlineDBPath   string
	offlinePassword string
	officerPassword string
	participants    string
	signers         string
	readerName      string
	appletAID       string
	fundAmount      string
	txValue         string
	skipFund        bool
	skipBroadcast   bool
	cleanupSE       bool
	isolateSE       bool
	debug           bool
}

type flow struct {
	opts         options
	outDir       string
	online       *apiClient
	offlineAdmin *apiClient
	offlineUsers map[string]*apiClient
	clients      map[string]*desktopClient
	coordinator  *desktopClient
	participants []string
	signers      []string
	runTag       string
}

type wallet struct {
	Label      string     `json:"label"`
	CaseID     uint       `json:"case_id"`
	CaseNo     string     `json:"case_no"`
	AccountID  uint       `json:"account_id"`
	Address    string     `json:"address"`
	TaskNo     string     `json:"task_no"`
	OfflineKey string     `json:"offline_key"`
	Shares     []keyShare `json:"-"`
}

type apiClient struct {
	baseURL string
	token   string
	client  *http.Client
}

type desktopClient struct {
	username string
	role     serverws.ClientRole
	token    string
	conn     *websocket.Conn
	security *services.SecurityService
	mpc      *services.MPCService
	cplc     string
}

type keyShare struct {
	Username       string `json:"username"`
	PartyIndex     int    `json:"party_index"`
	Address        string `json:"address"`
	PublicKey      string `json:"public_key"`
	CPLC           string `json:"cplc"`
	RecordID       string `json:"record_id"`
	EncryptedShard string `json:"encrypted_shard"`
}

type signOutcome struct {
	Username     string `json:"username"`
	SigningIndex int    `json:"signing_index"`
	Signature    string `json:"signature"`
}

type keyShardRecord struct {
	ShardID    string `json:"shard_id"`
	Username   string `json:"username"`
	ShardIndex int    `json:"shard_index"`
	RecordID   string `json:"record_id"`
	Status     string `json:"status"`
}

type standardResponse struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
	Error   string          `json:"error"`
}

type offlineLoginResponse struct {
	Token string `json:"token"`
	User  struct {
		Username string `json:"username"`
		Role     string `json:"role"`
	} `json:"user"`
}

func main() {
	var opts options
	flag.StringVar(&opts.onlineURL, "online-url", envDefault("ONLINE_URL", "http://127.0.0.1:8088"), "online system HTTP URL")
	flag.StringVar(&opts.offlineURL, "offline-url", envDefault("OFFLINE_URL", "http://127.0.0.1:8080"), "offline system HTTP URL")
	flag.StringVar(&opts.offlineWSURL, "offline-ws", envDefault("OFFLINE_WS_URL", "ws://127.0.0.1:8081/ws"), "offline system WebSocket URL")
	flag.StringVar(&opts.offlineDBPath, "offline-db", envDefault("OFFLINE_DB_PATH", ""), "offline SQLite DB path; auto-detected when empty")
	flag.StringVar(&opts.ganacheRPC, "ganache-rpc", envDefault("GANACHE_RPC_URL", "http://127.0.0.1:8545"), "Ganache JSON-RPC URL")
	flag.StringVar(&opts.outDir, "out-dir", "", "output directory; default runs/<timestamp>")
	flag.StringVar(&opts.onlineUser, "online-user", envDefault("ONLINE_USER", "admin"), "online username")
	flag.StringVar(&opts.onlinePassword, "online-password", envDefault("ONLINE_PASSWORD", "admin123"), "online password")
	flag.StringVar(&opts.offlineAdmin, "offline-admin", envDefault("OFFLINE_ADMIN", "admin"), "offline admin username")
	flag.StringVar(&opts.offlinePassword, "offline-password", envDefault("OFFLINE_PASSWORD", "admin123"), "offline admin password")
	flag.StringVar(&opts.officerPassword, "officer-password", envDefault("OFFICER_PASSWORD", "officer123"), "offline officer password for u1/u2/u3")
	flag.StringVar(&opts.participants, "participants", envDefault("OFFLINE_PARTICIPANTS", "u1,u2,u3"), "comma separated offline participant usernames")
	flag.StringVar(&opts.signers, "signers", envDefault("OFFLINE_SIGNERS", "u1,u2"), "comma separated offline signer usernames for 2-of-3 signing")
	flag.StringVar(&opts.readerName, "reader", envDefault("SE_READER", ""), "SE reader name substring; empty uses first available reader")
	flag.StringVar(&opts.appletAID, "aid", envDefault("SE_APPLET_AID", defaultAppletAID), "SE applet AID hex")
	flag.StringVar(&opts.fundAmount, "fund-amount", envDefault("GANACHE_FUND_AMOUNT", "50"), "Ganache funding amount in ETH for the first generated wallet")
	flag.StringVar(&opts.txValue, "tx-value", envDefault("E2E_TX_VALUE", "20 ETH"), "generated-wallet transfer value")
	flag.BoolVar(&opts.skipFund, "skip-fund", false, "skip Ganache funding step")
	flag.BoolVar(&opts.skipBroadcast, "skip-broadcast", false, "skip online broadcast step")
	flag.BoolVar(&opts.cleanupSE, "cleanup-se", false, "delete SE key records after the flow completes")
	flag.BoolVar(&opts.isolateSE, "isolate-se", true, "disable other active offline SE records so the current reader is selected")
	flag.BoolVar(&opts.debug, "debug", false, "enable SE debug mode")
	flag.Parse()

	if err := run(opts); err != nil {
		fmt.Fprintf(os.Stderr, "\n[FAIL] %v\n", err)
		os.Exit(1)
	}
}

func run(opts options) error {
	participants := parseParticipants(opts.participants)
	if len(participants) != 3 {
		return fmt.Errorf("participants must contain u1,u2,u3 style 3 users, got %v", participants)
	}
	signers := parseParticipants(opts.signers)
	if len(signers) < 2 {
		return fmt.Errorf("signers must contain at least 2 users, got %v", signers)
	}
	if err := ensureSubset(signers, participants); err != nil {
		return err
	}
	outDir, err := prepareOutputDir(opts.outDir)
	if err != nil {
		return err
	}

	f := &flow{
		opts:         opts,
		outDir:       outDir,
		online:       newAPIClient(opts.onlineURL),
		offlineAdmin: newAPIClient(opts.offlineURL),
		offlineUsers: map[string]*apiClient{},
		clients:      map[string]*desktopClient{},
		participants: participants,
		signers:      signers,
		runTag:       filepath.Base(outDir),
	}

	fmt.Println("Crypto custody online/offline E2E flow")
	fmt.Printf("Output: %s\n", outDir)
	fmt.Printf("Online: %s\n", opts.onlineURL)
	fmt.Printf("Offline: %s / %s\n", opts.offlineURL, opts.offlineWSURL)
	fmt.Printf("Participants: %s\n", strings.Join(participants, ","))
	fmt.Printf("Signers: %s\n", strings.Join(signers, ","))

	if err := f.loginAll(); err != nil {
		return err
	}
	cplc, err := f.prepareSE()
	if err != nil {
		return err
	}
	if err := f.connectWSClients(cplc); err != nil {
		return err
	}
	defer f.closeWSClients()

	source, err := f.generateWallet("source", "源托管钱包", "07_source")
	if err != nil {
		return err
	}
	target, err := f.generateWallet("target", "目标托管钱包", "13_target")
	if err != nil {
		return err
	}
	targetDestroyed := false
	if opts.cleanupSE {
		defer func() {
			wallets := []wallet{source}
			if !targetDestroyed {
				wallets = append(wallets, target)
			}
			cleanupWalletShares(f.clients[participants[0]].security, wallets...)
		}()
	}

	if !opts.skipFund {
		if err := f.fundCustodyAddress("19_ganache_fund_source.json", source.Address); err != nil {
			return err
		}
	}
	if _, err := f.syncAccountBalance("20_source_balance_after_fund.json", source.AccountID); err != nil {
		return err
	}

	txID, err := f.createTransaction("21_transaction_create.json", source.CaseID, source.CaseNo, source.AccountID, source.Address, target.Address)
	if err != nil {
		return err
	}
	signTask, signTaskNo, err := f.exportSignTask("22_sign_task_online.json", txID)
	if err != nil {
		return err
	}
	signTaskNo = f.offlineTaskNo(signTaskNo, "sign")
	signTask = clonePackageWithTaskNo(signTask, signTaskNo)
	if err := f.importOfflineTask("23_sign_task_offline_import.json", signTask); err != nil {
		return err
	}
	if _, err := f.runSign("24_sign_complete_ws.json", signTaskNo); err != nil {
		return err
	}
	signResult, err := f.downloadOfflineResult("25_sign_result_offline.json", signTaskNo)
	if err != nil {
		return err
	}
	if err := f.verifySignature("26_signature_verify_offline.json", signResult); err != nil {
		return err
	}
	if err := f.importSignature("27_signature_import_online.json", txID, signResult); err != nil {
		return err
	}
	if !opts.skipBroadcast {
		if err := f.broadcast("28_broadcast_online.json", txID); err != nil {
			return err
		}
	}
	sourceBalance, err := f.syncAccountBalance("29_source_balance_final.json", source.AccountID)
	if err != nil {
		return err
	}
	targetBalance, err := f.syncAccountBalance("30_target_balance_final.json", target.AccountID)
	if err != nil {
		return err
	}
	chainBalances, err := f.chainBalances("31_chain_balances_final.json", source.Address, target.Address)
	if err != nil {
		return err
	}
	transferReport, err := f.runShardTransfer("32_shard_transfer_offline.json", source, f.participants[0], f.opts.offlineAdmin)
	if err != nil {
		return err
	}
	destroyReport, err := f.runKeyDestroy("33_key_destroy_offline.json", target)
	if err != nil {
		return err
	}
	targetDestroyed = true

	summary := map[string]any{
		"source_wallet":    source,
		"target_wallet":    target,
		"transaction_id":   txID,
		"sign_task_no":     signTaskNo,
		"source_balance":   sourceBalance,
		"target_balance":   targetBalance,
		"chain_balances":   chainBalances,
		"shard_transfer":   transferReport,
		"key_destroy":      destroyReport,
		"output_directory": outDir,
		"completed_at":     time.Now().Format(time.RFC3339),
	}
	if err := writeJSON(filepath.Join(outDir, "summary.json"), summary); err != nil {
		return err
	}
	fmt.Println("\n[OK] full online/offline custody flow completed")
	fmt.Printf("[OK] source wallet: %s\n", source.Address)
	fmt.Printf("[OK] target wallet: %s\n", target.Address)
	fmt.Printf("[OK] output directory: %s\n", outDir)
	return nil
}

func (f *flow) loginAll() error {
	onlineRaw, err := f.online.postRaw("/api/login", map[string]any{
		"identifier": f.opts.onlineUser,
		"username":   f.opts.onlineUser,
		"password":   f.opts.onlinePassword,
	})
	if err != nil {
		return fmt.Errorf("online login: %w", err)
	}
	if err := writeRaw(filepath.Join(f.outDir, "01_online_login.json"), onlineRaw); err != nil {
		return err
	}
	token, err := onlineToken(onlineRaw)
	if err != nil {
		return err
	}
	f.online.token = token

	adminLogin, err := f.loginOfflineUser(f.opts.offlineAdmin, f.opts.offlinePassword, "02_offline_login_admin.json")
	if err != nil {
		return err
	}
	f.offlineAdmin.token = adminLogin.Token
	for i, username := range f.participants {
		client := newAPIClient(f.opts.offlineURL)
		login, err := f.loginOfflineUser(username, f.opts.officerPassword, fmt.Sprintf("%02d_offline_login_%s.json", i+3, username))
		if err != nil {
			return err
		}
		client.token = login.Token
		f.offlineUsers[username] = client
	}
	fmt.Println("[OK] online/offline users logged in")
	return nil
}

func (f *flow) loginOfflineUser(username, password, fileName string) (*offlineLoginResponse, error) {
	raw, err := f.offlineAdmin.postRaw("/user/login", map[string]any{
		"identifier": username,
		"username":   username,
		"password":   password,
	})
	if err != nil {
		return nil, fmt.Errorf("offline login %s: %w", username, err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return nil, err
	}
	var login offlineLoginResponse
	if err := json.Unmarshal(raw, &login); err != nil {
		return nil, err
	}
	if login.Token == "" {
		return nil, fmt.Errorf("offline login %s returned empty token", username)
	}
	return &login, nil
}

func (f *flow) prepareSE() (string, error) {
	cfg := f.clientConfig("probe")
	security, err := services.NewSecurityService(cfg)
	if err != nil {
		return "", err
	}
	cplcBytes, err := security.GetCPLC()
	if err != nil {
		return "", fmt.Errorf("probe SE: %w", err)
	}
	cplc := strings.ToUpper(hex.EncodeToString(cplcBytes))
	fmt.Printf("[OK] SE reachable, CPLC=%s\n", cplc)

	raw, err := f.offlineAdmin.getRaw("/se/list")
	if err != nil {
		return "", err
	}
	if err := writeRaw(filepath.Join(f.outDir, "06_offline_se_list.json"), raw); err != nil {
		return "", err
	}
	var listed struct {
		Data []struct {
			SeID   string `json:"se_id"`
			CPLC   string `json:"cplc"`
			Status string `json:"status"`
		} `json:"data"`
	}
	_ = json.Unmarshal(raw, &listed)
	active := make([]string, 0)
	hasProbe := false
	for _, se := range listed.Data {
		if se.Status != "active" {
			continue
		}
		active = append(active, se.SeID+"|"+strings.ToUpper(se.CPLC))
		if strings.EqualFold(se.CPLC, cplc) {
			hasProbe = true
		}
	}
	sort.Strings(active)
	if !hasProbe {
		createRaw, err := f.offlineAdmin.postRaw("/se/create", map[string]any{
			"se_id":            "00-E2E",
			"cplc":             cplc,
			"custody_location": "E2E automated flow",
		})
		if err != nil {
			return "", fmt.Errorf("register SE CPLC %s: %w", cplc, err)
		}
		if err := writeRaw(filepath.Join(f.outDir, "06_offline_se_create.json"), createRaw); err != nil {
			return "", err
		}
		active = append(active, "00-E2E|"+cplc)
		sort.Strings(active)
	}
	if f.opts.isolateSE {
		if err := f.isolateCurrentSE(cplc); err != nil {
			return "", err
		}
		return cplc, nil
	}
	if len(active) > 0 {
		firstCPLC := strings.SplitN(active[0], "|", 2)[1]
		if !strings.EqualFold(firstCPLC, cplc) {
			return "", fmt.Errorf("offline server's first active SE is %s, but local reader CPLC is %s; disable/remove the earlier active SE or use the matching reader", firstCPLC, cplc)
		}
	}
	return cplc, nil
}

func (f *flow) isolateCurrentSE(cplc string) error {
	dbPath, err := resolveOfflineDBPath(f.opts.offlineDBPath)
	if err != nil {
		return err
	}
	sql := fmt.Sprintf(
		"UPDATE ses SET status='disabled' WHERE status='active' AND upper(cplc) <> upper('%s'); UPDATE ses SET status='active' WHERE upper(cplc) = upper('%s');",
		escapeSQL(cplc),
		escapeSQL(cplc),
	)
	cmd := exec.Command("sqlite3", dbPath, sql)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("isolate SE records in %s: %w: %s", dbPath, err, strings.TrimSpace(string(output)))
	}
	if err := writeJSON(filepath.Join(f.outDir, "06_offline_se_isolate.json"), map[string]any{
		"database":    dbPath,
		"active_cplc": cplc,
		"action":      "disabled active SE records whose CPLC does not match the current reader",
	}); err != nil {
		return err
	}
	fmt.Printf("[OK] offline SE records isolated to current CPLC using %s\n", dbPath)
	return nil
}

func (f *flow) connectWSClients(cplc string) error {
	coordinator, err := dialWS(f.opts.offlineWSURL, f.opts.offlineAdmin, serverws.RoleAdmin, f.offlineAdmin.token, nil)
	if err != nil {
		return err
	}
	f.coordinator = coordinator

	for _, username := range f.participants {
		cfg := f.clientConfig(username)
		securityService, err := services.NewSecurityService(cfg)
		if err != nil {
			return fmt.Errorf("create SE service for %s: %w", username, err)
		}
		mpcService := services.NewMPCService(cfg, securityService)
		client, err := dialWS(f.opts.offlineWSURL, username, serverws.RoleOfficer, f.offlineUsers[username].token, func(c *desktopClient) {
			c.security = securityService
			c.mpc = mpcService
			c.cplc = cplc
		})
		if err != nil {
			return err
		}
		f.clients[username] = client
	}
	fmt.Println("[OK] offline WebSocket admin/u1/u2/u3 connected")
	return nil
}

func (f *flow) closeWSClients() {
	if f.coordinator != nil {
		f.coordinator.close()
	}
	for _, client := range f.clients {
		client.close()
	}
}

func (f *flow) generateWallet(label, caseName, prefix string) (wallet, error) {
	caseID, caseNo, err := f.createCase(label, caseName, prefix+"_case_create.json")
	if err != nil {
		return wallet{}, err
	}
	keygenTask, keygenTaskNo, err := f.createKeygenTask(caseID, prefix+"_keygen_task_online.json")
	if err != nil {
		return wallet{}, err
	}
	keygenTaskNo = f.offlineTaskNo(keygenTaskNo, label)
	keygenTask = clonePackageWithTaskNo(keygenTask, keygenTaskNo)
	if err := f.importOfflineTask(prefix+"_keygen_task_offline_import.json", keygenTask); err != nil {
		return wallet{}, err
	}
	shares, address, err := f.runKeygen(prefix+"_keygen_complete_ws.json", keygenTaskNo)
	if err != nil {
		return wallet{}, err
	}
	keygenResult, err := f.downloadOfflineResult(prefix+"_keygen_result_offline.json", keygenTaskNo)
	if err != nil {
		return wallet{}, err
	}
	accountID, err := f.importWalletResult(prefix+"_wallet_import_online.json", caseID, caseNo, keygenResult)
	if err != nil {
		return wallet{}, err
	}
	return wallet{
		Label:      label,
		CaseID:     caseID,
		CaseNo:     caseNo,
		AccountID:  accountID,
		Address:    address,
		TaskNo:     keygenTaskNo,
		OfflineKey: "OFFKEY-" + keygenTaskNo,
		Shares:     shares,
	}, nil
}

func (f *flow) offlineTaskNo(onlineTaskNo, label string) string {
	return sanitizeTaskNo(fmt.Sprintf("%s-%s-%s", onlineTaskNo, strings.ToUpper(label), f.runTag))
}

func clonePackageWithTaskNo(pkg map[string]any, taskNo string) map[string]any {
	raw, err := json.Marshal(pkg)
	if err != nil {
		cloned := make(map[string]any, len(pkg)+1)
		for key, value := range pkg {
			cloned[key] = value
		}
		cloned["task_no"] = taskNo
		return cloned
	}
	var cloned map[string]any
	if err := json.Unmarshal(raw, &cloned); err != nil {
		cloned = map[string]any{}
	}
	cloned["task_no"] = taskNo
	return cloned
}

func sanitizeTaskNo(value string) string {
	var builder strings.Builder
	builder.Grow(len(value))
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			builder.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			builder.WriteRune(r)
		case r >= '0' && r <= '9':
			builder.WriteRune(r)
		case r == '-' || r == '_':
			builder.WriteRune(r)
		default:
			builder.WriteByte('_')
		}
	}
	return strings.Trim(builder.String(), "_")
}

func cleanupWalletShares(securityService *services.SecurityService, wallets ...wallet) {
	for _, item := range wallets {
		if err := cleanupSERecords(securityService, item.Shares); err != nil {
			fmt.Fprintf(os.Stderr, "[WARN] cleanup SE records for %s: %v\n", item.Label, err)
		}
	}
}

func (f *flow) createCase(label, caseName, fileName string) (uint, string, error) {
	caseNo := "CASE-E2E-" + strings.ToUpper(label) + "-" + time.Now().Format("20060102150405")
	raw, err := f.online.postRaw("/api/cases", map[string]any{
		"caseNo":      caseNo,
		"name":        caseName + " " + time.Now().Format(time.RFC3339),
		"status":      "active",
		"description": "online/offline automated E2E flow",
	})
	if err != nil {
		return 0, "", fmt.Errorf("create case: %w", err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return 0, "", err
	}
	data, err := onlineDataMap(raw)
	if err != nil {
		return 0, "", err
	}
	id := uint(numberField(data, "ID", "id"))
	gotCaseNo := stringField(data, "CaseNo", "caseNo", "case_no")
	if id == 0 || gotCaseNo == "" {
		return 0, "", fmt.Errorf("create case response missing ID/CaseNo: %s", raw)
	}
	fmt.Printf("[OK] online case created: %s\n", gotCaseNo)
	return id, gotCaseNo, nil
}

func (f *flow) createKeygenTask(caseID uint, fileName string) (map[string]any, string, error) {
	raw, err := f.online.postRaw("/api/offline-tasks/custody-keygen", map[string]any{
		"caseId":          caseID,
		"coinType":        "ETH",
		"thresholdPolicy": "2_of_3",
	})
	if err != nil {
		return nil, "", fmt.Errorf("create keygen task: %w", err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return nil, "", err
	}
	data, err := onlineDataMap(raw)
	if err != nil {
		return nil, "", err
	}
	pkg, ok := data["package"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("keygen response missing package")
	}
	taskNo := stringField(pkg, "task_no")
	if taskNo == "" {
		return nil, "", fmt.Errorf("keygen package missing task_no")
	}
	fmt.Printf("[OK] online keygen task exported: %s\n", taskNo)
	return pkg, taskNo, nil
}

func (f *flow) importOfflineTask(fileName string, pkg map[string]any) error {
	raw, err := f.offlineAdmin.postRaw("/offline/tasks/import", pkg)
	if err != nil {
		return fmt.Errorf("offline import task %s: %w", stringField(pkg, "task_no"), err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return err
	}
	fmt.Printf("[OK] offline task imported: %s\n", stringField(pkg, "task_no"))
	return nil
}

func (f *flow) runKeygen(fileName, taskNo string) ([]keyShare, string, error) {
	raw, err := f.offlineAdmin.postRaw("/offline/tasks/"+url.PathEscape(taskNo)+"/keygen/start", map[string]any{
		"participants":   f.participants,
		"offline_key_id": "OFFKEY-" + taskNo,
	})
	if err != nil {
		return nil, "", fmt.Errorf("build keygen request: %w", err)
	}
	var response struct {
		Message map[string]any `json:"message"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, "", err
	}
	if err := f.coordinator.send(response.Message); err != nil {
		return nil, "", err
	}

	invites := map[string]serverws.KeyGenInviteMessage{}
	for _, username := range f.participants {
		invite, err := readWS[serverws.KeyGenInviteMessage](f.clients[username], 15*time.Second)
		if err != nil {
			return nil, "", err
		}
		invites[username] = invite
	}
	for _, username := range f.participants {
		invite := invites[username]
		if err := f.clients[username].send(serverws.KeyGenResponseMessage{
			BaseMessage: serverws.BaseMessage{Type: serverws.MsgKeyGenResponse},
			SessionKey:  invite.SessionKey,
			PartyIndex:  invite.PartyIndex,
			CPLC:        f.clients[username].cplc,
			Accept:      true,
		}); err != nil {
			return nil, "", err
		}
	}

	params := map[string]serverws.KeyGenParamsMessage{}
	for _, username := range f.participants {
		msg, err := readWS[serverws.KeyGenParamsMessage](f.clients[username], 15*time.Second)
		if err != nil {
			return nil, "", err
		}
		params[username] = msg
	}
	shares, err := performKeygenForAll(f.clients, params, f.participants)
	if err != nil {
		return nil, "", err
	}
	for _, share := range shares {
		if err := f.clients[share.Username].send(serverws.KeyGenResultMessage{
			BaseMessage:    serverws.BaseMessage{Type: serverws.MsgKeyGenResult},
			SessionKey:     params[share.Username].SessionKey,
			PartyIndex:     share.PartyIndex,
			Address:        share.Address,
			PublicKey:      share.PublicKey,
			CPLC:           share.CPLC,
			RecordID:       share.RecordID,
			EncryptedShard: share.EncryptedShard,
			Success:        true,
			Message:        "ok",
		}); err != nil {
			return nil, "", err
		}
		time.Sleep(250 * time.Millisecond)
	}
	complete, err := readWS[serverws.KeyGenCompleteMessage](f.coordinator, 30*time.Second)
	if err != nil {
		return nil, "", err
	}
	if !complete.Success || complete.Address == "" {
		return nil, "", fmt.Errorf("keygen failed: %+v", complete)
	}
	if err := writeJSON(filepath.Join(f.outDir, fileName), complete); err != nil {
		return nil, "", err
	}
	fmt.Printf("[OK] keygen completed: %s\n", complete.Address)
	return shares, complete.Address, nil
}

func (f *flow) downloadOfflineResult(fileName, taskNo string) (map[string]any, error) {
	raw, err := f.offlineAdmin.getRaw("/offline/results/" + url.PathEscape(taskNo) + "/download")
	if err != nil {
		return nil, fmt.Errorf("download offline result %s: %w", taskNo, err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return nil, err
	}
	var pkg map[string]any
	if err := json.Unmarshal(raw, &pkg); err != nil {
		return nil, err
	}
	fmt.Printf("[OK] offline result downloaded: %s\n", taskNo)
	return pkg, nil
}

func (f *flow) importWalletResult(fileName string, caseID uint, caseNo string, result map[string]any) (uint, error) {
	payload := mapField(result, "payload")
	raw, err := f.online.postRaw(fmt.Sprintf("/api/cases/%d/custody-wallet/import-result", caseID), map[string]any{
		"taskNo":         stringField(result, "task_no"),
		"caseNo":         caseNo,
		"coinType":       firstNonEmpty(stringField(payload, "coin_type"), "ETH"),
		"custodyAddress": stringField(payload, "custody_address"),
		"publicKey":      stringField(payload, "public_key"),
		"offlineRefNo":   stringField(payload, "offline_ref_no"),
		"completedAt":    stringField(payload, "completed_at"),
	})
	if err != nil {
		return 0, fmt.Errorf("online import wallet result: %w", err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return 0, err
	}
	data, err := onlineDataMap(raw)
	if err != nil {
		return 0, err
	}
	account := mapField(data, "account")
	accountID := uint(numberField(account, "ID", "id"))
	if accountID == 0 {
		return 0, fmt.Errorf("wallet import response missing account ID")
	}
	fmt.Printf("[OK] wallet imported into online case: account_id=%d\n", accountID)
	return accountID, nil
}

func (f *flow) fundCustodyAddress(fileName, address string) error {
	result, err := ganacheSend(f.opts.ganacheRPC, address, f.opts.fundAmount)
	if err != nil {
		return fmt.Errorf("Ganache fund custody address: %w", err)
	}
	if err := writeJSON(filepath.Join(f.outDir, fileName), result); err != nil {
		return err
	}
	fmt.Printf("[OK] Ganache funded custody address: %s ETH -> %s\n", f.opts.fundAmount, address)
	return nil
}

func (f *flow) createTransaction(fileName string, caseID uint, caseNo string, accountID uint, fromAddress, toAddress string) (uint, error) {
	raw, err := f.online.postRaw("/api/transactions", map[string]any{
		"caseId":        caseID,
		"caseNo":        caseNo,
		"txType":        "withdraw",
		"fromAccountId": accountID,
		"fromAddress":   fromAddress,
		"toAddress":     toAddress,
		"value":         f.opts.txValue,
		"coinType":      "ETH",
		"reason":        "E2E custody flow withdraw test",
	})
	if err != nil {
		return 0, fmt.Errorf("online create transaction: %w", err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return 0, err
	}
	data, err := onlineDataMap(raw)
	if err != nil {
		return 0, err
	}
	txID := uint(numberField(data, "ID", "id"))
	if txID == 0 {
		return 0, fmt.Errorf("transaction create response missing ID")
	}
	fmt.Printf("[OK] online transaction created: id=%d\n", txID)
	return txID, nil
}

func (f *flow) exportSignTask(fileName string, txID uint) (map[string]any, string, error) {
	raw, err := f.online.getRaw(fmt.Sprintf("/api/transactions/%d/export-sign-task", txID))
	if err != nil {
		return nil, "", fmt.Errorf("online export sign task: %w", err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return nil, "", err
	}
	data, err := onlineDataMap(raw)
	if err != nil {
		return nil, "", err
	}
	pkg, ok := data["package"].(map[string]any)
	if !ok {
		return nil, "", fmt.Errorf("sign export response missing package")
	}
	taskNo := stringField(pkg, "task_no")
	if taskNo == "" {
		return nil, "", fmt.Errorf("sign task package missing task_no")
	}
	fmt.Printf("[OK] online sign task exported: %s\n", taskNo)
	return pkg, taskNo, nil
}

func (f *flow) runSign(fileName, taskNo string) (string, error) {
	raw, err := f.offlineAdmin.postRaw("/offline/tasks/"+url.PathEscape(taskNo)+"/sign/start", map[string]any{
		"participants": f.signers,
	})
	if err != nil {
		return "", fmt.Errorf("build sign request: %w", err)
	}
	var response struct {
		Message map[string]any `json:"message"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return "", err
	}
	if err := f.coordinator.send(response.Message); err != nil {
		return "", err
	}
	invites := map[string]serverws.SignInviteMessage{}
	for _, username := range f.signers {
		invite, err := readWS[serverws.SignInviteMessage](f.clients[username], 15*time.Second)
		if err != nil {
			return "", err
		}
		invites[username] = invite
	}
	for _, username := range f.signers {
		invite := invites[username]
		if err := f.clients[username].send(serverws.SignResponseMessage{
			BaseMessage: serverws.BaseMessage{Type: serverws.MsgSignResponse},
			SessionKey:  invite.SessionKey,
			PartyIndex:  invite.PartyIndex,
			CPLC:        f.clients[username].cplc,
			Accept:      true,
		}); err != nil {
			return "", err
		}
	}
	params := map[string]serverws.SignParamsMessage{}
	for _, username := range f.signers {
		msg, err := readWS[serverws.SignParamsMessage](f.clients[username], 15*time.Second)
		if err != nil {
			return "", err
		}
		params[username] = msg
	}
	results, err := performSignForAll(f.clients, params, f.signers)
	if err != nil {
		return "", err
	}
	for _, result := range results {
		if err := f.clients[result.Username].send(serverws.SignResultMessage{
			BaseMessage:  serverws.BaseMessage{Type: serverws.MsgSignResult},
			SessionKey:   params[result.Username].SessionKey,
			SigningIndex: result.SigningIndex,
			Success:      true,
			Signature:    result.Signature,
			Message:      "ok",
		}); err != nil {
			return "", err
		}
		time.Sleep(250 * time.Millisecond)
	}
	complete, err := readWS[serverws.SignCompleteMessage](f.coordinator, 30*time.Second)
	if err != nil {
		return "", err
	}
	if !complete.Success || complete.Signature == "" {
		return "", fmt.Errorf("sign failed: %+v", complete)
	}
	if err := writeJSON(filepath.Join(f.outDir, fileName), complete); err != nil {
		return "", err
	}
	fmt.Printf("[OK] sign completed: %s\n", shortHex(complete.Signature))
	return complete.Signature, nil
}

func (f *flow) runShardTransfer(fileName string, item wallet, fromUsername, toUsername string) (map[string]any, error) {
	before, err := f.listOfflineShards(item.Address, fromUsername, "active")
	if err != nil {
		return nil, err
	}
	if len(before) != 1 {
		return nil, fmt.Errorf("expected one active shard for %s at %s, got %d", fromUsername, item.Address, len(before))
	}
	shard := before[0]
	raw, err := f.offlineAdmin.postRaw("/offline/shards/"+url.PathEscape(shard.ShardID)+"/transfer", map[string]any{
		"to_username": toUsername,
		"reason":      "E2E shard transfer",
	})
	if err != nil {
		return nil, fmt.Errorf("build shard transfer request: %w", err)
	}
	var response struct {
		Message map[string]any `json:"message"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	if err := f.coordinator.send(response.Message); err != nil {
		return nil, err
	}

	fromInvite, err := readWS[serverws.TransferInviteMessage](f.clients[fromUsername], 15*time.Second)
	if err != nil {
		return nil, err
	}
	if fromInvite.Type != serverws.MsgTransferInvite || fromInvite.ShardID != shard.ShardID || fromInvite.ToUsername != toUsername {
		return nil, fmt.Errorf("bad transfer invite for %s: %+v", fromUsername, fromInvite)
	}
	toInvite, err := readWS[serverws.TransferInviteMessage](f.coordinator, 15*time.Second)
	if err != nil {
		return nil, err
	}
	if toInvite.Type != serverws.MsgTransferInvite || toInvite.ShardID != shard.ShardID || toInvite.FromUsername != fromUsername {
		return nil, fmt.Errorf("bad transfer invite for %s: %+v", toUsername, toInvite)
	}

	if err := f.clients[fromUsername].send(serverws.TransferResponseMessage{
		BaseMessage: serverws.BaseMessage{Type: serverws.MsgTransferResponse},
		SessionKey:  fromInvite.SessionKey,
		ShardID:     shard.ShardID,
		Accept:      true,
	}); err != nil {
		return nil, err
	}
	if err := f.coordinator.send(serverws.TransferResponseMessage{
		BaseMessage: serverws.BaseMessage{Type: serverws.MsgTransferResponse},
		SessionKey:  toInvite.SessionKey,
		ShardID:     shard.ShardID,
		Accept:      true,
	}); err != nil {
		return nil, err
	}

	fromComplete, err := readWS[serverws.TransferCompleteMessage](f.clients[fromUsername], 15*time.Second)
	if err != nil {
		return nil, err
	}
	adminComplete1, err := readWS[serverws.TransferCompleteMessage](f.coordinator, 15*time.Second)
	if err != nil {
		return nil, err
	}
	adminComplete2, err := readWS[serverws.TransferCompleteMessage](f.coordinator, 15*time.Second)
	if err != nil {
		return nil, err
	}
	adminCompletes := []serverws.TransferCompleteMessage{adminComplete1, adminComplete2}
	for _, complete := range append([]serverws.TransferCompleteMessage{fromComplete}, adminCompletes...) {
		if complete.Type != serverws.MsgTransferComplete || !complete.Success || complete.ShardID != shard.ShardID {
			return nil, fmt.Errorf("bad transfer complete: %+v", complete)
		}
	}

	oldHolderShards, err := f.listOfflineShards(item.Address, fromUsername, "active")
	if err != nil {
		return nil, err
	}
	newHolderShards, err := f.listOfflineShards(item.Address, toUsername, "active")
	if err != nil {
		return nil, err
	}
	if len(oldHolderShards) != 0 {
		return nil, fmt.Errorf("expected no active shard for old holder %s, got %d", fromUsername, len(oldHolderShards))
	}
	if len(newHolderShards) != 1 || newHolderShards[0].ShardID != shard.ShardID {
		return nil, fmt.Errorf("expected shard %s to belong to %s, got %+v", shard.ShardID, toUsername, newHolderShards)
	}

	report := map[string]any{
		"address":         item.Address,
		"offline_key_id":  item.OfflineKey,
		"shard_id":        shard.ShardID,
		"shard_index":     shard.ShardIndex,
		"from_username":   fromUsername,
		"to_username":     toUsername,
		"from_complete":   fromComplete,
		"admin_complete":  adminCompletes,
		"new_holder_view": newHolderShards,
	}
	if err := writeJSON(filepath.Join(f.outDir, fileName), report); err != nil {
		return nil, err
	}
	fmt.Printf("[OK] shard transferred: %s %s -> %s\n", shard.ShardID, fromUsername, toUsername)
	return report, nil
}

func (f *flow) runKeyDestroy(fileName string, item wallet) (map[string]any, error) {
	raw, err := f.offlineAdmin.postRaw("/offline/keys/"+url.PathEscape(item.OfflineKey)+"/destroy", map[string]any{
		"reason": "E2E key destroy",
	})
	if err != nil {
		return nil, fmt.Errorf("build key destroy request: %w", err)
	}
	var response struct {
		Message map[string]any `json:"message"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	if err := f.coordinator.send(response.Message); err != nil {
		return nil, err
	}

	invites := map[string]serverws.DestroyInviteMessage{}
	for _, username := range f.participants {
		invite, err := readWS[serverws.DestroyInviteMessage](f.clients[username], 15*time.Second)
		if err != nil {
			return nil, err
		}
		if invite.Type != serverws.MsgDestroyInvite || !strings.EqualFold(invite.Address, item.Address) {
			return nil, fmt.Errorf("bad destroy invite for %s: %+v", username, invite)
		}
		invites[username] = invite
	}
	for _, username := range f.participants {
		invite := invites[username]
		if err := f.clients[username].send(serverws.DestroyResponseMessage{
			BaseMessage: serverws.BaseMessage{Type: serverws.MsgDestroyResponse},
			SessionKey:  invite.SessionKey,
			PartyIndex:  invite.PartyIndex,
			CPLC:        f.clients[username].cplc,
			Accept:      true,
		}); err != nil {
			return nil, err
		}
	}

	params := map[string]serverws.DestroyParamsMessage{}
	for _, username := range f.participants {
		msg, err := readWS[serverws.DestroyParamsMessage](f.clients[username], 15*time.Second)
		if err != nil {
			return nil, err
		}
		if msg.Type != serverws.MsgDestroyParams || !strings.EqualFold(msg.Address, item.Address) || msg.RecordID == "" || msg.Signature == "" {
			return nil, fmt.Errorf("bad destroy params for %s: %+v", username, msg)
		}
		params[username] = msg
	}

	deleted := []map[string]any{}
	for _, username := range f.participants {
		msg := params[username]
		signature, err := base64.StdEncoding.DecodeString(msg.Signature)
		if err != nil {
			return nil, fmt.Errorf("decode destroy signature for %s: %w", username, err)
		}
		if err := f.clients[username].security.DeleteData(msg.RecordID, msg.Address, signature); err != nil {
			return nil, fmt.Errorf("delete SE record for %s: %w", username, err)
		}
		if _, err := f.clients[username].security.ReadData(msg.RecordID, msg.Address, signature); err == nil {
			return nil, fmt.Errorf("SE record for %s remained readable after delete", username)
		}
		if err := f.clients[username].send(serverws.DestroyResultMessage{
			BaseMessage: serverws.BaseMessage{Type: serverws.MsgDestroyResult},
			SessionKey:  msg.SessionKey,
			PartyIndex:  msg.PartyIndex,
			Success:     true,
			Message:     "deleted and read-back rejected",
		}); err != nil {
			return nil, err
		}
		deleted = append(deleted, map[string]any{
			"username":    username,
			"party_index": msg.PartyIndex,
			"record_id":   msg.RecordID,
		})
		time.Sleep(250 * time.Millisecond)
	}

	complete, err := readWS[serverws.DestroyCompleteMessage](f.coordinator, 30*time.Second)
	if err != nil {
		return nil, err
	}
	if complete.Type != serverws.MsgDestroyComplete || !complete.Success || complete.Destroyed != len(f.participants) {
		return nil, fmt.Errorf("bad destroy complete: %+v", complete)
	}
	keySnapshot, err := f.offlineKeySnapshot(item.OfflineKey)
	if err != nil {
		return nil, err
	}
	if status, _ := keySnapshot["status"].(string); status != "destroyed" {
		return nil, fmt.Errorf("offline key status after destroy = %s", status)
	}
	activeShards, err := f.listOfflineShards(item.Address, "", "active")
	if err != nil {
		return nil, err
	}
	if len(activeShards) != 0 {
		return nil, fmt.Errorf("expected no active shards after destroy, got %d", len(activeShards))
	}

	report := map[string]any{
		"address":        item.Address,
		"offline_key_id": item.OfflineKey,
		"deleted":        deleted,
		"complete":       complete,
		"key_snapshot":   keySnapshot,
	}
	if err := writeJSON(filepath.Join(f.outDir, fileName), report); err != nil {
		return nil, err
	}
	fmt.Printf("[OK] key destroyed: %s\n", item.OfflineKey)
	return report, nil
}

func (f *flow) listOfflineShards(address, username, status string) ([]keyShardRecord, error) {
	query := url.Values{}
	if address != "" {
		query.Set("address", address)
	}
	if username != "" {
		query.Set("username", username)
	}
	if status != "" {
		query.Set("status", status)
	}
	path := "/offline/shards"
	if encoded := query.Encode(); encoded != "" {
		path += "?" + encoded
	}
	raw, err := f.offlineAdmin.getRaw(path)
	if err != nil {
		return nil, fmt.Errorf("list offline shards: %w", err)
	}
	var response struct {
		Shards []keyShardRecord `json:"shards"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	sort.Slice(response.Shards, func(i, j int) bool {
		return response.Shards[i].ShardIndex < response.Shards[j].ShardIndex
	})
	return response.Shards, nil
}

func (f *flow) offlineKeySnapshot(offlineKeyID string) (map[string]any, error) {
	raw, err := f.offlineAdmin.getRaw("/offline/keys/" + url.PathEscape(offlineKeyID))
	if err != nil {
		return nil, fmt.Errorf("get offline key %s: %w", offlineKeyID, err)
	}
	var response struct {
		Key map[string]any `json:"key"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return nil, err
	}
	if len(response.Key) == 0 {
		return nil, fmt.Errorf("offline key %s response did not include key", offlineKeyID)
	}
	return response.Key, nil
}

func (f *flow) verifySignature(fileName string, result map[string]any) error {
	payload := mapField(result, "payload")
	report, err := verifyEthereumSignature(
		stringField(payload, "message_hash"),
		stringField(payload, "signature"),
		stringField(payload, "from_address"),
	)
	if err != nil {
		report["success"] = false
		report["error"] = err.Error()
		if writeErr := writeJSON(filepath.Join(f.outDir, fileName), report); writeErr != nil {
			return writeErr
		}
		return err
	}
	report["success"] = true
	if err := writeJSON(filepath.Join(f.outDir, fileName), report); err != nil {
		return err
	}
	fmt.Printf("[OK] offline signature verified: recovered=%s\n", report["recovered_address"])
	return nil
}

func (f *flow) importSignature(fileName string, txID uint, result map[string]any) error {
	payload := mapField(result, "payload")
	raw, err := f.online.postRaw(fmt.Sprintf("/api/transactions/%d/import-signature", txID), map[string]any{
		"taskNo":      stringField(result, "task_no"),
		"messageHash": stringField(payload, "message_hash"),
		"signature":   stringField(payload, "signature"),
		"completedAt": stringField(payload, "completed_at"),
	})
	if err != nil {
		return fmt.Errorf("online import signature: %w", err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return err
	}
	fmt.Println("[OK] signature imported into online transaction")
	return nil
}

func (f *flow) broadcast(fileName string, txID uint) error {
	raw, err := f.online.postRaw(fmt.Sprintf("/api/transactions/%d/broadcast", txID), nil)
	if err != nil {
		return fmt.Errorf("online broadcast transaction: %w", err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return err
	}
	fmt.Println("[OK] online transaction broadcasted")
	return nil
}

func (f *flow) syncAccountBalance(fileName string, accountID uint) (map[string]any, error) {
	raw, err := f.online.postRaw(fmt.Sprintf("/api/accounts/%d/sync-balance", accountID), nil)
	if err != nil {
		return nil, fmt.Errorf("sync account %d balance: %w", accountID, err)
	}
	if err := writeRaw(filepath.Join(f.outDir, fileName), raw); err != nil {
		return nil, err
	}
	data, err := onlineDataMap(raw)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (f *flow) chainBalances(fileName string, addresses ...string) (map[string]any, error) {
	balances := make(map[string]any)
	for _, address := range addresses {
		balanceWei, err := ganacheBalance(f.opts.ganacheRPC, address)
		if err != nil {
			return nil, err
		}
		balances[address] = map[string]string{
			"wei": balanceWei.String(),
			"eth": weiToETH(balanceWei),
		}
	}
	if err := writeJSON(filepath.Join(f.outDir, fileName), balances); err != nil {
		return nil, err
	}
	return balances, nil
}

func performKeygenForAll(clients map[string]*desktopClient, params map[string]serverws.KeyGenParamsMessage, participants []string) ([]keyShare, error) {
	type outcome struct {
		share keyShare
		err   error
	}
	outcomes := make(chan outcome, len(participants))
	var wg sync.WaitGroup
	for _, username := range participants {
		username := username
		p := params[username]
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			address, publicKey, encryptedShard, err := clients[username].mpc.KeyGeneration(ctx, p.ManagerAddr, p.Room, p.Threshold, p.TotalParties, p.PartyIndex, p.FileName, p.RecordID)
			if err != nil {
				outcomes <- outcome{err: fmt.Errorf("%s keygen: %w", username, err)}
				return
			}
			outcomes <- outcome{share: keyShare{
				Username:       username,
				PartyIndex:     p.PartyIndex,
				Address:        address,
				PublicKey:      publicKey,
				CPLC:           clients[username].cplc,
				RecordID:       p.RecordID,
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
	return shares, ensureSameKey(shares)
}

func performSignForAll(clients map[string]*desktopClient, params map[string]serverws.SignParamsMessage, participants []string) ([]signOutcome, error) {
	type outcome struct {
		result signOutcome
		err    error
	}
	outcomes := make(chan outcome, len(participants))
	var wg sync.WaitGroup
	for _, username := range participants {
		username := username
		p := params[username]
		wg.Add(1)
		go func() {
			defer wg.Done()
			encryptedShard, err := base64.StdEncoding.DecodeString(p.EncryptedShard)
			if err != nil {
				outcomes <- outcome{err: fmt.Errorf("%s encrypted shard: %w", username, err)}
				return
			}
			seSignature, err := base64.StdEncoding.DecodeString(p.Signature)
			if err != nil {
				outcomes <- outcome{err: fmt.Errorf("%s SE signature: %w", username, err)}
				return
			}
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			signature, err := clients[username].mpc.SignMessage(ctx, p.ManagerAddr, p.Room, p.SigningIndex, p.Parties, p.MessageHash, p.FileName, p.RecordID, p.Address, encryptedShard, seSignature)
			if err != nil {
				outcomes <- outcome{err: fmt.Errorf("%s signing: %w", username, err)}
				return
			}
			outcomes <- outcome{result: signOutcome{Username: username, SigningIndex: p.SigningIndex, Signature: signature}}
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
			signature = result.result.Signature
		} else if signature != result.result.Signature {
			return nil, fmt.Errorf("signature mismatch: %s != %s", signature, result.result.Signature)
		}
		results = append(results, result.result)
	}
	sort.Slice(results, func(i, j int) bool { return results[i].SigningIndex < results[j].SigningIndex })
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

func dialWS(wsURL, username string, role serverws.ClientRole, token string, configure func(*desktopClient)) (*desktopClient, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("dial websocket %s: %w", username, err)
	}
	client := &desktopClient{username: username, role: role, token: token, conn: conn}
	if configure != nil {
		configure(client)
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
	ack, err := readWS[serverws.RegisterCompleteMessage](client, 10*time.Second)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	if !ack.Success {
		_ = conn.Close()
		return nil, fmt.Errorf("websocket register %s failed: %+v", username, ack)
	}
	return client, nil
}

func (c *desktopClient) send(msg any) error {
	return c.conn.WriteJSON(msg)
}

func (c *desktopClient) close() {
	if c != nil && c.conn != nil {
		_ = c.conn.Close()
	}
}

func readWS[T any](client *desktopClient, timeout time.Duration) (T, error) {
	var zero T
	if err := client.conn.SetReadDeadline(time.Now().Add(timeout)); err != nil {
		return zero, err
	}
	_, raw, err := client.conn.ReadMessage()
	if err != nil {
		return zero, fmt.Errorf("%s websocket read: %w", client.username, err)
	}
	var base serverws.BaseMessage
	if err := json.Unmarshal(raw, &base); err != nil {
		return zero, fmt.Errorf("%s decode websocket base: %w; raw=%s", client.username, err, raw)
	}
	if base.Type == serverws.MsgError {
		var errMsg serverws.ErrorMessage
		_ = json.Unmarshal(raw, &errMsg)
		return zero, fmt.Errorf("%s received server error: %s %s", client.username, errMsg.Message, errMsg.Details)
	}
	var msg T
	if err := json.Unmarshal(raw, &msg); err != nil {
		return zero, fmt.Errorf("%s decode websocket message: %w; raw=%s", client.username, err, raw)
	}
	return msg, nil
}

func (f *flow) clientConfig(name string) *clientcfg.Config {
	return &clientcfg.Config{
		Debug:          f.opts.debug,
		CardReaderName: f.opts.readerName,
		AppletAID:      f.opts.appletAID,
		TempDir:        filepath.Join(f.outDir, "client-temp", name),
		LogDir:         filepath.Join(f.outDir, "client-logs", name),
		ManagerAddr:    "http://127.0.0.1:8000",
	}
}

func newAPIClient(baseURL string) *apiClient {
	return &apiClient{baseURL: strings.TrimRight(baseURL, "/"), client: &http.Client{Timeout: 30 * time.Second}}
}

func (c *apiClient) getRaw(path string) ([]byte, error) {
	return c.doRaw(http.MethodGet, path, nil)
}

func (c *apiClient) postRaw(path string, body any) ([]byte, error) {
	return c.doRaw(http.MethodPost, path, body)
}

func (c *apiClient) doRaw(method, path string, body any) ([]byte, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequest(method, c.baseURL+path, reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.token != "" {
		req.Header.Set("Authorization", c.token)
	}
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(raw))
	}
	return raw, nil
}

func onlineToken(raw []byte) (string, error) {
	data, err := onlineDataMap(raw)
	if err != nil {
		return "", err
	}
	token := stringField(data, "token")
	if token == "" {
		return "", fmt.Errorf("online login returned empty token")
	}
	return token, nil
}

func onlineDataMap(raw []byte) (map[string]any, error) {
	var resp standardResponse
	if err := json.Unmarshal(raw, &resp); err != nil {
		return nil, err
	}
	if resp.Code >= 400 {
		return nil, fmt.Errorf("online API error %d: %s", resp.Code, resp.Message)
	}
	var data map[string]any
	if len(resp.Data) == 0 {
		return map[string]any{}, nil
	}
	if err := json.Unmarshal(resp.Data, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func ganacheAccounts(rpcURL string) ([]string, error) {
	var accounts []string
	if err := rpcCall(rpcURL, "eth_accounts", []any{}, &accounts); err != nil {
		return nil, fmt.Errorf("Ganache eth_accounts: %w", err)
	}
	return accounts, nil
}

func ganacheSend(rpcURL, toAddress, amountETH string) (map[string]any, error) {
	accounts, err := ganacheAccounts(rpcURL)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		return nil, fmt.Errorf("Ganache returned no unlocked accounts")
	}
	value, err := ethToWei(amountETH)
	if err != nil {
		return nil, err
	}
	args := map[string]any{
		"from":  accounts[0],
		"to":    toAddress,
		"value": "0x" + value.Text(16),
	}
	var txHash string
	if err := rpcCall(rpcURL, "eth_sendTransaction", []any{args}, &txHash); err != nil {
		return nil, err
	}
	return map[string]any{
		"rpc":        rpcURL,
		"from":       accounts[0],
		"to":         toAddress,
		"amount_eth": amountETH,
		"amount_wei": value.String(),
		"tx_hash":    txHash,
	}, nil
}

func ganacheBalance(rpcURL, address string) (*big.Int, error) {
	var hexBalance string
	if err := rpcCall(rpcURL, "eth_getBalance", []any{address, "latest"}, &hexBalance); err != nil {
		return nil, fmt.Errorf("Ganache eth_getBalance %s: %w", address, err)
	}
	value := new(big.Int)
	value.SetString(strings.TrimPrefix(hexBalance, "0x"), 16)
	return value, nil
}

func verifyEthereumSignature(messageHash, signature, expectedAddress string) (map[string]any, error) {
	report := map[string]any{
		"message_hash":       messageHash,
		"signature":          signature,
		"expected_address":   expectedAddress,
		"signature_format":   "ethereum_rsv",
		"tried_recovery_ids": []int{},
	}
	hashBytes, err := decodeFixedHex(messageHash, 32, "message_hash")
	if err != nil {
		return report, err
	}
	signatureBytes, err := decodeFixedHex(signature, 65, "signature")
	if err != nil {
		return report, err
	}
	if signatureBytes[64] == 27 || signatureBytes[64] == 28 {
		signatureBytes[64] -= 27
	}
	if signatureBytes[64] != 0 && signatureBytes[64] != 1 {
		return report, fmt.Errorf("signature recovery id invalid: %d", signatureBytes[64])
	}

	var recovered []map[string]any
	for _, recoveryID := range []byte{signatureBytes[64], 1 - signatureBytes[64]} {
		candidate := append([]byte(nil), signatureBytes...)
		candidate[64] = recoveryID
		pubKey, err := crypto.SigToPub(hashBytes, candidate)
		attempt := map[string]any{
			"recovery_id": int(recoveryID),
		}
		if err != nil {
			attempt["error"] = err.Error()
			recovered = append(recovered, attempt)
			continue
		}
		address := crypto.PubkeyToAddress(*pubKey).Hex()
		attempt["address"] = address
		recovered = append(recovered, attempt)
		if strings.EqualFold(address, expectedAddress) {
			report["recovered_address"] = address
			report["recovery_id"] = int(recoveryID)
			report["tried_recovery_ids"] = []int{int(signatureBytes[64]), int(1 - signatureBytes[64])}
			report["attempts"] = recovered
			return report, nil
		}
	}
	report["attempts"] = recovered
	return report, fmt.Errorf("signature does not recover expected address: expected=%s attempts=%v", expectedAddress, recovered)
}

func decodeFixedHex(value string, length int, label string) ([]byte, error) {
	cleaned := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	decoded, err := hex.DecodeString(cleaned)
	if err != nil {
		return nil, fmt.Errorf("decode %s: %w", label, err)
	}
	if len(decoded) != length {
		return nil, fmt.Errorf("%s length=%d, want %d bytes", label, len(decoded), length)
	}
	return decoded, nil
}

func rpcCall(rpcURL, method string, params []any, out any) error {
	payload := map[string]any{"jsonrpc": "2.0", "id": 1, "method": method, "params": params}
	raw, _ := json.Marshal(payload)
	resp, err := http.Post(rpcURL, "application/json", bytes.NewReader(raw))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, body)
	}
	var decoded struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &decoded); err != nil {
		return err
	}
	if decoded.Error != nil {
		return fmt.Errorf("RPC %d: %s", decoded.Error.Code, decoded.Error.Message)
	}
	return json.Unmarshal(decoded.Result, out)
}

func ethToWei(text string) (*big.Int, error) {
	value := new(big.Float)
	if _, ok := value.SetString(strings.TrimSpace(text)); !ok {
		return nil, fmt.Errorf("invalid ETH amount: %s", text)
	}
	weiFloat := new(big.Float).Mul(value, big.NewFloat(1e18))
	wei := new(big.Int)
	weiFloat.Int(wei)
	return wei, nil
}

func weiToETH(wei *big.Int) string {
	if wei == nil {
		return "0"
	}
	value := new(big.Float).SetInt(wei)
	eth := new(big.Float).Quo(value, big.NewFloat(1e18))
	return eth.Text('f', 18)
}

func prepareOutputDir(explicit string) (string, error) {
	outDir := explicit
	if outDir == "" {
		outDir = filepath.Join("runs", time.Now().Format("20060102-150405"))
	}
	abs, err := filepath.Abs(outDir)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return "", err
	}
	return abs, nil
}

func resolveOfflineDBPath(explicit string) (string, error) {
	if explicit != "" {
		if _, err := os.Stat(explicit); err != nil {
			return "", fmt.Errorf("offline DB path %s: %w", explicit, err)
		}
		return filepath.Abs(explicit)
	}
	root, err := findRepoRoot()
	if err != nil {
		return "", err
	}
	candidates := []string{
		filepath.Join(root, "offline-server-handoff", "data", "crypto-custody.db"),
		filepath.Join(root, "deploy", "offline-server", "data", "crypto-custody.db"),
		filepath.Join(root, "offline-server", "data", "crypto-custody.db"),
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err == nil {
			return filepath.Abs(candidate)
		}
	}
	return "", fmt.Errorf("offline DB not found; pass -offline-db /path/to/crypto-custody.db")
}

func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if exists(filepath.Join(wd, "offline-server")) && exists(filepath.Join(wd, "e2e", "custody-flow")) {
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

func escapeSQL(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}

func writeRaw(path string, raw []byte) error {
	var pretty bytes.Buffer
	if json.Indent(&pretty, raw, "", "  ") == nil {
		raw = pretty.Bytes()
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}

func writeJSON(path string, value any) error {
	raw, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(raw, '\n'), 0644)
}

func parseParticipants(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

func ensureSubset(values, allowed []string) error {
	allowedSet := make(map[string]bool, len(allowed))
	for _, value := range allowed {
		allowedSet[value] = true
	}
	for _, value := range values {
		if !allowedSet[value] {
			return fmt.Errorf("signer %s is not in participants %v", value, allowed)
		}
	}
	return nil
}

func mapField(m map[string]any, key string) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	if nested, ok := m[key].(map[string]any); ok {
		return nested
	}
	return map[string]any{}
}

func stringField(m map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			switch typed := value.(type) {
			case string:
				return typed
			case fmt.Stringer:
				return typed.String()
			}
		}
	}
	return ""
}

func numberField(m map[string]any, keys ...string) int64 {
	for _, key := range keys {
		if value, ok := m[key]; ok {
			switch typed := value.(type) {
			case float64:
				return int64(typed)
			case int:
				return int64(typed)
			case int64:
				return typed
			case json.Number:
				parsed, _ := typed.Int64()
				return parsed
			case string:
				parsed, _ := strconv.ParseInt(typed, 10, 64)
				return parsed
			}
		}
	}
	return 0
}

func ensureSameKey(shares []keyShare) error {
	if len(shares) == 0 {
		return errors.New("empty keygen results")
	}
	address := shares[0].Address
	publicKey := shares[0].PublicKey
	for _, share := range shares {
		if !strings.EqualFold(share.Address, address) {
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

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func shortHex(value string) string {
	if len(value) <= 18 {
		return value
	}
	return value[:18] + "..."
}

func envDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
