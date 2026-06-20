# Phase 1 Data Model: Cross-Platform Task Scheduler

Derived from the spec's Key Entities and Functional Requirements. All timestamps are stored in
**UTC**; timezone is applied only when computing/displaying local run times. Entities map to
SQLite tables in `internal/store`.

## Entity: Group

A named container forming a nested hierarchy (FR-019, FR-020).

| Field | Type | Notes |
|-------|------|-------|
| id | string (UUID) | Primary key |
| name | string | Required, non-empty |
| parent_id | string \| null | Self-reference; null = top-level. Cycles forbidden |
| enabled | bool | Default true; disabling cascades to descendants & tasks |
| created_at | timestamp (UTC) | |
| updated_at | timestamp (UTC) | |

- **Relationships**: parent → children (tree, arbitrary depth, ≥3 levels per SC-009); 1→N Tasks.
- **Rules**: setting `enabled=false` makes every contained task ineligible to run until the group
  (and ancestors) are re-enabled. Deleting a group requires it be empty or cascade explicitly.

## Entity: Task

A unit of work to execute (FR-001, FR-007, FR-011, FR-012, FR-017).

| Field | Type | Notes |
|-------|------|-------|
| id | string (UUID) | Primary key |
| name | string | Required |
| group_id | string \| null | FK → Group; null = ungrouped |
| command | string | Program/script to run |
| args | string[] (JSON) | Arguments |
| working_dir | string \| null | Defaults to a documented base dir |
| env | map<string,string> (JSON) | Extra environment variables |
| run_as | string \| null | Optional account; defaults to service account |
| enabled | bool | Default true |
| timezone | string (IANA) | Default = system local; e.g. `America/New_York` (FR-017) |
| schedule_id | string | FK → Schedule (one per task) |
| overlap_policy | enum | `queue_one` (default) \| `skip` \| `allow_concurrent` (FR-012) |
| catchup_policy | enum | `one` (default) \| `none` (FR-010, FR-011) |
| state | enum | `active` \| `completed` \| `disabled` (one-off → `completed` after run) |
| created_at / updated_at | timestamp (UTC) | |

- **Relationships**: N→1 Group; 1→1 Schedule; 1→N Run; may be the source/target of a Trigger.
- **State transitions**:
  - `active → completed`: a one-off task finishes its single run (FR-004a), or its recurrence
    `UNTIL`/`COUNT` is exhausted. Stays in history; re-schedulable back to `active`.
  - `active ↔ disabled`: user disables/enables (or an ancestor group does, effectively).
- **Validation**: `command` non-empty; `timezone` resolvable; `schedule` valid; a one-off whose
  time is already past at creation is rejected (Edge Cases) — except when discovered missed after
  downtime, where catch-up applies.

## Entity: Schedule

The timing definition for a task (FR-002, FR-003, FR-004, FR-004a, FR-007, FR-016).

| Field | Type | Notes |
|-------|------|-------|
| id | string (UUID) | Primary key |
| kind | enum | `one_off` \| `recurring` \| `event` |
| rrule | string \| null | RFC 5545 RRULE for `recurring` (rrule-go); null otherwise |
| run_at | timestamp (UTC) \| null | For `one_off`: the single fire time |
| trigger_id | string \| null | For `event`: FK → Trigger |
| human_summary | string | Cached plain-language description (FR-006) |
| anchor | timestamp (UTC) \| null | DTSTART for recurrence math |

- **Rules**: exactly one of (`rrule`+`anchor`), `run_at`, or `trigger_id` is set per `kind`.
  Next-run computation runs in the task's timezone, then normalizes DST (next-valid for
  spring-forward, first-occurrence for fall-back) and converts to UTC (FR-016, FR-018).

## Entity: Trigger (Event)

A task-completion event that fires another task, with deduplication (FR-007, FR-014). v1 source
is **another task's completion** only.

| Field | Type | Notes |
|-------|------|-------|
| id | string (UUID) | Primary key |
| source_task_id | string | FK → Task whose completion fires the event |
| on_outcome | enum | `success` \| `failure` \| `any` |
| target_task_id | string | FK → Task to run |
| dedup_key_template | string \| null | How to derive a dedup key for an event instance |
| dedup_window | duration | Window within which duplicates collapse to one run |

- **Relationships**: source Task → Trigger → target Task.
- **Backed by**: a **DedupLedger** record per delivered event (key + first-seen UTC) so
  at-least-once delivery yields one execution per logical event within the window.

## Entity: DedupLedger (supporting)

| Field | Type | Notes |
|-------|------|-------|
| trigger_id | string | FK → Trigger |
| dedup_key | string | Logical event identity |
| first_seen_at | timestamp (UTC) | Window start |
| executed | bool | Whether the target run was dispatched |

- **Rules**: unique on (trigger_id, dedup_key) within window; second delivery inside the window
  is a no-op for execution.

## Entity: Run (Execution Record)

A single attempted execution (FR-015, FR-013).

| Field | Type | Notes |
|-------|------|-------|
| id | string (UUID) | Primary key |
| task_id | string | FK → Task |
| scheduled_for | timestamp (UTC) | Intended fire time |
| started_at | timestamp (UTC) \| null | |
| ended_at | timestamp (UTC) \| null | |
| outcome | enum | `success` \| `failure` \| `skipped` \| `caught_up` \| `queued` |
| exit_code | int \| null | |
| output | text | Captured stdout/stderr (bounded/truncated) |
| trigger | enum | `schedule` \| `event` \| `catchup` \| `manual` |

- **Relationships**: N→1 Task.
- **Rules**: append-only history; powers calendar/timeline views (FR-023) and alerts.

## Entity: Alert / Notification

A surfaced condition shown in the GUI and reflected in logs (FR-013, FR-024).

| Field | Type | Notes |
|-------|------|-------|
| id | string (UUID) | Primary key |
| task_id | string \| null | Related task, if any |
| severity | enum | `info` \| `warning` \| `error` |
| kind | enum | `overlap_queued` \| `run_failed` \| `missed_run` \| `service` |
| message | string | Human-readable |
| created_at | timestamp (UTC) | |
| acknowledged | bool | User dismissal state |

- **Rules**: an `overlap_queued` warning is created (and logged) whenever the queue-one policy
  queues a pending run (FR-013); failures create `run_failed` alerts (FR-024).

## Relationships (summary)

```text
Group 1───N Group        (nested tree)
Group 1───N Task
Task  1───1 Schedule
Task  1───N Run
Task  1───N Trigger(source) ───N Task(target)
Trigger 1───N DedupLedger
Task  1───N Alert
```

## Config (not persisted as an entity — single schema, validated at startup)

Documented config file (`internal/config`) with fail-fast validation (constitution UX
principle): data directory, IPC socket/pipe path, admin group, default timezone resolution,
log level/format, output-capture size limit, worker-pool size.
