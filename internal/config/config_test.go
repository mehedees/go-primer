package config

import (
	"testing"
	"time"
)

// Tests live next to the code in *_test.go files and are run with `go test`.
// A test is any func TestXxx(t *testing.T). No classes, no fixtures framework.

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Addr != ":8080" {
		t.Errorf("Addr = %q, want %q", cfg.HTTP.Addr, ":8080")
	}
	if !cfg.IsDevelopment() {
		t.Errorf("expected development env by default")
	}
}

func TestLoadFromEnv(t *testing.T) {
	// t.Setenv sets a variable for this test only and restores it afterwards
	// (like pytest's monkeypatch.setenv).
	t.Setenv("APP_ENV", "production")
	t.Setenv("HTTP_ADDR", ":9000")
	t.Setenv("HTTP_READ_TIMEOUT", "2s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.HTTP.Addr != ":9000" {
		t.Errorf("Addr = %q, want :9000", cfg.HTTP.Addr)
	}
	if cfg.HTTP.ReadTimeout != 2*time.Second {
		t.Errorf("ReadTimeout = %v, want 2s", cfg.HTTP.ReadTimeout)
	}
}

func TestLoadInvalid(t *testing.T) {
	t.Setenv("APP_ENV", "staging")
	t.Setenv("HTTP_READ_TIMEOUT", "soon")

	if _, err := Load(); err == nil {
		t.Fatal("expected an error, got nil")
	}
}
