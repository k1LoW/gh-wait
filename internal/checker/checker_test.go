package checker

import (
	"testing"

	"github.com/k1LoW/gh-wait/internal/rule"
)

func TestCheckWithTransitionStateBased(t *testing.T) {
	r := &rule.WatchRule{ID: "r1"}

	// First match with stateKey records state and returns true
	if got := checkWithTransition(r, "approved", true, "true", false); !got {
		t.Error("expected true on first state match")
	}
	if !r.HasFiredForState("approved", "true") {
		t.Error("expected state to be recorded")
	}

	// Same stateKey again should be deduped
	if got := checkWithTransition(r, "approved", true, "true", false); got {
		t.Error("expected false on duplicate state (dedup)")
	}

	// State reverts (not matched) → clears fired state
	if got := checkWithTransition(r, "approved", false, "", false); got {
		t.Error("expected false when not matched")
	}
	if r.HasFiredForState("approved", "true") {
		t.Error("expected fired state to be cleared after revert")
	}

	// State transitions back (false→true) → should fire again
	if got := checkWithTransition(r, "approved", true, "true", false); !got {
		t.Error("expected true on re-transition (false→true)")
	}
}

func TestCheckWithTransitionDifferentStateKey(t *testing.T) {
	r := &rule.WatchRule{ID: "r1"}

	// First match with sha1
	if got := checkWithTransition(r, "ci-completed", true, "sha1", false); !got {
		t.Error("expected true on first state match")
	}

	// Different stateKey (new commit pushed) → should fire
	if got := checkWithTransition(r, "ci-completed", true, "sha2", false); !got {
		t.Error("expected true on new stateKey (state transition)")
	}

	// Same stateKey sha2 again → deduped
	if got := checkWithTransition(r, "ci-completed", true, "sha2", false); got {
		t.Error("expected false on duplicate stateKey")
	}
}

func TestCheckWithTransitionEventBased(t *testing.T) {
	r := &rule.WatchRule{ID: "r1"}

	// Event-based (empty stateKey) always passes through
	if got := checkWithTransition(r, "commented", true, "", false); !got {
		t.Error("expected true for event-based condition")
	}
	if got := checkWithTransition(r, "commented", true, "", false); !got {
		t.Error("expected true for repeated event-based condition")
	}

	// Not matched → returns false
	if got := checkWithTransition(r, "commented", false, "", false); got {
		t.Error("expected false when event not matched")
	}
}

func TestCheckWithTransitionSeeding(t *testing.T) {
	t.Run("state-based: seeding records state but returns false", func(t *testing.T) {
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		got := checkWithTransition(r, "approved", true, "true", false)
		if got {
			t.Error("expected false when seeding state-based condition")
		}
		if !r.IsSeededState("approved", "true") {
			t.Error("expected state to be recorded in SeededStates during seeding")
		}

		// After seeding, same state should NOT fire (pre-existing state is already known)
		r.Seeding = false
		got = checkWithTransition(r, "approved", true, "true", false)
		if got {
			t.Error("expected false on first non-seeding check with seeded state (pre-existing)")
		}

		// Subsequent check with same state should also be deduped
		got = checkWithTransition(r, "approved", true, "true", false)
		if got {
			t.Error("expected false on subsequent check with same state (dedup)")
		}
	})

	t.Run("state-based: fires after seeding when state transitions", func(t *testing.T) {
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		// Seed with current state
		checkWithTransition(r, "approved", true, "true", false)
		r.Seeding = false

		// State reverts
		checkWithTransition(r, "approved", false, "", false)

		// State transitions back → should fire
		got := checkWithTransition(r, "approved", true, "true", false)
		if !got {
			t.Error("expected true after false→true transition post-seeding")
		}
	})

	t.Run("event-based: seeding does not suppress", func(t *testing.T) {
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		got := checkWithTransition(r, "commented", true, "", false)
		if !got {
			t.Error("expected true for event-based condition even during seeding")
		}
	})

	t.Run("state-based: pre-existing state with multiple conditions does not fire", func(t *testing.T) {
		// Simulates: gh wait <PR> --approved --ci-completed --open
		// where PR is already approved and CI already completed at rule creation.
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		// Seed both conditions (first check)
		checkWithTransition(r, "approved", true, "true", false)
		checkWithTransition(r, "ci-completed", true, "sha1", false)
		r.Seeding = false

		// Second check: both conditions still hold but should NOT fire
		if got := checkWithTransition(r, "approved", true, "true", false); got {
			t.Error("expected false for pre-existing approved state after seeding")
		}
		if got := checkWithTransition(r, "ci-completed", true, "sha1", false); got {
			t.Error("expected false for pre-existing ci-completed state after seeding")
		}

		// New commit pushed → CI completes with new SHA → should fire
		if got := checkWithTransition(r, "ci-completed", true, "sha2", false); !got {
			t.Error("expected true when CI completes for new commit (genuine transition)")
		}
	})

	t.Run("state-based: stale seeded state does not suppress transition back to original key", func(t *testing.T) {
		// Simulates: CI completed with sha1 at rule creation, then new commit
		// pushed (sha2), then force-push back to sha1.
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		// Seed with sha1
		checkWithTransition(r, "ci-completed", true, "sha1", false)
		r.Seeding = false

		// New commit: CI completes with sha2 (different stateKey, fires)
		if got := checkWithTransition(r, "ci-completed", true, "sha2", false); !got {
			t.Error("expected true for new stateKey sha2")
		}

		// Force-push back to sha1: should fire (genuine sha2→sha1 transition),
		// not be suppressed by stale seeded state
		if got := checkWithTransition(r, "ci-completed", true, "sha1", false); !got {
			t.Error("expected true on transition back to sha1 (stale seeded state should not suppress)")
		}
	})

	t.Run("state-based: seeding cleared on revert allows subsequent real transition", func(t *testing.T) {
		// Simulates: PR already approved, then approval dismissed, then re-approved
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		// Seed approved state
		checkWithTransition(r, "approved", true, "true", false)
		r.Seeding = false

		// Seeded state consumed without firing
		checkWithTransition(r, "approved", true, "true", false)

		// Approval dismissed
		checkWithTransition(r, "approved", false, "", false)

		// Re-approved → genuine transition, should fire
		if got := checkWithTransition(r, "approved", true, "true", false); !got {
			t.Error("expected true on re-approval after dismiss (genuine false→true transition)")
		}

		// Same state again → deduped
		if got := checkWithTransition(r, "approved", true, "true", false); got {
			t.Error("expected false on duplicate after re-approval")
		}
	})
}

func TestCheckWithTransitionTerminal(t *testing.T) {
	t.Run("terminal: seeding does not suppress", func(t *testing.T) {
		r := &rule.WatchRule{ID: "r1", Seeding: true}

		got := checkWithTransition(r, "merged", true, "true", true)
		if !got {
			t.Error("expected true for terminal condition during seeding")
		}
	})

	t.Run("terminal: deduplicates after firing", func(t *testing.T) {
		r := &rule.WatchRule{ID: "r1"}

		got := checkWithTransition(r, "merged", true, "true", true)
		if !got {
			t.Error("expected true on first terminal match")
		}
		got = checkWithTransition(r, "merged", true, "true", true)
		if got {
			t.Error("expected false on duplicate terminal match")
		}
	})

	t.Run("terminal: not matched returns false", func(t *testing.T) {
		r := &rule.WatchRule{ID: "r1"}

		got := checkWithTransition(r, "merged", false, "", true)
		if got {
			t.Error("expected false when terminal condition not matched")
		}
	})
}
