# Phase 0 Research: GUI Task Editor UX Overhaul

All "NEEDS CLARIFICATION" items from Technical Context are resolved below. Each entry records the
decision, rationale, and the alternatives weighed.

## 1. Dialog container: keep `dialog.NewForm` vs move to `dialog.NewCustom`

- **Decision**: Replace `dialog.NewForm` with `dialog.NewCustom`, building the body from a
  `container.NewVBox` of section blocks (each a labeled `widget.Card`/separator + `widget.Form`).
  Provide our own **Save**/**Cancel** buttons in the body footer so Save can be enabled/disabled.
- **Rationale**: `dialog.NewForm` renders a flat list of `FormItem`s with fixed OK/Cancel buttons
  we cannot relabel-gate cleanly, and it cannot host collapsible sections or per-section headers.
  All of FR-001 (hide a field), FR-017 (sections), FR-018 (collapsible advanced), and FR-004
  (gate Save) need direct control of the layout and the action buttons.
- **Alternatives considered**:
  - *Keep `NewForm`, toggle item visibility* — Fyne `FormItem`s can't be hidden/removed
    individually after construction without rebuilding the dialog; rejected.
  - *Full custom `widget.Window`/modal* — heavier than needed; `dialog.NewCustom` already gives a
    modal with our content.

## 2. Mode-driven field visibility (FR-001/FR-002)

- **Decision**: Build Schedule+Preview and One-off rows as containers held in variables; on
  `mode.OnChanged`, call `.Show()`/`.Hide()` on the relevant container and `Refresh()` the dialog
  content. Keep both entry widgets alive (never destroyed) so their text persists across toggles.
- **Rationale**: Show/Hide on containers is the idiomatic Fyne way to swap visible inputs while
  preserving state, satisfying FR-002 (no data loss on toggle).
- **Alternatives**: Disabling instead of hiding — kept as the fallback the spec allows
  ("hidden or disabled"), but hiding reads cleaner and removes the ambiguous greyed box.

## 3. Required-field validation and Save gating (FR-003/FR-004/FR-005/FR-006)

- **Decision**: Attach `Validator` functions to Name, Command, Timezone, and the active time
  field. Maintain a `revalidate()` routine wired to each field's `OnChanged` that recomputes
  whether all *currently-relevant* fields are valid and enables/disables the custom Save button.
  Mark required fields with a visible "*" in their section labels.
- **Rationale**: Centralized `revalidate()` handles the mode-dependent required set (Schedule vs
  One-off) that per-widget validators alone can't express. Disabling Save is the clearest guard.
- **Alternatives**: Rely solely on `widget.Form` submit-gating — insufficient because the required
  set changes with Mode and we also gate on the timezone/one-off semantic checks.

## 4. Timezone input (FR-014)

- **Decision**: Use `widget.NewSelectEntry(commonZones)` — an editable combo that offers a curated
  dropdown yet accepts any typed value. Seed with a curated list (Local, UTC, and major IANA zones
  across regions). Validate on change via the existing `timezone.Resolve`.
- **Rationale**: `SelectEntry` is core Fyne (no new dependency), gives searchable suggestions, and
  preserves free-text entry of any valid IANA name (FR-014).
- **Alternatives**: A read-only `widget.Select` of the full IANA DB — too long, and blocks valid
  names not in the list; rejected.

## 5. One-off time input (FR-015)

- **Decision**: Provide a **date** entry + a **time** entry (with placeholders and live
  validation) that assemble into an RFC 3339 instant, plus a live label echoing the parsed local
  time and a clear error when unparseable/past. **Adopted** `fyne.io/x/fyne` `Calendar` for a
  graphical month picker, surfaced via a "Pick…" button next to the Date field; selecting a day
  fills the Date entry. Typing remains supported.
- **Rationale**: The spec accepts "a date/time picker and/or live parsed-time confirmation." The
  split date+time entries with live echo remove raw-RFC3339 hand-typing; the calendar adds the
  fuller graphical pick the user explicitly requested. `fyne.io/x/fyne` is the official Fyne
  extension repo (pure-Go widgets, no extra cgo), so it builds headlessly with the rest of `gui/`.
- **Dependency note**: `fyne.io/x/fyne` is now a direct module dependency (justified here per the
  constitution's "new dependency must be justified" rule). The calendar widget pulls no native
  libraries beyond what Fyne already requires.
- **Alternatives**: Core-widgets-only (no calendar) — kept as the fallback, but the user asked for
  the graphical picker, so the dependency is warranted.

## 6. Combined schedule + command preview (FR-007/FR-008/FR-009)

- **Decision**: Keep the async schedule Preview backend call (unchanged) and add a locally-computed
  **command-line preview** string assembled from Command + per-line Args using the same arg-split
  rules as submit (`splitArgs`). Render both in the Preview area: a schedule block (summary + next
  runs) and a "Will run:" command-line block. Seed the area with guidance text when empty and a
  "⚠" warning when the schedule is invalid.
- **Rationale**: The command line is fully derivable client-side (no backend needed), so it updates
  instantly as the user types; the schedule block reuses the existing `Backend.Preview` path. This
  satisfies FR-008 without new API surface.
- **Command-line rendering**: display each token; quote tokens containing spaces for readability
  (display-only — actual execution still passes the raw arg slice). Document this in the contract.

## 7. Human-readable Advanced Settings labels + collapsible (FR-018/FR-019)

- **Decision**: Put Overlap and Catch-up inside a `widget.Accordion` item titled "Advanced
  Settings", `Open=false` by default. Use display labels mapped to/from wire values:
  - Overlap: `Queue one run` ⇄ `queue_one`, `Skip this run` ⇄ `skip`,
    `Allow concurrent runs` ⇄ `allow_concurrent`.
  - Catch-up: `Run once to catch up` ⇄ `one`, `Skip missed runs` ⇄ `none`.
  Selects display labels; submit translates label→wire; editing translates wire→label.
- **Rationale**: `widget.Accordion` is the core collapsible primitive. Mapping keeps the wire/store
  contract and CLI unchanged (presentation-only), satisfying FR-019 and FR-022.
- **Alternatives**: A custom show/hide toggle — reinvents Accordion; rejected.

## 8. Hand/pointer cursor on buttons (FR-021)

- **Decision**: Add a small custom widget `tappableButton` (or `cursorButton`) embedding
  `widget.Button` and implementing `desktop.Cursorable` to return `desktop.PointerCursor`. Use it
  for the dialog's Save/Cancel (and reuse elsewhere as desired).
- **Rationale**: Core Fyne `widget.Button` does **not** change the cursor on hover; the documented
  way to get a pointer cursor is implementing `desktop.Cursorable`. This is the minimal, idiomatic
  approach and is unit-testable (the method returns a value).
- **Alternatives**: Theme-level cursor change — Fyne has no global "buttons use pointer" setting;
  per-widget `Cursorable` is the supported mechanism.

## 9. Anchor/start time for sub-daily intervals (FR-010–FR-013)

- **Decision**: Reuse the engine's existing `Schedule.Anchor`. Extend the human grammar with an
  optional trailing clause `starting at <time>` or `from <time>` accepted **only** for sub-daily
  intervals (seconds/minutes/hours). The parser computes the anchor instant in the task timezone
  (the chosen wall time on a reference day) and sets `sch.Anchor` to it; `finish()` stops blindly
  overwriting the anchor with `now` when an explicit anchor was parsed. When no clause is present,
  behavior is unchanged (anchor = now).
- **Rationale**: `nextRecurring` already sets `opt.Dtstart = sch.Anchor` and returns the first
  occurrence at/after `now`, so alignment to the anchor works with **zero** engine change. Carrying
  the anchor inside the phrase means `PreviewRequest`/`TaskCreateRequest`/CLI need no new fields —
  Preview and the CLI get the feature for free. Using `starting at`/`from` (not bare `at`) avoids
  collision with the existing rule that sub-daily intervals reject an `at <time-of-day>` clause.
- **Anchor reference day & "at or after now"**: choose the anchor's wall time on the current day in
  the task tz; rrule's `After(now)` then yields the next aligned slot (FR-011). A past anchor is
  fine — alignment is modular, so only the time-of-day phase matters for sub-daily intervals.
- **GUI surface**: in Recurring mode, when the typed schedule is a sub-daily interval, reveal an
  optional "Start at" time input; when filled, the GUI appends `starting at <HH:MM>` to the phrase
  sent to Preview/Create. The grammar-level clause remains the source of truth so typed phrases and
  the GUI field converge on one code path.
- **Alternatives considered**:
  - *Add an `Anchor` field to the API/Parse signature* — more invasive (touches server, client,
    CLI, request structs) for no functional gain; rejected in favor of the phrase-carried anchor.
  - *Anchor as full date+time* — unnecessary for sub-daily alignment (only time-of-day phase
    matters); keep it time-of-day to match the existing `at`/time-of-day parsing and avoid scope
    creep into daily-or-coarser anchoring (out of scope per spec).

## 10. Visual grouping & persistent Arguments caption (FR-017/FR-020)

- **Decision**: Three sections via labeled separators/cards: "What to run", "When", and the
  "Advanced Settings" accordion. Under the Arguments multiline entry, add a always-visible muted
  caption ("One argument per line") using a small `widget.Label` with the low-importance style.
- **Rationale**: Placeholders vanish once typing starts (the documented #1 confusion point); a
  persistent caption keeps the guidance. Cards/separators give clear visual grouping with core
  widgets.

## Cross-cutting confirmations

- **Backward compatibility**: No store/schema change; un-anchored interval tasks parse and run
  exactly as before (FR-012/FR-022/SC-007). Editing an existing task maps stored policy values back
  to labels and reconstructs any anchored phrase for display.
- **Timezone/DST**: Anchor is interpreted in the task's IANA zone; storage stays UTC, matching the
  existing `nextRecurring`/`timezone.WallTime` handling.
- **Testing strategy**: parser tests with injected `now` for anchor alignment (fail-first), GUI
  unit tests for label mapping, mode visibility, validation gating, and command-line assembly using
  the headless driver + fake backend.
