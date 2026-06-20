# go-scheduler

[![CI](https://github.com/shruggietech/go-scheduler/actions/workflows/ci.yml/badge.svg)](https://github.com/shruggietech/go-scheduler/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/shruggietech/go-scheduler)](https://github.com/shruggietech/go-scheduler/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

A cross-platform (Linux · macOS · Windows) **task scheduler** written in Go — cron-level power
without the cryptic syntax. CLI-first, with a Go-native Material Design desktop GUI built on top.

> **Status:** Active development (spec-driven via [Spec Kit](https://github.com/github/spec-kit)).
> The **CLI + daemon are functional**: human-readable & one-off scheduling, per-task timezones
> with DST handling, nested groups, event triggers, and downtime catch-up all work and are
> tested. The **Material Design GUI (US2) is not yet built** — it needs a C toolchain (OpenGL)
> for Fyne. See [TODO.md](TODO.md) and [CHANGELOG.md](CHANGELOG.md) for current state.

## Why

`cron` is powerful but its `*/15 * * * *` syntax is hard to read and easy to get wrong.
go-scheduler gives you the same scheduling power expressed in plain language — "every 15 minutes",
"every weekday at 9:00 AM", "the 3rd Wednesday of each month", or a single one-off run — with a
calendar/timeline GUI for managing it all.

## Features

Implemented (✅) and planned:

- ✅ **Human-readable schedules** — recurring and one-off, no cron strings.
- ✅ **Cron parity** — anything cron can express, this can too (intervals, ordinal weekdays, etc.).
- ✅ **Per-task timezones** with correct Daylight Saving Time handling; UTC backend.
- ✅ **Nested task groups** (groups within groups) with cascading enable/disable.
- ✅ **Event triggers** — run a task when another task completes (at-least-once, with dedup).
- ✅ **Downtime catch-up** — one catch-up run per task after missed runs, then resume.
- ✅ **Overlap control** — queue-one-pending by default, configurable per task, with alerts.
- ✅ **Starts on boot** — runs as a system-wide service (systemd / launchd / Windows Service);
  single-instance guarded.
- 🚧 **Material Design desktop GUI** — calendar/schedule views, guided task editor, live alerts.
  Opening the GUI never leaves a visible console window. *(Not yet built — needs a C toolchain.)*

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

## Install

Download the archive for your platform from the
[latest release](https://github.com/shruggietech/go-scheduler/releases/latest) (binaries are
provided for Linux, macOS, and Windows on amd64 and arm64), verify it against `SHA256SUMS.txt`,
extract, then register the system service:

```sh
sudo ./gosched service install   # admin/root required
sudo ./gosched service start
./gosched task add hello --command /usr/bin/true --schedule "every weekday at 09:00"
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
