package main

import (
	"os"
	"testing"
	"time"
)

func TestServerConfigFromEnvUsesDefaultsAndOverrides(t *testing.T) {
	os.Setenv("TENDER_API_KEY", "env-key")
	os.Setenv("TENDER_ENABLE_AUTH", "false")
	os.Setenv("TENDER_RATE_LIMIT", "12")
	os.Setenv("TENDER_RATE_WINDOW_SECONDS", "90")
	os.Setenv("TENDER_METRICS_PATH", "/custom/metrics")
	os.Setenv("TENDER_API_PORT", "9090")
	os.Setenv("TENDER_P2P_PORT", "4040")
	os.Setenv("TENDER_DATA_DIR", "/tmp/tender")
	os.Setenv("TENDER_CONSENSUS", "poa")
	os.Setenv("TENDER_STRICT_P2P", "false")
	defer os.Unsetenv("TENDER_API_KEY")
	defer os.Unsetenv("TENDER_ENABLE_AUTH")
	defer os.Unsetenv("TENDER_RATE_LIMIT")
	defer os.Unsetenv("TENDER_RATE_WINDOW_SECONDS")
	defer os.Unsetenv("TENDER_METRICS_PATH")
	defer os.Unsetenv("TENDER_API_PORT")
	defer os.Unsetenv("TENDER_P2P_PORT")
	defer os.Unsetenv("TENDER_DATA_DIR")
	defer os.Unsetenv("TENDER_CONSENSUS")
	defer os.Unsetenv("TENDER_STRICT_P2P")

	cfg := serverConfigFromEnv()

	if cfg.APIKey != "env-key" {
		t.Fatalf("expected API key from env, got %q", cfg.APIKey)
	}
	if cfg.EnableAuth {
		t.Fatal("expected auth to be disabled from env")
	}
	if cfg.RateLimit != 12 {
		t.Fatalf("expected rate limit 12, got %d", cfg.RateLimit)
	}
	if cfg.RateWindow != 90*time.Second {
		t.Fatalf("expected rate window 90s, got %v", cfg.RateWindow)
	}
	if cfg.MetricsPath != "/custom/metrics" {
		t.Fatalf("expected metrics path override, got %q", cfg.MetricsPath)
	}
	if cfg.APIPort != 9090 {
		t.Fatalf("expected API port 9090, got %d", cfg.APIPort)
	}
	if cfg.P2PPort != 4040 {
		t.Fatalf("expected P2P port 4040, got %d", cfg.P2PPort)
	}
	if cfg.DataDir != "/tmp/tender" {
		t.Fatalf("expected data-dir override, got %q", cfg.DataDir)
	}
	if cfg.Consensus != "poa" {
		t.Fatalf("expected consensus override poa, got %q", cfg.Consensus)
	}
	if cfg.StrictP2P {
		t.Fatal("expected strict P2P to be disabled from env")
	}
}

func TestServerConfigFromEnvUsesDefaultsWhenUnset(t *testing.T) {
	for _, key := range []string{"TENDER_API_KEY", "TENDER_ENABLE_AUTH", "TENDER_RATE_LIMIT", "TENDER_RATE_WINDOW_SECONDS", "TENDER_METRICS_PATH", "TENDER_API_PORT", "TENDER_P2P_PORT", "TENDER_DATA_DIR", "TENDER_CONSENSUS", "TENDER_STRICT_P2P"} {
		os.Unsetenv(key)
	}

	cfg := serverConfigFromEnv()

	if cfg.APIKey != "" {
		t.Fatalf("expected empty default API key, got %q", cfg.APIKey)
	}
	if !cfg.EnableAuth {
		t.Fatal("expected auth to be enabled by default")
	}
	if cfg.RateLimit != 60 {
		t.Fatalf("expected default rate limit 60, got %d", cfg.RateLimit)
	}
	if cfg.RateWindow != time.Minute {
		t.Fatalf("expected default rate window 1m, got %v", cfg.RateWindow)
	}
	if cfg.MetricsPath != "/metrics" {
		t.Fatalf("expected default metrics path, got %q", cfg.MetricsPath)
	}
	if cfg.APIPort != 8080 {
		t.Fatalf("expected default API port 8080, got %d", cfg.APIPort)
	}
	if cfg.P2PPort != 3030 {
		t.Fatalf("expected default P2P port 3030, got %d", cfg.P2PPort)
	}
	if cfg.DataDir != "./data" {
		t.Fatalf("expected default data dir ./data, got %q", cfg.DataDir)
	}
	if cfg.Consensus != "pos" {
		t.Fatalf("expected default consensus pos, got %q", cfg.Consensus)
	}
	if !cfg.StrictP2P {
		t.Fatal("expected strict P2P to be enabled by default")
	}
}
