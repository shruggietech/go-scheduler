# Implementation Plan: Cross-Platform Task Scheduler

**Branch**: `001-task-scheduler` | **Date**: 2026-06-19 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/001-task-scheduler/spec.md`

## Summary

A cross-platform (Linux/macOS/Windows) task scheduler written in Go with cron-equivalent power
expressed in human-readable terms. The architecture is **client/server within one machine**: a
long-lived **daemon** (`goschedd`) registered as a system-wide service hosts the scheduling
engine, persistence, and executor; a **CLI** (`gosched`) and a **Go-native desktop GUI**
(`gosched-gui`, built with Fyne/Material Design) are thin clients that talk to the daemon over a
local IPC API. This guarantees the CLI and GUI operate on identical state (FR-026) and lets the
engine keep running regardless of which client is open — or whether any is.

The GUI binary is built as a **windowless** application (`-H windowsgui` on Windows) and the
executor spawns task processes with **no console window**, satisfying the requirement that
opening the GUI never flashes or leaves a visible command prompt.

## Technical Context

**Language/Version**: Go 1.23+ (latest stable minor at implementation time; modules)

**Primary Dependencies**:
- `fyne.io/fyne/v2` — Go-native, Material Design desktop GUI (FR-022, FR-025)
- `github.com/kardianos/service` — cross-platform system-service install/boot registration (FR-009)
- `modernc.org/sqlite` — pure-Go (cgo-free) SQLite for queryable persistence (FR-008, FR-015)
- `github.com/teambition/rrule-go` — RFC 5545 recurrence engine for ordinal/interval rules (FR-002, FR-003)
- `github.com/spf13/cobra` — consistent verb-noun CLI command tree (FR-021)
- `github.com/Microsoft/go-winio` — Windows named-pipe transport for local IPC (Windows only)
- Standard library: `log/slog` (structured logging), `net/http` (API over local listener), `time`/`time/tzdata` (timezones/DST), `os/exec` + `syscall` (windowless task spawning)

**Storage**: Embedded SQLite database file in the daemon's data directory (tasks, groups,
schedules, triggers, run history). UTC-only timestamps in storage.

**Testing**: `go test -race`; table-driven unit tests with an injected `Clock` interface;
integration tests for persistence/recovery/concurrency under `test/integration/`; benchmarks
(`go test -bench`) for dispatch latency.

**Target Platform**: Linux, macOS, Windows (64-bit). Single codebase, per-OS service + windowing
adapters behind build tags.

**Project Type**: Multi-binary single Go module — `cmd/{goschedd,gosched,gosched-gui}` over
shared `internal/` packages (desktop-app + CLI + background service).

**Performance Goals**: Dispatch latency (scheduled time → execution start) p99 < 100ms under
nominal load (per constitution); no goroutine/memory leaks under sustained operation.

**Constraints**:
- No visible console window when the GUI runs; spawned task processes start with no console
  window on Windows (`CREATE_NO_WINDOW`/`HideWindow`).
- cgo-free build to keep cross-compilation simple (drives the pure-Go SQLite choice). (Fyne
  requires a C toolchain/OpenGL only for the GUI binary; daemon and CLI remain cgo-free.)
- Internal scheduling computed in UTC; per-task timezone applied at the edges with DST rules
  (next-valid / first-occurrence).

**Scale/Scope**: Target up to ~10,000 enabled tasks per machine with sub-second scheduling
accuracy; three supported OSes; single local user for the GUI in v1.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Derived from [constitution.md](../../.specify/memory/constitution.md) v1.0.0.

| Principle | Gate | Status |
|-----------|------|--------|
| I. Code Quality | `gofmt`/`go vet`/`golangci-lint` clean; errors wrapped with `%w`; every goroutine has a documented owner + termination path; `-race` passes | ✅ PASS — engine uses one scheduler goroutine + a bounded worker pool, both with explicit `context.Context` cancellation and shutdown drain |
| II. Testing (NON-NEGOTIABLE) | Injectable `Clock`; unit tests for scheduling/time/error paths; integration for persistence/recovery/concurrency; `-race`; ≥80% coverage on core packages | ✅ PASS — `Clock` interface is a first-class design element; time is never read via `time.Now()` directly in engine code |
| III. UX Consistency | CLI verb-noun + `--json` + stdout/stderr split + exit codes; single config schema with fail-fast validation; RFC 3339 times; structured logging | ✅ PASS — Cobra command tree, shared API client, `slog` JSON logs, one documented config file |
| IV. Performance | Documented dispatch-latency budget; benchmarks; bounded resources; no leaks | ✅ PASS — p99 < 100ms budget lives beside the dispatcher; benchmark + leak test included in plan |

**Dependency justifications** (constitution requires each new dependency be justified):
SQLite reimplementation, a cross-platform service manager, an RFC-5545 recurrence engine, and a
Material Design widget toolkit are all large, correctness-sensitive subsystems with no adequate
stdlib equivalent; reimplementing any would add more risk than it removes. `cobra` and
`go-winio` are near-standard, narrowly scoped, and license-clean. Structured logging uses stdlib
`log/slog` (no dependency).

**Result**: No violations. Complexity Tracking section intentionally empty.

## Project Structure

### Documentation (this feature)

```text
specs/001-task-scheduler/
├── plan.md              # This file (/speckit-plan command output)
├── research.md          # Phase 0 output — decisions & rationale
├── data-model.md        # Phase 1 output — entities, fields, state transitions
├── quickstart.md        # Phase 1 output — runnable validation guide
├── contracts/           # Phase 1 output — CLI command + local API contracts
│   ├── cli.md
│   └── local-api.md
└── tasks.md             # Phase 2 output (/speckit-tasks — NOT created here)
```

### Source Code (repository root)

```text
cmd/
├── goschedd/            # Daemon: hosts engine, runs as system service, serves local API
│   └── main.go
├── gosched/             # CLI client (cobra verb-noun commands)
│   └── main.go
└── gosched-gui/         # Go-native desktop GUI client (Fyne); built windowless
    └── main.go

