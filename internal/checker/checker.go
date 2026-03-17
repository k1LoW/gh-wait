package checker

import (
	"context"
	"regexp"

	"github.com/google/go-github/v83/github"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type Checker interface {
	Check(ctx context.Context, r *rule.WatchRule) (bool, error)
	CheckConditions(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error)
	// CheckState checks whether any of the given conditions currently hold,
	// without state-transition tracking. Used for until (termination) conditions
	// that should match whenever the state is true, not only on transitions.
	CheckState(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error)
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

// checkWithTransition applies state-transition tracking to a condition check result.
// State-based conditions (non-empty stateKey) only trigger once per state transition.
// Event-based conditions (empty stateKey) always pass through.
func checkWithTransition(r *rule.WatchRule, cond string, matched bool, stateKey string) bool {
	if !matched {
		// State reverted (e.g., reopened, approval dismissed) — clear to allow re-trigger
		r.ClearFiredState(cond)
		return false
	}
	// Event-based conditions (empty stateKey) always fire
	if stateKey == "" {
		return true
	}
	// One-shot rules are removed after first trigger, no dedup needed
	if r.IsOneShot() {
		return true
	}
	// State-based: only trigger on transition (new stateKey)
	if r.HasFiredForState(cond, stateKey) {
		return false
	}
	r.RecordFiredState(cond, stateKey)
	return true
}

// checkClosed checks whether the issue/PR is closed by someone other than an ignored user.
func checkClosed(ctx context.Context, client *github.Client, currentUser string, compiled []*regexp.Regexp, owner, repo string, number int) (bool, error) {
	issue, _, err := client.Issues.Get(ctx, owner, repo, number)
	if err != nil {
		return false, skipNotFound(err)
	}
	if issue.GetState() != "closed" {
		return false, nil
	}
	if shouldIgnoreUser(currentUser, compiled, issue.GetClosedBy().GetLogin()) {
		return false, nil
	}
	return true, nil
}
