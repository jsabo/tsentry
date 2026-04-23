# tsentry

tsentry watches the Teleport audit log for `session.summarized` events — the output of Teleport's AI session summarizer — and alerts you when a session is flagged as high risk. With `--auto-lock` it will also issue a `tctl lock` to block that user immediately.

## Quick look

No cluster needed. Pipe in a test event to see the output format:

```bash
echo '{
  "event": "session.summarized",
  "time": "2026-04-22T12:30:00Z",
  "cluster_name": "prod.example.com",
  "sid": "abc-123-def-456",
  "risk_level": "high",
  "username": "alice",
  "session_type": "ssh",
  "short_description": "User escalated to root via sudo -i, accessed /etc/shadow, and downloaded an unrecognized binary to /tmp."
}' | tsentry --stdin
```

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
[ALERT] 2026-04-22T12:30:00Z  risk=HIGH      type=ssh
  user:    alice
  cluster: prod.example.com
  session: abc-123-def-456
  summary: User escalated to root via sudo -i, accessed /etc/shadow, and
           downloaded an unrecognized binary to /tmp.

  ▶ tctl lock --user=alice --ttl=15m0s --message="tsentry: HIGH risk AI session summary (session: abc-123-def-456)"
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

## Requirements

- Teleport with AI session summarization enabled (enterprise feature)
- `tsh` logged in to your cluster
- A role with `event: [list, read]` — see [Permissions](#permissions) below
- For `--auto-lock`: also `lock: [create, update]`

## Installation

### Homebrew

```bash
brew install --cask jsabo/tap/tsentry
```

### Download a release

Download the binary for your platform from the [Releases](https://github.com/jsabo/tsentry/releases) page, extract it, and place `tsentry` on your `PATH`.

Verify the download:

```bash
sha256sum --check checksums.txt --ignore-missing
```

### Build from source

```bash
go install github.com/jsabo/tsentry/cmd/tsentry@latest
```

## Usage

### Alert mode (default)

Connects to Teleport using your active `tsh` session, polls for new `session.summarized` events every 30 seconds, and prints a formatted alert to stdout for every session at or above the risk threshold. The `tctl lock` command is printed but **not executed**.

```bash
tsentry
tsentry --look-back=1h --risk-threshold=medium
```

### Auto-lock mode

Same as alert mode, but also executes `tctl lock` for each session above the threshold. The lock expires after `--lock-ttl` (default 15 minutes).

```bash
tsentry --auto-lock
tsentry --auto-lock --lock-ttl=30m --risk-threshold=critical
```

### Stdin mode

Reads newline-delimited JSON events from stdin. No Teleport connection is required. Useful for testing output format, replaying exported audit logs, or piping events from other tooling. Auto-lock is always disabled in this mode.

```bash
cat audit-export.ndjson | tsentry --stdin
echo '{"event":"session.summarized","risk_level":"critical",...}' | tsentry --stdin --risk-threshold=medium
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `--proxy` | from tsh profile | Teleport proxy address |
| `--cluster` | from tsh profile | Teleport cluster name |
| `--poll-interval` | `30s` | How often to poll for new events |
| `--look-back` | `5m` | How far back to scan on first startup |
| `--risk-threshold` | `high` | Minimum risk level to alert/lock: `low` `medium` `high` `critical` |
| `--auto-lock` | `false` | Execute `tctl lock` (default: print command only) |
| `--lock-ttl` | `15m` | Duration of the lock when `--auto-lock` is set |
| `--stdin` | `false` | Read newline-delimited JSON events from stdin |
| `--hide-unscored` | `false` | Suppress events where risk level could not be determined |
| `--identity` | _(unset)_ | Path to tbot identity file (overrides tsh profile) |
| `--tsh-profile-dir` | `~/.tsh` | Override tsh profile directory |

## Permissions

Create a role for tsentry and assign it to the operator (or tbot) running the tool:

```yaml
kind: role
version: v7
metadata:
  name: tsentry
spec:
  allow:
    rules:
      # Required: read audit log events
      - resources: [event]
        verbs: [list, read]
      # Required only for --auto-lock
      - resources: [lock]
        verbs: [create, update]
```

```bash
tctl create tsentry-role.yaml
tctl users update <your-user> --set-roles=...,tsentry
```

## Teleport API dependency

tsentry uses the public [`github.com/gravitational/teleport/api`](https://github.com/gravitational/teleport/tree/master/api) module — the same client library used by all Teleport integrations and plugins. This module is open source (Apache 2.0) and carries no enterprise dependency.

The `go.mod` is pinned to a specific commit pseudo-version. When you upgrade Teleport, update the pin:

```bash
go get github.com/gravitational/teleport/api@<commit-or-pseudo-version>
go mod tidy
```

### Building against a local Teleport checkout

If you are developing tsentry alongside a local Teleport clone, use Go workspaces to override the pinned version without modifying `go.mod`:

```bash
# Layout expected:  ~/Source/teleport/  and  ~/Source/tsentry/
make workspace          # runs: go work init && go work use . ../teleport/api
go build ./cmd/tsentry  # uses local ../teleport/api automatically
```

`go.work` is gitignored and will not affect other contributors or CI.

## Releases

Releases are built by [GoReleaser](https://goreleaser.com) via GitHub Actions when a version tag is pushed. No additional secrets or variables are required.

**Platforms:** `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, `windows/amd64`

To cut a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Artifacts and a SHA-256 checksum file are attached to the GitHub Release automatically.

## Roadmap

- [ ] tbot machine identity (flag already wired: `--identity`)
- [ ] Notification hooks (Slack, PagerDuty)
- [ ] True event streaming if Teleport exposes a push-based audit log watch
- [ ] Per-restart lock deduplication (persist seen event IDs across restarts)

## License

[Apache 2.0](LICENSE)
