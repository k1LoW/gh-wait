package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:   "stop",
	Short: "Stop the gh-wait server",
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
