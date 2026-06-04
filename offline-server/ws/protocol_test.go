package ws

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"offline-server/manager"
	"offline-server/storage"
	"offline-server/storage/model"
	mem_storage "offline-server/ws/storage"
)

const testAddress = "0x1111111111111111111111111111111111111111"

func TestKeyGenProtocolUsesServerOwnedManagerAndRecordIDs(t *testing.T) {
	participants := []string{"u1", "u2", "u3"}
	seStore := newFakeSeStorage()
	for i, username := range participants {
		_ = username
		seStore.add(model.Se{
			SeID:   "SE0" + string(rune('1'+i)),
			CPLC:   "CPLC0" + string(rune('1'+i)),
			Status: model.SeStatusActive,
		})
	}

	shareStore := newFakeShareStorage()
	offlineKeyStore := newFakeOfflineKeyStorage()
	keyGenStore := newFakeKeyGenStorage()
	runtime := newFakeManagerRuntime()
	sessionManager := mem_storage.NewSessionManager()
	handler := NewKeyGenHandler(shareStore, seStore, offlineKeyStore, keyGenStore, fakeAuditStorage{}, sessionManager, runtime)

	hub := newTestHub()
	coordinator := addTestClient(hub, "coordinator", RoleAdmin)
	clients := map[string]*Client{}
	for _, participant := range participants {
		clients[participant] = addTestClient(hub, participant, RoleOfficer)
	}

	err := handler.handleKeyGenRequest(KeyGenRequestMessage{
		SessionKey:      "kg-session",
		OfflineKeyID:    "offline-key-1",
		CoinType:        "ETH",
		RequiredSigners: 2,
		TotalParties:    3,
		Participants:    participants,
	}, coordinator)
	if err != nil {
		t.Fatalf("handleKeyGenRequest failed: %v", err)
	}
	if !reflect.DeepEqual(runtime.starts, []string{"kg-session"}) {
		t.Fatalf("manager starts = %v", runtime.starts)
	}

	for i, participant := range participants {
		invite := readMessage[KeyGenInviteMessage](t, clients[participant])
		if invite.Type != MsgKeyGenInvite || invite.PartyIndex != i+1 || invite.SeID == "" || invite.CoinType != "ETH" {
			t.Fatalf("bad invite for %s: %+v", participant, invite)
		}
	}

	for i, participant := range participants {
		err := handler.handleKeyGenResponse(KeyGenResponseMessage{
			SessionKey: "kg-session",
			PartyIndex: i + 1,
			CPLC:       "CPLC0" + string(rune('1'+i)),
			Accept:     true,
		}, clients[participant])
		if err != nil {
			t.Fatalf("handleKeyGenResponse(%s) failed: %v", participant, err)
		}
	}

	for i, participant := range participants {
		params := readMessage[KeyGenParamsMessage](t, clients[participant])
		if params.Type != MsgKeyGenParams {
			t.Fatalf("bad params type: %+v", params)
		}
		if params.ManagerAddr != "http://127.0.0.1:18000" || params.Room != "room-kg-session" {
			t.Fatalf("manager params mismatch: %+v", params)
		}
		if params.Threshold != 1 || params.TotalParties != 3 || params.PartyIndex != i+1 {
			t.Fatalf("gg20 params mismatch: %+v", params)
		}
		if len(params.RecordID) != 64 {
			t.Fatalf("record_id should be 32-byte hex, got %q", params.RecordID)
		}
	}

	for i, participant := range participants {
		recordID := deriveRecordID("offline-key-1", i+1, 1)
		err := handler.handleKeyGenResult(KeyGenResultMessage{
			SessionKey:     "kg-session",
			PartyIndex:     i + 1,
			Address:        testAddress,
			PublicKey:      "public-key",
			CPLC:           "CPLC0" + string(rune('1'+i)),
			RecordID:       recordID,
			EncryptedShard: "encrypted-share-" + participant,
			Success:        true,
			Message:        "ok",
		}, clients[participant])
		if err != nil {
			t.Fatalf("handleKeyGenResult(%s) failed: %v", participant, err)
		}
	}

	if len(shareStore.created) != 3 {
		t.Fatalf("created shards = %d", len(shareStore.created))
	}
	for i, shard := range shareStore.created {
		if shard.ShardIndex != i+1 || shard.RecordID != deriveRecordID("offline-key-1", i+1, 1) {
			t.Fatalf("bad shard[%d]: %+v", i, shard)
		}
	}
	if shareStore.activatedKey == nil {
		t.Fatal("offline key metadata was not created")
	}
	if shareStore.activatedKey.RequiredSigners != 2 || shareStore.activatedKey.TotalParties != 3 {
		t.Fatalf("bad offline key metadata: %+v", shareStore.activatedKey)
	}
	if shareStore.activatedKey.CoinType != "ETH" {
		t.Fatalf("offline key coin_type = %q", shareStore.activatedKey.CoinType)
	}
	for _, shard := range shareStore.shards {
		if shard.Status != model.KeyShardStatusActive {
			t.Fatalf("keygen shard should be active after final commit: %+v", shard)
		}
	}
	if !reflect.DeepEqual(runtime.stops, []string{"kg-session"}) {
		t.Fatalf("manager stops = %v", runtime.stops)
	}

	complete := readMessage[KeyGenCompleteMessage](t, coordinator)
	if complete.Type != MsgKeyGenComplete || !complete.Success || complete.Address != testAddress {
		t.Fatalf("bad completion message: %+v", complete)
	}
}

