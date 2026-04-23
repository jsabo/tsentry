# tsentry

A Go tool that watches Teleport audit logs for `session.summarized` events and
alerts (or auto-locks) users whose AI session summaries are flagged HIGH or
CRITICAL risk.

## Project layout

```
cmd/tsentry/main.go          entry point, signal handling
internal/config/config.go    flags and Config struct, RiskLevel type
internal/client/client.go    Teleport API client construction
internal/watcher/watcher.go  polling loop, event dispatch, output formatting
internal/locker/locker.go    lock creation via Teleport API
```

## Teleport API dependency

The `go.mod` uses a `replace` directive pointing to `../teleport/api` (the local
Teleport OSS checkout). This is intentional for local development — do not remove it.

For CI/GitHub Actions, the replace is dropped and the public module is pinned via
`TELEPORT_API_VERSION` (a GitHub Actions variable). Update this variable when
upgrading Teleport.

The key types used from the API:
- `api/client.LoadProfile` — loads credentials from ~/.tsh
- `api/client.LoadIdentityFile` — loads tbot identity file (--identity flag)
- `api/client.Client.SearchEvents` — queries audit log
- `api/types/events.SessionSummarized` — the event struct (risk_level, username, session_id)
- `api/types.NewLock / LockSpecV2` — creates a Teleport lock

## Auth

Currently uses the operator's active tsh session (`~/.tsh/`). tbot machine
identity is stubbed: pass `--identity=/path/to/identity` to use an identity file
instead. No other code changes needed for tbot — the `client.go` credential
selection already handles it.

## Release

Tags trigger GoReleaser via `.github/workflows/release.yml`. Artifacts:
- `tsentry_vX.Y.Z_linux_amd64.tar.gz`
- `tsentry_vX.Y.Z_linux_arm64.tar.gz`
- `tsentry_vX.Y.Z_darwin_amd64.tar.gz`
- `tsentry_vX.Y.Z_darwin_arm64.tar.gz`
- `tsentry_vX.Y.Z_windows_amd64.zip`
- `checksums.txt`

To cut a release:
```
git tag v0.1.0
git push origin v0.1.0
```

## Usage

```
# Alert only (default) — requires active tsh login
tsentry

# Auto-lock users with HIGH or CRITICAL sessions
tsentry --auto-lock

# Tune threshold, TTL, and poll interval
tsentry --auto-lock --risk-threshold=critical --lock-ttl=30m --poll-interval=60s

# Use tbot identity instead of tsh profile
tsentry --auto-lock --identity=/var/lib/teleport/bot/identity
```

## Future work

- tbot machine identity (stubbed, works via --identity)
- True event streaming if Teleport adds push-based audit log watch
- Slack/PagerDuty notification hooks
- Per-user lock deduplication persistence across restarts
