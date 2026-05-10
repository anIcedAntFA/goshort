package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/anIcedAntFA/goshort/internal/cli"
	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats <code>",
	Short: "Show details for a short URL",
	Args:  cobra.ExactArgs(1),
	RunE:  runStats,
}

func init() {
	rootCmd.AddCommand(statsCmd)
}

func runStats(cmd *cobra.Command, args []string) error {
	code := args[0]
	client := cli.NewAPIClient(serverURL, apiKey)
	u, err := client.GetURL(cmd.Context(), code)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(u)
	}

	custom := "no"
	if u.IsCustom {
		custom = "yes"
	}
	expires := "never"
	if u.ExpiresAt != nil {
		expires = cli.FormatTime(*u.ExpiresAt)
	}

	fmt.Printf("  Code:       %s\n", u.ShortCode)
	fmt.Printf("  Original:   %s\n", u.OriginalURL)
	fmt.Printf("  Short URL:  %s\n", u.ShortURL)
	fmt.Printf("  Custom:     %s\n", custom)
	fmt.Printf("  Clicks:     %d\n", u.ClickCount)
	fmt.Printf("  Created:    %s\n", cli.FormatTime(u.CreatedAt))
	fmt.Printf("  Expires:    %s\n", expires)
	return nil
}
