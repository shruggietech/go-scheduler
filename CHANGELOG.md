# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Spec-driven development scaffolding via Spec Kit:
  - Project constitution (v1.0.0) — code quality, testing standards, UX consistency, performance.
  - Feature specification for the cross-platform task scheduler (`specs/001-task-scheduler/`),
    including clarifications and a one-off (non-recurring) scheduling mode.
  - Implementation plan, research, data model, CLI & local-API contracts, and quickstart.
  - Dependency-ordered task breakdown (78 tasks across 8 phases).
- Repository basics: Apache 2.0 license, README, changelog, and TODO.
- **Foundational implementation (Phases 1–2, tasks T001–T019):**
  - Go module, `golangci-lint` config, `Makefile`, and `.gitattributes`.
  - `internal/platform` — build-tagged data dirs and windowless process-spawn helper.
  - `internal/clock` — injectable `Clock` with real and deterministic fake implementations.
  - `internal/config` — single config schema, fail-fast validation, structured `slog` logging.
  - `internal/domain` + `internal/store` — core entities and durable SQLite persistence
    (pure-Go, cgo-free) with migrations and CRUD.
  - `internal/ipc` — local transport (Unix socket / Windows named pipe).
  - `internal/api` — local HTTP/JSON API server (health, error envelope) and shared client.
  - `cmd/goschedd` (daemon) and `cmd/gosched` (CLI): the daemon serves health over IPC and the
    CLI reaches it — end-to-end architecture verified.
- **User Story 1 — MVP (Phase 3, tasks T020–T037, T074–T078):**
  - `internal/timezone` — IANA resolution and DST rules (next-valid spring-forward,
    first-occurrence fall-back), verified against 2026 US transitions.
  - `internal/schedule` — RFC 5545 RRULE recurrence (rrule-go), one-off, and a human-readable
    parser with plain-language summaries (no cron syntax); cron-parity suite.
  - `internal/engine` — timer-driven scheduling loop over an injected clock, bounded worker
    pool, one-off completion, failure alerts; overlap policies (queue_one / skip /
    allow_concurrent) with warning + alert.
  - `internal/executor` — windowless command execution with bounded output capture; build-tagged
    `run_as` (Unix credential impersonation; rejected on Windows for now).
  - Local API: task CRUD + edit (PATCH), `schedules/preview`, `run-now`, enable/disable, and
    run/alert queries. Full cobra CLI: `task`, `runs`, `alerts`, `service`, `gui`, with `--json`
    and contract-compliant exit codes.
  - `internal/service` — cross-platform system-service control (install/start/stop/status) via
    kardianos; the daemon runs under the OS service manager (start on boot).
  - Verified end-to-end: create recurring + one-off tasks via CLI, run them, inspect history and
    failure alerts; DST handled correctly across the year.
- **User Story 3 — Nested task groups (Phase 5, tasks T049–T054):**
  - `internal/task` — pure, testable group-tree logic: cascading enabled-state resolution,
    descendant enumeration, cycle detection, forest building.
  - `internal/store` — group chain-enabled queries, parent validation, reparent with cycle
    rejection, rename, and tree retrieval.
  - Engine respects the group chain: disabling an ancestor group stops its tasks from being
    scheduled (without mutating each task's own enabled flag); re-enabling restores them.
  - Local API: group CRUD, tree view, reparent (PATCH), enable/disable. CLI: `group add/list
    [--tree]/enable/disable/rm`.
  - Verified end-to-end: 3-level hierarchy, cascade disable, cycle rejection.
  - Note: the GUI group tree (T055) is deferred until the US2 GUI exists.
- **User Story 4 — Event triggers (Phase 6, tasks T056–T061):**
  - `internal/trigger` — completion-event dispatcher: matches a source task's
    success/failure/any outcome to triggers and fires target tasks, with durable
    de-duplication (window + key) and at-least-once recovery across restarts.
  - `internal/store` — triggers and a dedup ledger (claim/mark-executed/pending),
    schema migration v2.
  - Engine wiring: a completion hook fires triggers after each run; a startup hook
    recovers unexecuted events. New `FireEvent` dispatches targets as event runs.
  - Local API: trigger CRUD; CLI: `trigger add/list/rm`.
  - Verified end-to-end: source completion fires the target once (recorded as an
    `event` run); duplicates within the window are de-duplicated.
  - Note: the GUI trigger editor field (T062) is deferred until the US2 GUI exists.
- **User Story 5 — Downtime catch-up (Phase 7, tasks T063–T066):**
  - `internal/catchup` — pure detection: given a task's schedule, last run, and
    policy, decide whether a scheduled run was missed during downtime.
  - Engine startup performs at most one catch-up run per eligible task (recorded
    as a `catchup` trigger at startup time, so a restart never re-triggers it),
    raises a `missed_run` alert, then resumes normal scheduling. Honors the
    per-task catch-up policy (`one` / `none`) and the overlap policy via dispatch.
  - Verified end-to-end: a short-interval task left across real downtime performs
    exactly one catch-up run and then resumes.
- **Polish & hardening (Phase 8, tasks T067–T071; T072/T073 partial):**
  - `internal/lock` — cross-platform single-instance guard (flock / LockFileEx); a
    second daemon now fails fast instead of double-executing every task (T070).
  - Goroutine-leak test (no leak after 500 executions) and a dispatch benchmark
    (~36µs per run — far under the 100ms budget) (T068, T069).
  - Test coverage raised to ≥80% statements on all core packages — engine, schedule,
    timezone, store, trigger, catchup (T071).
  - README updated to reflect functional CLI/daemon; daemon + CLI cross-compile
    cleanly for linux/macOS/windows on amd64 + arm64 (T067, T072 partial).
  - Deferred (need the US2 GUI): windowless-GUI verification (T072) and the GUI
    success criterion SC-008 (T073). Other success criteria verified via live CLI
    tests.
- **User Story 2 — Material Design desktop GUI (Phase 4, tasks T038–T048, T055, T062):**
  - `gui/` — Fyne desktop app with tabs for Tasks, Schedule (calendar/timeline),
    Groups (tree), Triggers, and Alerts, using Fyne's Material-style theme. The
    guided task editor shows a live plain-language schedule preview (FR-006); the
    alerts panel updates live and carries an unacknowledged badge.
  - `internal/events` — in-process pub/sub broker; API `GET /v1/events` streams
    run/alert events over SSE and `GET /v1/calendar` materializes occurrences.
  - `gui/viewmodel` — pure, unit-tested GUI state; the Fyne widget layer is
    cgo-free and unit-tested headlessly. Only `cmd/gosched-gui` (the OpenGL
    window) needs cgo; a cgo-free stub keeps `go build ./...` working everywhere.
  - CI builds the GUI with cgo + OpenGL and runs the headless GUI tests; releases
    publish `gosched-gui` for Linux, macOS, and Windows (windowless on Windows).
- **Zero-config desktop experience:**
  - `internal/autostart` — the GUI now starts the background daemon automatically
    (detached, windowless) if none is reachable, and reuses an already-running one
    (e.g. the installed service); the daemon's single-instance lock prevents
    duplicates.
  - Releases now publish a self-contained `go-scheduler-desktop_<os>_<arch>`
    archive bundling the GUI + daemon + CLI, so desktop users download one file and
    just run the GUI.

[Unreleased]: https://github.com/shruggietech/go-scheduler/commits/main
