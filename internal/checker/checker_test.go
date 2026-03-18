package checker

import (
	"testing"

	"github.com/k1LoW/gh-wait/internal/rule"
)

func TestCheckWithTransitionStateBased(t *testing.T) {
	r := &rule.WatchRule{ID: "r1"}

	// First match with stateKey records state and returns true
	if got := checkWithTransition(r, "approved", true, "true"); !got {
		t.Error("expected true on first state match")
	}
	if !r.HasFiredForState("approved", "true") {
		t.Error("expected state to be recorded")
	}

	// Same stateKey again should be deduped
	if got := checkWithTransition(r, "approved", true, "true"); got {
		t.Error("expected false on duplicate state (dedup)")
	}

	// State reverts (not matched) → clears fired state
	if got := checkWithTransition(r, "approved", false, ""); got {
		t.Error("expected false when not matched")
	}
	if r.HasFiredForState("approved", "true") {
		t.Error("expected fired state to be cleared after revert")
	}

	// State transitions back (false→true) → should fire again
	if got := checkWithTransition(r, "approved", true, "true"); !got {
		t.Error("expected true on re-transition (false→true)")
	}
}

func TestCheckWithTransitionDifferentStateKey(t *testing.T) {
	r := &rule.WatchRule{ID: "r1"}

	// First match with sha1
	if got := checkWithTransition(r, "ci-finished", true, "sha1"); !got {
		t.Error("expected true on first state match")
	}

	// Different stateKey (new commit pushed) → should fire
	if got := checkWithTransition(r, "ci-finished", true, "sha2"); !got {
		t.Error("expected true on new stateKey (state transition)")
	}

	// Same stateKey sha2 again → deduped
	if got := checkWithTransition(r, "ci-finished", true, "sha2"); got {
		t.Error("expected false on duplicate stateKey")
	}
}

func TestCheckWithTransitionEventBased(t *testing.T) {
	r := &rule.WatchRule{ID: "r1"}

	// Event-based (empty stateKey) always passes through
	if got := checkWithTransition(r, "commented", true, ""); !got {
		t.Error("expected true for event-based condition")
	}
	if got := checkWithTransition(r, "commented", true, ""); !got {
		t.Error("expected true for repeated event-based condition")
	}

	// Not matched → returns false
	if got := checkWithTransition(r, "commented", false, ""); got {
		t.Error("expected false when event not matched")
	}
}

func TestCheckWithTransitionSeeding(t *testing.T) {
	t.Run("state-based: seeding records state but returns false", func(t *testing.T) {
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		got := checkWithTransition(r, "approved", true, "true")
		if got {
			t.Error("expected false when seeding state-based condition")
		}
		if !r.HasFiredForState("approved", "true") {
			t.Error("expected state to be recorded even during seeding")
		}

		// After seeding, same state should still be deduped
		r.Seeding = false
		got = checkWithTransition(r, "approved", true, "true")
		if got {
			t.Error("expected false on same state after seeding (dedup)")
		}
	})

	t.Run("state-based: fires after seeding when state transitions", func(t *testing.T) {
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		// Seed with current state
		checkWithTransition(r, "approved", true, "true")
		r.Seeding = false

		// State reverts
		checkWithTransition(r, "approved", false, "")

		// State transitions back → should fire
		got := checkWithTransition(r, "approved", true, "true")
		if !got {
			t.Error("expected true after false→true transition post-seeding")
		}
	})

	t.Run("event-based: seeding does not suppress", func(t *testing.T) {
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		got := checkWithTransition(r, "commented", true, "")
		if !got {
			t.Error("expected true for event-based condition even during seeding")
		}
	})
}