func TestKeyGenFailureMarksPendingShardsFailed(t *testing.T) {
	participants := []string{"u1", "u2", "u3"}
	seStore := newFakeSeStorage()
	for i := range participants {
		seStore.add(model.Se{
			SeID:   "SE0" + string(rune('1'+i)),
			CPLC:   "CPLC0" + string(rune('1'+i)),
			Status: model.SeStatusActive,
		})
	}

	shareStore := newFakeShareStorage()
	offlineKeyStore := newFakeOfflineKeyStorage()
	keyGenStore := newFakeKeyGenStorage()
	runtime := newFakeManagerRuntime()
	sessionManager := mem_storage.NewSessionManager()
	handler := NewKeyGenHandler(shareStore, seStore, offlineKeyStore, keyGenStore, fakeAuditStorage{}, sessionManager, runtime)

	hub := newTestHub()
	coordinator := addTestClient(hub, "coordinator", RoleAdmin)
	clients := map[string]*Client{}
	for _, participant := range participants {
		clients[participant] = addTestClient(hub, participant, RoleOfficer)
	}

	if err := handler.handleKeyGenRequest(KeyGenRequestMessage{
		SessionKey:      "kg-failed",
		OfflineKeyID:    "offline-key-failed",
		CoinType:        "ETH",
		RequiredSigners: 2,
		TotalParties:    3,
		Participants:    participants,
	}, coordinator); err != nil {
		t.Fatalf("handleKeyGenRequest failed: %v", err)
	}
	for _, participant := range participants {
		_ = readMessage[KeyGenInviteMessage](t, clients[participant])
	}
	for i, participant := range participants {
		if err := handler.handleKeyGenResponse(KeyGenResponseMessage{
			SessionKey: "kg-failed",
			PartyIndex: i + 1,
			CPLC:       "CPLC0" + string(rune('1'+i)),
			Accept:     true,
		}, clients[participant]); err != nil {
			t.Fatalf("handleKeyGenResponse(%s) failed: %v", participant, err)
		}
	}
	for _, participant := range participants {
		_ = readMessage[KeyGenParamsMessage](t, clients[participant])
	}

	for i, participant := range []string{"u1", "u2"} {
		if err := handler.handleKeyGenResult(KeyGenResultMessage{
			SessionKey:     "kg-failed",
			PartyIndex:     i + 1,
			Address:        testAddress,
			PublicKey:      "public-key",
			CPLC:           "CPLC0" + string(rune('1'+i)),
			RecordID:       deriveRecordID("offline-key-failed", i+1, 1),
			EncryptedShard: "encrypted-share-" + participant,
			Success:        true,
			Message:        "ok",
		}, clients[participant]); err != nil {
			t.Fatalf("handleKeyGenResult(%s) failed: %v", participant, err)
		}
	}
	if err := handler.handleKeyGenResult(KeyGenResultMessage{
		SessionKey: "kg-failed",
		PartyIndex: 3,
		Success:    false,
		Message:    "se store failed",
	}, clients["u3"]); err != nil {
		t.Fatalf("failed keygen result should be accepted: %v", err)
	}

	if shareStore.activatedKey != nil {
		t.Fatalf("offline key should not be created on failed keygen: %+v", shareStore.activatedKey)
	}
	for _, shard := range shareStore.shards {
		if shard.Status != model.KeyShardStatusFailed {
			t.Fatalf("pending shard should be marked failed: %+v", shard)
		}
	}
	if _, err := shareStore.GetKeyShardForParticipant("u1", testAddress); err == nil {
		t.Fatal("failed keygen shard must not be available for signing")
	}
}

func TestSignProtocolUsesOriginalShardIndexesForAllTwoOfThreeCombinations(t *testing.T) {
	writeTestPrivateKey(t)

	cases := []struct {
		name       string
		users      []string
		parties    string
		signingIdx map[string]int
	}{
		{name: "1-2", users: []string{"u1", "u2"}, parties: "1,2", signingIdx: map[string]int{"u1": 1, "u2": 2}},
		{name: "1-3", users: []string{"u1", "u3"}, parties: "1,3", signingIdx: map[string]int{"u1": 1, "u3": 2}},
		{name: "2-3", users: []string{"u2", "u3"}, parties: "2,3", signingIdx: map[string]int{"u2": 1, "u3": 2}},
		{name: "1-2-3", users: []string{"u1", "u2", "u3"}, parties: "1,2,3", signingIdx: map[string]int{"u1": 1, "u2": 2, "u3": 3}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			shareStore := newFakeShareStorage()
			seStore := newFakeSeStorage()
			for i, username := range []string{"u1", "u2", "u3"} {
				shardIndex := i + 1
				cplc := "CPLC0" + string(rune('1'+i))
				recordID := hex.EncodeToString([]byte{
					byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
					byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
					byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
					byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
					byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
					byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
					byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
					byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
				})
				seStore.add(model.Se{SeID: "SE0" + string(rune('1'+i)), CPLC: cplc, Status: model.SeStatusActive})
				shareStore.shards[shardKey(username, testAddress)] = model.KeyShard{
					ShardID:       "shard-" + username,
					OfflineKeyID:  "offline-key-1",
					Username:      username,
					Address:       testAddress,
					ShardIndex:    shardIndex,
					RecordID:      recordID,
					SeCPLC:        cplc,
					EncryptedBlob: "encrypted-share-" + username,
					BlobType:      model.BlobTypeMPCShare,
					KeyVersion:    1,
					Status:        model.KeyShardStatusActive,
				}
			}

			offlineKeyStore := newFakeOfflineKeyStorage()
			offlineKeyStore.byAddress[testAddress] = model.OfflineKey{
				OfflineKeyID:    "offline-key-1",
				Address:         testAddress,
				RequiredSigners: 2,
				TotalParties:    3,
				Status:          model.OfflineKeyStatusActive,
			}
			signStore := newFakeSignStorage()
			runtime := newFakeManagerRuntime()
			sessionManager := mem_storage.NewSessionManager()
			handler := NewSignHandler(shareStore, seStore, offlineKeyStore, signStore, fakeAuditStorage{}, sessionManager, runtime)

			hub := newTestHub()
			coordinator := addTestClient(hub, "coordinator", RoleAdmin)
			clients := map[string]*Client{}
			for _, username := range []string{"u1", "u2", "u3"} {
				clients[username] = addTestClient(hub, username, RoleOfficer)
			}

			err := handler.handleSignRequest(SignRequestMessage{
				SessionKey:   "sign-session-" + tc.name,
				MessageHash:  "0000000000000000000000000000000000000000000000000000000000000001",
				Address:      testAddress,
				Participants: tc.users,
			}, coordinator)
			if err != nil {
				t.Fatalf("handleSignRequest failed: %v", err)
			}

			for _, username := range tc.users {
				invite := readMessage[SignInviteMessage](t, clients[username])
				if invite.PartyIndex != shareStore.shards[shardKey(username, testAddress)].ShardIndex {
					t.Fatalf("bad invite for %s: %+v", username, invite)
				}
			}

			for _, username := range tc.users {
				shard := shareStore.shards[shardKey(username, testAddress)]
				err := handler.handleSignResponse(SignResponseMessage{
					SessionKey: "sign-session-" + tc.name,
					PartyIndex: shard.ShardIndex,
					CPLC:       shard.SeCPLC,
					Accept:     true,
				}, clients[username])
				if err != nil {
					t.Fatalf("handleSignResponse(%s) failed: %v", username, err)
				}
			}

			for _, username := range tc.users {
				params := readMessage[SignParamsMessage](t, clients[username])
				if params.Parties != tc.parties {
					t.Fatalf("parties for %s = %q, want %q", username, params.Parties, tc.parties)
				}
				if params.SigningIndex != tc.signingIdx[username] {
					t.Fatalf("signing_index for %s = %d, want %d", username, params.SigningIndex, tc.signingIdx[username])
				}
				if params.PartyIndex != shareStore.shards[shardKey(username, testAddress)].ShardIndex {
					t.Fatalf("party_index for %s = %d", username, params.PartyIndex)
				}
				if params.ManagerAddr == "" || params.Room == "" || params.Signature == "" {
					t.Fatalf("missing sign params for %s: %+v", username, params)
				}
			}

			for _, username := range tc.users {
				err := handler.handleSignResult(SignResultMessage{
					SessionKey:   "sign-session-" + tc.name,
					SigningIndex: tc.signingIdx[username],
					Success:      true,
					Signature:    "0xsig",
					Message:      "ok",
				}, clients[username])
				if err != nil {
					t.Fatalf("handleSignResult(%s) failed: %v", username, err)
				}
			}
			if !reflect.DeepEqual(runtime.stops, []string{"sign-session-" + tc.name}) {
				t.Fatalf("manager stops = %v", runtime.stops)
			}
		})
	}
}

