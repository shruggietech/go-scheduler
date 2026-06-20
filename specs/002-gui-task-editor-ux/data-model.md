# Phase 1 Data Model: GUI Task Editor UX Overhaul

This feature is UI-centric; the persistent data model is **unchanged**. The entities below
describe in-memory editor state and the mappings the dialog applies. No SQLite migration.

## 1. Editor view state (in-dialog, transient)

Conceptual fields the dialog manages while open (already partly present as `taskForm` in
[gui/editor.go](../../gui/editor.go)):

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| name | text | yes | non-empty |
| command | text | yes | single executable, non-empty |
| args | text (multiline) | no | one argument per line; blank/whitespace lines dropped |
| timezone | text (combo) | no → `Local` | `Local` or valid IANA zone |
| mode | enum {Recurring, One-off} | yes | drives which time field is active |
| schedule | text | when Recurring | human phrase; may include `starting at`/`from` anchor |
| anchorTime | time-of-day (optional) | no | only for sub-daily interval schedules; GUI helper |
| oneOffDate / oneOffTime | date + time | when One-off | assembled into a future RFC 3339 instant |
| overlapLabel | enum (display label) | no → default | mapped to wire value on submit |
| catchupLabel | enum (display label) | no → default | mapped to wire value on submit |

**Validity rules (drive Save gating, FR-003/004/005/006):**
- `name` non-empty AND `command` non-empty.
- Recurring: `schedule` non-empty and parseable (preview did not error).
- One-off: assembled timestamp parses as RFC 3339 and is strictly in the future.
- `timezone` resolves via `timezone.Resolve` (`Local` or known IANA).
- Save is enabled only when all currently-relevant rules hold.

**State preservation (FR-002):** toggling `mode` hides/shows fields but never clears
`schedule`/`oneOff*` values.

## 2. Persistent Schedule (unchanged shape, anchor now user-settable)

`domain.Schedule` ([internal/domain/domain.go](../../internal/domain/domain.go)) — no field added:

| Field | Meaning in this feature |
|-------|-------------------------|
| Kind | `recurring` or `one_off` (as today) |
| RRULE | built by the parser (unchanged grammar except anchor clause) |
| **Anchor** | for sub-daily intervals, now set from the user's `starting at`/`from` time instead of always `now` |
| RunAt | one-off instant (unchanged) |
| HumanSummary | extended to mention the anchor, e.g. "Every 15 minutes starting at 09:00" |

State transition for anchor (parser):
- phrase has no anchor clause → `Anchor = now` (today's behavior, FR-012).
- phrase has `starting at <t>` / `from <t>` AND schedule is sub-daily interval →
  `Anchor = <wall time t in task tz, on the reference day>` (FR-010/FR-011/FR-013).
- anchor clause on a non-sub-daily schedule → validation error (FR-013, clause not applicable).

## 3. Policy label ⇄ wire-value mapping (presentation-only)

Bidirectional maps used by the dialog; **wire/store values are unchanged** (FR-019/FR-022).

**Overlap** (`domain.OverlapPolicy`):

| Display label | Wire value |
|---------------|-----------|
| Queue one run (default) | `queue_one` |
| Skip this run | `skip` |
| Allow concurrent runs | `allow_concurrent` |

**Catch-up** (`domain.CatchupPolicy`):

| Display label | Wire value |
|---------------|-----------|
| Run once to catch up (default) | `one` |
| Skip missed runs | `none` |

Rules:
- On **submit**: label → wire value (default applied if somehow unset).
- On **edit open**: wire value → label (any unknown/legacy value falls back to the default label
  and is logged, never crashes).

## 4. Curated timezone suggestion list (FR-014)

A static, ordered slice seeding the timezone `SelectEntry`. Representative, not exhaustive:
`Local`, `UTC`, `America/New_York`, `America/Chicago`, `America/Denver`, `America/Los_Angeles`,
`America/Sao_Paulo`, `Europe/London`, `Europe/Paris`, `Europe/Berlin`, `Europe/Moscow`,
`Asia/Kolkata`, `Asia/Shanghai`, `Asia/Tokyo`, `Australia/Sydney`, `Pacific/Auckland`.
Typed values outside the list remain valid if `timezone.Resolve` accepts them.

## 5. Command-line preview model (FR-008)

Derived, not stored. Given `command` and the split `args`, render a single display string:
`command` followed by each argument; tokens containing whitespace are shown quoted. Display-only —
execution still receives the raw `[]string` (no shell, no re-parsing).
