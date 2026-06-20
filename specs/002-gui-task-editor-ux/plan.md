# Implementation Plan: GUI Task Editor UX Overhaul

**Branch**: `002-gui-task-editor-ux` | **Date**: 2026-06-20 | **Spec**: [spec.md](spec.md)

**Input**: Feature specification from `specs/002-gui-task-editor-ux/spec.md`

## Summary

Rework the Fyne desktop **New Task / Edit Task** dialog ([gui/editor.go](../../gui/editor.go)) for
clarity and safety, and add an optional **anchor/start time** for sub-daily fixed-interval
schedules. The dialog moves from the layout-limited `dialog.NewForm` to a custom-built
`dialog.NewCustom` body so it can: show only the time field relevant to the chosen Mode, group
fields into "What to run" / "When" / collapsible "Advanced Settings" (overlap & catch-up with
human-readable labels), present a combined schedule-and-command Preview, validate required fields
inline and gate Save, offer a searchable timezone dropdown and a typo-proof one-off time entry,
expose schedule grammar help, keep a persistent "one argument per line" caption, and show a hand
cursor on buttons.

The anchor capability reuses the engine's **existing** `Schedule.Anchor` support: today
`finish()` overwrites the anchor with `now`; we extend the human grammar with a `starting at`/
`from` clause (valid only for sub-daily intervals) so the parser can set a chosen anchor instead.
No engine/recurrence change is required — `nextRecurring` already honors `sch.Anchor`.

## Technical Context

**Language/Version**: Go (latest stable; module already on the project toolchain)

**Primary Dependencies**: Fyne v2 (GUI: `widget`, `container`, `dialog`, `desktop`); `widget.Accordion`,
`widget.SelectEntry` from core Fyne; `teambition/rrule-go` (recurrence, already used). **Added**
`fyne.io/x/fyne` (official Fyne extension repo) for its `Calendar` widget — the graphical one-off
date picker; pure-Go, no extra native libraries.

**Storage**: SQLite via existing store; **no schema change** — `schedules.anchor` already persists
(`domain.Schedule.Anchor`).

**Testing**: `go test -race`; Fyne headless test driver (GUI unit tests already run cgo-free in
`gui/`); table-driven parser tests in `internal/schedule`.

**Target Platform**: Linux, macOS, Windows desktop (`gosched-gui`, built `-H windowsgui`).

**Project Type**: Desktop app (thin GUI client) over a local daemon; single Go module.

**Performance Goals**: GUI interactions feel instant; preview round-trips stay off the UI thread
(existing pattern). Engine dispatch budget (p99 < 100 ms) is unaffected — no hot-path change.

**Constraints**: Internal scheduling stays UTC; anchor interpreted in the task's IANA timezone with
DST handling identical to existing rules; backward compatible with un-anchored tasks.

**Scale/Scope**: One dialog (~170 lines today) plus a small parser extension and a curated
timezone list; no change to daemon scheduling loop.

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- **I. Code Quality** — PASS. Changes are idiomatic Fyne/Go; new helpers (label⇄policy mapping,
  anchor parsing, custom-cursor button) are small, single-purpose, and documented. No new panics;
  errors wrapped with context. The dialog refactor reduces a long function into composed builders.
- **II. Testing Standards** — PASS (planned). Parser anchor behavior gets table-driven unit tests
  (including a failing-first regression for alignment) under injected `now`. GUI logic
  (label mapping, mode-driven visibility, validation gating, command-line preview assembly) is
  unit-tested with the headless driver and a fake backend, mirroring existing `gui/app_test.go`.
  No reliance on wall-clock sleeps. Coverage on `internal/schedule` must not drop.
- **III. UX Consistency** — PASS, and directly advanced: clearer required-field feedback,
  consistent RFC 3339 handling preserved at the boundary, actionable validation messages naming the
  field. Human-readable labels are presentation-only; stored policy values and the CLI contract are
  unchanged.
- **IV. Performance** — PASS. No scheduling hot-path change. Preview calls remain async/off-thread.
  No new allocations on any dispatch path; the anchor is computed once at parse time.

No violations → Complexity Tracking left empty.

## Project Structure

### Documentation (this feature)

```text
specs/002-gui-task-editor-ux/
├── plan.md              # This file
├── spec.md              # Feature spec
├── research.md          # Phase 0 — decisions (Fyne widgets, anchor grammar, cursor)
├── data-model.md        # Phase 1 — editor view model, anchor, label mapping
├── quickstart.md        # Phase 1 — manual + automated validation guide
├── contracts/
│   ├── schedule-grammar.md   # extended human grammar (starting at / from)
│   └── editor-ui.md          # dialog UI contract (fields, sections, states)
└── checklists/
    └── requirements.md  # spec quality checklist (from /speckit-specify)
```

### Source Code (repository root)

```text
gui/
├── editor.go            # PRIMARY: rebuilt dialog (custom layout, sections, validation, preview)
├── editor_test.go       # NEW: unit tests for mapping, visibility, validation, cmdline preview
├── widgets.go           # NEW: custom-cursor button (desktop.Cursorable) + small helpers
├── app.go               # unchanged (Backend interface already exposes Preview)
└── app_test.go          # existing patterns reused by editor_test.go

internal/schedule/
├── parse.go             # extend parseInterval + finish: optional "starting at"/"from" anchor
├── parse_test.go        # add anchor-clause cases
└── recur_test.go        # add/confirm anchored alignment coverage (engine already supports it)

docs/
└── gui-fields.md        # update field reference (anchor clause, new inputs, advanced section)
```

**Structure Decision**: Single Go module, existing layout retained. The work concentrates in
`gui/` (dialog rebuild + new small widget file + tests) and a localized extension to
`internal/schedule/parse.go`. No API request/response shape change is required: the anchor travels
inside the existing schedule phrase, so `PreviewRequest`, `TaskCreateRequest`, and
`TaskUpdateRequest` are untouched and the CLI inherits the new grammar for free.

## Complexity Tracking

> No constitution violations — section intentionally empty.