func TestSignProtocolRejectsMismatchedFinalSignatures(t *testing.T) {
	shareStore := newFakeShareStorage()
	seStore := newFakeSeStorage()
	offlineKeyStore := newFakeOfflineKeyStorage()
	signStore := newFakeSignStorage()
	runtime := newFakeManagerRuntime()
	sessionManager := mem_storage.NewSessionManager()
	handler := NewSignHandler(shareStore, seStore, offlineKeyStore, signStore, fakeAuditStorage{}, sessionManager, runtime)

	session := model.SignSession{
		SessionKey:   "sign-mismatch",
		Initiator:    "coordinator",
		Address:      testAddress,
		MessageHash:  "0000000000000000000000000000000000000000000000000000000000000001",
		Participants: model.StringSlice{"u1", "u2"},
		Responses:    model.StringSlice{string(model.ParticipantAccepted), string(model.ParticipantAccepted)},
		Status:       model.StatusProcessing,
	}
	if _, err := signStore.CreateSession(session); err != nil {
		t.Fatal(err)
	}
	if _, err := sessionManager.CreateSignSession(session); err != nil {
		t.Fatal(err)
	}

	hub := newTestHub()
	coordinator := addTestClient(hub, "coordinator", RoleAdmin)
	u1 := addTestClient(hub, "u1", RoleOfficer)
	u2 := addTestClient(hub, "u2", RoleOfficer)
	_ = coordinator

	if err := handler.handleSignResult(SignResultMessage{
		SessionKey:   "sign-mismatch",
		SigningIndex: 1,
		Success:      true,
		Signature:    "0xsig-a",
		Message:      "ok",
	}, u1); err != nil {
		t.Fatalf("first sign result failed: %v", err)
	}

	err := handler.handleSignResult(SignResultMessage{
		SessionKey:   "sign-mismatch",
		SigningIndex: 2,
		Success:      true,
		Signature:    "0xsig-b",
		Message:      "ok",
	}, u2)
	if err == nil {
		t.Fatal("mismatched signature should fail")
	}

	cached := sessionManager.GetSignSession("sign-mismatch")
	if cached == nil || cached.Status != model.StatusFailed {
		t.Fatalf("session should be failed: %+v", cached)
	}
	if !reflect.DeepEqual(runtime.stops, []string{"sign-mismatch"}) {
		t.Fatalf("manager stops = %v", runtime.stops)
	}
	failure := readMessage[ErrorMessage](t, coordinator)
	if failure.Type != MsgError || failure.Message != "签名结果不一致" {
		t.Fatalf("bad failure message: %+v", failure)
	}
}

