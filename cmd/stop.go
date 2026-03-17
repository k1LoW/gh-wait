package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the gh-wait server",
	Long: `Stop the gh-wait background server gracefully.

The server saves its current state before shutting down, so watch rules
are preserved and will resume when the server is started again.

If the server is not running, this command does nothing.`,
	Example: `  # Stop the server
  gh wait stop`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		if _, err := c.ProbeStatus(); err != nil {
			fmt.Println("Server is not running")
			return nil
		}
		if err := c.Shutdown(); err != nil {
			return fmt.Errorf("failed to stop server: %w", err)
		}
		fmt.Println("Server stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(stopCmd)
}
