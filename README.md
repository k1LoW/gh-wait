# gh-wait

`gh-wait` is a GitHub CLI (`gh`) extension that watches pull requests and issues for specific conditions, then takes action when they are met.

It runs a lightweight background server that polls the GitHub API and can open your browser or notify you when conditions like approval, merge, CI completion, or new comments are detected.

## Usage

```bash
# Watch for PR approval, then open in browser
$ gh wait pr 123 --approved --open

# Watch for CI completion
$ gh wait pr 123 --ci-finished --open

# Watch for new comments on an issue
$ gh wait issue 456 --commented --open

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

### Managing Rules

```bash
# List all watch rules
$ gh wait list

# List in JSON format
$ gh wait list --json

# Start/stop/restart the background server
$ gh wait start
$ gh wait stop
$ gh wait restart
```

### List Output

```
ID        TYPE  REPO           NUMBER  CONDITIONS  UNTIL   COUNT  INTERVAL  ACTION  STATUS
00a12cf6  pr    k1LoW/gh-wait  1       commented   closed  0/0    30sec     open    watching
abc12345  pr    k1LoW/gh-wait  2       approved    merged  1/3    5min      open    watching
```

## Supported Conditions

### Pull Request

| Condition | Flag | Description |
|-----------|------|-------------|
| Approved | `--approved` | At least one approval review |
| Merged | `--merged` | PR has been merged |
| Closed | `--closed` | PR has been closed |
| Commented | `--commented` | New comments added (issue comments, review comments, or reviews with body) |
| CI Finished | `--ci-finished` | All CI checks completed |
| CI Failed | `--ci-failed` | At least one CI check failed |

### Issue

| Condition | Flag | Description |
|-----------|------|-------------|
| Commented | `--commented` | New comments added |
| Closed | `--closed` | Issue has been closed |

## Install

```bash
$ gh extension install k1LoW/gh-wait
```

## Command Line Options

### `gh wait pr <number>`

| Option | Description |
|--------|-------------|
| `--repo` | Select another repository using the `OWNER/REPO` format |
| `--approved` | Watch for approval |
| `--merged` | Watch for merge |
| `--closed` | Watch for close |
| `--commented` | Watch for new comments |
| `--ci-finished` | Watch for CI completion |
| `--ci-failed` | Watch for CI failure |
| `--open` | Open in browser when condition is met |
| `--until` | Termination condition (can be specified multiple times) |
| `--count` | Maximum number of triggers (0 = unlimited) |
| `--interval` | Polling interval (e.g., `30sec`, `5min`, `1h`). Default: `30sec` |

### `gh wait issue <number>`

| Option | Description |
|--------|-------------|
| `--repo` | Select another repository using the `OWNER/REPO` format |
| `--commented` | Watch for new comments |
| `--closed` | Watch for close |
| `--open` | Open in browser when condition is met |
| `--until` | Termination condition (can be specified multiple times) |
| `--count` | Maximum number of triggers (0 = unlimited) |
| `--interval` | Polling interval (e.g., `30sec`, `5min`, `1h`). Default: `30sec` |

### `gh wait list`

| Option | Description |
|--------|-------------|
| `--json` | Output results as JSON |

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
