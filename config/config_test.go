package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoad_DefaultsAndLegacyURL(t *testing.T) {
	tmp := t.TempDir()
	cfgDir := filepath.Join(tmp, "config")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatalf("mkdir config: %v", err)
	}

	content := `{
  "app": {
    "port": "8080",
    "read_timeout": 5,
    "write_timeout": 7
  },
  "elasticsearch": {
    "url": "http://localhost:9200"
  },
  "cors_allowed_origins": "*",
  "jwt": {}
}`
	if err := os.WriteFile(filepath.Join(cfgDir, "local.json"), []byte(content), 0o600); err != nil {
		t.Fatalf("write local config: %v", err)
	}

	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	defer func() { _ = os.Chdir(wd) }()
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}

	t.Setenv("APP_ENV", "unknown-env")
	cfg := Load()

	if cfg.App.Env != "local" {
		t.Fatalf("expected env local fallback, got %q", cfg.App.Env)
	}
	if len(cfg.Elasticsearch.Addresses) != 1 || cfg.Elasticsearch.Addresses[0] != "http://localhost:9200" {
		t.Fatalf("expected legacy elasticsearch.url fallback to addresses")
	}
	if cfg.App.ReadTimeout != 5*time.Second {
		t.Fatalf("expected read_timeout promoted to seconds, got %v", cfg.App.ReadTimeout)
	}
	if cfg.App.WriteTimeout != 7*time.Second {
		t.Fatalf("expected write_timeout promoted to seconds, got %v", cfg.App.WriteTimeout)
	}
	if cfg.JWT.Secret != DefaultJWTSecret {
		t.Fatalf("expected default jwt secret")
	}
	if cfg.JWT.AdminUsername != DefaultAdminUsername || cfg.JWT.AdminPassword != DefaultAdminPassword {
		t.Fatalf("expected default admin credentials")
	}
}

func TestBuildLogger(t *testing.T) {
	devCfg := &Config{App: AppConfig{Env: "local"}}
	prodCfg := &Config{App: AppConfig{Env: "production"}}

	dev := devCfg.BuildLogger()
	if dev == nil {
		t.Fatalf("expected dev logger")
	}
	_ = dev.Sync()

	prod := prodCfg.BuildLogger()
	if prod == nil {
		t.Fatalf("expected prod logger")
	}
	_ = prod.Sync()
}
