package cmd

import (
	"fmt"
	"os/exec"
)

func checkNotifyDeps() error {
	if _, err := exec.LookPath("terminal-notifier"); err != nil {
		return fmt.Errorf("terminal-notifier is required for --notify on macOS (install via: brew install terminal-notifier)")
	}
	return nil
}
