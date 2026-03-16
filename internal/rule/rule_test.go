package rule

import (
	"testing"
)

func TestGenerateID(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		id1 := GenerateID("pr", "owner/repo", 1, []string{"approved"})
		id2 := GenerateID("pr", "owner/repo", 1, []string{"approved"})
		if id1 != id2 {
			t.Errorf("expected same ID, got %s and %s", id1, id2)
		}
	})

	t.Run("order independent", func(t *testing.T) {
		id1 := GenerateID("pr", "owner/repo", 1, []string{"approved", "merged"})
		id2 := GenerateID("pr", "owner/repo", 1, []string{"merged", "approved"})
		if id1 != id2 {
			t.Errorf("expected same ID regardless of condition order, got %s and %s", id1, id2)
		}
	})

	t.Run("different for different inputs", func(t *testing.T) {
		id1 := GenerateID("pr", "owner/repo", 1, []string{"approved"})
		id2 := GenerateID("pr", "owner/repo", 2, []string{"approved"})
		if id1 == id2 {
			t.Errorf("expected different IDs for different numbers")
		}

		id3 := GenerateID("issue", "owner/repo", 1, []string{"approved"})
		if id1 == id3 {
			t.Errorf("expected different IDs for different types")
		}
	})

	t.Run("8 hex chars", func(t *testing.T) {
		id := GenerateID("pr", "owner/repo", 1, []string{"approved"})
		if len(id) != 8 {
			t.Errorf("expected 8 char ID, got %d: %s", len(id), id)
		}
	})
}

func TestSplitRepo(t *testing.T) {
	tests := []struct {
		input     string
		wantOwner string
		wantRepo  string
	}{
		{"owner/repo", "owner", "repo"},
		{"k1LoW/gh-wait", "k1LoW", "gh-wait"},
		{"invalid", "", ""},
		{"", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			owner, repo := SplitRepo(tt.input)
			if owner != tt.wantOwner || repo != tt.wantRepo {
				t.Errorf("SplitRepo(%q) = (%q, %q), want (%q, %q)", tt.input, owner, repo, tt.wantOwner, tt.wantRepo)
			}
		})
	}
}
