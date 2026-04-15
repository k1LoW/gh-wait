package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var workflowConditionFlags = []string{"completed", "succeeded", "failed"}

var workflowCmd = &cobra.Command{
	Use:   "workflow <run-id>",
	Short: "Watch a workflow run for conditions",
	Long: `Watch a GitHub Actions workflow run for one or more conditions and trigger
an action when any condition is met.

The workflow run ID is required. You can also pass a full workflow run URL
directly to gh-wait instead of using this subcommand.

Available conditions (defaults to --completed if none specified):
  --completed   Triggered when the workflow run reaches any terminal state.
  --succeeded   Triggered when the workflow run completes with success.
  --failed      Triggered when the workflow run completes with failure.

Multiple conditions are evaluated with OR logic — the rule triggers when
any one of the specified conditions is met.

Actions:
  By default, the trigger is logged to the server log. Use --notify to send
  an OS desktop notification, --open to open the workflow run in your default
  browser, or both together.

Termination:
  By default, a rule triggers once and stops. Use --count to allow
  multiple triggers, or --until to keep watching until a termination
  condition is met (e.g., --until succeeded).

Polling:
  The server polls the GitHub API at the interval specified by --interval
  (default: 1min). Accepts durations like 30sec, 5min, 1h.`,
	Example: `  # Watch a workflow run for completion (default)
  gh wait workflow 23424874935 --repo owner/repo

  # Watch a workflow run for failure, open browser when failed
  gh wait workflow 23424874935 --repo owner/repo --failed --open

  # Watch a workflow run for failure, stop when succeeded
  gh wait workflow 23424874935 --repo owner/repo --failed --until succeeded

  # Watch by URL (auto-detected)
  gh wait https://github.com/owner/repo/actions/runs/23424874935`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		runID, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid workflow run ID: %w", err)
		}

		return addWatchRule(cmd, "workflow", runID, workflowConditionFlags)
	},
}

func init() {
	rootCmd.AddCommand(workflowCmd)
	registerWatchFlags(workflowCmd, "workflow")
	workflowCmd.Flags().Bool("completed", false, "Watch for completion (any conclusion)")
	workflowCmd.Flags().Bool("succeeded", false, "Watch for success")
	workflowCmd.Flags().Bool("failed", false, "Watch for failure")
}
