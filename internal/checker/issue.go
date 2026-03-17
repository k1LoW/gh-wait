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
		comments, _, err := c.client.Issues.ListComments(ctx, owner, repo, r.Number,
			&github.IssueListCommentsOptions{Since: &since})
		if err != nil {
			return false, "", skipNotFound(err)
		}
		for _, comment := range comments {
			if !skipUserFilter && shouldIgnoreUser(c.currentUser, r.CompiledIgnoreUsers(), comment.GetUser().GetLogin()) {
				continue
			}
			return true, "", nil
		}
		return false, "", nil
	case "closed":
		matched, err := checkClosed(ctx, c.client, c.currentUser, r.CompiledIgnoreUsers(), owner, repo, r.Number, skipUserFilter)
		return matched, "true", err
	}
	return false, "", nil
}
