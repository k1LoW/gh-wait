package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/cli/go-gh/v2"
	"github.com/spf13/cobra"
)

var prConditionFlags = []string{"approved", "merged", "closed", "commented", "ci-completed", "ci-failed"}

var prCmd = &cobra.Command{
	Use:   "pr [number]",
	Short: "Watch a pull request for conditions",
	Long: `Watch a pull request for one or more conditions and trigger an action
when any condition is met.

If no PR number is given, gh-wait auto-detects the PR associated with
the current branch using "gh pr view".

Available conditions (at least one condition or --until is required):
  --approved      Triggered when the PR receives a new approval review.
  --merged        Triggered when the PR is merged.
  --closed        Triggered when the PR is closed (without merge).
  --commented     Triggered when a new comment, review comment, or review
                  with body is posted on the PR.
  --ci-completed  Triggered when all CI checks and commit statuses reach
                  a completed state (none pending).
  --ci-failed     Triggered when any CI check or commit status fails.

Multiple conditions are evaluated with OR logic — the rule triggers when
any one of the specified conditions is met.

Actions:
  By default, the trigger is logged to the server log. Use --notify to send
  an OS desktop notification, --open to open the PR in your default browser,
  or both together.

Termination:
  By default, a rule triggers once and stops. Use --count to allow
  multiple triggers, or --until to keep watching until a termination
  condition is met (e.g., --until merged). --until accepts the same
  condition names as the watch flags.

Polling:
  The server polls the GitHub API at the interval specified by --interval
  (default: 30sec). Accepts durations like 30sec, 5min, 1h.

Filtering:
  Use --ignore-user to exclude events from specific users. The value is
  a Go regular expression matched against the username. Can be specified
  multiple times.`,
	Example: `  # Watch the current branch's PR for approval
  gh wait pr --approved

  # Watch PR #42 for merge, open browser when merged
  gh wait pr 42 --merged --open

  # Watch PR for comments, stop when merged
  gh wait pr 42 --commented --until merged

  # Watch PR for CI completion with 1-minute polling
  gh wait pr 42 --ci-completed --interval 1min

  # Watch PR for approval, trigger up to 3 times, stop when closed
  gh wait pr 42 --approved --count 3 --until closed

  # Ignore bot users when watching for comments
  gh wait pr 42 --commented --ignore-user ".*\\[bot\\]" --ignore-user "dependabot"

  # Watch PR on a different repo
  gh wait pr 10 --approved --repo owner/repo`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		var number int
		if len(args) > 0 {
			n, err := strconv.Atoi(args[0])
			if err != nil {
				return fmt.Errorf("invalid PR number: %w", err)
			}
			number = n
		} else {
			n, err := detectCurrentPR()
			if err != nil {
				return fmt.Errorf("failed to detect PR for current branch (specify a PR number): %w", err)
			}
			number = n
		}

		return addWatchRule(cmd, "pr", number, prConditionFlags)
	},
}

func detectCurrentPR() (int, error) {
	stdout, _, err := gh.Exec("pr", "view", "--json", "number")
	if err != nil {
		return 0, fmt.Errorf("no PR found for current branch: %w", err)
	}
	var result struct {
		Number int `json:"number"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return 0, fmt.Errorf("failed to parse PR info: %w", err)
	}
	if result.Number == 0 {
		return 0, fmt.Errorf("no PR found for current branch")
	}
	return result.Number, nil
}

func init() {
	rootCmd.AddCommand(prCmd)
	registerWatchFlags(prCmd)
	prCmd.Flags().Bool("approved", false, "Watch for approval")
	prCmd.Flags().Bool("merged", false, "Watch for merge")
	prCmd.Flags().Bool("closed", false, "Watch for close")
	prCmd.Flags().Bool("commented", false, "Watch for new comments")
	prCmd.Flags().Bool("ci-completed", false, "Watch for CI completion")
	prCmd.Flags().Bool("ci-failed", false, "Watch for CI failure")
}
