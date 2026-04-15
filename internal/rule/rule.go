package rule

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/k1LoW/duration"
)

type WatchRule struct {
	ID              string            `json:"id"`
	Type            string            `json:"type"` // "pr", "issue", "workflow"
	Repo            string            `json:"repo"` // "owner/repo"
	Number          int               `json:"number"`
	Conditions      []string          `json:"conditions"`        // OR evaluation
	Actions         []string          `json:"actions,omitempty"` // e.g. ["open"], ["notify"], ["open","notify"]
	URL             string            `json:"url"`
	CreatedAt       time.Time         `json:"created_at"`
	Status          string            `json:"status"`              // "watching", "triggered", "stopped"
	Until           []string          `json:"until,omitempty"`     // termination conditions (any match ends the rule)
	MaxCount        int               `json:"max_count,omitempty"` // 0=unlimited, N=end after N triggers
	TriggerCount    int               `json:"trigger_count"`       // current trigger count
	LastCheckedAt   time.Time         `json:"last_checked_at,omitzero"`
	LastTriggeredAt time.Time         `json:"last_triggered_at,omitzero"`
	Interval        string            `json:"interval,omitempty"`      // polling interval (e.g., "30sec", "5min", "1h")
	IgnoreUsers     []string          `json:"ignore_users,omitempty"`  // regex patterns of users to ignore
	FiredStates     map[string]string `json:"fired_states,omitempty"`  // state-based condition dedup (condition -> stateKey)
	SeededStates    map[string]string `json:"seeded_states,omitempty"` // states recorded during seeding (first check)

	Seeding bool `json:"-"` // transient: when true, state-based conditions seed without triggering

	ignoreUsersOnce     sync.Once        `json:"-"`
	compiledIgnoreUsers []*regexp.Regexp `json:"-"`
}

// CompiledIgnoreUsers returns pre-compiled regexps for IgnoreUsers patterns.
// Invalid patterns are silently skipped (validated at rule creation time).
func (r *WatchRule) CompiledIgnoreUsers() []*regexp.Regexp {
	r.ignoreUsersOnce.Do(func() {
		for _, pattern := range r.IgnoreUsers {
			re, err := regexp.Compile(pattern)
			if err != nil {
				continue
			}
			r.compiledIgnoreUsers = append(r.compiledIgnoreUsers, re)
		}
	})
	return r.compiledIgnoreUsers
}

// Clone returns a deep copy of the rule.
// Compiled ignore-user regexps are shared (read-only) to avoid recompilation.
func (r *WatchRule) Clone() *WatchRule {
	cp := &WatchRule{
		ID:              r.ID,
		Type:            r.Type,
		Repo:            r.Repo,
		Number:          r.Number,
		Actions:         slices.Clone(r.Actions),
		URL:             r.URL,
		CreatedAt:       r.CreatedAt,
		Status:          r.Status,
		MaxCount:        r.MaxCount,
		TriggerCount:    r.TriggerCount,
		LastCheckedAt:   r.LastCheckedAt,
		LastTriggeredAt: r.LastTriggeredAt,
		Interval:        r.Interval,
		Seeding:         r.Seeding,
	}
	if r.Conditions != nil {
		cp.Conditions = make([]string, len(r.Conditions))
		copy(cp.Conditions, r.Conditions)
	}
	if r.Until != nil {
		cp.Until = make([]string, len(r.Until))
		copy(cp.Until, r.Until)
	}
	if r.IgnoreUsers != nil {
		cp.IgnoreUsers = make([]string, len(r.IgnoreUsers))
		copy(cp.IgnoreUsers, r.IgnoreUsers)
	}
	if r.FiredStates != nil {
		cp.FiredStates = make(map[string]string, len(r.FiredStates))
		maps.Copy(cp.FiredStates, r.FiredStates)
	}
	if r.SeededStates != nil {
		cp.SeededStates = make(map[string]string, len(r.SeededStates))
		maps.Copy(cp.SeededStates, r.SeededStates)
	}
	// Share compiled regexps (read-only) if already populated.
	// Avoid calling CompiledIgnoreUsers() here as it mutates the source rule.
	if r.compiledIgnoreUsers != nil {
		cp.compiledIgnoreUsers = r.compiledIgnoreUsers
		cp.ignoreUsersOnce.Do(func() {})
	}
	return cp
}

