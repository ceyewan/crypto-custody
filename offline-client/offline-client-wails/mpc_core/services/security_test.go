package services

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestParseRecordIDRequires32ByteHex(t *testing.T) {
	recordID := hex.EncodeToString([]byte("12345678901234567890123456789012"))

	parsed, err := parseRecordID(recordID)
	if err != nil {
		t.Fatalf("parseRecordID failed: %v", err)
	}
	if len(parsed) != 32 {
		t.Fatalf("parsed length = %d", len(parsed))
	}

	if _, err := parseRecordID("0x" + recordID); err != nil {
		t.Fatalf("parseRecordID with 0x failed: %v", err)
	}

	badValues := []string{
		"",
		"not-hex",
		strings.Repeat("0", 62),
		strings.Repeat("0", 66),
	}
	for _, value := range badValues {
		if _, err := parseRecordID(value); err == nil {
			t.Fatalf("parseRecordID(%q) should fail", value)
		}
	}
}

func TestParseAddressRequires20ByteHex(t *testing.T) {
	address := "0x1111111111111111111111111111111111111111"

	parsed, err := parseAddress(address)
	if err != nil {
		t.Fatalf("parseAddress failed: %v", err)
	}
	if len(parsed) != 20 {
		t.Fatalf("parsed length = %d", len(parsed))
	}

	if _, err := parseAddress(strings.Repeat("1", 40)); err != nil {
		t.Fatalf("parseAddress without 0x failed: %v", err)
	}

	for _, value := range []string{"", "0x1234", "not-hex"} {
		if _, err := parseAddress(value); err == nil {
			t.Fatalf("parseAddress(%q) should fail", value)
		}
	}
}
