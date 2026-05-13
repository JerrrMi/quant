package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadSaaSConfig_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "saas.yaml")
	content := []byte(`
database:
  dsn: "file:test.db"
redis:
  enable: false
  addr: "127.0.0.1:6379"
  db: 0
websocket:
  listen_addr: "0.0.0.0:9"
  agent_path: "/v1/agent"
scheduler:
  enable: false
  tick_interval_seconds: 120
model:
  id: "m"
  values:
    x: 1
logging:
  level: "warn"
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadSaaSConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.WebSocket.ListenAddr != "0.0.0.0:9" || cfg.Database.DSN != "file:test.db" || cfg.Logging.Level != "warn" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}

func TestSaaSConfig_Validate_listenRequired(t *testing.T) {
	var c SaaSConfig
	c.Logging.Level = "info"
	if err := c.Validate(); err == nil {
		t.Fatal("expected error for missing websocket.listen_addr")
	}
}

func TestLoadBacktestConfig_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bt.yaml")
	content := []byte(`
data:
  provider: file
  path: ./p
  symbol: ETHUSDT
fees:
  maker_bps: 1
  taker_bps: 4
slippage:
  bps: 0.5
replay:
  warmup_bars: 10
  window:
    start: ""
    end: ""
capital:
  initial_quote: "1000"
  currency: "USDT"
model:
  id: bt
logging:
  level: debug
`)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadBacktestConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	if cfg.Data.Symbol != "ETHUSDT" || cfg.Slippage.Bps != 0.5 {
		t.Fatalf("unexpected cfg %+v", cfg)
	}
}

func TestLoadConfig_wrappedErrorMentionsKind(t *testing.T) {
	_, err := LoadSaaSConfig("/nonexistent/saas.yml")
	if err != nil && !strings.Contains(err.Error(), "saas config file") {
		t.Fatalf("expected error to mention kind and file path, got: %v", err)
	}
}
