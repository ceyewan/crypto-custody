package handler

import (
	"bytes"
	"encoding/json"
	"testing"

	"offline-server/storage/model"
)

func TestHashPayloadUsesCanonicalJSON(t *testing.T) {
	hashA, err := hashPayload(json.RawMessage(`{ "case_no": "CASE-1", "n": 2 }`))
	if err != nil {
		t.Fatalf("hashPayload failed: %v", err)
	}
	hashB, err := hashPayload(json.RawMessage(`{"n":2,"case_no":"CASE-1"}`))
	if err != nil {
		t.Fatalf("hashPayload failed: %v", err)
	}
	if hashA != hashB {
		t.Fatalf("hash mismatch: %s != %s", hashA, hashB)
	}
}

func TestValidateOfflineTaskPackageRejectsWrongDirectionAndOldShape(t *testing.T) {
	validPayload := json.RawMessage(`{"case_no":"CASE-1"}`)
	hash, err := hashPayload(validPayload)
	if err != nil {
		t.Fatal(err)
	}
	pkg := offlinePackage{
		SchemaVersion: "1.0",
		PackageType:   "offline_task",
		TaskType:      "custody_keygen",
		TaskNo:        "TASK-1",
		SourceSystem:  "online",
		TargetSystem:  "offline",
		Payload:       validPayload,
		PayloadHash:   hash,
	}
	if err := validateOfflineTaskPackage(pkg); err != nil {
		t.Fatalf("valid package rejected: %v", err)
	}

	pkg.PackageType = "offline_result"
	if err := validateOfflineTaskPackage(pkg); err == nil {
		t.Fatal("offline_result should not be accepted by task import")
	}

	pkg.PackageType = "offline_task"
	pkg.TaskType = "legacy_keygen"
	if err := validateOfflineTaskPackage(pkg); err == nil {
		t.Fatal("legacy task_type should be rejected")
	}
}

func TestBuildResultPackageDoesNotLeakOfflineOnlyFields(t *testing.T) {
	result, payloadHash, err := buildResultPackage(
		offlinePackage{TaskNo: "TASK-1"},
		"custody_keygen_result",
		map[string]any{
			"case_no":         "CASE-1",
			"custody_address": "0x1111111111111111111111111111111111111111",
			"offline_ref_no":  "OFFKEY-TASK-1",
		},
		"coordinator",
	)
	if err != nil {
		t.Fatalf("buildResultPackage failed: %v", err)
	}
	if payloadHash == "" || result["package_type"] != "offline_result" || result["target_system"] != "online" {
		t.Fatalf("bad result package: %+v hash=%s", result, payloadHash)
	}
	payload := result["payload"].(map[string]any)
	for _, forbidden := range []string{"encrypted_shard", "cplc", "participants"} {
		if _, ok := payload[forbidden]; ok {
			t.Fatalf("result payload leaked %s: %+v", forbidden, payload)
		}
	}
}

func TestAuditorRoleIsValid(t *testing.T) {
	if !isValidRole("auditor") {
		t.Fatal("auditor role should be accepted")
	}
}

func TestOfflineKeyDTOShardSummaryDoesNotLeakEncryptedBlob(t *testing.T) {
	key := &model.OfflineKey{
		OfflineKeyID: "OFFKEY-1",
		Address:      "0x1111111111111111111111111111111111111111",
		CoinType:     "ETH",
		Algorithm:    model.AlgorithmGG20ECDSASECP256K1,
		Status:       model.OfflineKeyStatusActive,
	}
	dto := offlineKeyDTO(key, []model.KeyShard{
		{
			ShardID:       "OFFKEY-1:1",
			Username:      "u1",
			ShardIndex:    1,
			RecordID:      "record",
			SeCPLC:        "cplc",
			EncryptedBlob: "super-secret-ciphertext",
			BlobType:      model.BlobTypeMPCShare,
			KeyVersion:    1,
			Status:        model.KeyShardStatusActive,
		},
	})
	raw, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(raw, []byte("super-secret-ciphertext")) {
		t.Fatalf("encrypted blob leaked in response: %s", raw)
	}
	if len(dto.Shards) != 1 || dto.Shards[0].EncryptedBlobSize == 0 || dto.Shards[0].EncryptedBlobSHA256 == "" {
		t.Fatalf("missing shard summary: %+v", dto.Shards)
	}
}