// Label returns a human-readable short description of the rule target.
func (r *WatchRule) Label() string {
	if r.Type == "workflow" {
		return fmt.Sprintf("%s %s %d", r.Type, r.Repo, r.Number)
	}
	return fmt.Sprintf("%s %s#%d", r.Type, r.Repo, r.Number)
}

func GenerateID(typ, repo string, number int, conditions, until []string, maxCount int, ignoreUsers []string) string {
	sorted := make([]string, len(conditions))
	copy(sorted, conditions)
	sort.Strings(sorted)
	sortedUntil := make([]string, len(until))
	copy(sortedUntil, until)
	sort.Strings(sortedUntil)
	sortedIgnore := make([]string, len(ignoreUsers))
	copy(sortedIgnore, ignoreUsers)
	sort.Strings(sortedIgnore)
	key := fmt.Sprintf("%s:%s:%d:%s:until=%s:max=%d:ignore=%s", typ, repo, number, strings.Join(sorted, ","), strings.Join(sortedUntil, ","), maxCount, strings.Join(sortedIgnore, ","))
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h[:4])
}

// IsOneShot returns true if the rule should be removed after the first trigger (legacy behavior).
func (r *WatchRule) IsOneShot() bool {
	return r.MaxCount == 0 && len(r.Until) == 0
}

// SinceTime returns the appropriate "since" time for comment checks.
// Uses LastCheckedAt if set (continuous watch), otherwise CreatedAt.
func (r *WatchRule) SinceTime() time.Time {
	if !r.LastCheckedAt.IsZero() {
		return r.LastCheckedAt
	}
	return r.CreatedAt
}

const DefaultInterval = 1 * time.Minute
const DefaultIntervalStr = "1min"

// DefaultIntervalForType returns the default polling interval for the given rule type.
func DefaultIntervalForType(ruleType string) time.Duration {
	switch ruleType {
	case "issue", "discussion":
		return 30 * time.Minute
	default:
		return DefaultInterval
	}
}

// DefaultIntervalStrForType returns the default polling interval string for the given rule type.
func DefaultIntervalStrForType(ruleType string) string {
	switch ruleType {
	case "issue", "discussion":
		return "30min"
	default:
		return DefaultIntervalStr
	}
}

// PollInterval returns the rule's polling interval as time.Duration.
// Falls back to the type-specific default if not set or invalid.
func (r *WatchRule) PollInterval() time.Duration {
	if r.Interval == "" {
		return DefaultIntervalForType(r.Type)
	}
	d, err := duration.Parse(r.Interval)
	if err != nil {
		return DefaultIntervalForType(r.Type)
	}
	return d
}

// HasFiredForState returns true if the condition has already fired with the given stateKey.
func (r *WatchRule) HasFiredForState(condition, stateKey string) bool {
	if r.FiredStates == nil {
		return false
	}
	return r.FiredStates[condition] == stateKey
}

// RecordFiredState marks a condition as fired with the given stateKey.
func (r *WatchRule) RecordFiredState(condition, stateKey string) {
	if r.FiredStates == nil {
		r.FiredStates = make(map[string]string)
	}
	r.FiredStates[condition] = stateKey
}

// ClearFiredState removes a condition from fired states (e.g., state reverted).
func (r *WatchRule) ClearFiredState(condition string) {
	if r.FiredStates != nil {
		delete(r.FiredStates, condition)
	}
}

// RecordSeededState records a state observed during seeding (first check).
func (r *WatchRule) RecordSeededState(condition, stateKey string) {
	if r.SeededStates == nil {
		r.SeededStates = make(map[string]string)
	}
	r.SeededStates[condition] = stateKey
}

// IsSeededState returns true if the condition was seeded with the given stateKey.
func (r *WatchRule) IsSeededState(condition, stateKey string) bool {
	if r.SeededStates == nil {
		return false
	}
	return r.SeededStates[condition] == stateKey
}

// ClearSeededState removes a condition from seeded states after it has been consumed.
func (r *WatchRule) ClearSeededState(condition string) {
	if r.SeededStates != nil {
		delete(r.SeededStates, condition)
	}
}

func SplitRepo(repo string) (string, string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
