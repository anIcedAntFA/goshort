package cli

import (
	"os"

	"github.com/pelletier/go-toml/v2"
)

// Config holds settings read from ~/.goshort.toml.
type Config struct {
	ServerURL string `toml:"server_url"`
	APIKey    string `toml:"api_key"`
}

// LoadConfig reads the TOML config file at path.
// Missing file returns zero-value config without error.
func LoadConfig(path string) Config {
	data, err := os.ReadFile(path) //nolint:gosec // path comes from user-controlled flag
	if err != nil {
		return Config{}
	}
	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}
	}
	return cfg
}

// ResolveValue returns the first non-empty value from the list.
func ResolveValue(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
