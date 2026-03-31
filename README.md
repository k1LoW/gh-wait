# gh-wait

`gh-wait` is a GitHub CLI (`gh`) extension that watches pull requests, issues, discussions, and workflow runs for specific conditions, then takes action when they are met.

It runs a lightweight background server that polls the GitHub API and can open your browser or notify you when conditions like approval, merge, CI completion, workflow completion, new comments, or discussion answers are detected.

## Usage

You can pass a GitHub URL directly — `gh-wait` auto-detects whether it is a PR, an issue, a discussion, or a workflow run:

```bash
# Watch a PR by URL
$ gh wait https://github.com/owner/repo/pull/123 --approved --open

# Watch an issue by URL
$ gh wait https://github.com/owner/repo/issues/456 --commented --open

# Watch a discussion by URL
$ gh wait https://github.com/owner/repo/discussions/7 --answered --open

# Watch a workflow run by URL
$ gh wait https://github.com/owner/repo/actions/runs/23424874935
```

Or use the explicit subcommands:

```bash
# Watch for PR approval, then open in browser (auto-detect PR from current branch)
$ gh wait pr --approved --open

# Watch for PR approval with explicit number
$ gh wait pr 123 --approved --open

# Watch for CI completion
$ gh wait pr 123 --ci-completed --open

# Watch for new comments on an issue
$ gh wait issue 456 --commented --open

# Watch a discussion for answer
$ gh wait discussion 7 --answered --open --repo owner/repo

# Watch a workflow run for completion
$ gh wait workflow 23424874935 --repo owner/repo

# Watch a workflow run for failure, open browser when failed
$ gh wait workflow 23424874935 --repo owner/repo --failed --open

# Specify a repository explicitly
$ gh wait pr 123 --approved --open --repo owner/repo

# Custom polling interval
$ gh wait pr 123 --approved --open --interval 5min
$ gh wait issue 456 --commented --open --interval 1h
```

### Continuous Watch

By default, rules are **one-shot** — they trigger once and are removed. Use `--until` and `--count` for continuous watching:

```bash
# Notify on every new comment until the PR is closed
$ gh wait pr 123 --commented --open --until closed

# Notify on approval until merged
$ gh wait pr 123 --approved --open --until merged

# Notify on comments up to 3 times
$ gh wait issue 456 --commented --open --count 3

# Notify on comments up to 3 times or until merged (whichever comes first)
$ gh wait pr 123 --commented --open --until merged --count 3

# Wait for close (until-only mode: action executes when the until condition is met)
$ gh wait pr 123 --open --until closed
```

| `--count` | `--until` | Behavior |
|-----------|-----------|----------|
| (omitted) | (omitted) | One-shot: triggers once and removes the rule |
| (omitted) | `closed`  | Triggers on every condition match until closed |
| `3`       | (omitted) | Triggers up to 3 times then removes the rule |
| `3`       | `merged`  | Triggers up to 3 times or until merged (whichever comes first) |

### Ignoring Users

Use `--ignore-user` to exclude events from specific users (e.g., bots). The value is a Go regular expression matched against the username. Can be specified multiple times:

```bash
# Ignore bot users when watching for comments
$ gh wait pr 42 --commented --ignore-user ".*\[bot\]" --ignore-user "dependabot"
```

### Self-Filtering

`gh-wait` automatically detects the authenticated GitHub user and filters out events triggered by yourself. For example, if you approve your own PR or leave a comment, those events will not trigger actions (browser open, notification).

This prevents noise from your own activity while still tracking events from other users.

- **Trigger conditions** (`--approved`, `--commented`, etc.) — self-filtering is applied. Actions are skipped if all matching events are from yourself.
- **Until conditions** (`--until merged`, `--until closed`) — self-filtering is **not** applied. The rule terminates regardless of who caused the state change.
- **CI conditions** (`--ci-completed`, `--ci-failed`) — self-filtering is not applied, as these are system events.
- **Rule lifecycle** — even when actions are skipped due to self-filtering, the rule's trigger count is still incremented and one-shot rules are still removed.

### Managing Rules

```bash
# List all watch rules
$ gh wait list

# List in JSON format
$ gh wait list --json

# Delete a specific rule by ID
$ gh wait delete abc1234

# Delete all rules
$ gh wait delete --all

# Start/stop/restart the background server
$ gh wait start
$ gh wait stop
$ gh wait restart
```

### List Output

```
ID        URL                                        CONDITIONS  UNTIL   COUNT  INTERVAL  ACTION  STATUS    LAST_TRIGGERED_AT
00a12cf6  https://github.com/k1LoW/gh-wait/pull/1    commented   closed  0/3    30sec     open    watching  -
abc12345  https://github.com/k1LoW/gh-wait/pull/2    approved    merged  1/3    5min      open    watching  3 minutes ago
```

## Supported Conditions

### Pull Request

