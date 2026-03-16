package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the gh-wait server",
	RunE: func(cmd *cobra.Command, args []string) error {
		if foreground {
			return runForeground(cmd.Context())
		}
		if _, err := probeServer(); err == nil {
			fmt.Println("Server is already running")
			return nil
		}
		return startBackground()
	},
}

func init() {
	rootCmd.AddCommand(startCmd)
}
