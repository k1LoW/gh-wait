# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build & Test Commands

- **Test:** `make test` (runs `go test ./... -coverprofile=coverage.out -covermode=count -count=1`)
- **Lint:** `make lint` (runs `golangci-lint run ./...`)
- **Single test:** `go test ./internal/server -run TestCheckRulesMatched -v`
- **CI:** `make ci` (installs dev deps, then runs tests)

## Architecture

gh-wait is a GitHub CLI extension (`gh` extension) that watches PRs/issues/workflow runs for conditions and triggers actions. It uses a **client-server architecture**:

- **CLI (cmd/)** — Parses commands via Cobra, ensures the background server is running, and communicates with it over HTTP. `pr` subcommand supports auto-detecting the PR number from the current branch via `gh pr view`.
- **Background Server (internal/server/)** — Listens on `localhost:9248`, polls GitHub API every 30s, manages watch rules, persists state to `$XDG_STATE_HOME/gh-wait/gh-wait-{port}.json`. Thread-safe via RWMutex.
- **Checkers (internal/checker/)** — Evaluate conditions against GitHub API. PRChecker handles `approved`, `merged`, `closed`, `commented`, `ci-completed`, `ci-failed`. IssueChecker handles `commented`, `closed`. WorkflowChecker handles `completed`, `succeeded`, `failed`. Conditions use OR evaluation.
- **Rules (internal/rule/)** — WatchRule struct with deterministic SHA256-based IDs (4-byte truncated hex). Rules have lifecycle: watching → triggered/stopped. Only "watching" rules persist.
- **Actions (internal/action/)** — Execute on trigger (e.g., open browser via `cli/browser`).

Server is auto-started on first rule creation (`ensureServer()` in cmd/root.go) and can be manually controlled via `start`/`stop`/`restart` commands. Unix/Windows process detachment handled separately in `sysprocattr_*.go`.

## Key Dependencies

- `github.com/cli/go-gh/v2` — GitHub CLI integration
- `github.com/google/go-github/v83` — GitHub API client
- `github.com/k1LoW/go-github-client/v83` — GitHub client wrapper with auth
- `github.com/k1LoW/donegroup` — Goroutine coordination for graceful shutdown
- `github.com/spf13/cobra` — CLI framework