func TestDestroyProtocolDeletesAllActiveShardsBeforeMarkingKeyDestroyed(t *testing.T) {
	writeTestPrivateKey(t)

	shareStore := newFakeShareStorage()
	seStore := newFakeSeStorage()
	for i, username := range []string{"u1", "u2", "u3"} {
		shardIndex := i + 1
		cplc := "CPLC0" + string(rune('1'+i))
		recordID := hex.EncodeToString([]byte{
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
		})
		seStore.add(model.Se{SeID: "SE0" + string(rune('1'+i)), CPLC: cplc, Status: model.SeStatusActive})
		shareStore.shards[shardKey(username, testAddress)] = model.KeyShard{
			ShardID:       "shard-" + username,
			OfflineKeyID:  "offline-key-1",
			Username:      username,
			Address:       testAddress,
			ShardIndex:    shardIndex,
			RecordID:      recordID,
			SeCPLC:        cplc,
			EncryptedBlob: "encrypted-share-" + username,
			BlobType:      model.BlobTypeMPCShare,
			KeyVersion:    1,
			Status:        model.KeyShardStatusActive,
		}
	}

	offlineKeyStore := newFakeOfflineKeyStorage()
	offlineKeyStore.byAddress[testAddress] = model.OfflineKey{
		OfflineKeyID: "offline-key-1",
		Address:      testAddress,
		Status:       model.OfflineKeyStatusActive,
	}

	sessionManager := mem_storage.NewSessionManager()
	handler := NewDestroyHandler(shareStore, seStore, offlineKeyStore, fakeAuditStorage{}, fakeApprovalStorage{}, sessionManager)

	hub := newTestHub()
	admin := addTestClient(hub, "admin", RoleAdmin)
	clients := map[string]*Client{}
	for _, username := range []string{"u1", "u2", "u3"} {
		clients[username] = addTestClient(hub, username, RoleOfficer)
	}

	if err := handler.handleDestroyRequest(DestroyRequestMessage{
		SessionKey:   "destroy-session",
		OfflineKeyID: "offline-key-1",
		Reason:       "test destroy",
	}, admin); err != nil {
		t.Fatalf("handleDestroyRequest failed: %v", err)
	}

	for i, username := range []string{"u1", "u2", "u3"} {
		invite := readMessage[DestroyInviteMessage](t, clients[username])
		if invite.Type != MsgDestroyInvite || invite.PartyIndex != i+1 || invite.SeID != "SE0"+string(rune('1'+i)) {
			t.Fatalf("bad destroy invite for %s: %+v", username, invite)
		}
	}

	for i, username := range []string{"u1", "u2", "u3"} {
		err := handler.handleDestroyResponse(DestroyResponseMessage{
			SessionKey: "destroy-session",
			PartyIndex: i + 1,
			CPLC:       "CPLC0" + string(rune('1'+i)),
			Accept:     true,
		}, clients[username])
		if err != nil {
			t.Fatalf("handleDestroyResponse(%s) failed: %v", username, err)
		}
	}

	for i, username := range []string{"u1", "u2", "u3"} {
		params := readMessage[DestroyParamsMessage](t, clients[username])
		if params.Type != MsgDestroyParams || params.PartyIndex != i+1 || params.RecordID == "" || params.Signature == "" {
			t.Fatalf("bad destroy params for %s: %+v", username, params)
		}
	}

	for i, username := range []string{"u1", "u2", "u3"} {
		err := handler.handleDestroyResult(DestroyResultMessage{
			SessionKey: "destroy-session",
			PartyIndex: i + 1,
			Success:    true,
			Message:    "ok",
		}, clients[username])
		if err != nil {
			t.Fatalf("handleDestroyResult(%s) failed: %v", username, err)
		}
	}

	key, err := offlineKeyStore.GetOfflineKeyByID("offline-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if key.Status != model.OfflineKeyStatusDestroyed {
		t.Fatalf("offline key status = %s", key.Status)
	}
	for _, shard := range shareStore.shards {
		if shard.Status != model.KeyShardStatusDestroyed {
			t.Fatalf("shard not destroyed: %+v", shard)
		}
	}

	complete := readMessage[DestroyCompleteMessage](t, admin)
	if complete.Type != MsgDestroyComplete || !complete.Success || complete.Destroyed != 3 {
		t.Fatalf("bad destroy complete message: %+v", complete)
	}
}

