package checker

import (
	"context"

	"github.com/google/go-github/v83/github"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type IssueChecker struct {
	client      *github.Client
	currentUser string
}

func NewIssueChecker(client *github.Client, currentUser string) *IssueChecker {
	return &IssueChecker{client: client, currentUser: currentUser}
}

func (c *IssueChecker) Check(ctx context.Context, r *rule.WatchRule) (bool, error) {
	return c.CheckConditions(ctx, r, r.Conditions)
}

func (c *IssueChecker) CheckConditions(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error) {
	return evalConditions(ctx, r, conditions, c.checkCondition, true)
}

func (c *IssueChecker) CheckState(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error) {
	return evalConditions(ctx, r, conditions, c.checkCondition, false)
}

func (c *IssueChecker) checkCondition(ctx context.Context, owner, repo string, r *rule.WatchRule, cond string, skipUserFilter bool) (bool, string, error) {
	switch cond {
	case "commented":
		since := r.SinceTime()
		matched, err := checkIssueCommented(ctx, c.client, c.currentUser, r.CompiledIgnoreUsers(), owner, repo, r.Number, since, skipUserFilter)
		return matched, "", err
	case "closed":
		matched, err := checkClosed(ctx, c.client, c.currentUser, r.CompiledIgnoreUsers(), owner, repo, r.Number, skipUserFilter)
		return matched, "true", err
	}
	return false, "", nil
}
