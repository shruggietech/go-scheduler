# Quickstart: Validating the GUI Task Editor UX Overhaul

A run/validation guide proving the feature works end-to-end. Implementation details live in
`tasks.md`; this file is for verifying behavior. See [contracts/editor-ui.md](contracts/editor-ui.md)
and [contracts/schedule-grammar.md](contracts/schedule-grammar.md) for the exact contracts.

## Prerequisites

- Go toolchain and the cgo GUI toolchain (WinLibs MinGW GCC on Windows) per `CLAUDE.md`.
- Daemon (`goschedd`) running locally, or use the in-process fake backend for GUI unit tests.

## Automated checks

```bash
# Parser: anchor grammar + alignment (deterministic, injected clock)
go test ./internal/schedule/... -race

# GUI: label mapping, mode visibility, validation gating, command-line preview
go test ./gui/... -race

# Full suite + vet/format gate
go test ./... -race
go vet ./...
gofmt -l .
```

Expected: all green; coverage on `internal/schedule` not reduced.

## Manual validation (windowed GUI)

Build and launch the GUI, open **New Task**, and confirm each scenario.

### Scenario A — Mode-driven visibility (US1)
1. Open New Task (defaults to Recurring). **Expect**: Schedule + Preview visible; no active
   One-off time field.
2. Switch Mode → One-off. **Expect**: One-off date/time visible; Schedule/Preview hidden.
3. Switch back → Recurring. **Expect**: previously typed Schedule text still present.

### Scenario B — Required-field gating (US2)
1. Leave Name and Command empty. **Expect**: Save disabled; both fields flagged.
2. Fill Name + Command + a valid Schedule. **Expect**: Save enabled.
3. One-off mode with a past date. **Expect**: Save disabled with a "must be in the future" reason.

### Scenario C — Combined preview (US3)
1. Command `cmd`, Arguments lines `/c` and `echo hello world`. **Expect**: Preview shows
   `Will run: cmd /c "echo hello world"` (display quoting only).
2. Schedule `every day at 09:00`. **Expect**: Preview also shows the summary + next run times.
3. Clear Schedule. **Expect**: guidance text, not a blank row.

### Scenario D — Interval anchor (US4)
1. Schedule `every 15 minutes`. Note the "Start at" field appears (sub-daily interval).
2. Set Start at `09:00` (or type `every 15 minutes starting at 09:00`). **Expect**: previewed
   run times fall on :00/:15/:30/:45.
3. Schedule `every day at 09:00`. **Expect**: "Start at" field disappears (not applicable).
4. CLI parity: `gosched` creating `every 15 minutes starting at 09:00` yields the same aligned
   runs (anchor carried in the phrase, no API change).

### Scenario E — Easier inputs (US5)
1. Timezone field: type `Amer` → suggestions include `America/...`; pick one. Typing `UTC`
   also accepted. An invalid zone blocks Save.
2. One-off: set date + time via the inputs (no raw RFC 3339 typing); the echo shows the parsed
   local time.
3. Schedule "Examples" affordance lists intervals, daily, weekday sets, single weekday, monthly
   ordinals.

### Scenario F — Layout & polish (US6)
1. Sections "What to run" / "When" / "Advanced Settings" are visually separated.
2. Advanced Settings is collapsed initially; expand it to see Overlap/Catch-up with
   human-readable labels. Save a task, reopen it (Edit) — labels reflect stored values.
3. Type into Arguments — the "One argument per line" caption stays visible.
4. Hover Save/Cancel — cursor becomes a hand/pointer.

## Regression / compatibility
- Create a plain `every 15 minutes` task (no anchor) and confirm runs match prior behavior.
- Open and re-save a task created before this change; no field is lost or altered in meaning.
