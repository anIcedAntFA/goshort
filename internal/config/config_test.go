package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/anIcedAntFA/goshort/internal/config"
)

func TestLoad_Defaults(t *testing.T) {
	t.Parallel()

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") error = %v", err)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"server.port", cfg.Server.Port, 8080},
		{"server.base_url", cfg.Server.BaseURL, "http://localhost:8080"},
		{"storage.driver", cfg.Storage.Driver, "sqlite"},
		{"storage.sqlite_path", cfg.Storage.SQLitePath, "./data/goshort.db"},
		{"cache.driver", cfg.Cache.Driver, "none"},
		{"cache.redis_url", cfg.Cache.RedisURL, "redis://localhost:6379"},
		{"shortener.code_length", cfg.Shortener.CodeLength, 6},
		{"shortener.default_expiry", cfg.Shortener.DefaultExpiry, "0"},
		{"logging.level", cfg.Logging.Level, "info"},
		{"logging.format", cfg.Logging.Format, "json"},
		{"rate_limit.enabled", cfg.RateLimit.Enabled, false},
		{"rate_limit.requests_per_minute", cfg.RateLimit.RequestsPerMinute, 60},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.got != tc.want {
				t.Errorf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
}

func TestLoad_TOMLFile(t *testing.T) {
	t.Parallel()

	toml := `
[server]
port = 3000
base_url = "https://example.com"

[storage]
driver = "sqlite"
sqlite_path = "/tmp/test.db"

[cache]
driver = "memory"

[logging]
level = "debug"
format = "text"

[rate_limit]
enabled = true
requests_per_minute = 30
`
	path := filepath.Join(t.TempDir(), "goshort.toml")
	if err := os.WriteFile(path, []byte(toml), 0o600); err != nil {
		t.Fatalf("write toml file: %v", err)
	}

	cfg, err := config.Load(path)
	if err != nil {
		t.Fatalf("Load(%q) error = %v", path, err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("server.port: got %d, want 3000", cfg.Server.Port)
	}
	if cfg.Server.BaseURL != "https://example.com" {
		t.Errorf("server.base_url: got %q, want %q", cfg.Server.BaseURL, "https://example.com")
	}
	if cfg.Storage.SQLitePath != "/tmp/test.db" {
		t.Errorf("storage.sqlite_path: got %q, want %q", cfg.Storage.SQLitePath, "/tmp/test.db")
	}
	if cfg.Cache.Driver != "memory" {
		t.Errorf("cache.driver: got %q, want %q", cfg.Cache.Driver, "memory")
	}
	if cfg.Logging.Level != "debug" {
		t.Errorf("logging.level: got %q, want debug", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "text" {
		t.Errorf("logging.format: got %q, want text", cfg.Logging.Format)
	}
	if !cfg.RateLimit.Enabled {
		t.Error("rate_limit.enabled: got false, want true")
	}
	if cfg.RateLimit.RequestsPerMinute != 30 {
		t.Errorf("rate_limit.requests_per_minute: got %d, want 30", cfg.RateLimit.RequestsPerMinute)
	}
}

func TestLoad_EnvOverride(t *testing.T) {
	t.Setenv("GOSHORT_SERVER_PORT", "9090")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") error = %v", err)
	}

	if cfg.Server.Port != 9090 {
		t.Errorf("server.port: got %d, want 9090", cfg.Server.Port)
	}
}

func TestLoad_EnvPrefix(t *testing.T) {
	t.Setenv("GOSHORT_CACHE_DRIVER", "memory")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") error = %v", err)
	}

	if cfg.Cache.Driver != "memory" {
		t.Errorf("cache.driver: got %q, want memory", cfg.Cache.Driver)
	}
}

func TestLoad_EnvOverride_MultiWordField(t *testing.T) {
	t.Setenv("GOSHORT_CACHE_REDIS_URL", "redis://custom:6380")
	t.Setenv("GOSHORT_STORAGE_SQLITE_PATH", "/tmp/custom.db")
	t.Setenv("GOSHORT_SHORTENER_CODE_LENGTH", "8")
	t.Setenv("GOSHORT_SHORTENER_DEFAULT_EXPIRY", "30d")
	t.Setenv("GOSHORT_RATE_LIMIT_REQUESTS_PER_MINUTE", "120")
	t.Setenv("GOSHORT_AUTH_API_KEY", "sk_test_123")
	t.Setenv("GOSHORT_SERVER_BASE_URL", "https://short.example.com")

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load error: %v", err)
	}

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"cache.redis_url", cfg.Cache.RedisURL, "redis://custom:6380"},
		{"storage.sqlite_path", cfg.Storage.SQLitePath, "/tmp/custom.db"},
		{"shortener.code_length", cfg.Shortener.CodeLength, 8},
		{"shortener.default_expiry", cfg.Shortener.DefaultExpiry, "30d"},
		{"rate_limit.requests_per_minute", cfg.RateLimit.RequestsPerMinute, 120},
		{"auth.api_key", cfg.Auth.APIKey, "sk_test_123"},
		{"server.base_url", cfg.Server.BaseURL, "https://short.example.com"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.want {
				t.Errorf("got %v, want %v", tc.got, tc.want)
			}
		})
	}
}

func TestLoad_AutoDiscover(t *testing.T) {
	dir := t.TempDir()
	tomlContent := `[server]
port = 7777
`
	if err := os.WriteFile(filepath.Join(dir, "goshort.toml"), []byte(tomlContent), 0o600); err != nil {
		t.Fatalf("write goshort.toml: %v", err)
	}

	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") error = %v", err)
	}
	if cfg.Server.Port != 7777 {
		t.Errorf("auto-discover: server.port got %d, want 7777", cfg.Server.Port)
	}
}

func TestLoad_AutoDiscover_NoFile(t *testing.T) {
	dir := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	cfg, err := config.Load("")
	if err != nil {
		t.Fatalf("Load(\"\") with no file error = %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("no file: server.port got %d, want 8080 (default)", cfg.Server.Port)
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	t.Parallel()

	_, err := config.Load("nonexistent.toml")
	if err == nil {
		t.Error("expected error for nonexistent file, got nil")
	}
}

func TestConfig_Validate(t *testing.T) {
	t.Parallel()

	validConfig := func() *config.Config {
		t.Helper()
		cfg, err := config.Load("")
		if err != nil {
			t.Fatalf("Load: %v", err)
		}
		return cfg
	}

	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantErr bool
	}{
		{"defaults are valid", func(_ *config.Config) {}, false},
		{"invalid cache driver", func(c *config.Config) { c.Cache.Driver = "memcached" }, true},
		{"empty base_url", func(c *config.Config) { c.Server.BaseURL = "" }, true},
		{"code_length zero", func(c *config.Config) { c.Shortener.CodeLength = 0 }, true},
		{"code_length too large", func(c *config.Config) { c.Shortener.CodeLength = 256 }, true},
		{"negative port", func(c *config.Config) { c.Server.Port = -1 }, true},
		{"port too large", func(c *config.Config) { c.Server.Port = 65536 }, true},
		{"invalid storage driver", func(c *config.Config) { c.Storage.Driver = "postgres" }, true},
		{"invalid logging level", func(c *config.Config) { c.Logging.Level = "trace" }, true},
		{"invalid logging format", func(c *config.Config) { c.Logging.Format = "xml" }, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfg := validConfig()
			tc.mutate(cfg)
			err := cfg.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}
