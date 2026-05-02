// Package config loads application configuration from TOML files and env vars.
package config

import (
	"fmt"
	"os"
	"strings"

	toml "github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/confmap"
	env "github.com/knadh/koanf/providers/env/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config holds the complete application configuration.
type Config struct {
	Server    ServerConfig    `koanf:"server"`
	Storage   StorageConfig   `koanf:"storage"`
	Cache     CacheConfig     `koanf:"cache"`
	Auth      AuthConfig      `koanf:"auth"`
	RateLimit RateLimitConfig `koanf:"rate_limit"`
	Shortener ShortenerConfig `koanf:"shortener"`
	Logging   LoggingConfig   `koanf:"logging"`
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Port    int    `koanf:"port"`
	BaseURL string `koanf:"base_url"`
}

// StorageConfig holds storage backend settings.
type StorageConfig struct {
	Driver     string `koanf:"driver"`
	SQLitePath string `koanf:"sqlite_path"`
}

// CacheConfig holds cache backend settings.
type CacheConfig struct {
	Driver   string `koanf:"driver"`
	RedisURL string `koanf:"redis_url"`
}

// AuthConfig holds authentication settings.
type AuthConfig struct {
	APIKey string `koanf:"api_key"`
}

// RateLimitConfig holds rate limiting settings.
type RateLimitConfig struct {
	Enabled           bool `koanf:"enabled"`
	RequestsPerMinute int  `koanf:"requests_per_minute"`
}

// ShortenerConfig holds URL shortener settings.
type ShortenerConfig struct {
	CodeLength    int    `koanf:"code_length"`
	DefaultExpiry string `koanf:"default_expiry"`
}

// LoggingConfig holds logging settings.
type LoggingConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}

const defaultConfigFile = "goshort.toml"

var defaults = map[string]any{
	"server.port":                    8080,
	"server.base_url":                "http://localhost:8080",
	"storage.driver":                 "sqlite",
	"storage.sqlite_path":            "./data/goshort.db",
	"cache.driver":                   "none",
	"cache.redis_url":                "redis://localhost:6379",
	"shortener.code_length":          6,
	"shortener.default_expiry":       "0",
	"logging.level":                  "info",
	"logging.format":                 "json",
	"rate_limit.enabled":             false,
	"rate_limit.requests_per_minute": 60,
}

// Load builds a Config by layering: defaults → TOML file → env vars.
//
// path is the explicit config file. If empty, Load auto-discovers goshort.toml
// in the current directory (silently skipped when absent).
// Env vars are prefixed GOSHORT_ with _ as the level separator:
// GOSHORT_SERVER_PORT=9090 overrides server.port.
func Load(path string) (*Config, error) {
	k := koanf.New(".")

	if err := k.Load(confmap.Provider(defaults, "."), nil); err != nil {
		return nil, fmt.Errorf("load defaults: %w", err)
	}
	if err := loadTOML(k, resolveConfigPath(path)); err != nil {
		return nil, err
	}
	if err := k.Load(env.Provider(".", envOpts()), nil); err != nil {
		return nil, fmt.Errorf("load env vars: %w", err)
	}

	var cfg Config
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}

// resolveConfigPath returns the file path to load.
// An explicit path is returned as-is. When empty, returns defaultConfigFile
// if it exists in the current directory, otherwise "".
func resolveConfigPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if _, err := os.Stat(defaultConfigFile); err == nil {
		return defaultConfigFile
	}
	return ""
}

// loadTOML loads a TOML file into k. Empty path is a no-op.
func loadTOML(k *koanf.Koanf, path string) error {
	if path == "" {
		return nil
	}
	if err := k.Load(file.Provider(path), toml.Parser()); err != nil {
		return fmt.Errorf("load config file %s: %w", path, err)
	}
	return nil
}

// envOpts returns the provider options for GOSHORT_ env vars.
// GOSHORT_SERVER_PORT → server.port (strip prefix, lowercase, _ → .)
func envOpts() env.Opt {
	return env.Opt{
		Prefix:        "GOSHORT_",
		TransformFunc: envKeyTransform,
	}
}

func envKeyTransform(k, v string) (string, any) { //nolint:gocritic // plain types are clearer than named returns here
	return strings.ReplaceAll(
		strings.ToLower(strings.TrimPrefix(k, "GOSHORT_")),
		"_", ".",
	), v
}
