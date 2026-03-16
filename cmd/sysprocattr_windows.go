//go:build windows

package cmd

import "os/exec"

func setSysProcAttr(_ *exec.Cmd) {
	// On Windows, child processes are independent by default.
}
