package ws

import (
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"offline-server/manager"
	"offline-server/storage/model"
	"offline-server/tools"
	mem_storage "offline-server/ws/storage"

	"github.com/gorilla/websocket"
)

func TestWebSocketSmokeKeygenSignErrorAndReconnect(t *testing.T) {
	writeTestPrivateKey(t)

	server, wsURL, runtime := startSmokeWebSocketServer(t)

	coordinator := dialSmokeClient(t, wsURL, "coordinator", RoleAdmin)
	defer coordinator.close()
	u1 := dialSmokeClient(t, wsURL, "u1", RoleOfficer)
	defer u1.close()
	u2 := dialSmokeClient(t, wsURL, "u2", RoleOfficer)
	defer u2.close()
	u3 := dialSmokeClient(t, wsURL, "u3", RoleOfficer)
	defer u3.close()

	u3Reconnect := dialSmokeClient(t, wsURL, "u3", RoleOfficer)
	defer u3Reconnect.close()
	u3 = u3Reconnect
	time.Sleep(50 * time.Millisecond)
	if _, exists := server.hub.GetClient("u3"); !exists {
		t.Fatal("reconnected client u3 was not kept in the hub")
	}
	_, reconnections, _ := server.hub.GetConnectionStats()
	if reconnections == 0 {
		t.Fatal("expected reconnect counter to increase")
	}

	participants := map[string]*smokeClient{"u1": u1, "u2": u2, "u3": u3}
	keygenParams := runSmokeKeygen(t, coordinator, participants)
	if runtime.starts[0] != "ws-smoke-keygen" {
		t.Fatalf("keygen manager was not started: %v", runtime.starts)
	}

	runSmokeSignSuccess(t, coordinator, participants, keygenParams)
	runSmokeSignFailure(t, coordinator, participants)
	runSmokeServerError(t, coordinator)

	if !contains(runtime.stops, "ws-smoke-keygen") ||
		!contains(runtime.stops, "ws-smoke-sign") ||
		!contains(runtime.stops, "ws-smoke-sign-fail") {
		t.Fatalf("expected manager sessions to be stopped, got %v", runtime.stops)
	}
}

func startSmokeWebSocketServer(t *testing.T) (*Server, string, *fakeManagerRuntime) {
	t.Helper()

	shareStore := newFakeShareStorage()
	seStore := newFakeSeStorage()
	for i, username := range []string{"u1", "u2", "u3"} {
		_ = username
		seStore.add(model.Se{
			SeID:   fmt.Sprintf("SE%02d", i+1),
			CPLC:   fmt.Sprintf("CPLC%02d", i+1),
			Status: model.SeStatusActive,
		})
	}
	offlineKeyStore := newFakeOfflineKeyStorage()
	keyGenStore := newFakeKeyGenStorage()
	signStore := newFakeSignStorage()
	sessionManager := mem_storage.NewSessionManager()
	runtime := newFakeManagerRuntime()
	handler := &MessageHandler{
		shareStorage:      shareStore,
		seStorage:         seStore,
		offlineKeyStorage: offlineKeyStore,
		keyGenStorage:     keyGenStore,
		signStorage:       signStore,
		auditStorage:      fakeAuditStorage{},
		sessionManager:    sessionManager,
		managerRuntime:    runtime,
	}
	handler.keygenHandler = NewKeyGenHandler(shareStore, seStore, offlineKeyStore, keyGenStore, fakeAuditStorage{}, sessionManager, runtime)
	handler.signHandler = NewSignHandler(shareStore, seStore, offlineKeyStore, signStore, fakeAuditStorage{}, sessionManager, runtime)
	handler.destroyHandler = NewDestroyHandler(shareStore, seStore, offlineKeyStore, fakeAuditStorage{}, fakeApprovalStorage{}, sessionManager)

	addr := freeLocalAddr(t)
	server := NewServerWithConfig(addr, ServerConfig{
		PingInterval:     200 * time.Millisecond,
		ReadTimeout:      5 * time.Second,
		WriteTimeout:     5 * time.Second,
		MessageSizeLimit: MaxMessageSize,
	})
	server.handler = handler
	server.hub = NewHub(handler)
	if err := server.Start(); err != nil {
		t.Fatalf("start websocket server: %v", err)
	}
	t.Cleanup(func() {
		if err := server.Stop(); err != nil {
			t.Fatalf("stop websocket server: %v", err)
		}
	})

	return server, "ws://" + addr + "/ws", runtime
}