func TestDestroyProtocolCanRetryRemainingShardsAfterPartialFailure(t *testing.T) {
	writeTestPrivateKey(t)

	shareStore := newFakeShareStorage()
	seStore := newFakeSeStorage()
	for i, username := range []string{"u1", "u2", "u3"} {
		shardIndex := i + 1
		cplc := "CPLC0" + string(rune('1'+i))
		recordID := hex.EncodeToString([]byte{
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
			byte(shardIndex), byte(shardIndex), byte(shardIndex), byte(shardIndex),
		})
		seStore.add(model.Se{SeID: "SE0" + string(rune('1'+i)), CPLC: cplc, Status: model.SeStatusActive})
		shareStore.shards[shardKey(username, testAddress)] = model.KeyShard{
			ShardID:       "shard-" + username,
			OfflineKeyID:  "offline-key-1",
			Username:      username,
			Address:       testAddress,
			ShardIndex:    shardIndex,
			RecordID:      recordID,
			SeCPLC:        cplc,
			EncryptedBlob: "encrypted-share-" + username,
			BlobType:      model.BlobTypeMPCShare,
			KeyVersion:    1,
			Status:        model.KeyShardStatusActive,
		}
	}

	offlineKeyStore := newFakeOfflineKeyStorage()
	offlineKeyStore.byAddress[testAddress] = model.OfflineKey{
		OfflineKeyID: "offline-key-1",
		Address:      testAddress,
		Status:       model.OfflineKeyStatusActive,
	}

	sessionManager := mem_storage.NewSessionManager()
	handler := NewDestroyHandler(shareStore, seStore, offlineKeyStore, fakeAuditStorage{}, fakeApprovalStorage{}, sessionManager)

	hub := newTestHub()
	admin := addTestClient(hub, "admin", RoleAdmin)
	clients := map[string]*Client{}
	for _, username := range []string{"u1", "u2", "u3"} {
		clients[username] = addTestClient(hub, username, RoleOfficer)
	}

	if err := handler.handleDestroyRequest(DestroyRequestMessage{
		SessionKey:   "destroy-partial",
		OfflineKeyID: "offline-key-1",
	}, admin); err != nil {
		t.Fatalf("initial destroy request failed: %v", err)
	}
	key, err := offlineKeyStore.GetOfflineKeyByID("offline-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if key.Status != model.OfflineKeyStatusDestroying {
		t.Fatalf("offline key should be destroying after request, got %s", key.Status)
	}

	for i, username := range []string{"u1", "u2", "u3"} {
		invite := readMessage[DestroyInviteMessage](t, clients[username])
		if invite.Type != MsgDestroyInvite || invite.PartyIndex != i+1 {
			t.Fatalf("bad destroy invite for %s: %+v", username, invite)
		}
	}
	for i, username := range []string{"u1", "u2", "u3"} {
		if err := handler.handleDestroyResponse(DestroyResponseMessage{
			SessionKey: "destroy-partial",
			PartyIndex: i + 1,
			CPLC:       "CPLC0" + string(rune('1'+i)),
			Accept:     true,
		}, clients[username]); err != nil {
			t.Fatalf("destroy response %s failed: %v", username, err)
		}
	}
	for i, username := range []string{"u1", "u2", "u3"} {
		params := readMessage[DestroyParamsMessage](t, clients[username])
		if params.Type != MsgDestroyParams || params.PartyIndex != i+1 {
			t.Fatalf("bad destroy params for %s: %+v", username, params)
		}
	}
	for i, username := range []string{"u1", "u2"} {
		if err := handler.handleDestroyResult(DestroyResultMessage{
			SessionKey: "destroy-partial",
			PartyIndex: i + 1,
			Success:    true,
		}, clients[username]); err != nil {
			t.Fatalf("destroy success %s failed: %v", username, err)
		}
	}
	if err := handler.handleDestroyResult(DestroyResultMessage{
		SessionKey: "destroy-partial",
		PartyIndex: 3,
		Success:    false,
		Message:    "card unavailable",
	}, clients["u3"]); err != nil {
		t.Fatalf("destroy failure failed: %v", err)
	}
	failure := readMessage[ErrorMessage](t, admin)
	if failure.Type != MsgError || failure.Message == "" {
		t.Fatalf("bad failure message: %+v", failure)
	}
	key, err = offlineKeyStore.GetOfflineKeyByID("offline-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if key.Status != model.OfflineKeyStatusDestroyFailed {
		t.Fatalf("offline key should be destroy_failed after partial failure, got %s", key.Status)
	}
	for _, username := range []string{"u1", "u2"} {
		shard, err := shareStore.GetKeyShardByID("shard-" + username)
		if err != nil {
			t.Fatal(err)
		}
		if shard.Status != model.KeyShardStatusDestroyed {
			t.Fatalf("%s shard should be destroyed, got %s", username, shard.Status)
		}
	}
	if _, err := shareStore.GetKeyShardForParticipant("u3", testAddress); err != nil {
		t.Fatalf("u3 shard should remain active for retry: %v", err)
	}

	if err := handler.handleDestroyRequest(DestroyRequestMessage{
		SessionKey:   "destroy-retry",
		OfflineKeyID: "offline-key-1",
	}, admin); err != nil {
		t.Fatalf("retry destroy request failed: %v", err)
	}
	retry := sessionManager.GetDestroySession("destroy-retry")
	if retry == nil || !reflect.DeepEqual([]string(retry.Participants), []string{"u3"}) {
		t.Fatalf("retry should only include remaining active shard holder: %+v", retry)
	}
	invite := readMessage[DestroyInviteMessage](t, clients["u3"])
	if invite.Type != MsgDestroyInvite || invite.PartyIndex != 3 {
		t.Fatalf("bad retry invite: %+v", invite)
	}
	if err := handler.handleDestroyResponse(DestroyResponseMessage{
		SessionKey: "destroy-retry",
		PartyIndex: 3,
		CPLC:       "CPLC03",
		Accept:     true,
	}, clients["u3"]); err != nil {
		t.Fatalf("retry response failed: %v", err)
	}
	params := readMessage[DestroyParamsMessage](t, clients["u3"])
	if params.Type != MsgDestroyParams || params.PartyIndex != 3 {
		t.Fatalf("bad retry params: %+v", params)
	}
	if err := handler.handleDestroyResult(DestroyResultMessage{
		SessionKey: "destroy-retry",
		PartyIndex: 3,
		Success:    true,
	}, clients["u3"]); err != nil {
		t.Fatalf("retry result failed: %v", err)
	}
	key, err = offlineKeyStore.GetOfflineKeyByID("offline-key-1")
	if err != nil {
		t.Fatal(err)
	}
	if key.Status != model.OfflineKeyStatusDestroyed {
		t.Fatalf("offline key should be destroyed after retry, got %s", key.Status)
	}
	complete := readMessage[DestroyCompleteMessage](t, admin)
	if complete.Type != MsgDestroyComplete || !complete.Success || complete.Destroyed != 1 {
		t.Fatalf("bad retry complete message: %+v", complete)
	}
}

func TestTransferProtocolRequiresBothSidesBeforeMovingShard(t *testing.T) {
	shareStore := newFakeShareStorage()
	shareStore.shards[shardKey("u1", testAddress)] = model.KeyShard{
		ShardID:       "shard-u1",
		OfflineKeyID:  "offline-key-1",
		Username:      "u1",
		Address:       testAddress,
		ShardIndex:    2,
		RecordID:      "record-u1",
		SeCPLC:        "CPLC01",
		EncryptedBlob: "encrypted-share-u1",
		BlobType:      model.BlobTypeMPCShare,
		KeyVersion:    1,
		Status:        model.KeyShardStatusActive,
	}

	sessionManager := mem_storage.NewSessionManager()
	handler := NewTransferHandler(shareStore, fakeAuditStorage{}, fakeApprovalStorage{}, sessionManager)

	hub := newTestHub()
	admin := addTestClient(hub, "admin", RoleAdmin)
	u1 := addTestClient(hub, "u1", RoleOfficer)
	u2 := addTestClient(hub, "u2", RoleOfficer)

	if err := handler.handleTransferRequest(TransferRequestMessage{
		SessionKey:   "transfer-session",
		ShardID:      "shard-u1",
		CaseNo:       "CASE-1",
		FromUsername: "u1",
		ToUsername:   "u2",
		Reason:       "handover",
	}, admin); err != nil {
		t.Fatalf("handleTransferRequest failed: %v", err)
	}

	for username, client := range map[string]*Client{"u1": u1, "u2": u2} {
		invite := readMessage[TransferInviteMessage](t, client)
		if invite.Type != MsgTransferInvite || invite.ShardID != "shard-u1" ||
			invite.FromUsername != "u1" || invite.ToUsername != "u2" ||
			invite.CaseNo != "CASE-1" || invite.ShardIndex != 2 {
			t.Fatalf("bad transfer invite for %s: %+v", username, invite)
		}
	}

	if err := handler.handleTransferResponse(TransferResponseMessage{
		SessionKey: "transfer-session",
		ShardID:    "shard-u1",
		Accept:     true,
	}, u1); err != nil {
		t.Fatalf("first transfer response failed: %v", err)
	}
	if _, err := shareStore.GetKeyShardForParticipant("u1", testAddress); err != nil {
		t.Fatalf("shard should still belong to u1 before both confirmations: %v", err)
	}

	if err := handler.handleTransferResponse(TransferResponseMessage{
		SessionKey: "transfer-session",
		ShardID:    "shard-u1",
		Accept:     true,
	}, u2); err != nil {
		t.Fatalf("second transfer response failed: %v", err)
	}

	if _, err := shareStore.GetKeyShardForParticipant("u1", testAddress); err == nil {
		t.Fatal("u1 should no longer hold the shard after transfer")
	}
	updated, err := shareStore.GetKeyShardForParticipant("u2", testAddress)
	if err != nil {
		t.Fatalf("u2 should hold transferred shard: %v", err)
	}
	if updated.RecordID != "record-u1" || updated.SeCPLC != "CPLC01" {
		t.Fatalf("transfer should not change SE record or CPLC: %+v", updated)
	}

	for username, client := range map[string]*Client{"admin": admin, "u1": u1, "u2": u2} {
		complete := readMessage[TransferCompleteMessage](t, client)
		if complete.Type != MsgTransferComplete || !complete.Success || complete.ShardID != "shard-u1" {
			t.Fatalf("bad transfer completion for %s: %+v", username, complete)
		}
	}
}

