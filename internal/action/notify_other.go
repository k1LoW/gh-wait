//go:build !darwin

package action

import (
	"github.com/gen2brain/beeep"
	"github.com/k1LoW/gh-wait/internal/rule"
)

type NotifyAction struct{}

func (a *NotifyAction) Execute(r *rule.WatchRule) error {
	return beeep.Notify("gh-wait", r.Label(), "")
}
