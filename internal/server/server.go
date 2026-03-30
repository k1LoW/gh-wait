package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/k1LoW/donegroup"
	"github.com/k1LoW/gh-wait/internal/action"
	"github.com/k1LoW/gh-wait/internal/checker"
	"github.com/k1LoW/gh-wait/internal/rule"
	"github.com/k1LoW/gh-wait/version"
	factory "github.com/k1LoW/go-github-client/v83/factory"
	"github.com/shurcooL/githubv4"
)

const (
	pollTick    = 1 * time.Second
	backupDelay = 1 * time.Second
)

type State struct {
	mu         sync.RWMutex
	rules      []*rule.WatchRule
	shutdownCh chan struct{}
	backupCh   chan struct{}
	port       int
}

func NewState(port int) *State {
	return &State{
		rules:      make([]*rule.WatchRule, 0),
		shutdownCh: make(chan struct{}, 1),
		backupCh:   make(chan struct{}, 1),
		port:       port,
	}
}

func (s *State) AddRule(r *rule.WatchRule) {
	// Pre-compile ignore-user regexps so clones can share the cache.
	r.CompiledIgnoreUsers()
	s.mu.Lock()
	defer s.mu.Unlock()
	// Deduplicate by ID
	for i, existing := range s.rules {
		if existing.ID == r.ID {
			s.rules[i] = r
			s.notifyBackup()
			return
		}
	}
	s.rules = append(s.rules, r)
	s.notifyBackup()
}

func (s *State) Rules() []*rule.WatchRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*rule.WatchRule, len(s.rules))
	for i, r := range s.rules {
		result[i] = r.Clone()
	}
	return result
}

func (s *State) RemoveRule(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, r := range s.rules {
		if r.ID == id {
			s.rules = append(s.rules[:i], s.rules[i+1:]...)
			s.notifyBackup()
			return true
		}
	}
	return false
}

func (s *State) UpdateRule(r *rule.WatchRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.rules {
		if existing.ID == r.ID {
			s.rules[i] = r
			s.notifyBackup()
			return
		}
	}
}

// updateLastCheckedAt updates only LastCheckedAt without triggering a backup.
func (s *State) updateLastCheckedAt(id string, t time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.rules {
		if existing.ID == id {
			s.rules[i].LastCheckedAt = t
			return
		}
	}
}

// syncFiredStates deep-copies FiredStates and SeededStates from a cloned
// rule back to the original without triggering a backup.
func (s *State) syncFiredStates(r *rule.WatchRule) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, existing := range s.rules {
		if existing.ID == r.ID {
			if r.FiredStates != nil {
				cp := make(map[string]string, len(r.FiredStates))
				maps.Copy(cp, r.FiredStates)
				s.rules[i].FiredStates = cp
			} else {
				s.rules[i].FiredStates = nil
			}
			if r.SeededStates != nil {
				cp := make(map[string]string, len(r.SeededStates))
				maps.Copy(cp, r.SeededStates)
				s.rules[i].SeededStates = cp
			} else {
				s.rules[i].SeededStates = nil
			}
			return
		}
	}
}

func (s *State) WatchingRules() []*rule.WatchRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*rule.WatchRule
	for _, r := range s.rules {
		if r.Status == "watching" {
			result = append(result, r.Clone())
		}
	}
	return result
}

func (s *State) MarkTriggered(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, r := range s.rules {
		if r.ID == id {
			r.Status = "triggered"
			s.notifyBackup()
			return
		}
	}
}

func (s *State) notifyBackup() {
	select {
	case s.backupCh <- struct{}{}:
	default:
	}
}

// Persistence

func stateDir() (string, error) {
	if v := os.Getenv("XDG_STATE_HOME"); v != "" {
		return filepath.Join(v, "gh-wait"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "gh-wait"), nil
}

func statePath(port int) (string, error) {
	dir, err := stateDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fmt.Sprintf("gh-wait-%d.json", port)), nil
}

func (s *State) Save() error {
	p, err := statePath(s.port)
	if err != nil {
		return err
	}
	dir := filepath.Dir(p)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("failed to create state directory: %w", err)
	}

	rules := s.Rules()
	// Only save watching rules
	var watching []*rule.WatchRule
	for _, r := range rules {
		if r.Status == "watching" {
			watching = append(watching, r)
		}
	}

	b, err := json.Marshal(watching)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	tmp, err := os.CreateTemp(dir, "gh-wait-*.tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return fmt.Errorf("failed to write state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	if err := os.Rename(tmpName, p); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}
	return nil
}

