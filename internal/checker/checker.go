package checker

import (
	"context"
	"log/slog"
	"regexp"

	"github.com/google/go-github/v83/github"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type Checker interface {
	Check(ctx context.Context, r *rule.WatchRule) (bool, error)
	CheckConditions(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error)
}

// shouldIgnoreUser reports whether login should be ignored based on
// the authenticated user and the rule's ignore patterns.
func shouldIgnoreUser(currentUser string, ignoreUsers []string, login string) bool {
	if login == "" {
		return false
	}
	if currentUser != "" && login == currentUser {
		return true
	}
	for _, pattern := range ignoreUsers {
		re, err := regexp.Compile(pattern)
		if err != nil {
			slog.Warn("invalid ignore-users pattern", "pattern", pattern, "error", err)
			continue
		}
		if re.MatchString(login) {
			return true
		}
	}
	return false
}

// checkClosed checks whether the issue/PR is closed by someone other than an ignored user.
func checkClosed(client *github.Client, currentUser string, ignoreUsers []string, ctx context.Context, owner, repo string, number int) (bool, error) {
	issue, _, err := client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return false, skipNotFound(err)
	}
	if issue.GetState() != "closed" {
		return false, nil
	}
	if shouldIgnoreUser(currentUser, ignoreUsers, issue.GetClosedBy().GetLogin()) {
		return false, nil
	}
	return true, nil
}
