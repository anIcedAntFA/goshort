package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// version is set via -ldflags at build time.
var version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(_ *cobra.Command, _ []string) {
		fmt.Printf("goshort-cli %s\n", version)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
