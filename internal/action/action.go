package action

import (
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
