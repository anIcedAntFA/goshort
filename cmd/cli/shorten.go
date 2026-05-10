package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/anIcedAntFA/goshort/internal/cli"
	"github.com/spf13/cobra"
)

var shortenCmd = &cobra.Command{
	Use:   "shorten [url]",
	Short: "Create a short URL",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runShorten,
}

func init() {
	shortenCmd.Flags().String("alias", "", "custom alias (e.g., my-link)")
	shortenCmd.Flags().String("expires", "", "expiration duration (e.g., 7d, 30d)")
	rootCmd.AddCommand(shortenCmd)
}

func runShorten(cmd *cobra.Command, args []string) error {
	var rawURL string
	if len(args) > 0 {
		rawURL = args[0]
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		if scanner.Scan() {
			rawURL = strings.TrimSpace(scanner.Text())
		}
		if rawURL == "" {
			return fmt.Errorf("url is required (pass as argument or pipe via stdin)")
		}
	}

	alias, _ := cmd.Flags().GetString("alias")
	expires, _ := cmd.Flags().GetString("expires")

	client := cli.NewAPIClient(serverURL, apiKey)
	resp, err := client.CreateURL(cmd.Context(), cli.CreateRequest{
		URL:         rawURL,
		CustomAlias: alias,
		ExpiresIn:   expires,
	})
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(resp)
	}

	fmt.Println("Short URL created")
	fmt.Println()
	fmt.Printf("  Short URL:  %s\n", resp.ShortURL)
	fmt.Printf("  Original:   %s\n", resp.OriginalURL)
	fmt.Printf("  Code:       %s\n", resp.ShortCode)
	if resp.ExpiresAt != nil {
		fmt.Printf("  Expires:    %s\n", *resp.ExpiresAt)
	} else {
		fmt.Printf("  Expires:    never\n")
	}
	return nil
}
