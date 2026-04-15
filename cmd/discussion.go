package cmd

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var discussionConditionFlags = []string{"commented", "closed", "answered"}

var discussionCmd = &cobra.Command{
	Use:   "discussion <number>",
	Short: "Watch a discussion for conditions",
	Long: `Watch a GitHub discussion for one or more conditions and trigger an action
when any condition is met.

The discussion number is required (no auto-detection).

Available conditions (at least one condition or --until is required):
  --commented   Triggered when a new comment is posted on the discussion.
  --closed      Triggered when the discussion is closed.
  --answered    Triggered when the discussion is marked as answered.

Multiple conditions are evaluated with OR logic — the rule triggers when
any one of the specified conditions is met.

Actions:
  By default, the trigger is logged to the server log. Use --notify to send
  an OS desktop notification, --open to open the discussion in your default
  browser, or both together.

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
	Example: `  # Watch discussion #5 for new comments
  gh wait discussion 5 --commented

  # Watch discussion #5 for answer, open browser when answered
  gh wait discussion 5 --answered --open

  # Watch discussion for comments, stop when closed
  gh wait discussion 10 --commented --until closed

  # Watch discussion for comments with 5-minute polling, ignoring bots
  gh wait discussion 10 --commented --interval 5min --ignore-user ".*\\[bot\\]"

  # Watch discussion on a different repo
  gh wait discussion 5 --commented --repo owner/repo`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid discussion number: %w", err)
		}

		return addWatchRule(cmd, "discussion", number, discussionConditionFlags)
	},
}

func init() {
	rootCmd.AddCommand(discussionCmd)
	registerWatchFlags(discussionCmd, "discussion")
	discussionCmd.Flags().Bool("commented", false, "Watch for new comments")
	discussionCmd.Flags().Bool("closed", false, "Watch for close")
	discussionCmd.Flags().Bool("answered", false, "Watch for answer")
}
