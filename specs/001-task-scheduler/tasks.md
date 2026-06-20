---
description: "Task list for Cross-Platform Task Scheduler implementation"
---

# Tasks: Cross-Platform Task Scheduler

**Input**: Design documents from `specs/001-task-scheduler/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/](contracts/)

**Tests**: Tests are **REQUIRED** for this feature. The project
[constitution](../../.specify/memory/constitution.md) v1.0.0 makes testing non-negotiable
(injected `Clock`, `-race`, ≥80% coverage on core packages, benchmarks). Test tasks are therefore
included in every phase and, where they verify behavior, are written before implementation.

**Organization**: Tasks are grouped by user story (from spec.md priorities) for independent
implementation and testing.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies on incomplete tasks)
- **[Story]**: US1–US5 maps to the spec's user stories; Setup/Foundational/Polish carry no label
- Paths follow [plan.md](plan.md): `cmd/`, `internal/`, `gui/`, `test/integration/`

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and tooling.

- [ ] T001 Initialize Go module `go-scheduler` and create the directory skeleton (`cmd/{goschedd,gosched,gosched-gui}`, `internal/`, `gui/`, `test/integration/`) per plan.md
- [ ] T002 Add and pin dependencies in `go.mod`: `fyne.io/fyne/v2`, `github.com/kardianos/service`, `modernc.org/sqlite`, `github.com/teambition/rrule-go`, `github.com/spf13/cobra`, `github.com/Microsoft/go-winio`, plus `time/tzdata`
- [ ] T003 [P] Configure `golangci-lint` (`.golangci.yml`) and a `Makefile`/`Taskfile` with `fmt`, `vet`, `lint`, `test`, `bench`, `build` targets
- [ ] T004 [P] Configure CI to run `gofmt -l`, `go vet`, `golangci-lint`, `go test -race`, coverage gate (≥80% core), and dispatch benchmark on Linux/macOS/Windows
- [ ] T005 [P] Add `internal/platform` build-tagged stubs for per-OS data directories and windowless process-spawn flags (`platform_windows.go`, `platform_unix.go`)

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure every user story depends on.

**⚠️ CRITICAL**: No user story work begins until this phase is complete.

- [ ] T006 [P] Define the `Clock` interface (`Now`, `After`, `NewTimer`) with real + fake implementations in `internal/engine/clock.go`
- [ ] T007 [P] Unit-test the fake `Clock` (deterministic advance/timers) in `internal/engine/clock_test.go`
- [ ] T008 [P] Implement config schema, defaults, and fail-fast validation in `internal/config/config.go` (data dir, IPC path, admin group, default tz, log level/format, output cap, worker-pool size)
- [ ] T009 [P] Unit-test config validation (bad tz, bad paths, defaults) in `internal/config/config_test.go`
- [ ] T010 [P] Set up `log/slog` structured logging (JSON + human handlers, consistent fields) in `internal/config/logging.go`
- [ ] T011 Implement SQLite store bootstrap + schema migrations (modernc.org/sqlite) in `internal/store/store.go` and `internal/store/migrations/`
- [ ] T012 [P] Implement base domain models Group, Task, Schedule, Run, Alert (UTC timestamps, enums) in `internal/task/` and `internal/schedule/types.go`
- [ ] T013 Implement store CRUD for Group, Task, Schedule, Run, Alert (transactional) in `internal/store/*.go`
- [ ] T014 Integration-test store CRUD + durability across reopen in `test/integration/store_test.go`
- [ ] T015 Implement IPC transport: Unix domain socket (`ipc_unix.go`) and Windows named pipe via go-winio (`ipc_windows.go`) behind build tags in `internal/ipc/`
- [ ] T016 Implement local API server skeleton (router, JSON error envelope, `/v1/health`) over the IPC listener in `internal/api/server/server.go`
- [ ] T017 [P] Implement shared API client in `internal/api/client/client.go` (used by CLI + GUI)
- [ ] T018 [P] Contract-test `/v1/health` and error envelope in `internal/api/server/health_test.go`
- [ ] T019 Implement daemon skeleton wiring config → store → api server → (engine placeholder) in `cmd/goschedd/main.go`

**Checkpoint**: Daemon starts, serves health over local IPC, persists base entities.

---

## Phase 3: User Story 1 - Schedule a task without cron syntax (Priority: P1) 🎯 MVP

**Goal**: Create time-based and one-off tasks in human-readable terms; the daemon runs them
reliably, survives reboot via the system service, and records run history — no cron syntax.

**Independent Test**: Create a task with a human-readable recurrence and a one-off via the CLI;
verify both execute at expected times under the fake clock and in a live run; restart the service
and confirm resumption.

### Tests for User Story 1

- [ ] T020 [P] [US1] Unit-test human-readable schedule parsing + plain-language summary in `internal/schedule/parse_test.go`
- [ ] T021 [P] [US1] Unit-test RRULE next-run for interval and ordinal-weekday ("3rd Wednesday monthly") in `internal/schedule/recur_test.go`
- [ ] T022 [P] [US1] Unit-test one-off scheduling + past-time rejection + post-run `completed` state in `internal/schedule/oneoff_test.go`
- [ ] T023 [P] [US1] Unit-test timezone + DST resolution (next-valid spring-forward, first-occurrence fall-back) in `internal/timezone/dst_test.go`
- [ ] T024 [P] [US1] Unit-test all three overlap policies — `queue_one` (queue once, warn, drop extras), `skip`, `allow_concurrent` — in `internal/engine/overlap_test.go`
- [ ] T025 [P] [US1] Contract-test `POST /v1/tasks`, `POST /v1/schedules:preview`, `POST /v1/tasks/{id}:run-now` in `internal/api/server/tasks_test.go`
- [ ] T026 [US1] Integration-test: create recurring + one-off task → fake clock advances → correct runs recorded; restart daemon → schedule resumes, in `test/integration/scheduling_test.go`
- [ ] T074 [P] [US1] **Cron-parity equivalence suite** (SC-002): map representative cron patterns to human-readable configs and assert matching run times in `internal/schedule/cronparity_test.go`

### Implementation for User Story 1

- [ ] T027 [P] [US1] Implement timezone + DST resolver (UTC conversion, next-valid/first-occurrence rules) in `internal/timezone/timezone.go`
- [ ] T028 [US1] Implement schedule model: RRULE mapping (rrule-go), one-off, next-run computation in task tz → UTC in `internal/schedule/recur.go`
- [ ] T029 [US1] Implement human-readable parse + summarize + preview in `internal/schedule/parse.go`
- [ ] T030 [US1] Implement scheduling engine: timer-driven loop + dispatcher + bounded worker pool with `context` cancellation/drain in `internal/engine/engine.go`
- [ ] T031 [US1] Implement all three configurable overlap policies — `queue_one` (default: queue one pending + warning log + Alert creation hook), `skip`, and `allow_concurrent` — in `internal/engine/overlap.go`
- [ ] T032 [US1] Implement executor: spawn command **windowless** (CREATE_NO_WINDOW/HideWindow), capture stdout/stderr (bounded), write Run record, in `internal/executor/executor.go`
- [ ] T033 [US1] Wire one-off completion → task `state=completed` (no re-arm) in `internal/engine/engine.go`
- [ ] T034 [US1] Implement API task endpoints (`/v1/tasks` CRUD, `:run-now`, `:enable`/`:disable`, `schedules:preview`) in `internal/api/server/tasks.go`
- [ ] T035 [US1] Implement CLI `gosched task add/list/show/edit/enable/disable/rm/run-now` with `--json`, stdout/stderr split, exit codes (cobra) in `cmd/gosched/` and `internal/cli/task.go`
- [ ] T036 [US1] Implement `gosched service install/uninstall/start/stop/status` (kardianos/service, system-wide, boot registration) in `internal/service/service.go` and `internal/cli/service.go`
- [ ] T037 [US1] Integrate the engine into the daemon run loop (replace placeholder) and start it from the service entrypoint in `cmd/goschedd/main.go`
- [ ] T075 [US1] Implement run-history + alerts **query** API (`GET /v1/runs` with filters, `GET /v1/alerts`, `POST /v1/alerts/{id}:ack`) in `internal/api/server/runs.go` (G2)
- [ ] T076 [US1] Implement CLI `gosched runs` and `gosched alerts [--unacked] / alerts ack <id>` (cobra, `--json`) in `internal/cli/runs.go` (G2)
- [ ] T077 [US1] Implement per-task `run_as` impersonation behind build tags (`internal/executor/runas_windows.go`, `runas_unix.go`); default to the service account; validate the account at task-create time (G4)
- [ ] T078 [P] [US1] Unit/integration-test `run_as` (impersonation applied, invalid account rejected, default fallback) in `internal/executor/runas_test.go` (G4)

**Checkpoint**: MVP — recurring + one-off tasks schedule, run windowless, persist history, and
resume after a service restart, all via the CLI. Independently demoable.

---

## Phase 4: User Story 2 - Visual calendar management with alerts (Priority: P2)

**Goal**: A Go-native Material Design desktop GUI with calendar/timeline views, a guided task
editor showing a live plain-language schedule summary, and prominent alerts — launched without any
visible console window.

**Independent Test**: With the daemon running, launch `gosched gui`; view tasks on the calendar,
create one via the form editor (appears in `gosched task list`), and confirm overlap/failure
alerts surface — with no CMD window present.

### Tests for User Story 2

- [ ] T038 [P] [US2] Contract-test `GET /v1/calendar` and `GET /v1/events` (SSE) in `internal/api/server/calendar_test.go`
- [ ] T039 [P] [US2] Unit-test calendar occurrence materialization (past runs + computed future) in `internal/api/server/calendar_occurrences_test.go`
- [ ] T040 [P] [US2] Unit-test GUI view-model state (task list/editor/alerts reducers) in `gui/viewmodel/viewmodel_test.go`

### Implementation for User Story 2

- [ ] T041 [US2] Implement `GET /v1/calendar` (occurrence materialization) and alerts endpoints in `internal/api/server/calendar.go`
- [ ] T042 [US2] Implement `GET /v1/events` SSE stream pushing run-state changes + new alerts in `internal/api/server/events.go`
- [ ] T043 [US2] Implement alert creation on overlap-queued and run-failure conditions in `internal/engine/alerts.go`
- [ ] T044 [P] [US2] Scaffold the Fyne app (`gosched-gui`) built **windowless** (`-H windowsgui`) with Material theme + SSE client in `cmd/gosched-gui/main.go` and `gui/app.go`
- [ ] T045 [P] [US2] Implement calendar + schedule/timeline views in `gui/calendar.go`
- [ ] T046 [US2] Implement guided task editor with live plain-language preview (calls `schedules:preview`) in `gui/editor.go`
- [ ] T047 [US2] Implement alerts/notifications surface bound to the SSE stream in `gui/alerts.go`
- [ ] T048 [US2] Implement CLI `gosched gui` that launches the GUI detached, windowless (no console) in `internal/cli/gui.go`

**Checkpoint**: US1 + US2 work; tasks are manageable visually with alerts and no console window.

---

## Phase 5: User Story 3 - Nested task groups (Priority: P2)

**Goal**: Organize tasks into groups and sub-groups (≥3 levels); enable/disable cascades.

**Independent Test**: Build a 3-level group hierarchy, assign tasks, disable a parent, and confirm
all contained tasks stop — in both CLI and GUI.

### Tests for User Story 3

- [ ] T049 [P] [US3] Unit-test cascade enable/disable + parent-cycle rejection in `internal/task/group_test.go`
- [ ] T050 [US3] Integration-test 3-level nesting + disable cascade stops runs in `test/integration/groups_test.go`

### Implementation for User Story 3

- [ ] T051 [US3] Implement group tree queries (descendants) + cycle prevention in `internal/store/group.go`
- [ ] T052 [US3] Implement group cascade enable/disable + effective-enabled resolution used by the engine in `internal/task/group.go`
- [ ] T053 [US3] Implement API group endpoints (CRUD, tree, `:enable`/`:disable`) in `internal/api/server/groups.go`
- [ ] T054 [US3] Implement CLI `gosched group add/list --tree/enable/disable/rm` in `internal/cli/group.go`
- [ ] T055 [US3] Implement GUI group tree view + assign-task-to-group in `gui/groups.go`

**Checkpoint**: US1–US3 independently functional.

---

## Phase 6: User Story 4 - Event-triggered tasks (Priority: P3)

**Goal**: Run a task on another task's completion with at-least-once delivery and dedup-window
exactly-once effect.

**Independent Test**: Configure B to trigger on A's success; run A → B runs once; deliver a
duplicate completion within the window → B does not run again; verify delivery survives a restart.

### Tests for User Story 4

- [ ] T056 [P] [US4] Unit-test dedup ledger (window/key collapse, executed flag) in `internal/trigger/dedup_test.go`
- [ ] T057 [US4] Integration-test completion → single run; duplicate-in-window → no second run; at-least-once across restart, in `test/integration/triggers_test.go`

### Implementation for User Story 4

- [ ] T058 [P] [US4] Implement Trigger + DedupLedger models and store in `internal/trigger/model.go` and `internal/store/trigger.go`
- [ ] T059 [US4] Implement completion-event emission + trigger evaluation + at-least-once delivery with dedup in `internal/trigger/dispatcher.go` (wired into engine)
- [ ] T060 [US4] Implement API trigger endpoints in `internal/api/server/triggers.go`
- [ ] T061 [US4] Implement CLI `gosched trigger add/list/rm` in `internal/cli/trigger.go`
- [ ] T062 [US4] Add trigger configuration to the GUI task editor in `gui/editor.go`

**Checkpoint**: US1–US4 independently functional.

---

## Phase 7: User Story 5 - Downtime catch-up (Priority: P3)

**Goal**: After downtime that missed ≥1 run, perform exactly one catch-up per task, then resume;
per-task configurable (`one`/`none`).

**Independent Test**: Stop the service across ≥1 scheduled time, restart, and confirm exactly one
`caught_up` run then normal scheduling; with `--catchup none`, zero catch-up runs.

### Tests for User Story 5

- [ ] T063 [P] [US5] Unit-test missed-run detection + one-catchup vs none + overlap interaction (fake clock) in `internal/catchup/catchup_test.go`
- [ ] T064 [US5] Integration-test downtime → exactly one catch-up → resume in `test/integration/catchup_test.go`

### Implementation for User Story 5

- [ ] T065 [US5] Implement catch-up detection (last-run vs schedule) + one-run-per-task policy honoring overlap in `internal/catchup/catchup.go`
- [ ] T066 [US5] Wire catch-up evaluation into engine startup; emit `missed_run` alert + `caught_up` Run records in `internal/engine/engine.go`

**Checkpoint**: All user stories independently functional.

---

## Phase 8: Polish & Cross-Cutting Concerns

- [ ] T067 [P] Write `README.md` (build, install, usage) and refresh `CLAUDE.md` references
- [ ] T068 [P] Add goroutine/memory leak test for the engine under sustained load in `test/integration/leak_test.go`
- [ ] T069 [P] Add dispatch-latency benchmark (`BenchmarkDispatch`) asserting p99 < 100ms in `internal/engine/engine_bench_test.go`
- [ ] T070 Harden local IPC access control (socket/pipe permissions, admin group) per research §2 in `internal/ipc/` and `internal/service/`
- [ ] T071 Verify coverage ≥80% on core packages (`engine`, `schedule`, `timezone`, `store`, `trigger`, `catchup`) and close gaps
- [ ] T072 Verify cross-platform build incl. Windows windowless GUI (`-H windowsgui`) and no-console task spawn on all three OSes
- [ ] T073 Execute [quickstart.md](quickstart.md) end-to-end and confirm every Success Criterion (SC-001..SC-010)

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (P1)**: no dependencies.
- **Foundational (P2)**: depends on Setup; **blocks all user stories**.
- **User Stories (P3–P7)**: each depends on Foundational. US1 is the MVP. US2/US3 build on US1's
  task/API surface; US4/US5 extend the engine. Stories are independently testable.
- **Polish (P8)**: depends on the targeted stories being complete.

### User Story Dependencies

- **US1 (P1)**: after Foundational. No dependency on other stories.
- **US2 (P2)**: after US1 (consumes task/API + adds calendar/alerts/GUI). Independently testable.
- **US3 (P2)**: after Foundational; integrates with US1 task model + US2 GUI but testable alone.
- **US4 (P3)**: after US1 (needs run/completion lifecycle). Independently testable.
- **US5 (P3)**: after US1 (needs schedule + run history). Independently testable.

### Within Each User Story

- Behavioral tests written before implementation (constitution: Test-aligned).
- Models → services → API endpoints → CLI/GUI.
- `Clock` injected everywhere time is read.

### Parallel Opportunities

- Setup: T003, T004, T005 in parallel.
- Foundational: T006/T008/T010/T012/T017 (different files) in parallel; T011→T013→T014 sequential.
- US1 tests T020–T025 and T074 in parallel before implementation.
- After Foundational, with staffing, US1 and the non-dependent parts of US3 can proceed in parallel.

---

## Parallel Example: User Story 1

```bash
# Tests first, in parallel:
Task: "Unit-test schedule parsing/summary in internal/schedule/parse_test.go"      # T020
Task: "Unit-test RRULE next-run (ordinal weekday) in internal/schedule/recur_test.go"  # T021
Task: "Unit-test one-off + past rejection in internal/schedule/oneoff_test.go"     # T022
Task: "Unit-test DST resolution in internal/timezone/dst_test.go"                  # T023
Task: "Unit-test overlap queue-one in internal/engine/overlap_test.go"            # T024
Task: "Contract-test task endpoints in internal/api/server/tasks_test.go"          # T025

# Then parallel models/resolvers:
Task: "Implement timezone/DST resolver in internal/timezone/timezone.go"           # T027
```

---

## Implementation Strategy

### MVP First (User Story 1 only)

1. Phase 1 Setup → 2. Phase 2 Foundational (blocks everything) → 3. Phase 3 US1 →
4. **STOP & VALIDATE** US1 independently (recurring + one-off, windowless run, restart resume) →
5. Demo MVP.

### Incremental Delivery

Foundation → US1 (MVP, CLI) → US2 (GUI + alerts) → US3 (groups) → US4 (triggers) → US5 (catch-up).
Each story is tested independently and adds value without breaking prior stories.

---

## Notes

- [P] = different files, no incomplete dependencies. [US#] maps tasks to stories for traceability.
- Tests are required (constitution v1.0.0): `go test -race`, ≥80% core coverage, dispatch benchmark.
- The windowless-GUI / no-console-spawn requirement is exercised by T032, T044, T048, and T072.
- Remediation tasks from `/speckit-analyze`: T074 (cron-parity suite, SC-002), T075/T076 (runs &
  alerts query API + CLI), T077/T078 (`run_as` impersonation + test); T024/T031 widened to cover
  all three overlap policies.
- Commit after each task or logical group; stop at any checkpoint to validate a story.
