package checker

import (
	"context"
	"errors"
	"net/http"

	"github.com/google/go-github/v83/github"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type PRChecker struct {
	client      *github.Client
	currentUser string
}

func NewPRChecker(client *github.Client, currentUser string) *PRChecker {
	return &PRChecker{client: client, currentUser: currentUser}
}

func (c *PRChecker) Check(ctx context.Context, r *rule.WatchRule) (bool, error) {
	return c.CheckConditions(ctx, r, r.Conditions)
}

func (c *PRChecker) CheckConditions(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error) {
	owner, repo := rule.SplitRepo(r.Repo)
	for _, cond := range conditions {
		matched, stateKey, err := c.checkCondition(ctx, owner, repo, r, cond)
		if err != nil {
			return false, err
		}
		if checkWithTransition(r, cond, matched, stateKey) {
			return true, nil
		}
	}
	return false, nil
}

func (c *PRChecker) CheckState(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error) {
	owner, repo := rule.SplitRepo(r.Repo)
	for _, cond := range conditions {
		matched, _, err := c.checkCondition(ctx, owner, repo, r, cond)
		if err != nil {
			return false, err
		}
		if matched {
			return true, nil
		}
	}
	return false, nil
}

// checkCondition returns (matched, stateKey, error).
// stateKey is empty for event-based conditions (commented) — they bypass transition tracking.
// stateKey is non-empty for state-based conditions — used to detect transitions.
func (c *PRChecker) checkCondition(ctx context.Context, owner, repo string, r *rule.WatchRule, cond string) (bool, string, error) {
	switch cond {
	case "approved":
		matched, err := c.checkApproved(ctx, owner, repo, r)
		return matched, "true", err
	case "merged":
		matched, err := c.checkMerged(ctx, owner, repo, r)
		return matched, "true", err
	case "closed":
		matched, err := checkClosed(ctx, c.client, c.currentUser, r.CompiledIgnoreUsers(), owner, repo, r.Number)
		return matched, "true", err
	case "ci-finished":
		return c.checkCIFinished(ctx, owner, repo, r.Number)
	case "ci-failed":
		return c.checkCIFailed(ctx, owner, repo, r.Number)
	case "commented":
		matched, err := c.checkCommented(ctx, owner, repo, r)
		return matched, "", err
	}
	return false, "", nil
}

func (c *PRChecker) checkApproved(ctx context.Context, owner, repo string, r *rule.WatchRule) (bool, error) {
	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, r.Number, nil)
	if err != nil {
		return false, skipNotFound(err)
	}
	for _, review := range reviews {
		if review.GetState() == "APPROVED" {
			if shouldIgnoreUser(c.currentUser, r.CompiledIgnoreUsers(), review.GetUser().GetLogin()) {
				continue
			}
			return true, nil
		}
	}
	return false, nil
}

func (c *PRChecker) checkMerged(ctx context.Context, owner, repo string, r *rule.WatchRule) (bool, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, r.Number)
	if err != nil {
		return false, skipNotFound(err)
	}
	if !pr.GetMerged() {
		return false, nil
	}
	if shouldIgnoreUser(c.currentUser, r.CompiledIgnoreUsers(), pr.GetMergedBy().GetLogin()) {
		return false, nil
	}
	return true, nil
}

// checkCIFinished returns (matched, sha, error). SHA is the stateKey for CI conditions.
func (c *PRChecker) checkCIFinished(ctx context.Context, owner, repo string, number int) (bool, string, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return false, "", skipNotFound(err)
	}
	sha := pr.GetHead().GetSHA()
	if sha == "" {
		return false, "", nil
	}

	combined, _, err := c.client.Repositories.GetCombinedStatus(ctx, owner, repo, sha, nil)
	if err != nil {
		return false, sha, skipNotFound(err)
	}
	// Only block on commit statuses if there are actual statuses.
	// When no statuses exist, GetCombinedStatus returns "pending" by default.
	if len(combined.Statuses) > 0 && combined.GetState() == "pending" {
		return false, sha, nil
	}

	opts := &github.ListCheckRunsOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		checkRuns, resp, err := c.client.Checks.ListCheckRunsForRef(ctx, owner, repo, sha, opts)
		if err != nil {
			return false, sha, skipNotFound(err)
		}
		for _, cr := range checkRuns.CheckRuns {
			if cr.GetStatus() != "completed" {
				return false, sha, nil
			}
		}
		// Require at least one status or check run to be present
		if opts.Page == 0 && len(combined.Statuses) == 0 && checkRuns.GetTotal() == 0 {
			return false, sha, nil
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return true, sha, nil
}

// checkCIFailed returns (matched, sha, error). SHA is the stateKey for CI conditions.
func (c *PRChecker) checkCIFailed(ctx context.Context, owner, repo string, number int) (bool, string, error) {
	pr, _, err := c.client.PullRequests.Get(ctx, owner, repo, number)
	if err != nil {
		return false, "", skipNotFound(err)
	}
	sha := pr.GetHead().GetSHA()
	if sha == "" {
		return false, "", nil
	}

	combined, _, err := c.client.Repositories.GetCombinedStatus(ctx, owner, repo, sha, nil)
	if err != nil {
		return false, sha, skipNotFound(err)
	}
	for _, s := range combined.Statuses {
		if s.GetState() == "failure" {
			return true, sha, nil
		}
	}

	opts := &github.ListCheckRunsOptions{ListOptions: github.ListOptions{PerPage: 100}}
	for {
		checkRuns, resp, err := c.client.Checks.ListCheckRunsForRef(ctx, owner, repo, sha, opts)
		if err != nil {
			return false, sha, skipNotFound(err)
		}
		for _, cr := range checkRuns.CheckRuns {
			if cr.GetConclusion() == "failure" {
				return true, sha, nil
			}
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	return false, sha, nil
}

func (c *PRChecker) checkCommented(ctx context.Context, owner, repo string, r *rule.WatchRule) (bool, error) {
	since := r.SinceTime()
	issueComments, _, err := c.client.Issues.ListComments(ctx, owner, repo, r.Number,
		&github.IssueListCommentsOptions{Since: &since})
	if err != nil {
		return false, skipNotFound(err)
	}
	for _, comment := range issueComments {
		if shouldIgnoreUser(c.currentUser, r.CompiledIgnoreUsers(), comment.GetUser().GetLogin()) {
			continue
		}
		return true, nil
	}

	reviewComments, _, err := c.client.PullRequests.ListComments(ctx, owner, repo, r.Number,
		&github.PullRequestListCommentsOptions{Since: since})
	if err != nil {
		return false, skipNotFound(err)
	}
	for _, comment := range reviewComments {
		if shouldIgnoreUser(c.currentUser, r.CompiledIgnoreUsers(), comment.GetUser().GetLogin()) {
			continue
		}
		return true, nil
	}

	reviews, _, err := c.client.PullRequests.ListReviews(ctx, owner, repo, r.Number, nil)
	if err != nil {
		return false, skipNotFound(err)
	}
	for _, review := range reviews {
		if review.GetSubmittedAt().After(since) && review.GetBody() != "" {
			if shouldIgnoreUser(c.currentUser, r.CompiledIgnoreUsers(), review.GetUser().GetLogin()) {
				continue
			}
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
