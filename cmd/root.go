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
	Long:  `gh-wait watches for GitHub events (PR approvals, merges, comments, etc.) and notifies you when conditions are met.`,
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
