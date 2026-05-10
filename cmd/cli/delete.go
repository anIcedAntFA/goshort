package main

import (
	"fmt"

	"github.com/anIcedAntFA/goshort/internal/cli"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <code>",
	Short: "Delete a short URL",
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	code := args[0]
	client := cli.NewAPIClient(serverURL, apiKey)
	if err := client.DeleteURL(cmd.Context(), code); err != nil {
		return err
	}
	fmt.Printf("Deleted short URL: %s\n", code)
	return nil
}
