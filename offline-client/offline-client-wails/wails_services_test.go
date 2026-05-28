package main

import (
	"path/filepath"
	"testing"

	"offline-client-wails/mpc_core/config"
)

func TestApplyDesktopRuntimePathsReplacesRelativeDirs(t *testing.T) {
	cfg := &config.Config{
		TempDir: "./temp",
		LogDir:  "./logs",
	}

	if err := applyDesktopRuntimePaths(cfg); err != nil {
		t.Fatalf("apply runtime paths: %v", err)
	}

	if !filepath.IsAbs(cfg.TempDir) {
		t.Fatalf("TempDir should be absolute, got %q", cfg.TempDir)
	}
	if !filepath.IsAbs(cfg.LogDir) {
		t.Fatalf("LogDir should be absolute, got %q", cfg.LogDir)
	}
	if filepath.Base(filepath.Dir(cfg.TempDir)) != "mpc-temp" {
		t.Fatalf("TempDir should live under mpc-temp, got %q", cfg.TempDir)
	}
}

func TestApplyDesktopRuntimePathsKeepsAbsoluteDirs(t *testing.T) {
	tempDir := t.TempDir()
	logDir := t.TempDir()
	cfg := &config.Config{
		TempDir: tempDir,
		LogDir:  logDir,
	}

	if err := applyDesktopRuntimePaths(cfg); err != nil {
		t.Fatalf("apply runtime paths: %v", err)
	}

	if cfg.TempDir != tempDir {
		t.Fatalf("TempDir changed: want %q got %q", tempDir, cfg.TempDir)
	}
	if cfg.LogDir != logDir {
		t.Fatalf("LogDir changed: want %q got %q", logDir, cfg.LogDir)
	}
}
