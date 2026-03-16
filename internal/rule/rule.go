package rule

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"
)

type WatchRule struct {
	ID         string    `json:"id"`
	Type       string    `json:"type"`       // "pr", "issue"
	Repo       string    `json:"repo"`       // "owner/repo"
	Number     int       `json:"number"`
	Conditions []string  `json:"conditions"` // OR evaluation
	Action     string    `json:"action"`     // "open", "notify"
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"created_at"`
	Status     string    `json:"status"` // "watching", "triggered", "stopped"
}

func GenerateID(typ, repo string, number int, conditions []string) string {
	sorted := make([]string, len(conditions))
	copy(sorted, conditions)
	sort.Strings(sorted)
	key := fmt.Sprintf("%s:%s:%d:%s", typ, repo, number, strings.Join(sorted, ","))
	h := sha256.Sum256([]byte(key))
	return fmt.Sprintf("%x", h[:4])
}

func SplitRepo(repo string) (string, string) {
	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
