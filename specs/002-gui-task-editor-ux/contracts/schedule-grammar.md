# Contract: Extended Schedule Grammar (anchor clause)

Extends the human-readable schedule grammar in
[internal/schedule/parse.go](../../../internal/schedule/parse.go). Only the **interval** form
gains an optional anchor clause; all other forms are unchanged.

## New clause: `starting at <time>` / `from <time>`

```
every <N> <sub-daily-unit> [ (starting at | from) <time> ]
```

- **Valid only** when the unit is sub-daily: `second(s)/sec/s`, `minute(s)/min/m`, `hour(s)/hr/h`.
- `<time>` uses the existing time-of-day grammar: `14:00`, `9:00`, `9:00 AM`, `9am`, bare `9`.
- The clause sets the schedule **anchor** (first-cycle alignment); it does **not** add a
  `BYHOUR/BYMINUTE` constraint (that remains reserved for daily-or-coarser `at <time>`).

### Examples

| Phrase | Result |
|--------|--------|
| `every 15 minutes` | interval, anchor = now (unchanged behavior) |
| `every 15 minutes starting at 09:00` | interval anchored to :00/:15/:30/:45 phase |
| `every 30 minutes from 9am` | interval anchored to :00/:30 phase relative to 09:00 |
| `every 2 hours starting at 08:00` | interval anchored to 08:00, 10:00, 12:00 … phase |
| `every 15 minutes at 09:00` | **rejected** (bare `at` still invalid for sub-daily) |
| `every day starting at 09:00` | **rejected** (anchor clause not valid for daily) |
| `weekdays starting at 09:00` | **rejected** (use `weekdays at 09:00`) |

## Parser behavior

- `parseInterval` recognizes an optional trailing `(starting at|from) <time>` group for sub-daily
  units and returns the parsed anchor time-of-day alongside the interval.
- `finish` sets `sch.Anchor` from the parsed anchor (interpreted as the given wall time in the
  task timezone on the reference day) when present; otherwise `sch.Anchor = now` (today's default).
- `HumanSummary` includes the anchor, e.g. `Every 15 minutes starting at 09:00`.
- On a non-sub-daily schedule, an anchor clause yields a clear error:
  `schedule: 'starting at' only applies to interval schedules (seconds/minutes/hours)`.

## Engine interaction (no change)

`NextRun`/`nextRecurring` already set `opt.Dtstart = sch.Anchor` and return the first occurrence
strictly after `now`. With an aligned anchor, sub-daily occurrences fall on the anchored phase.
A past anchor is acceptable: the next run is the first anchor-aligned instant at/after now
(FR-011). No change to `internal/schedule/recur.go` is required.

## Backward compatibility

- Phrases without the new clause parse and run exactly as before (FR-012, SC-007).
- No change to `PreviewRequest`, `TaskCreateRequest`, `TaskUpdateRequest`, the CLI, or the store
  schema — the anchor is carried within the schedule phrase string.
