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
	owner, repo := rule.SplitRepo(r.Repo)
	for _, cond := range conditions {
		matched, err := c.checkCondition(ctx, owner, repo, r, cond)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

func (c *IssueChecker) checkCondition(ctx context.Context, owner, repo string, r *rule.WatchRule, cond string) (bool, error) {
	switch cond {
	case "commented":
		since := r.SinceTime()
		comments, _, err := c.client.Issues.ListComments(ctx, owner, repo, r.Number,
			&github.IssueListCommentsOptions{Since: &since})
		if err != nil {
			return false, skipNotFound(err)
		}
		for _, comment := range comments {
			if c.currentUser != "" && comment.GetUser().GetLogin() == c.currentUser {
				continue
			}
			return true, nil
		}
		return false, nil
	case "closed":
		issue, _, err := c.client.Issues.Get(ctx, owner, repo, r.Number)
		if err != nil {
			return false, skipNotFound(err)
		}
		if issue.GetState() != "closed" {
			return false, nil
		}
		if c.currentUser != "" && issue.GetClosedBy().GetLogin() == c.currentUser {
			return false, nil
		}
		return true, nil
	}
	return false, nil
}
