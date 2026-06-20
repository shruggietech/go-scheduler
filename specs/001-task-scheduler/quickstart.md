# Quickstart & Validation Guide: Cross-Platform Task Scheduler

This guide proves the feature works end-to-end. It references [data-model.md](data-model.md) and
[contracts/](contracts/) rather than restating them. Implementation lives in `tasks.md` (Phase 2).

## Prerequisites

- Go 1.23+ installed; on the **GUI** build host, a C toolchain + OpenGL dev headers for Fyne
  (daemon and CLI are cgo-free).
- Admin/root privileges to install the system service.

## Build

```bash
# Daemon and CLI (cgo-free)
go build -o bin/goschedd ./cmd/goschedd
go build -o bin/gosched  ./cmd/gosched

# GUI — windowless on Windows (no console)
#   Windows: go build -ldflags "-H windowsgui" -o bin/gosched-gui.exe ./cmd/gosched-gui
#   Linux/macOS:
go build -o bin/gosched-gui ./cmd/gosched-gui
```

## Install & start the service (start-on-boot, FR-009)

```bash
sudo ./bin/gosched service install   # registers systemd/launchd/Windows Service (admin required)
sudo ./bin/gosched service start
./bin/gosched service status         # expect: running
```

## Validation scenarios (map to spec Success Criteria)

### SC-001 / US1 — human-readable recurring schedule (no cron syntax)
```bash
./bin/gosched task add nightly-report \
  --command /usr/bin/make-report --schedule "3rd wednesday monthly at 14:00" --tz America/New_York
# Expect: CLI echoes summary "Runs the 3rd Wednesday of every month at 2:00 PM (America/New_York)".
./bin/gosched task show <id>   # next_runs lists the correct upcoming dates
```
**Pass**: created in under 2 minutes, no cron string used, summary matches intent.

### SC-010 / US1 — one-off task
```bash
./bin/gosched task add birthday-msg --command /usr/bin/send-card --at 2026-08-04T09:00:00Z
# Past --at is rejected with exit 2.
```
**Pass**: runs exactly once at the time; afterward `task show` reports state `completed`; no further runs.

### SC-007 — resumes after reboot
Reboot the machine; then `./bin/gosched service status` → running and enabled tasks have upcoming
`next_runs`. **Pass**: no manual steps required.

### SC-003 — one catch-up after downtime
Stop the service across ≥1 scheduled time of a task, restart, then inspect history:
```bash
./bin/gosched service stop ; sleep <past one or more runs> ; ./bin/gosched service start
./bin/gosched runs --task <id>
```
**Pass**: exactly one `caught_up` run, then normal `schedule` runs; with `--catchup none`, zero catch-up runs.

### SC-004 — DST correctness
Using the injected-clock test harness (`test/integration`), drive a task across spring-forward and
fall-back in a DST zone. **Pass**: skipped-hour task runs at next valid instant; fall-back task runs
once at the first occurrence; zero double/missed runs.

### SC-005 — overlap alert
Create a task whose command sleeps past its next trigger:
```bash
./bin/gosched task add slow --command /bin/long-job --schedule "every 30s" --overlap queue_one
./bin/gosched alerts --unacked     # expect an overlap_queued warning within seconds
```
**Pass**: exactly one pending run queued; warning logged; alert visible (also in GUI).

### SC-006 — event trigger with dedup
```bash
./bin/gosched trigger add --source <taskA> --target <taskB> --on success --dedup-window 5m
./bin/gosched task run-now <taskA>           # taskB runs once
# duplicate completion within window → taskB does NOT run again
```
**Pass**: one logical completion → exactly one taskB execution.

### SC-009 — nested groups
```bash
./bin/gosched group add Backups
./bin/gosched group add Database --parent <BackupsId>
./bin/gosched task edit <id> --group <DatabaseId>
./bin/gosched group disable <BackupsId>      # cascades: tasks stop running
```
**Pass**: ≥3 levels nest; disabling a parent stops contained tasks.

### SC-008 — GUI calendar, no visible console
```bash
./bin/gosched gui      # launches the desktop app
```
**Pass**: a Material Design window opens with calendar/timeline + guided task editor; **no CMD/
console window appears or remains open**; a task created in the GUI appears in `gosched task list`
(shared state, FR-026), and overlap/failure alerts show in the GUI.

## Constitution validation gates (run before merge)
```bash
gofmt -l . ; go vet ./... ; golangci-lint run
go test -race ./...
go test -bench=Dispatch ./internal/engine   # p99 dispatch latency < 100ms (Performance principle)
```
**Pass**: formatting/vet/lint clean; race-free; coverage ≥80% on core packages; benchmark within budget.
