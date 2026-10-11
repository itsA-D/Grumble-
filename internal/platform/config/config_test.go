package config

import (
	"strings"
	"testing"
	"time"
)

func validEnv() map[string]string {
	return map[string]string{
		"DATABASE_URL":       "postgres://user:pass@localhost:5432/grumble",
		"ANON_ID_SECRET":     "0123456789abcdef",
		"IP_HASH_SALT":       "fedcba9876543210",
		"SITE_URL":           "https://grumble.example",
		"PRODUCT_NAME":       "Grumble",
		"DATABASE_MAX_CONNS": "20",
		"SSE_MAX_CLIENTS":    "5000",
	}
}

func lookupFrom(m map[string]string) lookupFunc {
	return func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
}

func TestLoadValid(t *testing.T) {
	cfg, err := load(lookupFrom(validEnv()))
	if err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if cfg.ProductName != "Grumble" {
		t.Errorf("ProductName = %q, want Grumble", cfg.ProductName)
	}
	if cfg.Database.URL == "" {
		t.Error("Database.URL is empty")
	}
	if cfg.Database.DirectURL != cfg.Database.URL {
		t.Errorf("DirectURL = %q, want it to default to URL", cfg.Database.DirectURL)
	}
	if cfg.Database.MaxConns != 20 {
		t.Errorf("MaxConns = %d, want 20", cfg.Database.MaxConns)
	}
	if cfg.Realtime.MaxClients != 5000 {
		t.Errorf("MaxClients = %d, want 5000", cfg.Realtime.MaxClients)
	}
	if cfg.HTTP.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("ReadHeaderTimeout = %v, want 5s", cfg.HTTP.ReadHeaderTimeout)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	env := validEnv()
	delete(env, "DATABASE_URL")
	_, err := load(lookupFrom(env))
	if err == nil {
		t.Fatal("expected error for missing DATABASE_URL")
	}
	if !strings.Contains(err.Error(), "DATABASE_URL") {
		t.Errorf("error %q does not mention DATABASE_URL", err)
	}
}

func TestLoadShortSecret(t *testing.T) {
	env := validEnv()
	env["ANON_ID_SECRET"] = "short"
	_, err := load(lookupFrom(env))
	if err == nil || !strings.Contains(err.Error(), "ANON_ID_SECRET") {
		t.Fatalf("expected ANON_ID_SECRET error, got %v", err)
	}
}

func TestLoadInvalidEnum(t *testing.T) {
	env := validEnv()
	env["REALTIME_MODE"] = "carrier-pigeon"
	_, err := load(lookupFrom(env))
	if err == nil || !strings.Contains(err.Error(), "REALTIME_MODE") {
		t.Fatalf("expected REALTIME_MODE error, got %v", err)
	}
}

func TestRedisModeRequiresURL(t *testing.T) {
	env := validEnv()
	env["RATELIMIT_MODE"] = "redis"
	_, err := load(lookupFrom(env))
	if err == nil || !strings.Contains(err.Error(), "RATELIMIT_REDIS_URL") {
		t.Fatalf("expected RATELIMIT_REDIS_URL error, got %v", err)
	}
}

func TestDefaultsApplied(t *testing.T) {
	cfg, err := load(lookupFrom(validEnv()))
	if err != nil {
		t.Fatalf("load returned error: %v", err)
	}
	if cfg.Env != "dev" {
		t.Errorf("Env = %q, want dev", cfg.Env)
	}
	if cfg.Log.Level != "info" {
		t.Errorf("Log.Level = %q, want info", cfg.Log.Level)
	}
	if cfg.HTTP.Addr != ":3000" {
		t.Errorf("HTTP.Addr = %q, want :3000", cfg.HTTP.Addr)
	}
	if cfg.Realtime.Mode != "memory" {
		t.Errorf("Realtime.Mode = %q, want memory", cfg.Realtime.Mode)
	}
}
