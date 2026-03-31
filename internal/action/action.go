package action

import (
	"log/slog"

	"github.com/cli/go-gh/v2/pkg/browser"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type Action interface {
	Execute(r *rule.WatchRule) error
}

type OpenBrowserAction struct{}

func (a *OpenBrowserAction) Execute(r *rule.WatchRule) error {
	b := browser.New("", nil, nil)
	return b.Browse(r.URL)
}

type LogAction struct{}

func (a *LogAction) Execute(r *rule.WatchRule) error {
	slog.Info("action triggered", "rule_id", r.ID, "label", r.Label())
	return nil
}

