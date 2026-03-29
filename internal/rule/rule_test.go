package rule

import (
	"testing"
	"time"
)

func TestGenerateID(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		id1 := GenerateID("pr", "owner/repo", 1, []string{"approved"}, nil, 0, nil)
		id2 := GenerateID("pr", "owner/repo", 1, []string{"approved"}, nil, 0, nil)
		if id1 != id2 {
			t.Errorf("expected same ID, got %s and %s", id1, id2)
		}
	})

	t.Run("order independent", func(t *testing.T) {
		id1 := GenerateID("pr", "owner/repo", 1, []string{"approved", "merged"}, nil, 0, nil)
		id2 := GenerateID("pr", "owner/repo", 1, []string{"merged", "approved"}, nil, 0, nil)
		if id1 != id2 {
			t.Errorf("expected same ID regardless of condition order, got %s and %s", id1, id2)
		}
	})

	t.Run("different for different inputs", func(t *testing.T) {
		id1 := GenerateID("pr", "owner/repo", 1, []string{"approved"}, nil, 0, nil)
		id2 := GenerateID("pr", "owner/repo", 2, []string{"approved"}, nil, 0, nil)
		if id1 == id2 {
			t.Errorf("expected different IDs for different numbers")
		}

		id3 := GenerateID("issue", "owner/repo", 1, []string{"approved"}, nil, 0, nil)
		if id1 == id3 {
			t.Errorf("expected different IDs for different types")
		}
	})

	t.Run("8 hex chars", func(t *testing.T) {
		id := GenerateID("pr", "owner/repo", 1, []string{"approved"}, nil, 0, nil)
		if len(id) != 8 {
			t.Errorf("expected 8 char ID, got %d: %s", len(id), id)
		}
	})

	t.Run("different with until", func(t *testing.T) {
		id1 := GenerateID("pr", "owner/repo", 1, []string{"commented"}, nil, 0, nil)
		id2 := GenerateID("pr", "owner/repo", 1, []string{"commented"}, []string{"closed"}, 0, nil)
		if id1 == id2 {
			t.Errorf("expected different IDs when until differs")
		}
	})

	t.Run("different with maxCount", func(t *testing.T) {
		id1 := GenerateID("pr", "owner/repo", 1, []string{"commented"}, []string{"closed"}, 0, nil)
		id2 := GenerateID("pr", "owner/repo", 1, []string{"commented"}, []string{"closed"}, 3, nil)
		if id1 == id2 {
			t.Errorf("expected different IDs when maxCount differs")
		}
	})

	t.Run("until order independent", func(t *testing.T) {
		id1 := GenerateID("pr", "owner/repo", 1, []string{"commented"}, []string{"closed", "merged"}, 0, nil)
		id2 := GenerateID("pr", "owner/repo", 1, []string{"commented"}, []string{"merged", "closed"}, 0, nil)
		if id1 != id2 {
			t.Errorf("expected same ID regardless of until order, got %s and %s", id1, id2)
		}
	})
}

func TestIsOneShot(t *testing.T) {
	tests := []struct {
		name     string
		rule     *WatchRule
		expected bool
	}{
		{"default oneshot", &WatchRule{}, true},
		{"with until", &WatchRule{Until: []string{"closed"}}, false},
		{"with maxCount", &WatchRule{MaxCount: 3}, false},
		{"with both", &WatchRule{Until: []string{"closed"}, MaxCount: 3}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rule.IsOneShot(); got != tt.expected {
				t.Errorf("IsOneShot() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestSinceTime(t *testing.T) {
	created := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	lastChecked := time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC)

	t.Run("uses CreatedAt when LastCheckedAt is zero", func(t *testing.T) {
		r := &WatchRule{CreatedAt: created}
		if got := r.SinceTime(); !got.Equal(created) {
			t.Errorf("SinceTime() = %v, want %v", got, created)
		}
	})

	t.Run("uses LastCheckedAt when set", func(t *testing.T) {
		r := &WatchRule{CreatedAt: created, LastCheckedAt: lastChecked}
		if got := r.SinceTime(); !got.Equal(lastChecked) {
			t.Errorf("SinceTime() = %v, want %v", got, lastChecked)
		}
	})
}

func TestClone(t *testing.T) {
	original := &WatchRule{
		ID:              "r1",
		Type:            "pr",
		Repo:            "owner/repo",
		Number:          42,
		Conditions:      []string{"approved", "merged"},
		Actions:         []string{"open"},
		URL:             "https://github.com/owner/repo/pull/42",
		CreatedAt:       time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Status:          "watching",
		Until:           []string{"closed"},
		MaxCount:        3,
		TriggerCount:    1,
		LastCheckedAt:   time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		LastTriggeredAt: time.Date(2025, 1, 2, 0, 0, 0, 0, time.UTC),
		Interval:        "5min",
		IgnoreUsers:     []string{"bot-.*", "dependabot"},
		FiredStates:     map[string]string{"approved": "user1"},
	}

	cp := original.Clone()

	t.Run("scalar fields match", func(t *testing.T) {
		if cp.ID != original.ID || cp.Type != original.Type || cp.Repo != original.Repo ||
			cp.Number != original.Number || cp.URL != original.URL ||
			cp.Status != original.Status || cp.MaxCount != original.MaxCount ||
			cp.TriggerCount != original.TriggerCount || cp.Interval != original.Interval ||
			!cp.CreatedAt.Equal(original.CreatedAt) || !cp.LastCheckedAt.Equal(original.LastCheckedAt) ||
			!cp.LastTriggeredAt.Equal(original.LastTriggeredAt) {
			t.Error("scalar fields do not match")
		}
	})

	t.Run("slices are deep copies", func(t *testing.T) {
		cp.Conditions[0] = "changed"
		if original.Conditions[0] == "changed" {
			t.Error("mutating clone's Conditions affected original")
		}
		cp.Until[0] = "changed"
		if original.Until[0] == "changed" {
			t.Error("mutating clone's Until affected original")
		}
		cp.IgnoreUsers[0] = "changed"
		if original.IgnoreUsers[0] == "changed" {
			t.Error("mutating clone's IgnoreUsers affected original")
		}
	})

	t.Run("map is a deep copy", func(t *testing.T) {
		cp.FiredStates["approved"] = "user2"
		if original.FiredStates["approved"] != "user1" {
			t.Error("mutating clone's FiredStates affected original")
		}
	})

	t.Run("compiled regexps are shared", func(t *testing.T) {
		// Trigger compilation on original first
		origCompiled := original.CompiledIgnoreUsers()
		cp2 := original.Clone()
		cp2Compiled := cp2.CompiledIgnoreUsers()
		if len(origCompiled) != len(cp2Compiled) {
			t.Fatalf("expected %d compiled regexps, got %d", len(origCompiled), len(cp2Compiled))
		}
		for i := range origCompiled {
			if origCompiled[i] != cp2Compiled[i] {
				t.Errorf("expected shared regexp pointer at index %d", i)
			}
		}
	})

	t.Run("nil fields stay nil", func(t *testing.T) {
		minimal := &WatchRule{ID: "r2"}
		minCp := minimal.Clone()
		if minCp.Conditions != nil || minCp.Until != nil || minCp.IgnoreUsers != nil || minCp.FiredStates != nil {
			t.Error("nil slice/map fields should remain nil in clone")
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