func runSmokeKeygen(t *testing.T, coordinator *smokeClient, participants map[string]*smokeClient) map[string]KeyGenParamsMessage {
	t.Helper()

	coordinator.send(KeyGenRequestMessage{
		BaseMessage:     BaseMessage{Type: MsgKeyGenRequest},
		SessionKey:      "ws-smoke-keygen",
		OfflineKeyID:    "offline-key-smoke",
		RequiredSigners: 2,
		TotalParties:    3,
		Participants:    []string{"u1", "u2", "u3"},
	})

	invites := make(map[string]KeyGenInviteMessage)
	for _, username := range []string{"u1", "u2", "u3"} {
		invite := readSmokeMessage[KeyGenInviteMessage](t, participants[username])
		if invite.Type != MsgKeyGenInvite || invite.SessionKey != "ws-smoke-keygen" {
			t.Fatalf("bad keygen invite for %s: %+v", username, invite)
		}
		invites[username] = invite
	}

	for _, username := range []string{"u1", "u2", "u3"} {
		invite := invites[username]
		participants[username].send(KeyGenResponseMessage{
			BaseMessage: BaseMessage{Type: MsgKeyGenResponse},
			SessionKey:  invite.SessionKey,
			PartyIndex:  invite.PartyIndex,
			CPLC:        fmt.Sprintf("CPLC%02d", invite.PartyIndex),
			Accept:      true,
		})
	}

	paramsByUser := make(map[string]KeyGenParamsMessage)
	for _, username := range []string{"u1", "u2", "u3"} {
		params := readSmokeMessage[KeyGenParamsMessage](t, participants[username])
		if params.Type != MsgKeyGenParams || params.ManagerAddr == "" || params.Room == "" || len(params.RecordID) != 64 {
			t.Fatalf("bad keygen params for %s: %+v", username, params)
		}
		paramsByUser[username] = params
	}

	for _, username := range []string{"u1", "u2", "u3"} {
		params := paramsByUser[username]
		participants[username].send(KeyGenResultMessage{
			BaseMessage:    BaseMessage{Type: MsgKeyGenResult},
			SessionKey:     params.SessionKey,
			PartyIndex:     params.PartyIndex,
			Address:        testAddress,
			PublicKey:      "public-key-smoke",
			CPLC:           fmt.Sprintf("CPLC%02d", params.PartyIndex),
			RecordID:       params.RecordID,
			EncryptedShard: "encrypted-share-" + username,
			Success:        true,
			Message:        "ok",
		})
	}

	complete := readSmokeMessage[KeyGenCompleteMessage](t, coordinator)
	if complete.Type != MsgKeyGenComplete || !complete.Success || complete.Address != testAddress {
		t.Fatalf("bad keygen complete: %+v", complete)
	}
	return paramsByUser
}

func runSmokeSignSuccess(t *testing.T, coordinator *smokeClient, participants map[string]*smokeClient, _ map[string]KeyGenParamsMessage) {
	t.Helper()

	coordinator.send(SignRequestMessage{
		BaseMessage:   BaseMessage{Type: MsgSignRequest},
		SessionKey:    "ws-smoke-sign",
		OfflineKeyID:  "offline-key-smoke",
		MessageHash:   strings.Repeat("0", 63) + "1",
		Address:       testAddress,
		Participants:  []string{"u1", "u3"},
		TransactionNo: "tx-smoke",
	})

	inviteU1 := readSmokeMessage[SignInviteMessage](t, participants["u1"])
	inviteU3 := readSmokeMessage[SignInviteMessage](t, participants["u3"])
	if inviteU1.Type != MsgSignInvite || inviteU1.PartyIndex != 1 || inviteU3.PartyIndex != 3 {
		t.Fatalf("bad sign invites: u1=%+v u3=%+v", inviteU1, inviteU3)
	}

	participants["u1"].send(SignResponseMessage{BaseMessage: BaseMessage{Type: MsgSignResponse}, SessionKey: inviteU1.SessionKey, PartyIndex: inviteU1.PartyIndex, CPLC: "CPLC01", Accept: true})
	participants["u3"].send(SignResponseMessage{BaseMessage: BaseMessage{Type: MsgSignResponse}, SessionKey: inviteU3.SessionKey, PartyIndex: inviteU3.PartyIndex, CPLC: "CPLC03", Accept: true})

	paramsU1 := readSmokeMessage[SignParamsMessage](t, participants["u1"])
	paramsU3 := readSmokeMessage[SignParamsMessage](t, participants["u3"])
	if paramsU1.Type != MsgSignParams || paramsU1.Parties != "1,3" || paramsU1.SigningIndex != 1 || paramsU1.Signature == "" {
		t.Fatalf("bad sign params u1: %+v", paramsU1)
	}
	if paramsU3.Type != MsgSignParams || paramsU3.Parties != "1,3" || paramsU3.SigningIndex != 2 || paramsU3.Signature == "" {
		t.Fatalf("bad sign params u3: %+v", paramsU3)
	}

	participants["u1"].send(SignResultMessage{BaseMessage: BaseMessage{Type: MsgSignResult}, SessionKey: paramsU1.SessionKey, SigningIndex: paramsU1.SigningIndex, Success: true, Signature: "0xsmokesig", Message: "ok"})
	participants["u3"].send(SignResultMessage{BaseMessage: BaseMessage{Type: MsgSignResult}, SessionKey: paramsU3.SessionKey, SigningIndex: paramsU3.SigningIndex, Success: true, Signature: "0xsmokesig", Message: "ok"})

	complete := readSmokeMessage[SignCompleteMessage](t, coordinator)
	if complete.Type != MsgSignComplete || !complete.Success || complete.Signature != "0xsmokesig" {
		t.Fatalf("bad sign complete: %+v", complete)
	}
}

