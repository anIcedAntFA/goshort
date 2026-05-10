package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/anIcedAntFA/goshort/internal/cli"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List short URLs (paginated)",
	Args:  cobra.NoArgs,
	RunE:  runList,
}

func init() {
	listCmd.Flags().Int("page", 1, "page number")
	listCmd.Flags().Int("per-page", 20, "items per page")
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, _ []string) error {
	page, _ := cmd.Flags().GetInt("page")
	perPage, _ := cmd.Flags().GetInt("per-page")

	client := cli.NewAPIClient(serverURL, apiKey)
	resp, err := client.ListURLs(cmd.Context(), page, perPage)
	if err != nil {
		return err
	}

	if jsonOut {
		return json.NewEncoder(os.Stdout).Encode(resp)
	}

	if len(resp.Data) == 0 {
		fmt.Printf("No URLs found (page %d)\n", page)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "CODE\tORIGINAL URL\tCLICKS\tCREATED\tEXPIRES")

	for _, u := range resp.Data {
		created := cli.FormatTime(u.CreatedAt)
		expires := "never"
		if u.ExpiresAt != nil {
			expires = cli.FormatTime(*u.ExpiresAt)
		}
		originalURL := u.OriginalURL
		if len(originalURL) > 50 {
			originalURL = originalURL[:47] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%d\t%s\t%s\n",
			u.ShortCode, originalURL, u.ClickCount, created, expires)
	}

	_ = w.Flush()

	p := resp.Pagination
	fmt.Printf("\nPage %d of %d  (%d total)\n", p.Page, p.TotalPages, p.Total)
	return nil
}