func (s *State) Load() error {
	p, err := statePath(s.port)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read state file: %w", err)
	}
	var rules []*rule.WatchRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}
	for _, r := range rules {
		r.CompiledIgnoreUsers()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = rules
	return nil
}

func (s *State) backupLoop(ctx context.Context) {
	var timer *time.Timer
	for {
		select {
		case <-s.backupCh:
			if timer != nil {
				timer.Stop()
			}
			timer = time.NewTimer(backupDelay)
		case <-func() <-chan time.Time {
			if timer != nil {
				return timer.C
			}
			return make(chan time.Time)
		}():
			if err := s.Save(); err != nil {
				slog.Error("failed to save state", "error", err)
			}
			timer = nil
		case <-ctx.Done():
			if timer != nil {
				timer.Stop()
			}
			if err := s.Save(); err != nil {
				slog.Error("failed to save state on shutdown", "error", err)
			}
			return
		}
	}
}

// HTTP Handlers

func NewHandler(state *State) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /_/api/status", handleStatus(state))
	mux.HandleFunc("POST /_/api/rules", handleAddRule(state))
	mux.HandleFunc("GET /_/api/rules", handleListRules(state))
	mux.HandleFunc("DELETE /_/api/rules/{id}", handleDeleteRule(state))
	mux.HandleFunc("POST /_/api/shutdown", handleShutdown(state))
	return mux
}

func handleStatus(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rules := state.Rules()
		watchingCount := 0
		for _, r := range rules {
			if r.Status == "watching" {
				watchingCount++
			}
		}
		resp := struct {
			Version       string `json:"version"`
			PID           int    `json:"pid"`
			RuleCount     int    `json:"rule_count"`
			WatchingCount int    `json:"watching_count"`
		}{
			Version:       version.Version,
			PID:           os.Getpid(),
			RuleCount:     len(rules),
			WatchingCount: watchingCount,
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(resp)
	}
}

func handleAddRule(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var wr rule.WatchRule
		if err := json.NewDecoder(r.Body).Decode(&wr); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		state.AddRule(&wr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(&wr)
	}
}

func handleListRules(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rules := state.Rules()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rules)
	}
}

func handleDeleteRule(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if state.RemoveRule(id) {
			w.WriteHeader(http.StatusNoContent)
		} else {
			http.Error(w, "rule not found", http.StatusNotFound)
		}
	}
}

func handleShutdown(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
		select {
		case state.shutdownCh <- struct{}{}:
		default:
		}
	}
}

// Server lifecycle

func Run(ctx context.Context, addr string, port int) error {
	state := NewState(port)

	if err := state.Load(); err != nil {
		slog.Warn("failed to load state", "error", err)
	}

	// Initialize GitHub client for checkers
	ghClient, err := factory.NewGithubClient()
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Get current authenticated user to ignore self-triggered events
	var currentUser string
	if u, _, err := ghClient.Users.Get(ctx, ""); err == nil {
		currentUser = u.GetLogin()
		slog.Info("authenticated user", "login", currentUser)
	} else {
		slog.Warn("failed to get authenticated user, self-filtering disabled", "error", err)
	}

	_, _, _, v4ep := factory.GetTokenAndEndpoints()
	var v4Client *githubv4.Client
	if v4ep == "https://api.github.com/graphql" || v4ep == "" {
		v4Client = githubv4.NewClient(ghClient.Client())
	} else {
		v4Client = githubv4.NewEnterpriseClient(v4ep, ghClient.Client())
	}

	checkers := map[string]checker.Checker{
		"pr":         checker.NewPRChecker(ghClient, currentUser),
		"issue":      checker.NewIssueChecker(ghClient, currentUser),
		"workflow":   checker.NewWorkflowChecker(ghClient, currentUser),
		"discussion": checker.NewDiscussionChecker(v4Client, currentUser),
	}
	actions := map[string]action.Action{
		"log":    &action.LogAction{},
		"open":   &action.OpenBrowserAction{},
		"notify": &action.NotifyAction{},
	}

	handler := NewHandler(state)
	srv := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := donegroup.WithCancel(ctx)
	defer cancel()

	// Backup loop
	donegroup.Go(ctx, func() error {
		state.backupLoop(ctx)
		return nil
	})

	// Polling loop
	donegroup.Go(ctx, func() error {
		pollLoop(ctx, state, checkers, actions)
		return nil
	})

	// Shutdown listener
	donegroup.Go(ctx, func() error {
		select {
		case <-state.shutdownCh:
			cancel()
		case <-ctx.Done():
		}
		return nil
	})

	// Graceful shutdown via cleanup
	if err := donegroup.Cleanup(ctx, func() error {
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		return srv.Shutdown(shutdownCtx)
	}); err != nil {
		return fmt.Errorf("failed to register cleanup: %w", err)
	}

	slog.Info("server starting", "addr", addr, "pid", os.Getpid())
	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		return fmt.Errorf("server error: %w", err)
	}

	if err := donegroup.WaitWithTimeout(ctx, 5*time.Second); err != nil {
		slog.Error("shutdown error", "error", err)
	}
	slog.Info("server stopped")
	return nil
}

