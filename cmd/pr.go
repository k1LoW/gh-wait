package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/k1LoW/duration"
	"github.com/k1LoW/gh-wait/internal/rule"
	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:   "pr <number>",
	Short: "Watch a pull request for conditions",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		number, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("invalid PR number: %w", err)
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

		id := rule.GenerateID("pr", repo, number, conditions, until, count)
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
