package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/anIcedAntFA/goshort/internal/cli"
	"github.com/spf13/cobra"
)

// Resolved configuration values used by all commands.
var (
	serverURL string
	apiKey    string
	jsonOut   bool
)

var rootCmd = &cobra.Command{
	Use:           "goshort-cli",
	Short:         "CLI client for GoShort URL shortener",
	Long:          "Command-line interface for interacting with a GoShort server.",
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP("config", "c", "", "config file (default: ~/.goshort.toml)")
	rootCmd.PersistentFlags().StringP("server", "s", "", "server URL")
	rootCmd.PersistentFlags().StringP("api-key", "k", "", "API key")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "output raw JSON")
}

func initConfig() {
	configPath, _ := rootCmd.PersistentFlags().GetString("config")
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			configPath = filepath.Join(home, ".goshort.toml")
		}
	}

	cfg := cli.LoadConfig(configPath)

	// Resolution: flag > env > config > default
	serverURL = cli.ResolveValue(
		flagString("server"),
		os.Getenv("GOSHORT_SERVER_URL"),
		cfg.ServerURL,
		"http://localhost:8080",
	)
	apiKey = cli.ResolveValue(
		flagString("api-key"),
		os.Getenv("GOSHORT_API_KEY"),
		cfg.APIKey,
		"",
	)
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// flagString returns the value of a persistent flag, or "" if not set.
func flagString(name string) string {
	v, _ := rootCmd.PersistentFlags().GetString(name)
	return v
}
