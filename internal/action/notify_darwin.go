package action

import (
	"fmt"
	"os/exec"

	"github.com/k1LoW/gh-wait/internal/rule"
)

type NotifyAction struct{}

func (a *NotifyAction) Execute(r *rule.WatchRule) error {
	title := "gh-wait"
	message := r.Label()

	// Prefer terminal-notifier because it works from background (setsid) processes,
	// whereas osascript display-notification silently drops notifications.
	if path, err := exec.LookPath("terminal-notifier"); err == nil {
		cmd := exec.Command(path, "-title", title, "-message", message, "-group", "gh-wait")
		if err := cmd.Run(); err == nil {
			return nil
		}
	}

	// Fallback to osascript (works in foreground sessions).
	if path, err := exec.LookPath("osascript"); err == nil {
		script := fmt.Sprintf("display notification %q with title %q", message, title)
		return exec.Command(path, "-e", script).Run()
	}

	return fmt.Errorf("no notification tool available (install terminal-notifier for background support)")
}
