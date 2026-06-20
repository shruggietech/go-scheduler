# Phase 0 Research: Cross-Platform Task Scheduler

All Technical Context unknowns are resolved below. Each decision records what was chosen, why,
and the alternatives rejected.

## 1. Process architecture: daemon + thin clients

- **Decision**: A single long-lived **daemon** (`goschedd`) hosts the scheduling engine,
  persistence, and executor. The **CLI** (`gosched`) and **GUI** (`gosched-gui`) are thin
  clients that connect to the daemon over a local IPC API. No client embeds the engine.
- **Rationale**: The spec requires the scheduler to run regardless of login state (FR-009) and
  requires CLI and GUI to share identical state (FR-026). A daemon-owns-engine model makes both
  fall out naturally: clients are stateless views/controllers, and the engine's lifecycle is
  independent of any UI. It also directly enables the user's "GUI connects to it" requirement.
- **Alternatives rejected**:
  - *Engine embedded in each client* — would mean two processes racing on the same database and
    duplicate scheduling; violates single-source-of-truth.
  - *GUI embeds engine, CLI is a separate tool* — breaks FR-026 and the "starts on boot" model.

## 2. Local IPC transport

- **Decision**: HTTP/JSON served by `net/http` over a **Unix domain socket** (Linux/macOS) and a
  **named pipe** (Windows, via `github.com/Microsoft/go-winio`). A shared `internal/api/client`
  package wraps it; CLI and GUI both use it.
- **Rationale**: `net/http` serves over any `net.Listener`, so the same handler code works on
  both transports. Local sockets/pipes avoid exposing a TCP port and let the OS enforce access
  via filesystem/pipe permissions. JSON keeps the contract debuggable (constitution: text I/O
  aids debuggability) and `--json` CLI output trivial.
- **Access control**: the socket/pipe is restricted by OS permissions. Because the daemon runs
  system-wide (often as root/SYSTEM) while the GUI runs as the logged-in user, the daemon sets
  socket ownership/ACL to a `goschedd` admin group; management requires membership (or an
  elevated client). Captured as a v1 security decision, not a blocker.
- **Alternatives rejected**:
  - *gRPC* — heavier toolchain (protoc), less human-debuggable; not warranted for a local,
    single-host API.
  - *Localhost TCP* — opens a network port other processes/users could reach; weaker isolation.

## 3. Recurrence engine (cron parity without cron syntax)

- **Decision**: Represent schedules internally with **RFC 5545 (iCalendar RRULE)** via
  `github.com/teambition/rrule-go`. A human-readable layer in `internal/schedule` parses guided
  inputs into RRULE and renders a plain-language summary back (FR-006). One-off tasks are
  `COUNT=1` rules (or a dedicated single-fire schedule); intervals map to `FREQ=…;INTERVAL=X`;
  "3rd Wednesday monthly" maps to `FREQ=MONTHLY;BYDAY=+3WE`.
- **Rationale**: RRULE is a mature standard that already expresses everything cron can plus
  ordinal weekdays, counts, and until-dates — covering FR-002/FR-003/FR-004a and SC-002
  (cron parity) without inventing a recurrence grammar. Keeping cron-style syntax out of the
  user surface is the product's core differentiator; RRULE stays purely internal.
- **DST handling**: next-run computation is done in the task's timezone (`time.LoadLocation`),
  then converted to UTC for storage and dispatch. The skipped/repeated-hour rules
  (next-valid / first-occurrence) are applied in `internal/timezone` as a normalization step
  over the rrule output. `time/tzdata` is imported so the zoneinfo DB ships in the binary
  (consistent behavior across OSes without relying on a system tzdata).
- **Alternatives rejected**:
  - *`robfig/cron`* — exposes cron semantics, weaker ordinal-weekday and DST handling.
  - *Hand-rolled recurrence math* — high risk for DST/ordinal correctness; reinvents a standard.

## 4. Persistence

- **Decision**: Embedded **SQLite** via `modernc.org/sqlite` (pure Go, cgo-free), one database
  file under the daemon's per-OS data directory. Stores tasks, groups (self-referencing tree),
  schedules, triggers, a dedup ledger, and run history. All timestamps stored in UTC.
- **Rationale**: The calendar/timeline views (FR-023) and run-history queries (FR-015) want
  relational queries and indexes; SQLite provides durability and transactional updates for
  catch-up/dedup bookkeeping. The pure-Go driver preserves cgo-free cross-compilation for the
  daemon and CLI.
- **Alternatives rejected**:
  - *bbolt / embedded KV* — no SQL/range queries; calendar and history filtering become
    hand-built indexes.
  - *cgo `mattn/go-sqlite3`* — best-known driver but reintroduces cgo, complicating
    cross-compilation and CI.
  - *JSON/flat files* — no transactional integrity for concurrent run/dedup updates.

## 5. Go-native GUI with Material Design

