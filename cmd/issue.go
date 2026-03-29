package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var issueConditionFlags = []string{"commented", "closed"}

var issueCmd = &cobra.Command{
	Use:   "issue <number>",
	Short: "Watch an issue for conditions",
	Long: `Watch a GitHub issue for one or more conditions and trigger an action
when any condition is met.

Unlike the "pr" subcommand, the issue number is required (no auto-detection).

Available conditions (at least one condition or --until is required):
  --commented   Triggered when a new comment is posted on the issue.
  --closed      Triggered when the issue is closed.

Multiple conditions are evaluated with OR logic — the rule triggers when
any one of the specified conditions is met.

Actions:
  By default, the trigger is logged to the server log. Use --notify to send
  an OS desktop notification, --open to open the issue in your default browser,
  or both together.

Termination:
  By default, a rule triggers once and stops. Use --count to allow
  multiple triggers, or --until to keep watching until a termination
  condition is met (e.g., --until closed).

Polling:
  The server polls the GitHub API at the interval specified by --interval
  (default: 30sec). Accepts durations like 30sec, 5min, 1h.

Filtering:
  Use --ignore-user to exclude events from specific users. The value is
  a Go regular expression matched against the username. Can be specified
  multiple times.`,
	Example: `  # Watch issue #5 for new comments
  gh wait issue 5 --commented

  # Watch issue #5 for close, open browser when closed
  gh wait issue 5 --closed --open

  # Watch issue for comments, stop when closed
  gh wait issue 10 --commented --until closed

  # Watch issue for comments with 5-minute polling, ignoring bots
  gh wait issue 10 --commented --interval 5min --ignore-user ".*\\[bot\\]"

  # Watch issue on a different repo
  gh wait issue 5 --commented --repo owner/repo`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid issue number: %w", err)
		}

		return addWatchRule(cmd, "issue", number, issueConditionFlags)
	},
}

func init() {
	rootCmd.AddCommand(issueCmd)
	registerWatchFlags(issueCmd)
	issueCmd.Flags().Bool("commented", false, "Watch for new comments")
	issueCmd.Flags().Bool("closed", false, "Watch for close")
}
