package cmd

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/k1LoW/gh-wait/internal/client"
	"github.com/k1LoW/gh-wait/internal/server"
	"github.com/k1LoW/gh-wait/version"
	"github.com/spf13/cobra"
)

const (
	defaultPort         = 9248
	probeTimeout        = 2 * time.Second
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

You can pass a GitHub PR or issue URL directly instead of using
subcommands — gh-wait will auto-detect the type:

  gh wait https://github.com/owner/repo/pull/42 --approved --open
  gh wait https://github.com/owner/repo/issues/5 --commented

It uses a client-server architecture: a background server polls the GitHub
API at configurable intervals and evaluates watch rules. The server is
automatically started when you create the first watch rule.

Multiple conditions on a single rule are evaluated with OR logic — if any
condition is met, the rule triggers. Use --until to set termination
conditions so the rule automatically stops watching.

Watch rules and server state are persisted to disk, so rules survive
server restarts.`,
	Example: `  # Watch a PR by URL
  gh wait https://github.com/owner/repo/pull/42 --approved --open

  # Watch an issue by URL
  gh wait https://github.com/owner/repo/issues/5 --commented

  # Watch the current branch's PR for approval, open browser when approved
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
	Version:      version.Version,
}

func Execute() {
	if transformed, ok := transformURLArgs(os.Args[1:]); ok {
		rootCmd.SetArgs(transformed)
	}
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

// transformURLArgs detects a GitHub PR/issue URL in the arguments and
// rewrites them into the equivalent subcommand form so that Cobra routes
// to the correct handler.
//
//	gh-wait https://github.com/owner/repo/pull/42 --approved
//	  → pr 42 --repo owner/repo --approved
//
//	gh-wait https://github.com/owner/repo/issues/5 --commented
//	  → issue 5 --repo owner/repo --commented
func transformURLArgs(args []string) ([]string, bool) {
	// Scan all non-flag arguments for a GitHub URL. We skip anything
	// starting with "-" (flags and their values) and try parseGitHubURL
	// on each candidate. This avoids fragile heuristics about which
	// flags are boolean vs value-bearing.
	for i, a := range args {
		if strings.HasPrefix(a, "-") {
			continue
		}
		subcommand, repo, number, normalizedURL, ok := parseGitHubURL(a)
		if !ok {
			continue
		}
		newArgs := make([]string, 0, len(args)+5)
		newArgs = append(newArgs, args[:i]...)
		newArgs = append(newArgs, subcommand, strconv.Itoa(number), "--repo", repo, "--url", normalizedURL)
		newArgs = append(newArgs, args[i+1:]...)
		return newArgs, true
	}
	return nil, false
}

// parseGitHubURL extracts the subcommand ("pr" or "issue"), "owner/repo",
// the number, and a normalized URL from a GitHub URL. Returns ok=false if
// the URL is not a recognized PR or issue URL.
func parseGitHubURL(raw string) (subcommand, repo string, number int, normalizedURL string, ok bool) {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", "", 0, "", false
	}

	// Accept github.com and GitHub Enterprise hosts.
	// Path format: /{owner}/{repo}/pull/{number} or /{owner}/{repo}/issues/{number}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 4 {
		return "", "", 0, "", false
	}

	owner := parts[0]
	repoName := parts[1]
	kind := parts[2] // "pull" or "issues"
	numStr := parts[3]

	n, err := strconv.Atoi(numStr)
	if err != nil || n <= 0 {
		return "", "", 0, "", false
	}

	switch kind {
	case "pull":
		subcommand = "pr"
	case "issues":
		subcommand = "issue"
	default:
		return "", "", 0, "", false
	}

	normalizedURL = fmt.Sprintf("%s://%s/%s/%s/%s/%d", u.Scheme, u.Host, owner, repoName, kind, n)
	return subcommand, owner + "/" + repoName, n, normalizedURL, true
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