- **Decision**: **Fyne** (`fyne.io/fyne/v2`) for the desktop GUI. Views: calendar, schedule/
  timeline, guided task editor with live plain-language schedule preview, group tree, and an
  alerts/notifications surface.
- **Rationale**: Fyne is genuinely Go-native (not a web view), cross-platform on all three target
  OSes, and its design language is Material Design — a direct match for the clarified decision
  (Go-native desktop app) and FR-025. It renders its own widgets, so no browser/CMD window is
  involved.
- **Alternatives rejected**:
  - *Wails / webview* — renders a web frontend; the user explicitly rejected a browser-based UI.
  - *Gio* — capable but lower-level immediate-mode; more effort to reach a polished Material UI.

## 6. No visible console window (explicit requirement)

- **Decision**:
  - The **GUI** binary is built windowless: `go build -ldflags "-H windowsgui"` on Windows (no
    attached console). On macOS it is bundled as a `.app`; on Linux it is a normal GUI binary.
  - The **executor** spawns task processes with no console window: on Windows set
    `syscall.SysProcAttr{HideWindow: true, CreationFlags: CREATE_NO_WINDOW}`; this logic lives in
    `internal/platform` behind build tags so non-Windows builds are unaffected.
  - The **daemon** runs as a service with no attached console by construction.
- **Rationale**: Directly satisfies "when the GUI is open, the CMD is not visibly open" and
  prevents per-task console flashing on Windows — a common, messy failure mode for schedulers
  that shell out.
- **Alternatives rejected**:
  - *Console-attached GUI* — leaves a visible terminal; rejected by requirement.
  - *Hiding the window post-launch* (e.g., ShowWindow SW_HIDE) — flickers a console first; the
    link-time `-H windowsgui` approach never creates one.

## 7. System service / start-on-boot

- **Decision**: `github.com/kardianos/service` to install, start, and run `goschedd` as a
  **system-wide** service: systemd system unit (Linux), launchd daemon (macOS), Windows Service
  (Windows). `gosched service install/uninstall/start/stop/status` wraps it.
- **Rationale**: One abstraction over three very different init systems; system-wide scope means
  tasks run with no user logged in (clarified decision, FR-009). Install requires admin
  privileges, as captured in the spec assumptions.
- **Alternatives rejected**:
  - *Per-OS hand-written unit/plist/service code* — large surface, easy to get boot ordering and
    restart semantics wrong.
  - *Per-user autostart (login agent)* — explicitly rejected during clarification.

## 8. CLI framework & UX

- **Decision**: `github.com/spf13/cobra` for the verb-noun command tree (`gosched task add`,
  `gosched task list`, `gosched group …`, `gosched run-now`, `gosched service …`), with a global
  `--json` flag, results to stdout, diagnostics/errors to stderr, and conventional exit codes.
- **Rationale**: Cobra is the de-facto standard for Go CLIs and gives consistent help,
  subcommands, and flag handling — meeting FR-021 and the constitution's UX-consistency principle
  with less bespoke code than stdlib `flag`.
- **Alternatives rejected**: stdlib `flag` — workable but more boilerplate for a multi-noun tree
  and inconsistent help output.

## 9. Time, clock injection, and testability

- **Decision**: Define a `Clock` interface (`Now() time.Time`, `After(d) <-chan time.Time`,
  `NewTimer`) in `internal/engine`. Production uses a real clock; tests inject a fake clock to
  drive deterministic scheduling, catch-up, and DST scenarios without real sleeps.
- **Rationale**: The constitution makes this non-negotiable (Testing principle): timing tests
  must not depend on wall-clock sleeps. It also makes DST and downtime/catch-up scenarios
  deterministically testable (SC-003, SC-004).
- **Alternatives rejected**: direct `time.Now()`/`time.Sleep` in engine code — untestable,
  flaky, and forbidden by the constitution.

## 10. Overlap, catch-up, and at-least-once/dedup semantics

- **Decision**:
  - **Overlap**: default "queue one pending" — if a task is still running at its next trigger,
    enqueue exactly one pending run, log a warning (`slog`), and raise a GUI alert; drop further
    triggers while one is queued. Policy is per-task configurable (FR-012/FR-013).
  - **Catch-up**: on startup the engine compares each task's last-run vs the schedule; if ≥1 run
    was missed and catch-up is enabled, run exactly once, then resume (FR-010/FR-011).
  - **Triggers**: task-completion events are written to a durable dedup ledger keyed by
    (trigger, dedup-key) within the dedup window; the triggered task fires at-least-once but
    effectively once per logical event (FR-014).
- **Rationale**: All three are reliability behaviors that must survive restarts, so they are
  backed by transactional store state rather than in-memory only. Determinism is verified with
  the injected clock.
- **Alternatives rejected**: in-memory-only tracking — loses overlap/catch-up/dedup state across
  the very restarts those features exist to handle.

## Residual unknowns

- None blocking. The local-API access-control model (admin group vs elevation) is a reasonable
  v1 default and is documented in the contracts; it can be revisited if multi-user management is
  added later.
