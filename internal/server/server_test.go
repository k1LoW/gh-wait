package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/k1LoW/gh-wait/internal/checker"
	"github.com/k1LoW/gh-wait/internal/rule"
)

func TestStateAddAndListRules(t *testing.T) {
	s := NewState(0)
	r := &rule.WatchRule{ID: "abc", Type: "pr", Status: "watching"}
	s.AddRule(r)

	rules := s.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "abc" {
		t.Errorf("expected ID abc, got %s", rules[0].ID)
	}
}

func TestStateDeduplication(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "abc", Type: "pr", Action: "notify", Status: "watching"})
	s.AddRule(&rule.WatchRule{ID: "abc", Type: "pr", Action: "open", Status: "watching"})

	rules := s.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule after dedup, got %d", len(rules))
	}
	if rules[0].Action != "open" {
		t.Errorf("expected updated action 'open', got %s", rules[0].Action)
	}
}

func TestStateRemoveRule(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "abc", Status: "watching"})
	s.AddRule(&rule.WatchRule{ID: "def", Status: "watching"})

	if !s.RemoveRule("abc") {
		t.Error("expected RemoveRule to return true")
	}
	if s.RemoveRule("nonexistent") {
		t.Error("expected RemoveRule to return false for nonexistent")
	}

	rules := s.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "def" {
		t.Errorf("expected remaining rule ID 'def', got %s", rules[0].ID)
	}
}

func TestStateWatchingRules(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "a", Status: "watching"})
	s.AddRule(&rule.WatchRule{ID: "b", Status: "triggered"})
	s.AddRule(&rule.WatchRule{ID: "c", Status: "watching"})

	watching := s.WatchingRules()
	if len(watching) != 2 {
		t.Fatalf("expected 2 watching rules, got %d", len(watching))
	}
}

func TestStateMarkTriggered(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "abc", Status: "watching"})
	s.MarkTriggered("abc")

	rules := s.Rules()
	if rules[0].Status != "triggered" {
		t.Errorf("expected status 'triggered', got %s", rules[0].Status)
	}
}

func TestStateUpdateRule(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "abc", Type: "pr", Status: "watching", TriggerCount: 0})

	updated := &rule.WatchRule{ID: "abc", Type: "pr", Status: "watching", TriggerCount: 2}
	s.UpdateRule(updated)

	rules := s.Rules()
	if rules[0].TriggerCount != 2 {
		t.Errorf("expected TriggerCount 2, got %d", rules[0].TriggerCount)
	}
}

func TestStateSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	s := NewState(19999)
	s.AddRule(&rule.WatchRule{ID: "abc", Type: "pr", Repo: "owner/repo", Number: 1, Status: "watching"})
	s.AddRule(&rule.WatchRule{ID: "def", Type: "issue", Status: "triggered"})

	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := NewState(19999)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	rules := s2.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 watching rule after load, got %d", len(rules))
	}
	if rules[0].ID != "abc" {
		t.Errorf("expected ID abc, got %s", rules[0].ID)
	}
}

func TestStateSaveAndLoadWithContinuousWatch(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	s := NewState(19999)
	s.AddRule(&rule.WatchRule{
		ID: "cont1", Type: "pr", Repo: "owner/repo", Number: 1, Status: "watching",
		Until: []string{"closed"}, MaxCount: 3, TriggerCount: 1,
		LastCheckedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
	})

	if err := s.Save(); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	s2 := NewState(19999)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	rules := s2.Rules()
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	r := rules[0]
	if len(r.Until) != 1 || r.Until[0] != "closed" {
		t.Errorf("expected Until [closed], got %v", r.Until)
	}
	if r.MaxCount != 3 {
		t.Errorf("expected MaxCount 3, got %d", r.MaxCount)
	}
	if r.TriggerCount != 1 {
		t.Errorf("expected TriggerCount 1, got %d", r.TriggerCount)
	}
}

func TestStateLoadNonexistent(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	s := NewState(19999)
	if err := s.Load(); err != nil {
		t.Fatalf("Load should not fail for nonexistent file: %v", err)
	}
	if len(s.Rules()) != 0 {
		t.Error("expected 0 rules")
	}
}

