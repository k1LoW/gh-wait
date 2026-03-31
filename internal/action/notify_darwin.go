package action

import (
	"fmt"
	"os/exec"

	"github.com/k1LoW/gh-wait/internal/rule"
)

type NotifyAction struct{}

func (a *NotifyAction) Execute(r *rule.WatchRule) error {
	// Use terminal-notifier directly because it works from background (setsid) processes,
	// whereas osascript display-notification silently drops notifications.
	path, err := exec.LookPath("terminal-notifier")
	if err != nil {
		return fmt.Errorf("terminal-notifier not found: %w (install via: brew install terminal-notifier)", err)
	}
	return exec.Command(path, "-title", "gh-wait", "-message", r.Label(), "-group", "gh-wait").Run()
}
