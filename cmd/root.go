package cmd

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/k1LoW/gh-wait/internal/client"
	"github.com/k1LoW/gh-wait/internal/server"
	"github.com/spf13/cobra"
)

const (
	defaultPort        = 9248
	probeTimeout       = 2 * time.Second
	waitForReadyTimeout = 10 * time.Second
)

var (
	port       int
	foreground bool
)

var rootCmd = &cobra.Command{
	Use:   "gh-wait",
	Short: "Wait for GitHub events and get notified",
	Long: `gh-wait is a GitHub CLI extension that watches pull requests and issues
for specific conditions (approvals, merges, CI completion, comments, etc.)
and triggers actions (desktop notification, open in browser) when those
conditions are met.

It uses a client-server architecture: a background server polls the GitHub
API at configurable intervals and evaluates watch rules. The server is
automatically started when you create the first watch rule.

Multiple conditions on a single rule are evaluated with OR logic — if any
condition is met, the rule triggers. Use --until to set termination
conditions so the rule automatically stops watching.

Watch rules and server state are persisted to disk, so rules survive
server restarts.`,
	Example: `  # Watch the current branch's PR for approval, open browser when approved
  gh wait pr --approved --open

  # Watch PR #42 for merge or close
  gh wait pr 42 --merged --closed

  # Watch PR #42 for comments, stop watching when merged, poll every 1 min
  gh wait pr 42 --commented --until merged --interval 1min

  # Watch PR #10 for CI completion, trigger at most 3 times
  gh wait pr 10 --ci-finished --count 3

  # Watch PR for approval, ignoring bot users
  gh wait pr 42 --approved --ignore-user ".*\\[bot\\]"

  # Watch issue #5 for new comments on a specific repo
  gh wait issue 5 --commented --repo owner/repo

  # List all watch rules
  gh wait list

  # List all watch rules in JSON format
  gh wait list --json

  # Delete a specific rule by ID
  gh wait delete abc1234

  # Delete all rules
  gh wait delete --all

  # Manage the background server
  gh wait start
  gh wait stop
  gh wait restart`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if foreground {
			return runForeground(cmd.Context())
		}
		return cmd.Help()
	},
	SilenceUsage: true,
}

func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().IntVar(&port, "port", defaultPort, "Server port")
	rootCmd.PersistentFlags().BoolVar(&foreground, "foreground", false, "Run server in foreground")
}

func serverAddr() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
}

func probeServer() (*client.StatusResponse, error) {
	c := client.New(serverAddr())
	return c.ProbeStatus()
}

func ensureServer() error {
	if _, err := probeServer(); err == nil {
		return nil
	}
	return startBackground()
}

func startBackground() error {
	binPath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("cannot find binary: %w", err)
	}

	cmd := exec.Command(binPath, "--foreground", "--port", strconv.Itoa(port))
	setSysProcAttr(cmd)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start server process: %w", err)
	}

	pid := cmd.Process.Pid
	if err := cmd.Process.Release(); err != nil {
		return fmt.Errorf("failed to release process: %w", err)
	}

	if err := waitForReady(); err != nil {
		return fmt.Errorf("server failed to start (pid=%d): %w", pid, err)
	}

	fmt.Fprintf(os.Stderr, "gh-wait server started (pid=%d, port=%d)\n", pid, port)
	return nil
}

func waitForReady() error {
	deadline := time.Now().Add(waitForReadyTimeout)
	for time.Now().Before(deadline) {
		if _, err := probeServer(); err == nil {
			return nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return fmt.Errorf("timeout waiting for server to be ready")
}

func runForeground(ctx context.Context) error {
	addr := serverAddr()
	return server.Run(ctx, addr, port)
}

func newClient() *client.Client {
	return client.New(serverAddr())
}
