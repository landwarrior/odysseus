package config

import (
	"path/filepath"
	"testing"
)

func TestDefaultListenPortIsChosenByOS(t *testing.T) {
	t.Setenv("ODYSSEUS_HTTP_ADDR", "")
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != "127.0.0.1:0" {
		t.Fatalf("HTTPAddr = %s", cfg.HTTPAddr)
	}
	if cfg.BrowserURL() != "http://127.0.0.1:0" {
		t.Fatalf("BrowserURL = %s", cfg.BrowserURL())
	}
}

func TestExplicitListenAddr(t *testing.T) {
	t.Setenv("ODYSSEUS_HTTP_ADDR", "127.0.0.1:8080")
	cfg, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if cfg.HTTPAddr != "127.0.0.1:8080" {
		t.Fatalf("HTTPAddr = %s", cfg.HTTPAddr)
	}
}

func TestDevMode(t *testing.T) {
	t.Setenv("ODYSSEUS_DEV", "1")
	base := t.TempDir()
	cfg, err := Load(base)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Dev {
		t.Fatal("Dev が真になるはずです")
	}
	want := filepath.Join(base, "internal", "web")
	if cfg.WebDir != want {
		t.Fatalf("WebDir = %s, want %s", cfg.WebDir, want)
	}
}
