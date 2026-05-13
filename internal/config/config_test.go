package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSaaSConfig_roundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "saas.yaml")
	content := []byte(`
http:
  listen_addr: "0.0.0.0:9"
database:
  dsn: "file:test.db"
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
	if cfg.HTTP.ListenAddr != "0.0.0.0:9" || cfg.Database.DSN != "file:test.db" || cfg.Logging.Level != "warn" {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
}
