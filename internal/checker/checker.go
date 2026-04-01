package checker

import (
	"context"
	"regexp"
	"time"

	"github.com/google/go-github/v83/github"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type Checker interface {
	Check(ctx context.Context, r *rule.WatchRule) (bool, bool, error)
	CheckConditions(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, bool, error)
	// CheckState checks whether any of the given conditions currently hold,
	// without state-transition tracking. Used for until (termination) conditions
	// that should match whenever the state is true, not only on transitions.
	CheckState(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, bool, error)
}

// shouldIgnoreUser reports whether login should be ignored based on
// the authenticated user and the rule's pre-compiled ignore patterns.
func shouldIgnoreUser(currentUser string, compiled []*regexp.Regexp, login string) bool {
	if login == "" {
		return false
	}
	if currentUser != "" && login == currentUser {
		return true
	}
	for _, re := range compiled {
		if re.MatchString(login) {
			return true
		}
	}
	return false
}

// checkConditionFunc is the signature of a per-checker condition evaluator.
// When skipUserFilter is true, shouldIgnoreUser checks are skipped (used for until/termination conditions).
type checkConditionFunc func(ctx context.Context, owner, repo string, r *rule.WatchRule, cond string, skipUserFilter bool) (matched bool, stateKey string, selfFiltered bool, err error)

// evalConditions iterates conditions, calling checkFn for each.
// When trackTransition is true, state-transition tracking is applied so that
// state-based conditions only fire once per transition. When false, the raw
// match result is used and user filtering is skipped (suitable for until/termination checks).
func evalConditions(ctx context.Context, r *rule.WatchRule, conditions []string, checkFn checkConditionFunc, trackTransition bool) (bool, bool, error) {
	skipUserFilter := !trackTransition
	owner, repo := rule.SplitRepo(r.Repo)
	anySelfFiltered := false
	for _, cond := range conditions {
		matched, stateKey, selfFiltered, err := checkFn(ctx, owner, repo, r, cond, skipUserFilter)
		if err != nil {
			return false, false, err
		}
		if trackTransition {
			matched = checkWithTransition(r, cond, matched, stateKey)
		}
		if matched {
			if !selfFiltered {
				return true, false, nil
			}
			anySelfFiltered = true
		}
	}
	if anySelfFiltered {
		return true, true, nil
	}
	return false, false, nil
}

// checkWithTransition applies state-transition tracking to a condition check result.
// State-based conditions (non-empty stateKey) only trigger once per state transition.
// Event-based conditions (empty stateKey) always pass through.
func checkWithTransition(r *rule.WatchRule, cond string, matched bool, stateKey string) bool {
	if !matched {
		// State reverted (e.g., reopened, approval dismissed) — clear to allow re-trigger
		r.ClearFiredState(cond)
		r.ClearSeededState(cond)
		return false
	}
	// Event-based conditions (empty stateKey) always fire
	if stateKey == "" {
		return true
	}
	// When seeding (first check), record state but don't trigger.
	// This prevents pre-existing states from causing false triggers.
	if r.Seeding {
		r.RecordSeededState(cond, stateKey)
		return false
	}
	// If this state was seeded with the same stateKey, treat it as
	// already-known and don't fire. This prevents pre-existing states
	// from triggering actions after the seeding (first) check.
	if r.IsSeededState(cond, stateKey) {
		r.ClearSeededState(cond)
		r.RecordFiredState(cond, stateKey)
		return false
	}
	// Seeding is over: clear any stale seeded state for this condition so that
	// it cannot suppress legitimate transitions in long-running tracking.
	r.ClearSeededState(cond)
	// State-based: only trigger on transition (new stateKey)
	if r.HasFiredForState(cond, stateKey) {
		return false
	}
	r.RecordFiredState(cond, stateKey)
	return true
}

// checkIssueCommented checks for issue comments (conversation thread) updated after since.
func checkIssueCommented(ctx context.Context, client *github.Client, currentUser string, compiled []*regexp.Regexp, owner, repo string, number int, since time.Time, skipUserFilter bool) (matched bool, selfFiltered bool, err error) {
	opts := &github.IssueListCommentsOptions{
		Since:       &since,
		ListOptions: github.ListOptions{PerPage: 100},
	}
	anySelfFiltered := false
	for {
		comments, resp, err := client.Issues.ListComments(ctx, owner, repo, number, opts)
		if err != nil {
			return false, false, skipNotFound(err)
		}
		for _, comment := range comments {
			if !skipUserFilter && shouldIgnoreUser(currentUser, compiled, comment.GetUser().GetLogin()) {
				anySelfFiltered = true
				continue
			}
			return true, false, nil
		}
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}
	if anySelfFiltered {
		return true, true, nil
	}
	return false, false, nil
}

// checkClosed checks whether the issue/PR is closed by someone other than an ignored user.
func checkClosed(ctx context.Context, client *github.Client, currentUser string, compiled []*regexp.Regexp, owner, repo string, number int, skipUserFilter bool) (matched bool, selfFiltered bool, err error) {
	issue, _, err := client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return false, false, skipNotFound(err)
	}
	if issue.GetState() != "closed" {
		return false, false, nil
	}
	if !skipUserFilter && shouldIgnoreUser(currentUser, compiled, issue.GetClosedBy().GetLogin()) {
		return true, true, nil
	}
	return true, false, nil
}
