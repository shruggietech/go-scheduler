# go-scheduler

[![CI](https://github.com/shruggietech/go-scheduler/actions/workflows/ci.yml/badge.svg)](https://github.com/shruggietech/go-scheduler/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/shruggietech/go-scheduler)](https://github.com/shruggietech/go-scheduler/releases/latest)
[![License](https://img.shields.io/badge/license-Apache--2.0-blue)](LICENSE)

A cross-platform (Linux · macOS · Windows) **task scheduler** written in Go — cron-level power
without the cryptic syntax. CLI-first, with a Go-native Material Design desktop GUI built on top.

> **Status:** Feature-complete (spec-driven via [Spec Kit](https://github.com/github/spec-kit)).
> The CLI, daemon, and **Material Design desktop GUI** are all implemented and tested:
> human-readable & one-off scheduling, per-task timezones with DST handling, nested groups,
> event triggers, downtime catch-up, and a Fyne GUI (calendar, guided editor with live preview,
> live alerts). See [CHANGELOG.md](CHANGELOG.md). The GUI requires a C toolchain + OpenGL to
> build (CI/releases handle this); the daemon and CLI are cgo-free.

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
- ✅ **Material Design desktop GUI** — calendar/schedule views, guided task editor with live
  schedule preview, group tree, trigger config, and live alerts. Opening the GUI never leaves a
  visible console window (`gosched gui`).

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

Each [release](https://github.com/shruggietech/go-scheduler/releases/latest) ships two kinds of
archive. Verify downloads against `SHA256SUMS.txt`.

### Desktop (recommended) — one self-contained download

`go-scheduler-desktop_<ver>_<os>_<arch>` bundles the **GUI + daemon + CLI** together (Linux,
macOS, Windows). Extract it and run the GUI — it **auto-starts the background daemon** the first
time, so there's nothing to configure:

```sh
./gosched-gui        # opens the window; starts the daemon in the background if needed
```

> The auto-started daemon keeps running so your tasks fire even after you close the window. For
> a daemon that also **starts on boot**, install it as a system service (below).

### Server / headless — daemon + CLI only

`go-scheduler_<ver>_<os>_<arch>` contains just `goschedd` + `gosched` (all platforms, amd64 +
arm64). Register the service to start on boot:

```sh
sudo ./gosched service install   # admin/root required
sudo ./gosched service start
./gosched health                 # expect: daemon ok
./gosched task add hello --command /usr/bin/true --schedule "every weekday at 09:00"
```

**Windows users:** see the step-by-step [Windows install guide](docs/INSTALL-windows.md).

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