internal/
├── engine/              # Scheduling loop, dispatcher, worker pool, Clock interface
├── schedule/            # Schedule model + human-readable parse/summarize (rrule-backed); one-off + recurrence
├── task/                # Task & Group domain model, lifecycle/state
├── trigger/             # Task-completion event triggers + at-least-once + dedup window/key
├── store/               # SQLite persistence (tasks, groups, schedules, runs, dedup ledger)
├── executor/            # Runs commands; windowless spawn; captures output/outcome
├── catchup/             # Missed-run detection + one-catch-up policy
├── timezone/            # Per-task tz + DST resolution (next-valid / first-occurrence)
├── api/                 # Local API: server (in daemon) + Go client (shared by CLI & GUI)
│   ├── server/
│   └── client/
├── ipc/                 # Transport: unix domain socket (unix) / named pipe (windows, build-tagged)
├── service/             # kardianos/service integration: install/boot/run lifecycle
├── config/              # Single config schema, defaults, fail-fast validation
└── platform/            # Build-tagged OS specifics (windowless spawn flags, data dirs)

gui/                     # Fyne views: calendar view, schedule/timeline, task editor, alerts
test/
└── integration/         # Cross-package integration tests (persistence, recovery, concurrency)
```

**Structure Decision**: A single Go module with three binaries under `cmd/` sharing `internal/`
packages. The engine lives only in the daemon; CLI and GUI never touch the store directly — they
go through `internal/api/client`, which guarantees one source of truth (FR-026) and keeps the
GUI/CLI thin. OS-specific behavior (service registration, windowless process spawning, named-pipe
vs unix-socket IPC) is isolated behind build tags in `internal/platform`, `internal/ipc`, and
`internal/service`.

## Complexity Tracking

> No constitution violations — section intentionally empty.
