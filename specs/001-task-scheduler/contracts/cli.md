# Contract: CLI (`gosched`)

Verb-noun command tree (cobra). Global rules (constitution UX-consistency principle, FR-021):

- Results → **stdout**; diagnostics/errors → **stderr**.
- Global `--json` flag: machine-readable output for every command.
- Exit codes: `0` success; `1` runtime error; `2` usage/validation error.
- Times accepted/printed in **RFC 3339**; durations in Go duration syntax (`30s`, `15m`, `24h`).
- All commands operate via `internal/api/client` against the daemon — never the store directly.

## Task commands

| Command | Purpose | Key flags |
|---------|---------|-----------|
| `gosched task add <name>` | Create a task | `--command`, `--arg` (repeatable), `--cwd`, `--env K=V`, `--group`, `--tz`, `--schedule <spec>` or `--at <RFC3339>` (one-off), `--overlap queue_one\|skip\|allow`, `--catchup one\|none` |
| `gosched task list` | List tasks | `--group`, `--state`, `--json` |
| `gosched task show <id>` | Show task detail + next runs + recent history | `--json` |
| `gosched task edit <id>` | Modify fields | same as `add` |
| `gosched task enable\|disable <id>` | Toggle state | |
| `gosched task rm <id>` | Delete task | `--force` |
| `gosched task run-now <id>` | Trigger a manual run immediately | |

- `--schedule <spec>` accepts human-readable forms (e.g. `"every 15m"`, `"weekdays at 09:00"`,
  `"3rd wednesday monthly at 14:00"`). The CLI echoes the resulting plain-language summary before
  confirming (FR-006). `--at` creates a one-off (FR-004a); a past `--at` is rejected with exit 2.

## Group commands

| Command | Purpose |
|---------|---------|
| `gosched group add <name> [--parent <id>]` | Create group / sub-group (FR-019) |
| `gosched group list [--tree]` | List groups (tree view) |
| `gosched group enable\|disable <id>` | Toggle (cascades to descendants, FR-020) |
| `gosched group rm <id> [--recursive]` | Delete |

## Trigger commands

| Command | Purpose |
|---------|---------|
| `gosched trigger add --source <taskId> --target <taskId> [--on success\|failure\|any] [--dedup-window 5m]` | Chain on task completion (FR-007, FR-014) |
| `gosched trigger list` / `gosched trigger rm <id>` | Manage triggers |

## Run / history & alerts

| Command | Purpose |
|---------|---------|
| `gosched runs [--task <id>] [--since <RFC3339>] [--outcome ...]` | Query run history (FR-015) |
| `gosched alerts [--unacked]` / `gosched alerts ack <id>` | View/ack alerts (FR-024) |

## Service & GUI

| Command | Purpose |
|---------|---------|
| `gosched service install\|uninstall\|start\|stop\|status` | Manage the system-wide service (FR-009); install needs admin |
| `gosched gui` | Launch the desktop GUI client (`gosched-gui`) detached, **without a visible console window** |

- `gosched gui` starts the windowless GUI binary and returns; the GUI connects to the running
  daemon over the local IPC API. No CMD window remains open (explicit requirement).

## Error contract

- Validation failures (bad timezone, past one-off, malformed schedule, cycle in group parent)
  return exit `2` with an actionable stderr message naming the offending field.
- Daemon-unreachable returns exit `1` with guidance to check `gosched service status`.
