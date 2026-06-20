# Contract: Local API (daemon ↔ clients)

The daemon (`goschedd`) serves an HTTP/JSON API over a **local transport** consumed by both the
CLI and the GUI through the shared `internal/api/client`:

- **Linux/macOS**: Unix domain socket (e.g. `/var/run/goschedd.sock` or XDG runtime dir).
- **Windows**: named pipe (e.g. `\\.\pipe\goschedd`) via `go-winio`.
- Not a TCP port — access is governed by OS socket/pipe permissions (admin group for management).
- Content type `application/json`; timestamps RFC 3339 (UTC); errors use a consistent envelope.

This is a **local, single-host** contract (not a public network API). Versioned via `/v1`.

## Error envelope

```json
{ "error": { "code": "validation_failed", "field": "timezone", "message": "unknown IANA zone 'Mars/Phobos'" } }
```

Codes: `validation_failed` (→ CLI exit 2), `not_found`, `conflict`, `internal`.

## Resources & operations

### Tasks
- `GET    /v1/tasks` — list (filters: `group`, `state`)
- `POST   /v1/tasks` — create (body = task + schedule spec). Server validates timezone, schedule,
  and rejects past one-offs.
- `GET    /v1/tasks/{id}` — detail incl. computed `next_runs` (UTC + rendered local) and recent runs
- `PATCH  /v1/tasks/{id}` — update
- `DELETE /v1/tasks/{id}` — delete
- `POST   /v1/tasks/{id}:run-now` — manual run
- `POST   /v1/tasks/{id}:enable` / `:disable`

### Schedules (preview)
- `POST   /v1/schedules:preview` — body = human-readable schedule spec; returns `{ rrule,
  human_summary, next_runs[] }` so CLI/GUI can show the plain-language summary before saving (FR-006).

### Groups
- `GET/POST /v1/groups`, `GET/PATCH/DELETE /v1/groups/{id}`, `:enable`/`:disable`
- Tree returned with `parent_id`; server rejects cycles (`conflict`).

### Triggers
- `GET/POST /v1/triggers`, `DELETE /v1/triggers/{id}`

### Runs (history)
- `GET /v1/runs` — filters: `task`, `since`, `until`, `outcome`. Powers calendar/timeline (FR-023).

### Alerts
- `GET /v1/alerts` (filter `unacked`), `POST /v1/alerts/{id}:ack`

### Calendar
- `GET /v1/calendar?from=<RFC3339>&to=<RFC3339>` — materialized occurrences (past runs + computed
  future runs) for the GUI calendar/timeline views.

### Health
- `GET /v1/health` — daemon liveness + version (used by CLI to detect "daemon unreachable").

## Streaming (GUI live updates)
- `GET /v1/events` (Server-Sent Events) — pushes run-state changes and new alerts so the GUI can
  surface overlap/failure alerts within seconds (SC-005) without polling.

## Notes
- The server is the only writer to the SQLite store; clients hold no scheduling state (FR-026).
- All mutating endpoints are transactional; overlap/catch-up/dedup bookkeeping is persisted before
  acknowledging the request.
