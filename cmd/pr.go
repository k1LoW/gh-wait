package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2"
	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/k1LoW/duration"
	"github.com/k1LoW/gh-wait/internal/rule"
	"github.com/spf13/cobra"
)

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
  --ci-finished   Triggered when all CI checks and commit statuses reach
                  a completed state (none pending).
  --ci-failed     Triggered when any CI check or commit status fails.

Multiple conditions are evaluated with OR logic — the rule triggers when
any one of the specified conditions is met.

Actions:
  By default, a desktop notification is sent. Use --open to instead open
  the PR in your default browser.

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
  gh wait pr 42 --ci-finished --interval 1min

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

		repo, _ := cmd.Flags().GetString("repo")
		if repo == "" {
			r, err := repository.Current()
			if err != nil {
				return fmt.Errorf("failed to detect repository (use --repo): %w", err)
			}
			repo = fmt.Sprintf("%s/%s", r.Owner, r.Name)
		}

		var conditions []string
		for _, flag := range []string{"approved", "merged", "closed", "commented", "ci-finished", "ci-failed"} {
			if v, _ := cmd.Flags().GetBool(flag); v {
				conditions = append(conditions, flag)
			}
		}
		until, _ := cmd.Flags().GetStringSlice("until")
		count, _ := cmd.Flags().GetInt("count")
		ignoreUsers, _ := cmd.Flags().GetStringSlice("ignore-user")
		for _, pattern := range ignoreUsers {
			if _, err := regexp.Compile(pattern); err != nil {
				return fmt.Errorf("invalid --ignore-user pattern %q: %w", pattern, err)
			}
		}
		interval, _ := cmd.Flags().GetString("interval")
		if _, err := duration.Parse(interval); err != nil {
			return fmt.Errorf("invalid interval %q: %w", interval, err)
		}

		if len(conditions) == 0 && len(until) == 0 {
			return fmt.Errorf("at least one condition flag or --until is required (--approved, --merged, --closed, --commented, --ci-finished, --ci-failed, --until)")
		}

		actionFlag := "notify"
		if v, _ := cmd.Flags().GetBool("open"); v {
			actionFlag = "open"
		}

		owner, repoName := rule.SplitRepo(repo)
		url := fmt.Sprintf("https://github.com/%s/%s/pull/%d", owner, repoName, number)

		id := rule.GenerateID("pr", repo, number, conditions, until, count, ignoreUsers)
		wr := &rule.WatchRule{
			ID:         id,
			Type:       "pr",
			Repo:       repo,
			Number:     number,
			Conditions: conditions,
			Action:     actionFlag,
			URL:        url,
			CreatedAt:  time.Now(),
			Status:     "watching",
			Until:       until,
			MaxCount:    count,
			IgnoreUsers: ignoreUsers,
			Interval:    interval,
		}

		if err := ensureServer(); err != nil {
			return err
		}

		c := newClient()
		if err := c.AddRule(wr); err != nil {
			return fmt.Errorf("failed to add rule: %w", err)
		}

		fmt.Printf("Watching PR #%d on %s for: %s (action: %s)\n", number, repo, strings.Join(conditions, ", "), actionFlag)
		return nil
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
	prCmd.Flags().String("repo", "", "Repository (owner/repo)")
	prCmd.Flags().Bool("approved", false, "Watch for approval")
	prCmd.Flags().Bool("merged", false, "Watch for merge")
	prCmd.Flags().Bool("closed", false, "Watch for close")
	prCmd.Flags().Bool("commented", false, "Watch for new comments")
	prCmd.Flags().Bool("ci-finished", false, "Watch for CI completion")
	prCmd.Flags().Bool("ci-failed", false, "Watch for CI failure")
	prCmd.Flags().Bool("open", false, "Open in browser when condition is met")
	prCmd.Flags().StringSlice("until", nil, "Termination condition (e.g., closed, merged). Can be specified multiple times")
	prCmd.Flags().Int("count", 0, "Maximum number of triggers (0 = unlimited)")
	prCmd.Flags().StringSlice("ignore-user", nil, "Regex pattern of users to ignore (can be specified multiple times)")
	prCmd.Flags().String("interval", rule.DefaultIntervalStr, "Polling interval (e.g., 30sec, 5min, 1h)")
}
