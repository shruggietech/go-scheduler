---
description: "Task list for GUI Task Editor UX Overhaul"
---

# Tasks: GUI Task Editor UX Overhaul

**Input**: Design documents from `specs/002-gui-task-editor-ux/`

**Prerequisites**: [plan.md](plan.md), [spec.md](spec.md), [research.md](research.md),
[data-model.md](data-model.md), [contracts/](contracts/), [quickstart.md](quickstart.md)

**Tests**: INCLUDED — the project constitution makes testing NON-NEGOTIABLE (Principle II), so
each behavioral change ships with tests (parser tests under injected clock; GUI logic tests under
the Fyne headless driver + fake backend).

**Organization**: Tasks are grouped by user story. NOTE: most GUI stories modify the single file
`gui/editor.go`, so they are largely **sequential** (not `[P]`) with respect to each other; `[P]`
is used only where files genuinely differ (e.g. `internal/schedule/parse.go`, `gui/widgets.go`,
`docs/`).

## Path Conventions

Single Go module at repo root. GUI in `gui/`, schedule parser in `internal/schedule/`, docs in
`docs/`.

---

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Test scaffolding and shared constants/helpers used across stories.

- [x] T001 [P] Create `gui/editor_test.go` with a test harness reusing the fake backend pattern from `gui/app_test.go` (headless app, helper to open the editor and access field widgets).
- [x] T002 [P] Add label⇄wire maps and the curated timezone list to a new `gui/editor_data.go`: overlap/catch-up display↔`domain` value maps (per [data-model.md](data-model.md) §3) and the ordered `commonZones` slice (§4).
- [x] T003 [P] Add unit tests in `gui/editor_data_test.go` asserting every `domain.OverlapPolicy`/`domain.CatchupPolicy` value round-trips through the label maps and that unknown values fall back to the default label.

**Checkpoint**: Test harness + mapping data exist and compile.

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Restructure the dialog onto a layout that can host sections, a collapsible advanced
panel, hidden fields, and a gate-able Save button. Without this, no GUI story can be implemented.

**⚠️ CRITICAL**: Blocks US1, US2, US3, US5, US6.