| Condition | Flag | Description |
|-----------|------|-------------|
| Approved | `--approved` | At least one approval review |
| Merged | `--merged` | PR has been merged |
| Closed | `--closed` | PR has been closed |
| Commented | `--commented` | New comments added (issue comments, review comments, or reviews with body) |
| CI Completed | `--ci-completed` | All CI checks and commit statuses reach a completed state (none pending) |
| CI Failed | `--ci-failed` | At least one CI check or commit status failed |

### Issue

| Condition | Flag | Description |
|-----------|------|-------------|
| Commented | `--commented` | New comments added |
| Closed | `--closed` | Issue has been closed |

### Discussion

| Condition | Flag | Description |
|-----------|------|-------------|
| Commented | `--commented` | New comments added |
| Closed | `--closed` | Discussion has been closed |
| Answered | `--answered` | Discussion has been marked as answered |

### Workflow Run

| Condition | Flag | Description |
|-----------|------|-------------|
| Completed | `--completed` | Workflow run reaches any terminal state (default if no condition specified) |
| Succeeded | `--succeeded` | Workflow run completes with success |
| Failed | `--failed` | Workflow run completes with failure |

## Install

```bash
$ gh extension install k1LoW/gh-wait
```

### macOS: `--notify` requires `terminal-notifier`

```bash
$ brew install terminal-notifier
```

## Command Line Options

### `gh wait pr [number]`

If `number` is omitted, the PR associated with the current branch is automatically detected via `gh pr view`.

| Option | Description |
|--------|-------------|
| `--repo` | Select another repository using the `OWNER/REPO` format |
| `--approved` | Watch for approval |
| `--merged` | Watch for merge |
| `--closed` | Watch for close |
| `--commented` | Watch for new comments |
| `--ci-completed` | Watch for CI completion |
| `--ci-failed` | Watch for CI failure |
| `--open` | Open in browser when condition is met |
| `--notify` | Send OS desktop notification when condition is met |
| `--until` | Termination condition (can be specified multiple times) |
| `--count` | Maximum number of triggers (0 = unlimited) |
| `--ignore-user` | Regex pattern of users to ignore (can be specified multiple times) |
| `--interval` | Polling interval (e.g., `30sec`, `5min`, `1h`). Default: `30sec` |

### `gh wait issue <number>`

| Option | Description |
|--------|-------------|
| `--repo` | Select another repository using the `OWNER/REPO` format |
| `--commented` | Watch for new comments |
| `--closed` | Watch for close |
| `--open` | Open in browser when condition is met |
| `--notify` | Send OS desktop notification when condition is met |
| `--until` | Termination condition (can be specified multiple times) |
| `--count` | Maximum number of triggers (0 = unlimited) |
| `--ignore-user` | Regex pattern of users to ignore (can be specified multiple times) |
| `--interval` | Polling interval (e.g., `30sec`, `5min`, `1h`). Default: `30sec` |

### `gh wait discussion <number>`

| Option | Description |
|--------|-------------|
| `--repo` | Select another repository using the `OWNER/REPO` format |
| `--commented` | Watch for new comments |
| `--closed` | Watch for close |
| `--answered` | Watch for answer |
| `--open` | Open in browser when condition is met |
| `--notify` | Send OS desktop notification when condition is met |
| `--until` | Termination condition (can be specified multiple times) |
| `--count` | Maximum number of triggers (0 = unlimited) |
| `--ignore-user` | Regex pattern of users to ignore (can be specified multiple times) |
| `--interval` | Polling interval (e.g., `30sec`, `5min`, `1h`). Default: `30sec` |

### `gh wait workflow <run-id>`

The workflow run ID is required. You can also pass a full workflow run URL directly to `gh-wait` instead of using this subcommand.

| Option | Description |
|--------|-------------|
| `--repo` | Select another repository using the `OWNER/REPO` format |
| `--completed` | Watch for completion (any conclusion). Default if no condition specified |
| `--succeeded` | Watch for success |
| `--failed` | Watch for failure |
| `--open` | Open in browser when condition is met |
| `--notify` | Send OS desktop notification when condition is met |
| `--until` | Termination condition (can be specified multiple times) |
| `--count` | Maximum number of triggers (0 = unlimited) |
| `--ignore-user` | Regex pattern of users to ignore (can be specified multiple times) |
| `--interval` | Polling interval (e.g., `30sec`, `5min`, `1h`). Default: `30sec` |

### `gh wait list`

| Option | Description |
|--------|-------------|
| `--json` | Output results as JSON |

### `gh wait delete [id...]`

| Option | Description |
|--------|-------------|
| `--all` | Delete all watch rules |

### Global Options

| Option | Description |
|--------|-------------|
| `--port` | Server port (default: 9248) |
| `--foreground` | Run server in foreground mode |

## Architecture

`gh-wait` uses a client-server architecture:

1. **Background Server** — A lightweight HTTP server runs on `localhost:9248` (configurable). It is automatically started when you create your first watch rule.
2. **Polling** — The server polls the GitHub API at each rule's configured interval (default: 30 seconds) to check conditions.
3. **State Persistence** — Rules are persisted to `$XDG_STATE_HOME/gh-wait/` (or `~/.local/state/gh-wait/`) and survive server restarts.
