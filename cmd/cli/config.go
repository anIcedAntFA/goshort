package main

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

// CLIConfig holds settings read from ~/.goshort.toml.
type CLIConfig struct {
	ServerURL string `toml:"server_url"`
	APIKey    string `toml:"api_key"`
}

// loadConfig reads the TOML config file at path.
// Missing file returns zero-value config without error.
func loadConfig(path string) CLIConfig {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from user-controlled flag
	if err != nil {
		return CLIConfig{}
	}
	var cfg CLIConfig
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return CLIConfig{}
	}
	return cfg
}
