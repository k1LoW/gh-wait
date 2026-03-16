package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var restartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the gh-wait server",
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
