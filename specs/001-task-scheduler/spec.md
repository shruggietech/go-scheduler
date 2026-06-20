# Feature Specification: Cross-Platform Task Scheduler

**Feature Branch**: `001-task-scheduler`

**Created**: 2026-06-19

**Status**: Draft

**Input**: User description: "A cross-platform (Linux/Mac/Windows) task scheduler, CLI-first with a user-friendly GUI on top. Cron-equivalent power without the cryptic syntax. Starts on boot. Calendar/schedule views. Flexible recurrence (every X seconds/minutes/days/weeks, Nth weekday of month, event-triggered). Nested task groups. Missed-run catch-up (one per task). Per-task timezones with DST handling, UTC backend. Event-driven at-least-once delivery with dedup. Queue-one-pending overlap policy with alerting. Material-design GUI following UI/UX best practices."

## Clarifications

### Session 2026-06-19

- Q: How should the scheduler run on each machine — does it need to execute tasks when no user is logged in? → A: System-wide service (runs at boot regardless of login state; closest to cron parity; requires install-time admin privileges).
- Q: How should the Material Design GUI be delivered? → A: Go-native desktop application (packaged native window, no browser).
- Q: Which event-trigger sources should v1 support? → A: Another task's completion only (external CLI/API triggers and file/folder watch are out of scope for v1).
- Q: How should tasks behave across DST transitions for skipped (spring-forward) or repeated (fall-back) times? → A: Next-valid / first-occurrence — skipped-hour tasks run at the next valid instant; repeated-hour tasks run once, on the first occurrence.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Schedule a task without cron syntax (Priority: P1)

