package cmd

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/k1LoW/duration"
	"github.com/k1LoW/gh-wait/internal/rule"
	"github.com/spf13/cobra"
)

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
  By default, a desktop notification is sent. Use --open to instead open
  the issue in your default browser.

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

		repo, _ := cmd.Flags().GetString("repo")
		if repo == "" {
			r, err := repository.Current()
			if err != nil {
				return fmt.Errorf("failed to detect repository (use --repo): %w", err)
			}
			repo = fmt.Sprintf("%s/%s", r.Owner, r.Name)
		}

		var conditions []string
		for _, flag := range []string{"commented", "closed"} {
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
			return fmt.Errorf("at least one condition flag or --until is required (--commented, --closed, --until)")
		}

		actionFlag := "notify"
		if v, _ := cmd.Flags().GetBool("open"); v {
			actionFlag = "open"
		}

		owner, repoName := rule.SplitRepo(repo)
		url := fmt.Sprintf("https://github.com/%s/%s/issues/%d", owner, repoName, number)

		id := rule.GenerateID("issue", repo, number, conditions, until, count, ignoreUsers)
		wr := &rule.WatchRule{
			ID:         id,
			Type:       "issue",
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

		fmt.Printf("Watching Issue #%d on %s for: %s (action: %s)\n", number, repo, strings.Join(conditions, ", "), actionFlag)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(issueCmd)
	issueCmd.Flags().String("repo", "", "Repository (owner/repo)")
	issueCmd.Flags().Bool("commented", false, "Watch for new comments")
	issueCmd.Flags().Bool("closed", false, "Watch for close")
	issueCmd.Flags().Bool("open", false, "Open in browser when condition is met")
	issueCmd.Flags().StringSlice("until", nil, "Termination condition (e.g., closed). Can be specified multiple times")
	issueCmd.Flags().Int("count", 0, "Maximum number of triggers (0 = unlimited)")
	issueCmd.Flags().StringSlice("ignore-user", nil, "Regex pattern of users to ignore (can be specified multiple times)")
	issueCmd.Flags().String("interval", rule.DefaultIntervalStr, "Polling interval (e.g., 30sec, 5min, 1h)")
}
