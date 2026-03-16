package client

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/k1LoW/gh-wait/internal/rule"
)

func newTestServer(t *testing.T) (*httptest.Server, *[]*rule.WatchRule) {
	t.Helper()
	rules := &[]*rule.WatchRule{}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /_/api/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"version":        "0.0.0",
			"pid":            1,
			"rule_count":     len(*rules),
			"watching_count": len(*rules),
		})
	})
	mux.HandleFunc("POST /_/api/rules", func(w http.ResponseWriter, r *http.Request) {
		var wr rule.WatchRule
		_ = json.NewDecoder(r.Body).Decode(&wr)
		*rules = append(*rules, &wr)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(&wr)
	})
	mux.HandleFunc("GET /_/api/rules", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(*rules)
	})
	mux.HandleFunc("DELETE /_/api/rules/{id}", func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		for i, wr := range *rules {
			if wr.ID == id {
				*rules = append((*rules)[:i], (*rules)[i+1:]...)
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		http.Error(w, "not found", http.StatusNotFound)
	})
	mux.HandleFunc("POST /_/api/shutdown", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv, rules
}

func TestClientProbeStatus(t *testing.T) {
	srv, _ := newTestServer(t)
	c := New(srv.Listener.Addr().String())

	status, err := c.ProbeStatus()
	if err != nil {
		t.Fatalf("ProbeStatus failed: %v", err)
	}
	if status.Version != "0.0.0" {
		t.Errorf("expected version 0.0.0, got %s", status.Version)
	}
}

func TestClientAddAndListRules(t *testing.T) {
	srv, _ := newTestServer(t)
	c := New(srv.Listener.Addr().String())

	wr := &rule.WatchRule{
		ID:     "test1",
		Type:   "pr",
		Repo:   "owner/repo",
		Number: 1,
		Status: "watching",
	}
	if err := c.AddRule(wr); err != nil {
		t.Fatalf("AddRule failed: %v", err)
	}

	rules, err := c.ListRules()
	if err != nil {
		t.Fatalf("ListRules failed: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(rules))
	}
	if rules[0].ID != "test1" {
		t.Errorf("expected ID test1, got %s", rules[0].ID)
	}
}

func TestClientDeleteRule(t *testing.T) {
	srv, rules := newTestServer(t)
	*rules = append(*rules, &rule.WatchRule{ID: "del1", Status: "watching"})
	c := New(srv.Listener.Addr().String())

	if err := c.DeleteRule("del1"); err != nil {
		t.Fatalf("DeleteRule failed: %v", err)
	}
	if len(*rules) != 0 {
		t.Error("expected 0 rules after delete")
	}

	if err := c.DeleteRule("nonexistent"); err == nil {
		t.Error("expected error for nonexistent rule")
	}
}

func TestClientShutdown(t *testing.T) {
	srv, _ := newTestServer(t)
	c := New(srv.Listener.Addr().String())

	if err := c.Shutdown(); err != nil {
		t.Fatalf("Shutdown failed: %v", err)
	}
}

func TestClientProbeStatusNotRunning(t *testing.T) {
	c := New("127.0.0.1:1") // port 1 should not be listening
	_, err := c.ProbeStatus()
	if err == nil {
		t.Error("expected error for non-running server")
	}
}