A user wants a command or job to run on a schedule — either recurring or just once. Using
plain-language options (e.g., "every 15 minutes", "every weekday at 9:00 AM", "the 3rd
Wednesday of each month", or a single one-off date/time like "Aug 4 at 9:00 AM"), they create
a task, and the scheduler runs it reliably at those times. The task persists across reboots
and resumes automatically when the system starts.

**Why this priority**: This is the core promise of the product — cron-level scheduling power
expressed in human-readable terms. Without it, nothing else matters. It is the minimum
viable product on its own.

**Independent Test**: Create a task via the CLI with a human-readable recurrence, verify it
executes at the expected times, reboot the machine, and confirm the task resumes on the next
scheduled occurrence without manual intervention.

**Acceptance Scenarios**:

1. **Given** no existing tasks, **When** the user creates a task that runs "every 15
   minutes", **Then** the task executes at each 15-minute boundary and its run history is
   recorded.
2. **Given** a task scheduled for "every weekday at 9:00 AM", **When** a weekend day passes,
   **Then** the task does not run, and it runs again on the next weekday at 9:00 AM.
3. **Given** a task scheduled for "the 3rd Wednesday of each month", **When** a month has its
   3rd Wednesday, **Then** the task runs exactly once on that day at the configured time.
4. **Given** a one-off task scheduled for a single specific date and time (e.g., a birthday
   message), **When** that moment arrives, **Then** the task runs exactly once and does not
   run again or recur.
5. **Given** an enabled task, **When** the system reboots, **Then** the scheduler restarts
   automatically on boot and the task continues on its normal schedule.

---

### User Story 2 - Manage tasks visually with calendar and schedule views (Priority: P2)

A user opens the GUI and sees their scheduled tasks laid out in a calendar/schedule view.
They can see what is scheduled and when, create and edit tasks through guided forms (not raw
syntax), and understand at a glance the upcoming run timeline. The GUI surfaces alerts (e.g.,
overlapping runs, failures) prominently.

**Why this priority**: The GUI is what makes the product approachable for non-experts and
delivers the "user-friendly" differentiator. It builds directly on the P1 engine but is not
required for the engine to deliver value, so it is P2.

**Independent Test**: With the scheduling engine running, open the GUI, view existing tasks
on a calendar, create a new task through the form-based editor, and confirm it appears on the
calendar and subsequently executes — all without typing schedule expressions.

**Acceptance Scenarios**:

1. **Given** several scheduled tasks, **When** the user opens the calendar view, **Then**
   upcoming runs are displayed on their scheduled dates/times in a readable layout.
2. **Given** the task editor, **When** the user builds a recurrence using guided controls,
   **Then** a plain-language summary of the schedule is shown before saving (e.g., "Runs the
   3rd Wednesday of every month at 2:00 PM").
3. **Given** a task whose previous run is still executing at its next trigger, **When** the
   overlap occurs, **Then** the GUI displays a visible alert for that task.
4. **Given** the GUI, **When** any task fails or is skipped, **Then** the condition is shown
   as an alert/notification with enough detail to act on.

---

### User Story 3 - Organize tasks into nested groups (Priority: P2)

A user with many tasks organizes them into groups, and groups within groups, mirroring how
they think about their work (e.g., "Backups" → "Database" → individual jobs). They can view,
enable/disable, and manage tasks at the group level.

**Why this priority**: Grouping is essential for managing scale and is a stated requirement,
but the scheduler delivers value before grouping exists, so it ranks alongside the GUI rather
than ahead of the core engine.

**Independent Test**: Create a nested group hierarchy, assign tasks to groups, and verify
tasks can be browsed, filtered, and enabled/disabled by group in both CLI and GUI.

**Acceptance Scenarios**:

1. **Given** a set of tasks, **When** the user creates a group and a sub-group and assigns
   tasks, **Then** the hierarchy is persisted and reflected in both CLI and GUI listings.
2. **Given** a group containing tasks, **When** the user disables the group, **Then** all
   tasks within it (including sub-groups) stop running until re-enabled.

---

### User Story 4 - Event-triggered tasks (Priority: P3)

A user configures a task to run in response to another task's completion rather than on a
clock schedule. The completion event is delivered at-least-once, and duplicate deliveries
within a configured dedup window/key are de-duplicated so the task does not run twice for the
same logical event.

**Why this priority**: Event-driven (task-completion) chaining extends the product beyond
time-based cron parity and is valuable, but it depends on the core engine and task model
being in place, so it follows the time-based capabilities.

**Independent Test**: Configure Task B to trigger on completion of Task A, run Task A, and
confirm Task B runs once; deliver a duplicate completion event within the dedup window and
confirm Task B does not run a second time.

**Acceptance Scenarios**:

1. **Given** Task B configured to trigger on Task A completion, **When** Task A finishes,
   **Then** Task B starts.
2. **Given** an event-triggered task with a dedup key/window, **When** the same completion
   event is delivered twice within the window, **Then** the task executes only once.
3. **Given** an event-triggered task, **When** a completion event occurs, **Then** delivery
   is guaranteed at least once even if the scheduler was briefly unavailable when the event
   occurred.

---

### User Story 5 - Recover gracefully from downtime with catch-up runs (Priority: P3)

After the machine or scheduler was off and one or more scheduled runs were missed, the
scheduler performs exactly one catch-up execution per affected task (if at least one run was
missed) and then resumes the normal schedule. Users can adjust this behavior per task via
advanced configuration.

**Why this priority**: Downtime handling is important for reliability but is a refinement of
the core execution loop; the scheduler must exist and run tasks before catch-up behavior is
meaningful.

**Independent Test**: Stop the scheduler across one or more scheduled times for a task,
restart it, and confirm exactly one catch-up run occurs followed by normal scheduling; then
change the per-task catch-up setting and confirm the alternate behavior.

**Acceptance Scenarios**:

1. **Given** a task that missed three runs during downtime, **When** the scheduler restarts,
   **Then** the task runs exactly once as catch-up and then continues on its normal schedule.
2. **Given** a task that missed zero runs, **When** the scheduler restarts, **Then** no
   catch-up run occurs.
3. **Given** a task with catch-up disabled in advanced settings, **When** runs were missed
   during downtime, **Then** no catch-up run occurs and only future scheduled runs execute.

---

### Edge Cases

- **Daylight Saving Time transitions**: When a task's local time falls in a skipped hour
  (spring-forward), the task runs at the next valid instant (e.g., a 2:30 AM task runs at
  3:00 AM). When it falls in a repeated hour (fall-back), the task runs exactly once, on the
  first occurrence of that wall-clock time. Each intended occurrence runs exactly once.
- **Per-task timezone differs from system**: A task pinned to a specific timezone runs at the
  correct local moment in that zone regardless of the host's timezone.
- **Overlapping runs (still running at next trigger)**: With the default policy, one pending
  run is queued, a warning is logged, and an alert appears in the GUI; additional triggers
  while one is already queued are dropped (or handled per the configured advanced policy).
- **Catch-up vs. overlap interaction**: A catch-up run that would itself overlap a running
  instance respects the overlap policy.
- **Invalid or impossible schedules** (e.g., "5th Wednesday" in a month that has only four):
  the occurrence is skipped for that month, with clear behavior shown in the schedule preview.
- **One-off scheduled for a past time**: creating a one-off task whose date/time is already in
  the past MUST be rejected with a clear message (or, if the time passed while the scheduler was
  down, handled by the task's catch-up policy like any missed run).
- **One-off after it has run**: once a one-off task has executed, it is marked complete and is
  not re-armed; it remains visible in history and can be re-enabled/re-scheduled by the user.
- **System clock changes** (manual adjustment, NTP correction): scheduling is anchored to UTC
  to avoid double-runs or skips from wall-clock jumps.
- **Task command fails / does not exist**: the run is recorded as failed with captured output
  and surfaced as an alert; scheduling continues.
- **Group disabled while a task within it is running**: the in-flight run completes; no new
  runs start until re-enabled.
- **Event trigger storms**: rapid duplicate events collapse to a single execution within the
  dedup window.

## Requirements *(mandatory)*

### Functional Requirements

**Scheduling engine & recurrence**

- **FR-001**: System MUST allow users to define recurring schedules using human-readable
  options without requiring cron expressions or asterisk/number syntax.
- **FR-002**: System MUST support fixed-interval recurrence in seconds, minutes, hours, days,
  and weeks (e.g., "every X seconds/minutes/days/weeks").
- **FR-003**: System MUST support calendar-relative recurrence including specific days of the
  week, specific days of the month, and ordinal weekday patterns (e.g., "3rd Wednesday of each
  month", "last Friday of the month").
- **FR-004**: System MUST support time-of-day specification for recurrences (e.g., run at
  9:00 AM).
- **FR-004a**: System MUST support one-off (non-recurring) schedules that run a task exactly
  once at a single specified date and time and then do not recur. After the one-off run
  completes, the task MUST NOT be scheduled again (it is marked complete/inactive rather than
  re-armed).
- **FR-005**: System MUST be capable of expressing any schedule that standard cron can express,
  so no time-based scheduling capability is lost relative to cron.
- **FR-006**: System MUST display a plain-language summary of any configured schedule so users
  can confirm intent before saving.
- **FR-007**: System MUST support event-triggered tasks that run in response to another task's
  completion, in addition to time-based schedules. (External CLI/API triggers and file/folder
  watching are out of scope for v1 — see Assumptions.)

**Reliability, downtime & overlap**

- **FR-008**: System MUST persist task definitions and run history so they survive restarts and
  reboots.
- **FR-009**: System MUST run as a system-wide background service that starts automatically
  when the operating system boots, on all supported platforms, executing tasks regardless of
  whether a user is logged in. Registering the service requires install-time administrative
  privileges.
- **FR-010**: System MUST, after downtime in which at least one run was missed for a task,
  perform exactly one catch-up execution for that task and then resume the normal schedule.
- **FR-011**: System MUST allow per-task advanced configuration of catch-up behavior (including
  disabling catch-up).
- **FR-012**: System MUST, by default, queue at most one pending run when a task's prior run is
  still executing at its next trigger time ("queue one pending"), and allow this overlap policy
  to be configured per task as an advanced option.
- **FR-013**: System MUST log a warning when an overlap/queued-run condition occurs and surface
  it as an alert in the GUI.
- **FR-014**: System MUST guarantee at-least-once delivery of task-completion events, with a
  configurable deduplication key and/or window so a single logical completion event causes a
  single execution of the triggered task.
- **FR-015**: System MUST record each run's outcome (success, failure, skipped, caught-up) with
  enough detail (timing, captured output) to diagnose problems.

**Time & timezones**

- **FR-016**: System MUST store and compute scheduling internally in UTC ("Zulu time").
- **FR-017**: System MUST support a per-task timezone, defaulting to the local system timezone
  when none is specified.
- **FR-018**: System MUST correctly account for Daylight Saving Time transitions so each
  intended occurrence runs exactly once at the correct local moment, using the next-valid-instant
  rule for skipped (spring-forward) times and the first-occurrence rule for repeated (fall-back)
  times.

**Organization & grouping**

- **FR-019**: Users MUST be able to organize tasks into groups, and groups MUST support nesting
  (sub-groups to arbitrary depth).
- **FR-020**: Users MUST be able to enable or disable tasks individually and at the group level,
  where disabling a group disables all contained tasks and sub-groups.

**Interfaces (CLI & GUI)**

- **FR-021**: System MUST provide a CLI as the primary interface for creating, editing,
  listing, enabling/disabling, running-now, and deleting tasks and groups.
- **FR-022**: System MUST provide a GUI, delivered as a Go-native desktop application (a
  packaged native window, not a browser-based UI), built on top of the same scheduling engine
  and offering equivalent task-management capabilities through guided, form-based controls
  rather than raw schedule syntax.
- **FR-023**: GUI MUST provide calendar and/or schedule (timeline) views showing upcoming and
  past runs for easy management.
- **FR-024**: GUI MUST surface alerts/notifications for overlap conditions, failures, and other
  conditions requiring user attention.
- **FR-025**: GUI MUST follow established UI/UX best practices and present a Material Design
  visual appearance.
- **FR-026**: CLI and GUI MUST operate on the same underlying task definitions and state, so a
  change made in one is reflected in the other.

**Cross-platform**

- **FR-027**: System MUST run on Linux, macOS, and Windows with consistent scheduling behavior
  across all three.

### Key Entities *(include if feature involves data)*

- **Task**: A unit of work to be executed (e.g., a command/job), with a name, the action to
  run, an enabled/disabled state, a schedule or trigger, a timezone, overlap and catch-up
  policies, group membership, and run history.
- **Schedule**: The timing definition for a task — one-off (a single date/time), interval-based,
  calendar-relative, ordinal-weekday, or event-triggered — expressible in human-readable terms
  and convertible to concrete future run times in UTC.
- **Trigger / Event**: The condition that initiates an event-driven task — in v1, another
  task's completion — including the source task, dedup key, and dedup window.
- **Run (Execution Record)**: A single attempted execution of a task, with start/end times,
  outcome (success/failure/skipped/caught-up), and captured output/diagnostics.
- **Group**: A named container for tasks and other groups, forming a nested hierarchy, with an
  enabled/disabled state that cascades to its contents.
- **Alert / Notification**: A surfaced condition (overlap, failure, missed run) shown in the
  GUI and reflected in logs.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A user can create a correct recurring schedule (including an ordinal-weekday
  pattern such as "3rd Wednesday monthly") in under 2 minutes using guided options, without
  writing any cron-style expression.
- **SC-002**: Any schedule expressible in standard cron can be reproduced in this system, as
  demonstrated by a coverage suite mapping representative cron patterns to equivalent
  configurations with matching run times.
- **SC-003**: After an unplanned shutdown spanning one or more scheduled times, each affected
  task performs exactly one catch-up run and then resumes normal scheduling, with zero
  duplicate or missed catch-ups.
- **SC-004**: Across a DST transition (both spring-forward and fall-back), every intended
  occurrence runs exactly once at the correct local time, with zero double-runs or skipped
  runs.
- **SC-005**: When a task is still running at its next trigger time, exactly one run is queued,
  a warning is logged, and an alert is visible in the GUI within seconds of the condition.
- **SC-006**: For event-triggered tasks, a single logical event results in exactly one
  execution even when the trigger is delivered multiple times within the dedup window
  (at-least-once delivery, exactly-once effect within the window).
- **SC-007**: The scheduler resumes automatically after a reboot on all three supported
  platforms with no manual steps.
- **SC-008**: A new user can locate, schedule, group, and verify a task entirely through the
  GUI calendar/editor on their first session without consulting schedule-syntax documentation.
- **SC-009**: Tasks can be organized into at least three levels of nested groups and managed
  (enable/disable, view) at any level.
- **SC-010**: A user can schedule a one-off task for a single future date/time; it runs exactly
  once at that moment and never recurs, with zero additional runs afterward.

## Assumptions

- The scheduler runs as a system-wide background service/daemon per machine; tasks are scoped
  to the machine on which they are defined (no multi-machine/distributed coordination in this
  version).
- "Tasks" execute local commands/scripts/programs; integrating with arbitrary external job
  systems is out of scope for v1.
- Event triggers in v1 are limited to another task's completion. External trigger sources
  (CLI/API-delivered events) and file/folder watching are out of scope for v1.
- The GUI is a Go-native desktop application for the local machine's scheduler (not a hosted
  multi-tenant web service); remote/multi-user access is out of scope for v1.
- "Starts on boot" registers a system-wide service via each platform's standard mechanism
  (systemd system unit / launchd daemon / Windows Service) and assumes the installing user has
  the administrative privileges required to register it.
- Default overlap policy is "queue one pending"; default catch-up is "one catch-up run if any
  were missed"; default timezone is the local system timezone — all changeable per task.
- Authentication/authorization for the GUI is assumed to be single local user for v1 unless
  later specified.
- Notifications are surfaced in-app (GUI alerts) and in logs; external notification channels
  (email, push, webhooks) are out of scope for v1 unless later specified.
- Daylight Saving Time rules are sourced from the platform/standard timezone database.
