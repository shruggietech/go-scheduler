# go-scheduler

A cross-platform (Linux · macOS · Windows) **task scheduler** written in Go — cron-level power
without the cryptic syntax. CLI-first, with a Go-native Material Design desktop GUI built on top.

> **Status:** Early development. The specification, plan, and task breakdown are complete
> (spec-driven development via [Spec Kit](https://github.com/github/spec-kit)); implementation is
> just beginning. See [the spec](specs/001-task-scheduler/spec.md) and [TODO.md](TODO.md).

## Why

`cron` is powerful but its `*/15 * * * *` syntax is hard to read and easy to get wrong.
go-scheduler gives you the same scheduling power expressed in plain language — "every 15 minutes",
"every weekday at 9:00 AM", "the 3rd Wednesday of each month", or a single one-off run — with a
calendar/timeline GUI for managing it all.

## Features (planned)

- **Human-readable schedules** — recurring and one-off, no cron strings.
- **Cron parity** — anything cron can express, this can too (intervals, ordinal weekdays, etc.).
- **Starts on boot** — runs as a system-wide service (systemd / launchd / Windows Service).
- **Per-task timezones** with correct Daylight Saving Time handling; UTC backend.
- **Nested task groups** (groups within groups) with cascading enable/disable.
- **Event triggers** — run a task when another task completes (at-least-once, with dedup).
- **Downtime catch-up** — one catch-up run per task after missed runs, then resume.
- **Overlap control** — queue-one-pending by default, configurable per task, with alerts.
- **Material Design desktop GUI** — calendar/schedule views, guided task editor, live alerts.
  Opening the GUI never leaves a visible console window.

## Architecture

A single background **daemon** (`goschedd`) hosts the scheduling engine, persistence, and
executor, and is registered as a system service. The **CLI** (`gosched`) and **desktop GUI**
(`gosched-gui`) are thin clients that talk to the daemon over local IPC (Unix socket / Windows
named pipe), so both operate on identical state and the engine keeps running regardless of which
client is open.

See [specs/001-task-scheduler/plan.md](specs/001-task-scheduler/plan.md) for the full design.

## Project layout (target)

```text
cmd/        goschedd (daemon) · gosched (CLI) · gosched-gui (Fyne GUI)
internal/   engine · schedule · task · trigger · store · executor · catchup · timezone · api · ipc · service · config · platform
gui/        Fyne views (calendar, editor, alerts, groups)
specs/      spec-driven development artifacts (spec, plan, tasks, contracts)
```

## Development

Spec-driven via Spec Kit. The source of truth lives under
[`specs/001-task-scheduler/`](specs/001-task-scheduler/):

- [`spec.md`](specs/001-task-scheduler/spec.md) — what & why
- [`plan.md`](specs/001-task-scheduler/plan.md) — architecture & tech choices
- [`tasks.md`](specs/001-task-scheduler/tasks.md) — dependency-ordered task list
- [`contracts/`](specs/001-task-scheduler/contracts/) — CLI & local API contracts

Engineering standards are governed by the project
[constitution](.specify/memory/constitution.md): `gofmt`/`go vet`/`golangci-lint` clean,
`go test -race`, ≥80% coverage on core packages, and a documented dispatch-latency budget.

## License

Licensed under the [Apache License 2.0](LICENSE). © 2026 ShruggieTech.
