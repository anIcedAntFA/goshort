package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "goshort.toml")
	if err := os.WriteFile(path, []byte(`
server_url = "http://myserver:9090"
api_key = "sk_test_abc"
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := loadConfig(path)
	if cfg.ServerURL != "http://myserver:9090" {
		t.Errorf("ServerURL = %q, want http://myserver:9090", cfg.ServerURL)
	}
	if cfg.APIKey != "sk_test_abc" {
		t.Errorf("APIKey = %q, want sk_test_abc", cfg.APIKey)
	}
}

func TestLoadConfig_MissingFile(t *testing.T) {
	t.Parallel()

	cfg := loadConfig("/nonexistent/path/goshort.toml")
	if cfg.ServerURL != "" {
		t.Errorf("ServerURL = %q, want empty for missing file", cfg.ServerURL)
	}
	if cfg.APIKey != "" {
		t.Errorf("APIKey = %q, want empty for missing file", cfg.APIKey)
	}
}

func TestLoadConfig_InvalidTOML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "bad.toml")
	if err := os.WriteFile(path, []byte("not = valid = toml !!!"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg := loadConfig(path)
	if cfg.ServerURL != "" || cfg.APIKey != "" {
		t.Error("expected zero-value config for invalid TOML")
	}
}

func TestResolveValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		values []string
		want   string
	}{
		{"first wins", []string{"a", "b", "c"}, "a"},
		{"skip empty first", []string{"", "b", "c"}, "b"},
		{"all empty returns empty", []string{"", "", ""}, ""},
		{"single non-empty", []string{"only"}, "only"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := resolveValue(tc.values...)
			if got != tc.want {
				t.Errorf("resolveValue(%v) = %q, want %q", tc.values, got, tc.want)
			}
		})
	}
}
