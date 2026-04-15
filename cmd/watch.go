package cmd

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/repository"
	"github.com/k1LoW/duration"
	"github.com/k1LoW/gh-wait/internal/rule"
	"github.com/spf13/cobra"
)

// registerWatchFlags registers the common flags shared by all watch subcommands.
func registerWatchFlags(cmd *cobra.Command, ruleType string) {
	cmd.Flags().String("repo", "", "Repository (owner/repo)")
	cmd.Flags().String("url", "", "Override URL for the watch rule")
	cmd.Flags().Bool("open", false, "Open in browser when condition is met")
	cmd.Flags().Bool("notify", false, "Send OS notification when condition is met")
	cmd.Flags().StringSlice("until", nil, "Termination condition (e.g., closed, merged). Can be specified multiple times")
	cmd.Flags().Int("count", 0, "Maximum number of triggers (0 = unlimited)")
	cmd.Flags().StringSlice("ignore-user", nil, "Regex pattern of users to ignore (can be specified multiple times)")
	cmd.Flags().String("interval", rule.DefaultIntervalStrForType(ruleType), "Polling interval (e.g., 30sec, 5min, 1h)")
	// --url is set internally by transformURLArgs; hide it from help.
	_ = cmd.Flags().MarkHidden("url")
}

// resolveRepo returns the repo from the --repo flag, falling back to the
// current directory's git remote.
func resolveRepo(cmd *cobra.Command) (string, error) {
	repo, _ := cmd.Flags().GetString("repo")
	if repo != "" {
		return repo, nil
	}
	r, err := repository.Current()
	if err != nil {
		return "", fmt.Errorf("failed to detect repository (use --repo): %w", err)
	}
	return fmt.Sprintf("%s/%s", r.Owner, r.Name), nil
}

// parseWatchFlags reads and validates the common watch flags from the command.
func parseWatchFlags(cmd *cobra.Command, conditionFlags []string) (conditions []string, until []string, count int, ignoreUsers []string, interval string, actions []string, err error) {
	for _, flag := range conditionFlags {
		if v, _ := cmd.Flags().GetBool(flag); v {
			conditions = append(conditions, flag)
		}
	}

	until, _ = cmd.Flags().GetStringSlice("until")
	count, _ = cmd.Flags().GetInt("count")

	ignoreUsers, _ = cmd.Flags().GetStringSlice("ignore-user")
	for _, pattern := range ignoreUsers {
		if _, compileErr := regexp.Compile(pattern); compileErr != nil {
			return nil, nil, 0, nil, "", nil, fmt.Errorf("invalid --ignore-user pattern %q: %w", pattern, compileErr)
		}
	}

	interval, _ = cmd.Flags().GetString("interval")
	if _, parseErr := duration.Parse(interval); parseErr != nil {
		return nil, nil, 0, nil, "", nil, fmt.Errorf("invalid interval %q: %w", interval, parseErr)
	}

	if v, _ := cmd.Flags().GetBool("open"); v {
		actions = append(actions, "open")
	}
	if v, _ := cmd.Flags().GetBool("notify"); v {
		if err := checkNotifyDeps(); err != nil {
			return nil, nil, 0, nil, "", nil, err
		}
		actions = append(actions, "notify")
	}
	if len(actions) == 0 {
		actions = []string{"log"}
	}

	return conditions, until, count, ignoreUsers, interval, actions, nil
}

// buildWatchURL returns the URL for a watch rule. If --url is set (e.g. from
// a GitHub URL argument), it is used as-is. Otherwise the URL is constructed
// from the repo and number assuming github.com.
func buildWatchURL(cmd *cobra.Command, ruleType, repo string, number int) string {
	if u, _ := cmd.Flags().GetString("url"); u != "" {
		return u
	}
	owner, repoName := rule.SplitRepo(repo)
	if ruleType == "workflow" {
		return fmt.Sprintf("https://github.com/%s/%s/actions/runs/%d", owner, repoName, number)
	}
	var pathSegment string
	switch ruleType {
	case "pr":
		pathSegment = "pull"
	case "issue":
		pathSegment = "issues"
	case "discussion":
		pathSegment = "discussions"
	}
	return fmt.Sprintf("https://github.com/%s/%s/%s/%d", owner, repoName, pathSegment, number)
}

// addWatchRule builds a WatchRule, registers it with the server, and prints
// a confirmation message.
func addWatchRule(cmd *cobra.Command, ruleType string, number int, conditionFlags []string) error {
	repo, err := resolveRepo(cmd)
	if err != nil {
		return err
	}

	conditions, until, count, ignoreUsers, interval, actions, err := parseWatchFlags(cmd, conditionFlags)
	if err != nil {
		return err
	}

	if len(conditions) == 0 && len(until) == 0 {
		if ruleType == "workflow" {
			conditions = []string{"completed"}
		} else {
			return fmt.Errorf("at least one condition flag or --until is required (%s, --until)",
				"--"+strings.Join(conditionFlags, ", --"))
		}
	}

	url := buildWatchURL(cmd, ruleType, repo, number)

	id := rule.GenerateID(ruleType, repo, number, conditions, until, count, ignoreUsers)
	wr := &rule.WatchRule{
		ID:          id,
		Type:        ruleType,
		Repo:        repo,
		Number:      number,
		Conditions:  conditions,
		Actions:     actions,
		URL:         url,
		CreatedAt:   time.Now(),
		Status:      "watching",
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

	var typeLabel string
	switch ruleType {
	case "pr":
		typeLabel = "PR"
	case "issue":
		typeLabel = "Issue"
	case "discussion":
		typeLabel = "Discussion"
	case "workflow":
		typeLabel = "Workflow run"
	}
	actionSuffix := ""
	if len(actions) > 0 {
		actionSuffix = fmt.Sprintf(" (actions: %s)", strings.Join(actions, ", "))
	}
	if ruleType == "workflow" {
		fmt.Printf("Watching %s %d on %s for: %s%s\n", typeLabel, number, repo, strings.Join(conditions, ", "), actionSuffix)
	} else {
		fmt.Printf("Watching %s #%d on %s for: %s%s\n", typeLabel, number, repo, strings.Join(conditions, ", "), actionSuffix)
	}
	return nil
}
