package checker

import (
	"context"

	"github.com/google/go-github/v83/github"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type WorkflowChecker struct {
	client      *github.Client
	currentUser string
}

func NewWorkflowChecker(client *github.Client, currentUser string) *WorkflowChecker {
	return &WorkflowChecker{client: client, currentUser: currentUser}
}

func (c *WorkflowChecker) Check(ctx context.Context, r *rule.WatchRule) (bool, error) {
	return c.CheckConditions(ctx, r, r.Conditions)
}

func (c *WorkflowChecker) CheckConditions(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error) {
	return evalConditions(ctx, r, conditions, c.checkCondition, true)
}

func (c *WorkflowChecker) CheckState(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error) {
	return evalConditions(ctx, r, conditions, c.checkCondition, false)
}

func (c *WorkflowChecker) checkCondition(ctx context.Context, owner, repo string, r *rule.WatchRule, cond string, _ bool) (bool, string, error) {
	run, _, err := c.client.Actions.GetWorkflowRunByID(ctx, owner, repo, int64(r.Number))
	if err != nil {
		return false, "", skipNotFound(err)
	}

	status := run.GetStatus()
	conclusion := run.GetConclusion()

	switch cond {
	case "completed":
		if status != "completed" {
			return false, "", nil
		}
		return true, conclusion, nil
	case "succeeded":
		matched := status == "completed" && conclusion == "success"
		if !matched {
			return false, "", nil
		}
		return true, "true", nil
	case "failed":
		matched := status == "completed" && conclusion == "failure"
		if !matched {
			return false, "", nil
		}
		return true, "true", nil
	}
	return false, "", nil
}
