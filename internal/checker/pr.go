package checker

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/go-github/v83/github"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type PRChecker struct {
	client *github.Client
}

func NewPRChecker(client *github.Client) *PRChecker {
	return &PRChecker{client: client}
}

func (c *PRChecker) Check(ctx context.Context, r *rule.WatchRule) (bool, error) {
	owner, repo := rule.SplitRepo(r.Repo)
	for _, cond := range r.Conditions {
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

func (c *PRChecker) checkCondition(ctx context.Context, owner, repo string, r *rule.WatchRule, cond string) (bool, error) {
	switch cond {
	case "approved":
		return c.checkApproved(ctx, owner, repo, r.Number)
	case "merged":
		return c.checkMerged(ctx, owner, repo, r.Number)
	case "closed":
		return c.checkClosed(ctx, owner, repo, r.Number)
	case "ci-finished":
		return c.checkCIFinished(ctx, owner, repo, r.Number)
	case "ci-failed":
		return c.checkCIFailed(ctx, owner, repo, r.Number)
	case "commented":
		return c.checkCommented(ctx, owner, repo, r)
	}
	return false, nil
}

func (c *PRChecker) checkApproved(ctx context.Context, owner, repo string, number int) (bool, error) {
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, number, nil)
	if err != nil {
		return false, skipNotFound(err)
	}
	for _, review := range reviews {
		if review.GetState() == "APPROVED" {
			return true, nil
		}
	}
	return false, nil
}

func (c *PRChecker) checkMerged(ctx context.Context, owner, repo string, number int) (bool, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return false, skipNotFound(err)
	}
	return pr.GetMerged(), nil
}

func (c *PRChecker) checkClosed(ctx context.Context, owner, repo string, number int) (bool, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return false, skipNotFound(err)
	}
	return pr.GetState() == "closed", nil
}

func (c *PRChecker) checkCIFinished(ctx context.Context, owner, repo string, number int) (bool, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return false, skipNotFound(err)
	}
	sha := pr.GetHead().GetSHA()
	if sha == "" {
		return false, nil
	}

	combined, _, err := c.client.Repositories.GetCombinedStatus(ctx, owner, repo, sha, nil)
	if err != nil {
		return false, skipNotFound(err)
	}
	if combined.GetState() == "pending" {
		return false, nil
	}

	checkRuns, _, err := c.client.Checks.ListCheckRunsForRef(ctx, owner, repo, sha, nil)
	if err != nil {
		return false, skipNotFound(err)
	}
	for _, cr := range checkRuns.CheckRuns {
		if cr.GetStatus() != "completed" {
			return false, nil
		}
	}
	return true, nil
}

func (c *PRChecker) checkCIFailed(ctx context.Context, owner, repo string, number int) (bool, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return false, skipNotFound(err)
	}
	sha := pr.GetHead().GetSHA()
	if sha == "" {
		return false, nil
	}

	combined, _, err := c.client.Repositories.GetCombinedStatus(ctx, owner, repo, sha, nil)
	if err != nil {
		return false, skipNotFound(err)
	}
	for _, s := range combined.Statuses {
		if s.GetState() == "failure" {
			return true, nil
		}
	}

	checkRuns, _, err := c.client.Checks.ListCheckRunsForRef(ctx, owner, repo, sha, nil)
	if err != nil {
		return false, skipNotFound(err)
	}
	for _, cr := range checkRuns.CheckRuns {
		if cr.GetConclusion() == "failure" {
			return true, nil
		}
	}
	return false, nil
}

func (c *PRChecker) checkCommented(ctx context.Context, owner, repo string, r *rule.WatchRule) (bool, error) {
	since := r.CreatedAt
	issueComments, _, err := c.client.Issues.ListComments(ctx, owner, repo, r.Number,
		&github.IssueListCommentsOptions{Since: &since})
	if err != nil {
		return false, skipNotFound(err)
	}
	if len(issueComments) > 0 {
		return true, nil
	}

	reviewComments, _, err := c.client.PullRequests.ListComments(ctx, owner, repo, r.Number,
		&github.PullRequestListCommentsOptions{Since: since})
	if err != nil {
		return false, skipNotFound(err)
	}
	if len(reviewComments) > 0 {
		return true, nil
	}

	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, r.Number, nil)
	if err != nil {
		return false, skipNotFound(err)
	}
	for _, review := range reviews {
		if review.GetSubmittedAt().After(since) && review.GetBody() != "" {
			return true, nil
		}
	}
	return false, nil
}

func skipNotFound(err error) error {
	var errResp *github.ErrorResponse
	if errors.As(err, &errResp) && errResp.Response.StatusCode == http.StatusNotFound {
		return nil
	}
	return err
}