func pollLoop(ctx context.Context, state *State, checkers map[string]checker.Checker, act map[string]action.Action) {
	ticker := time.NewTicker(pollTick)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			CheckRules(ctx, state, checkers, act)
		case <-ctx.Done():
			return
		}
	}
}

func CheckRules(ctx context.Context, state *State, checkers map[string]checker.Checker, act map[string]action.Action) {
	rules := state.WatchingRules()
	now := time.Now()
	for _, r := range rules {
		// Skip if the rule's polling interval hasn't elapsed
		interval := r.PollInterval()
		if !r.LastCheckedAt.IsZero() && now.Sub(r.LastCheckedAt) < interval {
			continue
		}

		c, ok := checkers[r.Type]
		if !ok {
			continue
		}

		// Detect first check before updating LastCheckedAt.
		isFirstCheck := r.LastCheckedAt.IsZero()

		// Update state's LastCheckedAt for interval scheduling without
		// touching the clone, so SinceTime() returns the previous check
		// time during condition evaluation.
		state.updateLastCheckedAt(r.ID, now)

		// On the first check, enable seeding mode so that
		// checkWithTransition records state-based conditions without
		// triggering actions, while event-based conditions (e.g.,
		// commented) still fire normally.
		if isFirstCheck {
			r.Seeding = true
		}

		// Step 1: Check until (termination) conditions.
		// Use CheckState (no transition tracking) because until conditions
		// should match whenever the state holds, not only on transitions.
		if len(r.Until) > 0 {
			untilMatched, err := c.CheckState(ctx, r, r.Until)
			if err != nil {
				slog.Error("until check failed", "rule_id", r.ID, "error", err)
				if isFirstCheck {
					state.updateLastCheckedAt(r.ID, time.Time{})
				}
				r.Seeding = false
				continue
			}
			if untilMatched {
				if !isFirstCheck {
					slog.Info("until condition matched", "rule_id", r.ID, "type", r.Type, "repo", r.Repo, "number", r.Number)
					if len(r.Conditions) == 0 || conditionsOverlapUntil(r.Conditions, r.Until) {
						// Also execute when conditions overlap with until, because
						// the transition-based check would miss state already present
						// at seeding time.
						executeAction(act, r)
					}
					state.MarkTriggered(r.ID)
					state.RemoveRule(r.ID)
					continue
				}
				slog.Info("first check: seeding until state", "rule_id", r.ID, "type", r.Type, "repo", r.Repo, "number", r.Number)
			}
		}

		// Step 2: Check trigger conditions
		if len(r.Conditions) == 0 {
			r.Seeding = false
			continue
		}

		matched, err := c.Check(ctx, r)
		r.Seeding = false
		// Sync FiredStates back (deep copy) after checker may have mutated them.
		state.syncFiredStates(r)
		if err != nil {
			slog.Error("check failed", "rule_id", r.ID, "error", err)
			if isFirstCheck {
				state.updateLastCheckedAt(r.ID, time.Time{})
			}
			continue
		}
		if matched {
			slog.Info("condition matched", "rule_id", r.ID, "type", r.Type, "repo", r.Repo, "number", r.Number)
			executeAction(act, r)

			r.TriggerCount++
			r.LastTriggeredAt = now
			r.LastCheckedAt = now
			state.UpdateRule(r)

			// Step 3: Determine if rule should be removed
			if r.IsOneShot() || (r.MaxCount > 0 && r.TriggerCount >= r.MaxCount) {
				state.MarkTriggered(r.ID)
				state.RemoveRule(r.ID)
			}
		}
	}
}

func conditionsOverlapUntil(conditions, until []string) bool {
	for _, c := range conditions {
		if slices.Contains(until, c) {
			return true
		}
	}
	return false
}

func executeAction(actions map[string]action.Action, r *rule.WatchRule) {
	if len(r.Actions) == 0 {
		slog.Warn("rule has no actions configured", "rule_id", r.ID)
		return
	}
	for _, name := range r.Actions {
		a, ok := actions[name]
		if !ok {
			slog.Warn("unknown action", "rule_id", r.ID, "action", name)
			continue
		}
		if err := a.Execute(r); err != nil {
			slog.Error("action failed", "rule_id", r.ID, "action", name, "error", err)
		}
	}
}
