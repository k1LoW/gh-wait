# Changelog

## [v0.10.6](https://github.com/k1LoW/gh-wait/compare/v0.10.5...v0.10.6) - 2026-04-15
### Other Changes
- feat: use type-specific default polling interval by @k1LoW in https://github.com/k1LoW/gh-wait/pull/51

## [v0.10.5](https://github.com/k1LoW/gh-wait/compare/v0.10.4...v0.10.5) - 2026-04-15
### Dependency Updates ⬆️
- chore(deps): bump softprops/action-gh-release from 2 to 3 in the dependencies group by @dependabot[bot] in https://github.com/k1LoW/gh-wait/pull/48
### Other Changes
- feat: show polling interval in watch rule registration message by @k1LoW in https://github.com/k1LoW/gh-wait/pull/50

## [v0.10.4](https://github.com/k1LoW/gh-wait/compare/v0.10.3...v0.10.4) - 2026-04-02
### Fix bug 🐛
- fix: terminal conditions bypass seeding to prevent permanent suppression by @k1LoW in https://github.com/k1LoW/gh-wait/pull/46

## [v0.10.3](https://github.com/k1LoW/gh-wait/compare/v0.10.2...v0.10.3) - 2026-04-01
### Fix bug 🐛
- fix: prevent pre-existing states from triggering actions after seeding by @k1LoW in https://github.com/k1LoW/gh-wait/pull/44

## [v0.10.2](https://github.com/k1LoW/gh-wait/compare/v0.10.1...v0.10.2) - 2026-03-31
### Fix bug 🐛
- fix: notify action not working from background server on macOS by @k1LoW in https://github.com/k1LoW/gh-wait/pull/42

## [v0.10.1](https://github.com/k1LoW/gh-wait/compare/v0.10.0...v0.10.1) - 2026-03-31
### New Features 🎉
- fix: self-filtered conditions still match but skip action execution by @k1LoW in https://github.com/k1LoW/gh-wait/pull/40

## [v0.10.0](https://github.com/k1LoW/gh-wait/compare/v0.9.3...v0.10.0) - 2026-03-31
### New Features 🎉
- refactor: run each watch rule in its own polling goroutine by @k1LoW in https://github.com/k1LoW/gh-wait/pull/38

## [v0.9.3](https://github.com/k1LoW/gh-wait/compare/v0.9.2...v0.9.3) - 2026-03-30
### Fix bug 🐛
- fix: fire state-based conditions that were already true at seeding time by @k1LoW in https://github.com/k1LoW/gh-wait/pull/36

## [v0.9.2](https://github.com/k1LoW/gh-wait/compare/v0.9.1...v0.9.2) - 2026-03-30
### Fix bug 🐛
- fix: execute action when conditions and until overlap by @k1LoW in https://github.com/k1LoW/gh-wait/pull/34

## [v0.9.1](https://github.com/k1LoW/gh-wait/compare/v0.9.0...v0.9.1) - 2026-03-30

## [v0.9.0](https://github.com/k1LoW/gh-wait/compare/v0.8.0...v0.9.0) - 2026-03-29
### New Features 🎉
- feat: add --notify flag for OS notifications via beeep by @k1LoW in https://github.com/k1LoW/gh-wait/pull/31

## [v0.8.0](https://github.com/k1LoW/gh-wait/compare/v0.7.1...v0.8.0) - 2026-03-25
### New Features 🎉
- feat: add GitHub Discussion support by @k1LoW in https://github.com/k1LoW/gh-wait/pull/28
### Other Changes
- fix: add pagination to all checker API calls by @k1LoW in https://github.com/k1LoW/gh-wait/pull/30

## [v0.7.1](https://github.com/k1LoW/gh-wait/compare/v0.7.0...v0.7.1) - 2026-03-23
### Fix bug 🐛
- fix: use previous LastCheckedAt for SinceTime() during condition checks by @k1LoW in https://github.com/k1LoW/gh-wait/pull/27

## [v0.7.0](https://github.com/k1LoW/gh-wait/compare/v0.6.0...v0.7.0) - 2026-03-23
### Breaking Changes 🛠
- feat: add workflow run watching and rename ci-finished to ci-completed by @k1LoW in https://github.com/k1LoW/gh-wait/pull/24

## [v0.6.0](https://github.com/k1LoW/gh-wait/compare/v0.5.0...v0.6.0) - 2026-03-19
### New Features 🎉
- feat: replace LAST_CHECKED_AT with LAST_TRIGGERED_AT in list output by @k1LoW in https://github.com/k1LoW/gh-wait/pull/22

## [v0.5.0](https://github.com/k1LoW/gh-wait/compare/v0.4.2...v0.5.0) - 2026-03-18
### New Features 🎉
- feat: replace TYPE/REPO/NUMBER columns with URL and add LAST_CHECKED_AT by @k1LoW in https://github.com/k1LoW/gh-wait/pull/21
### Fix bug 🐛
- fix: skip action execution on first check to avoid false triggers by @k1LoW in https://github.com/k1LoW/gh-wait/pull/19

## [v0.4.2](https://github.com/k1LoW/gh-wait/compare/v0.4.1...v0.4.2) - 2026-03-17
### Fix bug 🐛
- fix: skip user filtering for until (termination) conditions by @k1LoW in https://github.com/k1LoW/gh-wait/pull/18

## [v0.4.1](https://github.com/k1LoW/gh-wait/compare/v0.4.0...v0.4.1) - 2026-03-17
### Fix bug 🐛
- fix: until conditions stuck due to state-transition tracking by @k1LoW in https://github.com/k1LoW/gh-wait/pull/15

## [v0.4.0](https://github.com/k1LoW/gh-wait/compare/v0.3.2...v0.4.0) - 2026-03-17
### New Features 🎉
- feat: auto-detect PR number from current branch by @k1LoW in https://github.com/k1LoW/gh-wait/pull/12
- feat: allow GitHub PR/issue URL as direct argument by @k1LoW in https://github.com/k1LoW/gh-wait/pull/14

## [v0.3.2](https://github.com/k1LoW/gh-wait/compare/v0.3.1...v0.3.2) - 2026-03-17
### Fix bug 🐛
- feat: state-transition tracking for state-based conditions by @k1LoW in https://github.com/k1LoW/gh-wait/pull/10

## [v0.3.1](https://github.com/k1LoW/gh-wait/compare/v0.3.0...v0.3.1) - 2026-03-17
### Fix bug 🐛
- fix: ci-finished not triggering for repos using only GitHub Actions by @k1LoW in https://github.com/k1LoW/gh-wait/pull/8

## [v0.3.0](https://github.com/k1LoW/gh-wait/compare/v0.2.0...v0.3.0) - 2026-03-17
### New Features 🎉
- feat: add --ignore-user flag for regex-based user filtering by @k1LoW in https://github.com/k1LoW/gh-wait/pull/7

## [v0.2.0](https://github.com/k1LoW/gh-wait/compare/v0.1.0...v0.2.0) - 2026-03-17
### New Features 🎉
- feat: add delete command to remove watch rules by @k1LoW in https://github.com/k1LoW/gh-wait/pull/3
- feat: add per-rule --interval flag for configurable polling interval by @k1LoW in https://github.com/k1LoW/gh-wait/pull/4
- feat: ignore self-triggered events in condition checks by @k1LoW in https://github.com/k1LoW/gh-wait/pull/5

## [v0.1.0](https://github.com/k1LoW/gh-wait/commits/v0.1.0) - 2026-03-17