func writeTestPrivateKey(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	if err := os.MkdirAll("private_keys", 0755); err != nil {
		t.Fatal(err)
	}
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	pemData := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: der})
	if err := os.WriteFile(filepath.Join("private_keys", "ec_private_key.pem"), pemData, 0600); err != nil {
		t.Fatal(err)
	}
}

func newTestHub() *Hub {
	return &Hub{clients: make(map[string]*Client)}
}

func addTestClient(hub *Hub, username string, role ClientRole) *Client {
	client := &Client{
		username:  username,
		role:      role,
		hub:       hub,
		writeChan: make(chan []byte, 32),
	}
	hub.clients[username] = client
	return client
}

func readMessage[T any](t *testing.T, client *Client) T {
	t.Helper()
	select {
	case raw := <-client.writeChan:
		var msg T
		if err := json.Unmarshal(raw, &msg); err != nil {
			t.Fatalf("unmarshal message: %v\nraw=%s", err, string(raw))
		}
		return msg
	default:
		t.Fatalf("no message for %s", client.username)
		var zero T
		return zero
	}
}

type fakeManagerRuntime struct {
	starts []string
	stops  []string
}

func newFakeManagerRuntime() *fakeManagerRuntime {
	return &fakeManagerRuntime{}
}

func (f *fakeManagerRuntime) StartSession(sessionKey string) (manager.Session, error) {
	f.starts = append(f.starts, sessionKey)
	return manager.Session{
		SessionKey: sessionKey,
		ManagerURL: "http://127.0.0.1:18000",
		Room:       "room-" + sessionKey,
		Port:       18000,
	}, nil
}

func (f *fakeManagerRuntime) StopSession(sessionKey string) error {
	f.stops = append(f.stops, sessionKey)
	return nil
}

func (f *fakeManagerRuntime) StopAll() error {
	return nil
}

type fakeShareStorage struct {
	mu           sync.RWMutex
	shards       map[string]model.KeyShard
	created      []model.KeyShard
	activatedKey *model.OfflineKey
	offlineKeys  *fakeOfflineKeyStorage
}

func newFakeShareStorage() *fakeShareStorage {
	return &fakeShareStorage{shards: make(map[string]model.KeyShard)}
}

func (f *fakeShareStorage) attachOfflineKeyStorage(offlineKeys *fakeOfflineKeyStorage) {
	f.offlineKeys = offlineKeys
}

func (f *fakeShareStorage) CreateKeyShard(shard model.KeyShard) (*model.KeyShard, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, shard)
	f.shards[shardKey(shard.Username, shard.Address)] = shard
	return &shard, nil
}

func (f *fakeShareStorage) GetKeyShardForParticipant(username, address string) (*model.KeyShard, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	shard, ok := f.shards[shardKey(username, address)]
	if !ok || shard.Status != model.KeyShardStatusActive {
		return nil, storage.ErrRecordNotFound
	}
	return &shard, nil
}

