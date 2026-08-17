package config

import (
	"os"
	"testing"
	"time"
)

func reset() {
	os.Setenv("FORUMGO_AUTH_TOKEN", "")
}

func TestLoadDefaults(t *testing.T) {
	reset()
	cfg := Load()
	if cfg.Addr != ":8080" {
		t.Errorf("Addr = %q", cfg.Addr)
	}
	if cfg.MaxBody != 4096 {
		t.Errorf("MaxBody = %d", cfg.MaxBody)
	}
	if cfg.Timeout != 10*time.Second {
		t.Errorf("Timeout = %v", cfg.Timeout)
	}
	if cfg.RateLimit != 100 {
		t.Errorf("RateLimit = %d", cfg.RateLimit)
	}
	if cfg.RateWindow != 1*time.Minute {
		t.Errorf("RateWindow = %v", cfg.RateWindow)
	}
}

func TestLoadEnvOverrides(t *testing.T) {
	reset()
	os.Setenv("FORUMGO_ADDR", ":9090")
	os.Setenv("FORUMGO_MAX_BODY", "8192")
	os.Setenv("FORUMGO_TIMEOUT", "30s")
	os.Setenv("FORUMGO_RATE_LIMIT", "50")
	defer reset()
	cfg := Load()
	if cfg.Addr != ":9090" {
		t.Errorf("Addr = %q", cfg.Addr)
	}
	if cfg.MaxBody != 8192 {
		t.Errorf("MaxBody = %d", cfg.MaxBody)
	}
	if cfg.Timeout != 30*time.Second {
		t.Errorf("Timeout = %v", cfg.Timeout)
	}
	if cfg.RateLimit != 50 {
		t.Errorf("RateLimit = %d", cfg.RateLimit)
	}
}
