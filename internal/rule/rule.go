package rule

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/k1LoW/duration"
)

type WatchRule struct {
	ID            string    `json:"id"`
	Type          string    `json:"type"`                    // "pr", "issue"
	Repo          string    `json:"repo"`                    // "owner/repo"
	Number        int       `json:"number"`
	Conditions    []string  `json:"conditions"`              // OR evaluation
	Action        string    `json:"action"`                  // "open", "notify"
	URL           string    `json:"url"`
	CreatedAt     time.Time `json:"created_at"`
	Status        string    `json:"status"`                  // "watching", "triggered", "stopped"
	Until         []string  `json:"until,omitempty"`         // termination conditions (any match ends the rule)
	MaxCount      int       `json:"max_count,omitempty"`     // 0=unlimited, N=end after N triggers
	TriggerCount  int       `json:"trigger_count"`           // current trigger count
	LastCheckedAt time.Time `json:"last_checked_at,omitzero"`
	Interval      string    `json:"interval,omitempty"`     // polling interval (e.g., "30sec", "5min", "1h")
	IgnoreUsers   []string  `json:"ignore_users,omitempty"` // regex patterns of users to ignore
}

func GenerateID(typ, repo string, number int, conditions, until []string, maxCount int) string {
	sorted := make([]string, len(conditions))
	copy(sorted, conditions)
	sort.Strings(sorted)
	sortedUntil := make([]string, len(until))
	copy(sortedUntil, until)
	sort.Strings(sortedUntil)
	key := fmt.Sprintf("%s:%s:%d:%s:until=%s:max=%d", typ, repo, number, strings.Join(sorted, ","), strings.Join(sortedUntil, ","), maxCount)
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

const DefaultInterval = 30 * time.Second
const DefaultIntervalStr = "30sec"

// PollInterval returns the rule's polling interval as time.Duration.
// Falls back to DefaultInterval if not set or invalid.
func (r *WatchRule) PollInterval() time.Duration {
	if r.Interval == "" {
		return DefaultInterval
	}
	d, err := duration.Parse(r.Interval)
	if err != nil {
		return DefaultInterval
	}
	return d
}

func SplitRepo(repo string) (string, string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