func (f *fakeShareStorage) GetKeyShardByID(shardID string) (*model.KeyShard, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	for _, shard := range f.shards {
		if shard.ShardID == shardID {
			return &shard, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (f *fakeShareStorage) ListActiveKeyShardsByAddress(address string) ([]model.KeyShard, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var shards []model.KeyShard
	for _, shard := range f.shards {
		if shard.Address == address && shard.Status == model.KeyShardStatusActive {
			shards = append(shards, shard)
		}
	}
	return shards, nil
}

func (f *fakeShareStorage) ListKeyShardsByAddress(address string) ([]model.KeyShard, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var shards []model.KeyShard
	for _, shard := range f.shards {
		if shard.Address == address {
			shards = append(shards, shard)
		}
	}
	return shards, nil
}

func (f *fakeShareStorage) ListKeyShardsByUsername(username string) ([]model.KeyShard, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	var shards []model.KeyShard
	for _, shard := range f.shards {
		if shard.Username == username {
			shards = append(shards, shard)
		}
	}
	return shards, nil
}

func (f *fakeShareStorage) ListKeyShards() ([]model.KeyShard, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	shards := make([]model.KeyShard, 0, len(f.shards))
	for _, shard := range f.shards {
		shards = append(shards, shard)
	}
	return shards, nil
}

func (f *fakeShareStorage) UpdateKeyShardStatus(shardID string, status model.KeyShardStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key, shard := range f.shards {
		if shard.ShardID == shardID {
			shard.Status = status
			f.shards[key] = shard
			return nil
		}
	}
	return storage.ErrRecordNotFound
}

func (f *fakeShareStorage) UpdateKeyShardsStatusByOfflineKey(offlineKeyID string, from, to model.KeyShardStatus) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key, shard := range f.shards {
		if shard.OfflineKeyID == offlineKeyID && shard.Status == from {
			shard.Status = to
			f.shards[key] = shard
		}
	}
	return nil
}

func (f *fakeShareStorage) CreateOfflineKeyAndActivatePendingShards(key model.OfflineKey, expectedShardCount int) (*model.OfflineKey, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	count := 0
	for _, shard := range f.shards {
		if shard.OfflineKeyID == key.OfflineKeyID && shard.Address == key.Address && shard.Status == model.KeyShardStatusPending {
			count++
		}
	}
	if count != expectedShardCount {
		return nil, storage.ErrOperationFailed
	}
	for mapKey, shard := range f.shards {
		if shard.OfflineKeyID == key.OfflineKeyID && shard.Address == key.Address && shard.Status == model.KeyShardStatusPending {
			shard.Status = model.KeyShardStatusActive
			f.shards[mapKey] = shard
		}
	}
	f.activatedKey = &key
	if f.offlineKeys != nil {
		if _, err := f.offlineKeys.CreateOfflineKey(key); err != nil {
			return nil, err
		}
	}
	return &key, nil
}

func (f *fakeShareStorage) TransferKeyShard(shardID, newUsername string) (*model.KeyShard, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for key, shard := range f.shards {
		if shard.ShardID == shardID {
			delete(f.shards, key)
			shard.Username = newUsername
			f.shards[shardKey(newUsername, shard.Address)] = shard
			return &shard, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func shardKey(username, address string) string {
	return username + "|" + address
}

type fakeSeStorage struct {
	byID   map[string]model.Se
	byCPLC map[string]model.Se
}

func newFakeSeStorage() *fakeSeStorage {
	return &fakeSeStorage{byID: make(map[string]model.Se), byCPLC: make(map[string]model.Se)}
}

func (f *fakeSeStorage) add(se model.Se) {
	f.byID[se.SeID] = se
	f.byCPLC[se.CPLC] = se
}

func (f *fakeSeStorage) CreateSe(seID, cplc, custodyLocation, registeredBy string) (*model.Se, error) {
	se := model.Se{SeID: seID, CPLC: cplc, CustodyLocation: custodyLocation, RegisteredBy: registeredBy, Status: model.SeStatusActive}
	f.add(se)
	return &se, nil
}

func (f *fakeSeStorage) GetSeBySeId(seID string) (*model.Se, error) {
	se, ok := f.byID[seID]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &se, nil
}

func (f *fakeSeStorage) GetSeByCPLC(cplc string) (*model.Se, error) {
	se, ok := f.byCPLC[cplc]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &se, nil
}

func (f *fakeSeStorage) GetAllSe() ([]model.Se, error) {
	ses := make([]model.Se, 0, len(f.byID))
	for _, se := range f.byID {
		ses = append(ses, se)
	}
	return ses, nil
}

func (f *fakeSeStorage) GetActiveSeIds(n int) ([]string, error) {
	ids := make([]string, 0, n)
	for _, seID := range []string{"SE01", "SE02", "SE03", "SE04", "SE05"} {
		se, ok := f.byID[seID]
		if ok && se.Status == model.SeStatusActive {
			ids = append(ids, seID)
			if len(ids) == n {
				return ids, nil
			}
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (f *fakeSeStorage) UpdateSeStatus(seID string, status model.SeStatus) error {
	se, ok := f.byID[seID]
	if !ok {
		return storage.ErrRecordNotFound
	}
	se.Status = status
	f.add(se)
	return nil
}

func (f *fakeSeStorage) TouchSeLastUsedByCPLC(cplc string) error {
	se, ok := f.byCPLC[cplc]
	if !ok {
		return storage.ErrRecordNotFound
	}
	now := time.Now()
	se.LastUsedAt = &now
	f.add(se)
	return nil
}

func (f *fakeSeStorage) DeleteSe(seID string) error {
	se, ok := f.byID[seID]
	if !ok {
		return storage.ErrRecordNotFound
	}
	delete(f.byID, seID)
	delete(f.byCPLC, se.CPLC)
	return nil
}

type fakeOfflineKeyStorage struct {
	byAddress map[string]model.OfflineKey
	created   *model.OfflineKey
}

func newFakeOfflineKeyStorage() *fakeOfflineKeyStorage {
	return &fakeOfflineKeyStorage{byAddress: make(map[string]model.OfflineKey)}
}

func (f *fakeOfflineKeyStorage) CreateOfflineKey(key model.OfflineKey) (*model.OfflineKey, error) {
	f.created = &key
	f.byAddress[key.Address] = key
	return &key, nil
}

func (f *fakeOfflineKeyStorage) GetOfflineKeyByID(offlineKeyID string) (*model.OfflineKey, error) {
	for _, key := range f.byAddress {
		if key.OfflineKeyID == offlineKeyID {
			return &key, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (f *fakeOfflineKeyStorage) GetOfflineKeyByAddress(address string) (*model.OfflineKey, error) {
	key, ok := f.byAddress[address]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &key, nil
}

func (f *fakeOfflineKeyStorage) GetOfflineKeyByTaskNo(taskNo string) (*model.OfflineKey, error) {
	for _, key := range f.byAddress {
		if key.TaskNo == taskNo {
			return &key, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (f *fakeOfflineKeyStorage) ListOfflineKeys() ([]model.OfflineKey, error) {
	keys := make([]model.OfflineKey, 0, len(f.byAddress))
	for _, key := range f.byAddress {
		keys = append(keys, key)
	}
	return keys, nil
}

func (f *fakeOfflineKeyStorage) UpdateOfflineKeyOwner(offlineKeyID, logicalOwner string) error {
	for address, key := range f.byAddress {
		if key.OfflineKeyID == offlineKeyID {
			key.LogicalOwner = logicalOwner
			f.byAddress[address] = key
			return nil
		}
	}
	return storage.ErrRecordNotFound
}

func (f *fakeOfflineKeyStorage) UpdateOfflineKeyStatus(offlineKeyID string, status model.OfflineKeyStatus) error {
	for address, key := range f.byAddress {
		if key.OfflineKeyID == offlineKeyID {
			key.Status = status
			f.byAddress[address] = key
			return nil
		}
	}
	return storage.ErrRecordNotFound
}

type fakeKeyGenStorage struct {
	sessions map[string]model.KeyGenSession
}

func newFakeKeyGenStorage() *fakeKeyGenStorage {
	return &fakeKeyGenStorage{sessions: make(map[string]model.KeyGenSession)}
}

func (f *fakeKeyGenStorage) CreateSession(session model.KeyGenSession) (*model.KeyGenSession, error) {
	f.sessions[session.SessionKey] = session
	return &session, nil
}

func (f *fakeKeyGenStorage) GetSession(sessionKey string) (*model.KeyGenSession, error) {
	session, ok := f.sessions[sessionKey]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &session, nil
}

func (f *fakeKeyGenStorage) GetSessionByAccountAddr(accountAddr string) (*model.KeyGenSession, error) {
	for _, session := range f.sessions {
		if session.AccountAddr == accountAddr {
			return &session, nil
		}
	}
	return nil, storage.ErrRecordNotFound
}

func (f *fakeKeyGenStorage) UpdateStatus(sessionKey string, status model.SessionStatus) error {
	session := f.sessions[sessionKey]
	session.Status = status
	f.sessions[sessionKey] = session
	return nil
}

func (f *fakeKeyGenStorage) UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error {
	session := f.sessions[sessionKey]
	if len(session.Responses) > index {
		session.Responses[index] = string(status)
	}
	f.sessions[sessionKey] = session
	return nil
}

func (f *fakeKeyGenStorage) UpdateAccountAddr(sessionKey, accountAddr string) error {
	session := f.sessions[sessionKey]
	session.AccountAddr = accountAddr
	f.sessions[sessionKey] = session
	return nil
}

func (f *fakeKeyGenStorage) DeleteSession(sessionKey string) error {
	delete(f.sessions, sessionKey)
	return nil
}

func (f *fakeKeyGenStorage) UpdateSeIDs(sessionKey string, seIDs []string) error {
	session := f.sessions[sessionKey]
	session.SeIDs = model.StringSlice(seIDs)
	f.sessions[sessionKey] = session
	return nil
}

func (f *fakeKeyGenStorage) AllKeyGenInvitationsAccepted(sessionKey string) bool {
	return true
}

func (f *fakeKeyGenStorage) AllKeyGenPartsCompleted(sessionKey string) bool {
	return true
}

type fakeSignStorage struct {
	sessions map[string]model.SignSession
}

func newFakeSignStorage() *fakeSignStorage {
	return &fakeSignStorage{sessions: make(map[string]model.SignSession)}
}

func (f *fakeSignStorage) CreateSession(session model.SignSession) (*model.SignSession, error) {
	f.sessions[session.SessionKey] = session
	return &session, nil
}

func (f *fakeSignStorage) GetSession(sessionKey string) (*model.SignSession, error) {
	session, ok := f.sessions[sessionKey]
	if !ok {
		return nil, storage.ErrRecordNotFound
	}
	return &session, nil
}

func (f *fakeSignStorage) GetSessionByTaskNo(taskNo string) (*model.SignSession, error) {
	for _, session := range f.sessions {
		if session.TaskNo == taskNo {
			return &session, nil
		}
	}
	return nil, storage.ErrSessionNotFound
}

func (f *fakeSignStorage) UpdateStatus(sessionKey string, status model.SessionStatus) error {
	session := f.sessions[sessionKey]
	session.Status = status
	f.sessions[sessionKey] = session
	return nil
}

func (f *fakeSignStorage) UpdateParticipantStatus(sessionKey string, index int, status model.ParticipantStatus) error {
	session := f.sessions[sessionKey]
	if len(session.Responses) > index {
		session.Responses[index] = string(status)
	}
	f.sessions[sessionKey] = session
	return nil
}

func (f *fakeSignStorage) UpdateSignature(sessionKey, signature string) error {
	session := f.sessions[sessionKey]
	session.Signature = signature
	session.Status = model.StatusCompleted
	f.sessions[sessionKey] = session
	return nil
}

func (f *fakeSignStorage) DeleteSession(sessionKey string) error {
	delete(f.sessions, sessionKey)
	return nil
}

func (f *fakeSignStorage) UpdateSeIDs(sessionKey string, seIDs []string) error {
	session := f.sessions[sessionKey]
	session.SeIDs = model.StringSlice(seIDs)
	f.sessions[sessionKey] = session
	return nil
}

func (f *fakeSignStorage) AllKeyGenInvitationsAccepted(sessionKey string) bool {
	return true
}

func (f *fakeSignStorage) AllKeyGenPartsCompleted(sessionKey string) bool {
	return true
}

type fakeAuditStorage struct{}

func (fakeAuditStorage) CreateAuditLog(log model.AuditLog) error {
	return nil
}

func (fakeAuditStorage) ListAuditLogs(limit int) ([]model.AuditLog, error) {
	return nil, nil
}

func (fakeAuditStorage) SearchAuditLogs(filter storage.AuditLogFilter) ([]model.AuditLog, int64, error) {
	return nil, 0, nil
}

type fakeApprovalStorage struct{}

func (fakeApprovalStorage) CreateApproval(approval model.Approval) (*model.Approval, error) {
	return &approval, nil
}

func (fakeApprovalStorage) ListApprovals(limit int) ([]model.Approval, error) {
	return nil, nil
}

func (fakeApprovalStorage) ListApprovalsPage(page, pageSize int) ([]model.Approval, int64, error) {
	return nil, 0, nil
}
