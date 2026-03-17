package checker

import (
	"context"

	"github.com/google/go-github/v83/github"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type Checker interface {
	Check(ctx context.Context, r *rule.WatchRule) (bool, error)
	CheckConditions(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error)
}

// isSelf reports whether login belongs to the current authenticated user.
func isSelf(currentUser, login string) bool {
	return currentUser != "" && login == currentUser
}

// checkClosed checks whether the issue/PR is closed by someone other than the current user.
func checkClosed(client *github.Client, currentUser string, ctx context.Context, owner, repo string, number int) (bool, error) {
	issue, _, err := client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return false, skipNotFound(err)
	}
	if issue.GetState() != "closed" {
		return false, nil
	}
	if isSelf(currentUser, issue.GetClosedBy().GetLogin()) {
		return false, nil
	}
	return true, nil
}
