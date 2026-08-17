package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds application configuration from environment variables.
type Config struct {
	Addr       string
	DataDir    string
	AuthToken  string
	MaxBody    int64
	Timeout    time.Duration
	RateLimit  int
	RateWindow time.Duration
}

// Load returns a Config populated from FORUMGO_* environment variables.
func Load() *Config {
	return &Config{
		Addr:       getEnv("FORUMGO_ADDR", ":8080"),
		DataDir:    getEnv("FORUMGO_DATA_DIR", ""),
		AuthToken:  getEnv("FORUMGO_AUTH_TOKEN", ""),
		MaxBody:    int64Env("FORUMGO_MAX_BODY", 4096),
		Timeout:    durationEnv("FORUMGO_TIMEOUT", 10*time.Second),
		RateLimit:  intEnv("FORUMGO_RATE_LIMIT", 100),
		RateWindow: durationEnv("FORUMGO_RATE_WINDOW", 1*time.Minute),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func intEnv(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return fallback
}

func int64Env(key string, fallback int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return fallback
}