- [x] T004 Add a custom pointer-cursor button widget in `gui/widgets.go` embedding `widget.Button` and implementing `desktop.Cursorable` returning `desktop.PointerCursor` (per [research.md](research.md) §8), with doc comment.
- [x] T005 [P] Add `gui/widgets_test.go` verifying the custom button reports `desktop.PointerCursor` and still invokes its tap callback.
- [x] T006 Rebuild `showTaskEditor` in `gui/editor.go` to use `dialog.NewCustom` with a `container.NewVBox` body composed of three section blocks ("What to run", "When", "Advanced Settings") and a footer with custom-cursor **Cancel**/**Save** buttons; preserve all existing fields and the existing `submitTask`/`taskForm` flow (no behavior change yet). Keep `d.Resize(580x620)`.
- [x] T007 [US-foundation] Wire a central `revalidate()` + managed Save-enabled state into the rebuilt dialog (initially always-valid) so later stories can plug field validity in; add a smoke test in `gui/editor_test.go` that the rebuilt dialog opens, lists the expected fields, and Save is reachable.

**Checkpoint**: Dialog renders with sectioned layout and custom buttons; existing create/edit still works; tests green.

---

## Phase 3: User Story 1 — Only the relevant time field is shown (Priority: P1) 🎯 MVP

**Goal**: Mode toggles which time input is visible; the other is hidden and never looks active;
values persist across toggles.

**Independent Test**: Open editor, toggle Mode Recurring↔One-off, confirm only the relevant field
shows and prior text survives the round-trip.

- [x] T008 [US1] Wrap Schedule+Preview and the One-off time input in their own containers held as variables in `gui/editor.go`; in `mode.OnChanged`, `Show()`/`Hide()` the relevant container and `Refresh()` the content (per [contracts/editor-ui.md](contracts/editor-ui.md) state machine).
- [x] T009 [US1] Ensure both time entries are constructed once and never cleared on toggle so values persist (FR-002).
- [x] T010 [US1] Add tests in `gui/editor_test.go`: (a) Recurring shows Schedule, hides One-off; (b) switching to One-off inverts visibility; (c) text entered in each field survives a Recurring→One-off→Recurring round-trip.

**Checkpoint**: US1 fully functional and independently testable.

---

## Phase 4: User Story 2 — Guided required-field validation before save (Priority: P1)

**Goal**: Name, Command, and the active time field are marked required; Save is blocked with inline
feedback until all relevant fields are valid.

**Independent Test**: Attempt Save with empty Name/Command → blocked + flagged; fill all required →
Save enabled.

- [x] T011 [US2] Attach `Validator`s to Name, Command, and Timezone (via `timezone.Resolve`) in `gui/editor.go`; mark required fields with a visible "*".
- [x] T012 [US2] Extend `revalidate()` to compute the mode-dependent required set (Schedule non-empty+parseable for Recurring; valid future timestamp for One-off) and enable/disable Save accordingly (FR-003/004/005/006).
- [x] T013 [US2] Surface inline reasons (field-level) before Save is pressed; keep `submitTask` as the final guard.
- [x] T014 [US2] Add tests in `gui/editor_test.go`: Save disabled when Name/Command empty; disabled for One-off past/blank time; disabled for unknown timezone; enabled when all valid.

**Checkpoint**: US2 functional; failed-save-by-empty-field is impossible.

---

## Phase 5: User Story 3 — Combined schedule and command preview (Priority: P1)

**Goal**: Preview shows both the schedule summary + next runs AND the resolved command line;
guidance text when empty; warning when invalid.

**Independent Test**: Enter command + multi-line args + valid schedule → both blocks render; clear
schedule → guidance, not blank.

- [x] T015 [US3] Add a command-line preview builder (Command + split args, cosmetic quoting of tokens with spaces) — implement as a small pure function in `gui/editor.go` (or `gui/editor_data.go`) reusing `splitArgs` (per [data-model.md](data-model.md) §5).
- [x] T016 [P] [US3] Unit-test the command-line builder in `gui/editor_data_test.go` (no-args, spaced args quoted, blank lines dropped).
- [x] T017 [US3] Update the Preview area in `gui/editor.go` to render a "Will run:" command-line block (updates on Command/Args change) plus the existing async schedule block; seed empty-state guidance text and an "⚠ <reason>" invalid state (FR-007/008/009).
- [x] T018 [US3] Add tests in `gui/editor_test.go`: command-line block reflects edits to Command/Args; empty schedule shows guidance not blank.

**Checkpoint**: US3 functional; Preview row is always meaningful.

---

## Phase 6: User Story 4 — Anchor/start time for fixed-interval schedules (Priority: P2)

**Goal**: Sub-daily interval schedules accept an optional anchor (`starting at`/`from`), aligning
runs to a chosen phase; GUI offers a "Start at" field only for interval schedules.

**Independent Test**: `every 15 minutes starting at 09:00` previews runs on :00/:15/:30/:45; CLI
parity; non-interval schedules don't offer the field.

### Tests first (constitution: regression must fail before fix)

- [x] T019 [P] [US4] Add failing table-driven tests in `internal/schedule/parse_test.go` for the anchor clause: `every 15 minutes starting at 09:00` and `every 30 minutes from 9am` set `Anchor` to the expected wall time; `every 15 minutes at 09:00` still rejected; `every day starting at 09:00` rejected with a clear message.
- [x] T020 [P] [US4] Add tests in `internal/schedule/recur_test.go` (or extend) asserting anchored alignment under an injected `now` (e.g. now=09:07 → next run 09:15) and that an un-anchored `every 15 minutes` keeps prior behavior.

### Implementation

- [x] T021 [US4] Extend `reInterval`/`parseInterval` in `internal/schedule/parse.go` to recognize an optional trailing `(starting at|from) <time>` for sub-daily units only, returning the parsed anchor time-of-day; reject the clause for non-sub-daily schedules (per [contracts/schedule-grammar.md](contracts/schedule-grammar.md)).
- [x] T022 [US4] Update `finish` in `internal/schedule/parse.go` to set `sch.Anchor` from the parsed anchor (wall time in task tz on the reference day) when present, else keep `anchor = now`; include the anchor in `HumanSummary`.
- [x] T023 [US4] In `gui/editor.go`, reveal an optional "Start at" time input only when the current Schedule text parses as a sub-daily interval; when filled, append `starting at <HH:MM>` to the phrase sent to Preview/Create (single grammar code path).
- [x] T024 [US4] Add GUI test in `gui/editor_test.go`: "Start at" appears for `every 15 minutes`, hidden for `every day at 09:00`, and its value reaches the submitted schedule phrase.

**Checkpoint**: US4 functional in GUI and CLI; engine unchanged; backward compatible.

---

## Phase 7: User Story 5 — Easier inputs: timezone dropdown, time picker, schedule help (Priority: P2)

**Goal**: Searchable timezone combo; one-off time set without raw RFC 3339 + parsed-local echo;
discoverable schedule examples.

**Independent Test**: Timezone offers suggestions yet accepts typed IANA; one-off set via inputs
with live echo; examples reachable.

- [x] T025 [US5] Replace the Timezone `Entry` with `widget.NewSelectEntry(commonZones)` in `gui/editor.go`, default `Local`, validated via `timezone.Resolve` (FR-014).
- [x] T026 [US5] Replace the raw RFC 3339 One-off `Entry` with date + time inputs that assemble a future RFC 3339 instant, plus a live label echoing the parsed local time and a clear past/invalid message (FR-015); keep `submitTask` parsing compatible.
- [x] T027 [US5] Add a Schedule "Examples" help affordance (pointer-cursor button opening an info dialog) listing supported forms incl. the new `starting at` anchor (FR-016).
- [x] T028 [US5] Add tests in `gui/editor_test.go`: timezone combo accepts a typed valid zone and rejects an invalid one; one-off assembly produces a valid future instant and flags a past one; examples affordance is present.

**Checkpoint**: US5 functional; no raw-format memorization needed for common cases.

---

## Phase 8: User Story 6 — Cleaner, less error-prone layout (Priority: P3)

**Goal**: Visual sections; collapsed Advanced Settings with human-readable overlap/catch-up;
persistent Arguments caption; pointer cursor on buttons.

**Independent Test**: Sections visible; Advanced collapsed by default with friendly labels that
persist correct wire values; Arguments caption persists; buttons show hand cursor.

- [x] T029 [US6] Put Overlap and Catch-up inside a `widget.Accordion` "Advanced Settings" item (`Open=false`) in `gui/editor.go`, using the display labels from `gui/editor_data.go`; translate label→wire on submit and wire→label on edit-open (FR-018/019).
- [x] T030 [US6] Add the persistent muted "One argument per line" caption under the Arguments entry (FR-020).
- [x] T031 [US6] Ensure all dialog buttons (Save, Cancel, Examples) use the custom pointer-cursor widget (FR-021); confirm section separators/headers render for the three groups (FR-017).
- [x] T032 [US6] Add tests in `gui/editor_test.go`: advanced section default-collapsed; selecting a friendly overlap/catch-up label submits the correct wire value; editing a task with stored `skip`/`none` shows the matching labels.

**Checkpoint**: All user stories independently functional.

---

## Phase 9: Polish & Cross-Cutting Concerns

- [x] T033 [P] Update [docs/gui-fields.md](../../docs/gui-fields.md): document the Mode-driven visibility, Advanced Settings + friendly labels, combined preview, timezone combo, one-off picker, schedule examples, and the new `starting at`/`from` anchor clause.
- [x] T034 [P] Update the Schedule grammar doc/help text and `Parse` doc comment in `internal/schedule/parse.go` to mention the anchor clause.
- [x] T035 Run `gofmt -l .`, `go vet ./...`, and `go test ./... -race`; fix any findings; confirm `internal/schedule` coverage is not reduced.
- [x] T036 Execute [quickstart.md](quickstart.md) scenarios A–F manually against the windowed GUI and record results.

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: no dependencies.
- **Foundational (Phase 2)**: depends on Setup; **blocks** US1, US2, US3, US5, US6.
- **US4 (Phase 6)**: the parser half (T019–T022) depends only on Setup (independent of the GUI
  rebuild); the GUI half (T023–T024) depends on Foundational.
- **Polish (Phase 9)**: after all targeted stories.

### User Story Dependencies

- US1, US2, US3, US5, US6 all share `gui/editor.go` → implement **sequentially** (recommended
  order P1s → P2s → P3): US1 → US2 → US3 → US5 → US6.
- US4 parser work is **independent** of the GUI file and can proceed in parallel with the GUI
  stories; its GUI field (T023) should land after US3's preview wiring to share the phrase path.

### Within Each Story

- Tests written before/with implementation; parser regression tests (T019–T020) MUST fail first.
- Commit after each task or logical group.

### Parallel Opportunities

- Setup: T001, T002, T003 in parallel ([P]).
- T005 (widget test) parallel with other Phase-2 prep; T004→T006→T007 sequential (same file).
- US4 parser tests T019, T020 in parallel; T016 (cmdline builder test) parallel within US3.
- Polish docs T033, T034 in parallel.

---

## Parallel Example: Setup

```bash
# Independent files — safe to do together:
Task: "Create gui/editor_test.go harness"          # T001
Task: "Add label maps + timezone list (gui/editor_data.go)"  # T002
Task: "Add mapping round-trip tests (gui/editor_data_test.go)"  # T003
```

## Parallel Example: User Story 4 (parser, independent of GUI)

```bash
Task: "Failing anchor-grammar tests in internal/schedule/parse_test.go"   # T019
Task: "Anchored-alignment tests in internal/schedule/recur_test.go"        # T020
```

---

## Implementation Strategy

### MVP First (the three P1 stories)

1. Phase 1 Setup → Phase 2 Foundational (dialog rebuild + custom button).
2. US1 (mode visibility) → US2 (validation gating) → US3 (combined preview).
3. **STOP and VALIDATE**: quickstart Scenarios A–C. This alone resolves the worst usability
   issues and is shippable.

### Incremental Delivery

4. US4 anchor (parser + GUI) → validate Scenario D (+ CLI parity).
5. US5 easier inputs → Scenario E.
6. US6 layout/advanced/cursor polish → Scenario F.
7. Phase 9 polish, docs, full `-race` suite.

---

## Notes

- `[P]` = different files, no incomplete-task dependency. Most GUI tasks are NOT `[P]` because they
  edit `gui/editor.go`.
- The anchor feature requires **no** engine/store/API change — verify this stays true (the phrase
  carries the anchor); if a test pushes you toward changing `recur.go` or request structs, stop and
  re-read [contracts/schedule-grammar.md](contracts/schedule-grammar.md).
- Keep wire/store values unchanged — friendly labels are presentation-only (FR-022).
- Run `go test -race` throughout; no wall-clock sleeps in tests (inject `now`).