// HTTP handler tests

func TestHandleStatus(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "a", Status: "watching"})
	s.AddRule(&rule.WatchRule{ID: "b", Status: "triggered"})

	handler := NewHandler(s)
	req := httptest.NewRequest(http.MethodGet, "/_/api/status", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var resp struct {
		Version       string `json:"version"`
		RuleCount     int    `json:"rule_count"`
		WatchingCount int    `json:"watching_count"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if resp.RuleCount != 2 {
		t.Errorf("expected rule_count 2, got %d", resp.RuleCount)
	}
	if resp.WatchingCount != 1 {
		t.Errorf("expected watching_count 1, got %d", resp.WatchingCount)
	}
}

func TestHandleAddAndListRules(t *testing.T) {
	s := NewState(0)
	handler := NewHandler(s)

	body := `{"id":"r1","type":"pr","repo":"o/r","number":1,"conditions":["approved"],"action":"open","status":"watching"}`
	req := httptest.NewRequest(http.MethodPost, "/_/api/rules", strings.NewReader(body))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "/_/api/rules", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var rules []*rule.WatchRule
	if err := json.NewDecoder(w.Body).Decode(&rules); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "r1" {
		t.Errorf("expected ID r1, got %s", rules[0].ID)
	}
}

func TestHandleDeleteRule(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "r1", Status: "watching"})
	handler := NewHandler(s)

	req := httptest.NewRequest(http.MethodDelete, "/_/api/rules/r1", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}

	if len(s.Rules()) != 0 {
		t.Error("expected 0 rules after delete")
	}

	req = httptest.NewRequest(http.MethodDelete, "/_/api/rules/nonexistent", nil)
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleShutdown(t *testing.T) {
	s := NewState(0)
	handler := NewHandler(s)

	req := httptest.NewRequest(http.MethodPost, "/_/api/shutdown", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d", w.Code)
	}

	select {
	case <-s.shutdownCh:
	default:
		t.Error("expected shutdown signal")
	}
}

// CheckRules test with mock checker

type mockChecker struct {
	result          bool
	err             error
	conditionResult map[string]bool // per-condition results for CheckConditions
}

func (m *mockChecker) Check(ctx context.Context, r *rule.WatchRule) (bool, error) {
	return m.CheckConditions(ctx, r, r.Conditions)
}

func (m *mockChecker) CheckConditions(_ context.Context, _ *rule.WatchRule, conditions []string) (bool, error) {
	if m.conditionResult == nil {
		return m.result, m.err
	}
	for _, c := range conditions {
		if m.conditionResult[c] {
			return true, m.err
		}
	}
	return false, m.err
}

func (m *mockChecker) CheckState(ctx context.Context, r *rule.WatchRule, conditions []string) (bool, error) {
	return m.CheckConditions(ctx, r, conditions)
}

type mockAction struct {
	executed []string
}

func (m *mockAction) Execute(r *rule.WatchRule) error {
	m.executed = append(m.executed, r.ID)
	return nil
}

func TestCheckRulesFirstCheckSeeding(t *testing.T) {
	s := NewState(0)
	// LastCheckedAt is zero → first check
	s.AddRule(&rule.WatchRule{ID: "r1", Type: "pr", Action: "open", Status: "watching", Conditions: []string{"approved"}, Interval: "0s"})

	mc := &mockChecker{result: true}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": mc}

	CheckRules(context.Background(), s, checkers, ma)

	// First check should NOT execute action (seeding state)
	if len(ma.executed) != 0 {
		t.Errorf("expected no action on first check (seeding), got %v", ma.executed)
	}
	// Rule should remain
	rules := s.Rules()
	if len(rules) != 1 {
		t.Error("expected rule to remain after first check seeding")
	}
	// LastCheckedAt should now be set
	if rules[0].LastCheckedAt.IsZero() {
		t.Error("expected LastCheckedAt to be set after first check")
	}

	// Second check should trigger normally
	CheckRules(context.Background(), s, checkers, ma)
	if len(ma.executed) != 1 || ma.executed[0] != "r1" {
		t.Errorf("expected action on second check, got %v", ma.executed)
	}
}

func TestCheckRulesMatched(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "r1", Type: "pr", Action: "open", Status: "watching", Conditions: []string{"approved"}, LastCheckedAt: time.Now().Add(-time.Minute)})

	mc := &mockChecker{result: true}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": mc}

	CheckRules(context.Background(), s, checkers, ma)

	if len(ma.executed) != 1 || ma.executed[0] != "r1" {
		t.Errorf("expected action executed for r1, got %v", ma.executed)
	}
	if len(s.Rules()) != 0 {
		t.Error("expected rule to be removed after trigger")
	}
}

func TestCheckRulesNotMatched(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "r1", Type: "pr", Action: "open", Status: "watching", Conditions: []string{"approved"}, LastCheckedAt: time.Now().Add(-time.Minute)})

	mc := &mockChecker{result: false}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": mc}

	CheckRules(context.Background(), s, checkers, ma)

	if len(ma.executed) != 0 {
		t.Error("expected no action")
	}
	if len(s.Rules()) != 1 {
		t.Error("expected rule to remain")
	}
}

func TestCheckRulesNotifyAction(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "r1", Type: "pr", Action: "notify", Status: "watching", Conditions: []string{"approved"}, LastCheckedAt: time.Now().Add(-time.Minute)})

	mc := &mockChecker{result: true}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": mc}

	CheckRules(context.Background(), s, checkers, ma)

	// Action should not be executed for "notify" (browser open is only for "open")
	if len(ma.executed) != 0 {
		t.Error("expected no browser action for notify")
	}
	// But rule should still be removed
	if len(s.Rules()) != 0 {
		t.Error("expected rule to be removed after trigger")
	}
}

func TestCheckRulesMultipleTypes(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{ID: "pr1", Type: "pr", Action: "open", Status: "watching", Conditions: []string{"approved"}, LastCheckedAt: time.Now().Add(-time.Minute)})
	s.AddRule(&rule.WatchRule{ID: "issue1", Type: "issue", Action: "open", Status: "watching", Conditions: []string{"closed"}, LastCheckedAt: time.Now().Add(-time.Minute)})

	prMock := &mockChecker{result: true}
	issueMock := &mockChecker{result: false}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": prMock, "issue": issueMock}

	CheckRules(context.Background(), s, checkers, ma)

	if len(ma.executed) != 1 || ma.executed[0] != "pr1" {
		t.Errorf("expected only pr1 action, got %v", ma.executed)
	}
	rules := s.Rules()
	if len(rules) != 1 || rules[0].ID != "issue1" {
		t.Error("expected only issue1 to remain")
	}
}

// Continuous watch tests

func TestCheckRulesContinuousWithUntil(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{
		ID: "r1", Type: "pr", Action: "open", Status: "watching",
		Conditions:    []string{"commented"},
		Until:         []string{"closed"},
		LastCheckedAt: time.Now().Add(-time.Minute),
	})

	// commented matches, closed does not
	mc := &mockChecker{conditionResult: map[string]bool{"commented": true, "closed": false}}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": mc}

	CheckRules(context.Background(), s, checkers, ma)

	if len(ma.executed) != 1 {
		t.Fatalf("expected action executed, got %v", ma.executed)
	}
	// Rule should remain (continuous watch)
	rules := s.Rules()
	if len(rules) != 1 {
		t.Fatal("expected rule to remain for continuous watch")
	}
	if rules[0].TriggerCount != 1 {
		t.Errorf("expected TriggerCount 1, got %d", rules[0].TriggerCount)
	}
	if rules[0].LastCheckedAt.IsZero() {
		t.Error("expected LastCheckedAt to be set")
	}
}

func TestCheckRulesUntilMatched(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{
		ID: "r1", Type: "pr", Action: "open", Status: "watching",
		Conditions:    []string{"commented"},
		Until:         []string{"closed"},
		LastCheckedAt: time.Now().Add(-time.Minute),
	})

	// closed matches → rule should be removed without executing action
	mc := &mockChecker{conditionResult: map[string]bool{"commented": true, "closed": true}}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": mc}

	CheckRules(context.Background(), s, checkers, ma)

	// No action for conditions (until takes precedence)
	if len(ma.executed) != 0 {
		t.Errorf("expected no action when until matched, got %v", ma.executed)
	}
	if len(s.Rules()) != 0 {
		t.Error("expected rule to be removed when until matched")
	}
}

func TestCheckRulesUntilOnlyMode(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{
		ID: "r1", Type: "pr", Action: "open", Status: "watching",
		Conditions:    nil,
		Until:         []string{"closed"},
		LastCheckedAt: time.Now().Add(-time.Minute),
	})

	// closed matches → should execute action and remove rule
	mc := &mockChecker{conditionResult: map[string]bool{"closed": true}}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": mc}

	CheckRules(context.Background(), s, checkers, ma)

	if len(ma.executed) != 1 {
		t.Errorf("expected action executed in until-only mode, got %v", ma.executed)
	}
	if len(s.Rules()) != 0 {
		t.Error("expected rule to be removed")
	}
}

func TestCheckRulesMaxCount(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{
		ID: "r1", Type: "pr", Action: "open", Status: "watching",
		Conditions:    []string{"commented"},
		MaxCount:      2,
		TriggerCount:  0,
		Interval:      "0s",
		LastCheckedAt: time.Now().Add(-time.Minute),
	})

	mc := &mockChecker{conditionResult: map[string]bool{"commented": true}}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": mc}

	// First trigger
	CheckRules(context.Background(), s, checkers, ma)
	rules := s.Rules()
	if len(rules) != 1 {
		t.Fatal("expected rule to remain after first trigger")
	}
	if rules[0].TriggerCount != 1 {
		t.Errorf("expected TriggerCount 1, got %d", rules[0].TriggerCount)
	}

	// Second trigger → should be removed
	ma.executed = nil
	CheckRules(context.Background(), s, checkers, ma)
	if len(ma.executed) != 1 {
		t.Errorf("expected action on second trigger, got %v", ma.executed)
	}
	if len(s.Rules()) != 0 {
		t.Error("expected rule to be removed after reaching MaxCount")
	}
}

func TestCheckRulesMaxCountWithUntil(t *testing.T) {
	s := NewState(0)
	s.AddRule(&rule.WatchRule{
		ID: "r1", Type: "pr", Action: "open", Status: "watching",
		Conditions:    []string{"commented"},
		Until:         []string{"merged"},
		MaxCount:      3,
		TriggerCount:  2,
		LastCheckedAt: time.Now().Add(-time.Minute),
	})

	// commented matches, merged does not → third trigger reaches MaxCount
	mc := &mockChecker{conditionResult: map[string]bool{"commented": true, "merged": false}}
	ma := &mockAction{}
	checkers := map[string]checker.Checker{"pr": mc}

	CheckRules(context.Background(), s, checkers, ma)

	if len(ma.executed) != 1 {
		t.Errorf("expected action executed, got %v", ma.executed)
	}
	if len(s.Rules()) != 0 {
		t.Error("expected rule to be removed after reaching MaxCount")
	}
}

func TestBackupLoop(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	s := NewState(29999)
	s.AddRule(&rule.WatchRule{ID: "abc", Type: "pr", Status: "watching"})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		s.backupLoop(ctx)
		close(done)
	}()

	// Trigger backup
	s.notifyBackup()
	time.Sleep(2 * time.Second)
	cancel()
	<-done // Wait for backupLoop to finish its final Save before TempDir cleanup

	// Verify file was written
	s2 := NewState(29999)
	if err := s2.Load(); err != nil {
		t.Fatalf("Load failed: %v", err)
	}
	if len(s2.Rules()) != 1 {
		t.Errorf("expected 1 rule, got %d", len(s2.Rules()))
	}
}
