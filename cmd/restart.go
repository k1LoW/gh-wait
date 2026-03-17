package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the gh-wait server",
	Long: `Restart the gh-wait background server.

This stops the server gracefully (if running), then starts a new
background server process. Watch rules are preserved across restarts.

Useful after updating gh-wait to pick up the new binary.`,
	Example: `  # Restart the server
  gh wait restart

  # Restart on a custom port
  gh wait restart --port 9999`,
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		if _, err := c.ProbeStatus(); err == nil {
			if err := c.Shutdown(); err != nil {
				return fmt.Errorf("failed to stop server: %w", err)
			}
			fmt.Println("Server stopped")
		}
		return startBackground()
	},
}

func init() {
	rootCmd.AddCommand(restartCmd)
}
