package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the gh-wait server",
	Long: `Start the gh-wait background server.

The server listens on localhost (default port 9248) and polls the GitHub
API to evaluate watch rules. It is normally started automatically when
you create the first watch rule, but you can also start it explicitly.

Use --foreground to run the server in the foreground (useful for debugging).
Use --port to specify a custom port.

If the server is already running, this command does nothing.

Server state (watch rules) is persisted to:
  $XDG_STATE_HOME/gh-wait/gh-wait-{port}.json`,
	Example: `  # Start the server in the background
  gh wait start

  # Start the server in the foreground
  gh wait start --foreground

  # Start the server on a custom port
  gh wait start --port 9999`,
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
