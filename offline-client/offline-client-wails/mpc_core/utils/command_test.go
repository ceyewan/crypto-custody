package utils

import (
	"reflect"
	"testing"

	"offline-client-wails/mpc_core/config"
)

func TestBuildKeygenArgsUsesSessionManagerAndRoom(t *testing.T) {
	cfg := &config.Config{ManagerAddr: "http://fallback:8000"}

	args := buildKeygenArgs(cfg, "http://offline-server:18001", "room-1", 1, 3, 2, "/tmp/keygen.json")

	want := []string{
		"--address", "http://offline-server:18001",
		"--threshold", "1",
		"--number-of-parties", "3",
		"--index", "2",
		"--output", "/tmp/keygen.json",
		"--room", "room-1",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestBuildSigningArgsUsesSigningIndexAndOriginalParties(t *testing.T) {
	cfg := &config.Config{ManagerAddr: "http://fallback:8000"}

	args := buildSigningArgs(cfg, "http://offline-server:18002", "room-2", 2, "1,3", "00ff", "/tmp/share.json")

	want := []string{
		"--address", "http://offline-server:18002",
		"--index", "2",
		"--parties", "1,3",
		"--data-to-sign", "00ff",
		"--local-share", "/tmp/share.json",
		"--room", "room-2",
	}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("args = %#v, want %#v", args, want)
	}
}

func TestBuildArgsFallbackToConfiguredManager(t *testing.T) {
	cfg := &config.Config{ManagerAddr: "http://configured:8000"}

	keygenArgs := buildKeygenArgs(cfg, "", "", 1, 2, 1, "/tmp/keygen.json")
	if keygenArgs[1] != "http://configured:8000" {
		t.Fatalf("keygen manager addr = %s", keygenArgs[1])
	}

	signArgs := buildSigningArgs(cfg, "", "", 1, "1,2", "00", "/tmp/share.json")
	if signArgs[1] != "http://configured:8000" {
		t.Fatalf("sign manager addr = %s", signArgs[1])
	}
}