func runSmokeSignFailure(t *testing.T, coordinator *smokeClient, participants map[string]*smokeClient) {
	t.Helper()

	coordinator.send(SignRequestMessage{
		BaseMessage:  BaseMessage{Type: MsgSignRequest},
		SessionKey:   "ws-smoke-sign-fail",
		OfflineKeyID: "offline-key-smoke",
		MessageHash:  strings.Repeat("0", 63) + "2",
		Address:      testAddress,
		Participants: []string{"u1", "u2"},
	})

	inviteU1 := readSmokeMessage[SignInviteMessage](t, participants["u1"])
	inviteU2 := readSmokeMessage[SignInviteMessage](t, participants["u2"])
	participants["u1"].send(SignResponseMessage{BaseMessage: BaseMessage{Type: MsgSignResponse}, SessionKey: inviteU1.SessionKey, PartyIndex: inviteU1.PartyIndex, CPLC: "CPLC01", Accept: true})
	participants["u2"].send(SignResponseMessage{BaseMessage: BaseMessage{Type: MsgSignResponse}, SessionKey: inviteU2.SessionKey, PartyIndex: inviteU2.PartyIndex, CPLC: "CPLC02", Accept: true})
	paramsU1 := readSmokeMessage[SignParamsMessage](t, participants["u1"])
	_ = readSmokeMessage[SignParamsMessage](t, participants["u2"])

	participants["u1"].send(SignResultMessage{
		BaseMessage:  BaseMessage{Type: MsgSignResult},
		SessionKey:   paramsU1.SessionKey,
		SigningIndex: paramsU1.SigningIndex,
		Success:      false,
		Signature:    "",
		Message:      "desktop signing failed",
	})

	errMsg := readSmokeMessage[ErrorMessage](t, coordinator)
	if errMsg.Type != MsgError || !strings.Contains(errMsg.Message, "签名失败") || errMsg.Details != "desktop signing failed" {
		t.Fatalf("bad sign failure error: %+v", errMsg)
	}
}

func runSmokeServerError(t *testing.T, coordinator *smokeClient) {
	t.Helper()

	coordinator.send(map[string]string{"type": "unsupported_smoke_message"})
	errMsg := readSmokeMessage[ErrorMessage](t, coordinator)
	if errMsg.Type != MsgError || !strings.Contains(errMsg.Message, "不支持的消息类型") {
		t.Fatalf("bad unsupported-message error: %+v", errMsg)
	}
}

type smokeClient struct {
	username string
	conn     *websocket.Conn
}

func dialSmokeClient(t *testing.T, wsURL, username string, role ClientRole) *smokeClient {
	t.Helper()

	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial %s: %v", username, err)
	}
	client := &smokeClient{username: username, conn: conn}

	token, err := tools.GenerateToken(username, string(role), time.Hour)
	if err != nil {
		t.Fatalf("generate token for %s: %v", username, err)
	}
	client.send(RegisterMessage{
		BaseMessage: BaseMessage{Type: MsgRegister},
		Username:    username,
		Role:        role,
		Token:       token,
	})
	ack := readSmokeMessage[RegisterCompleteMessage](t, client)
	if ack.Type != MsgRegisterComplete || !ack.Success {
		t.Fatalf("bad register ack for %s: %+v", username, ack)
	}
	return client
}

func (c *smokeClient) send(msg any) {
	if err := c.conn.WriteJSON(msg); err != nil {
		panic(fmt.Sprintf("send from %s: %v", c.username, err))
	}
}

func (c *smokeClient) close() {
	_ = c.conn.Close()
}

func readSmokeMessage[T any](t *testing.T, client *smokeClient) T {
	t.Helper()

	if err := client.conn.SetReadDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatalf("set read deadline for %s: %v", client.username, err)
	}
	_, raw, err := client.conn.ReadMessage()
	if err != nil {
		t.Fatalf("read message for %s: %v", client.username, err)
	}
	var msg T
	if err := json.Unmarshal(raw, &msg); err != nil {
		t.Fatalf("unmarshal message for %s: %v\nraw=%s", client.username, err, string(raw))
	}
	return msg
}

func freeLocalAddr(t *testing.T) string {
	t.Helper()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("allocate tcp port: %v", err)
	}
	addr := listener.Addr().String()
	if err := listener.Close(); err != nil {
		t.Fatalf("close temp listener: %v", err)
	}
	return addr
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

var _ manager.SessionRuntime = (*fakeManagerRuntime)(nil)
