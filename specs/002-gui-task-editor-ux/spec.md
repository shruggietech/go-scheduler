# Feature Specification: GUI Task Editor UX Overhaul

**Feature Branch**: `002-gui-task-editor-ux`

**Created**: 2026-06-20

**Status**: Draft

**Input**: User description: "GUI New Task / Edit Task dialog UX overhaul — mode-driven field visibility, advanced-settings collapsible with human-readable policy labels, combined schedule + command preview, required-field validation, timezone dropdown, one-off date/time picker, schedule help, visual grouping, persistent argument caption, hand cursor on buttons, and an optional anchor/start time for fixed-interval recurring schedules."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Only the relevant time field is shown (Priority: P1)

A user opening the **New Task** dialog picks a **Mode** (Recurring or One-off). The form
shows only the time input that matters for that mode: in Recurring mode they see **Schedule**
and its **Preview**; in One-off mode they see the **One-off time** picker. The unused field is
not visible (or is clearly inactive), so the user is never confused about which box to fill in.

**Why this priority**: This is the single biggest source of confusion in the current form —
both time fields are always shown and look active. Fixing it delivers immediate clarity and is
independently demonstrable.

**Independent Test**: Open the dialog, toggle Mode between Recurring and One-off, and confirm
the visible time field switches accordingly with no leftover inactive-but-active-looking field.

**Acceptance Scenarios**:

1. **Given** the dialog is open in Recurring mode, **When** the user views the form, **Then**
   Schedule and Preview are shown and One-off time is hidden or disabled.
2. **Given** the dialog is in Recurring mode, **When** the user switches Mode to One-off,
   **Then** Schedule and Preview are hidden/disabled and One-off time becomes the active input.
3. **Given** the dialog is in One-off mode, **When** the user switches back to Recurring,
   **Then** the Schedule and Preview reappear with any previously entered values intact.

---

### User Story 2 - Guided required-field validation before save (Priority: P1)

A user filling out the form sees which fields are required (**Name**, **Command**, and the
active time field). The **Save** action is guarded — it cannot succeed while a required field
is empty or invalid — and the user gets inline feedback about what is missing rather than only
discovering it after pressing Save.

**Why this priority**: Prevents the most common failed-save path and removes the
error-after-the-fact frustration. Independently valuable and testable.

**Independent Test**: Open the dialog, attempt to save with Name and/or Command empty, and
confirm Save is blocked with a clear indication of the missing field; fill them in and confirm
Save becomes available.

**Acceptance Scenarios**:

1. **Given** an empty form, **When** the user attempts to Save, **Then** the save is prevented
   and the missing required fields are indicated inline.
2. **Given** Recurring mode with an empty Schedule, **When** the user attempts to Save,
   **Then** save is prevented until a Schedule is entered.
3. **Given** One-off mode with an empty or past One-off time, **When** the user attempts to
   Save, **Then** save is prevented with feedback that a valid future time is required.
4. **Given** all required fields are valid, **When** the user views the form, **Then** Save is
   enabled.

---

### User Story 3 - Combined schedule and command preview (Priority: P1)

As the user types, the **Preview** area shows two things at once: (a) a plain-language summary
of the schedule plus the next few run times, and (b) the fully resolved command line — the
command and each argument exactly as the task will be invoked. The user can confirm both *when*
and *what* will run before saving.

**Why this priority**: Closes the loop on the form's most error-prone inputs (schedule phrasing
and per-line arguments) and makes the previously-empty Preview row useful.

**Independent Test**: Enter a command with several arguments and a valid schedule, and confirm
the Preview shows both the human schedule summary with next runs and the assembled command line.

**Acceptance Scenarios**:

1. **Given** a valid Schedule, **When** the user finishes typing, **Then** the Preview shows a
   plain-language summary and the next several run times.
2. **Given** a Command and one-argument-per-line Arguments, **When** the user edits either,
   **Then** the Preview shows the resolved command line reflecting those exact arguments.
3. **Given** no schedule has been entered yet, **When** the dialog first opens, **Then** the
   Preview area shows helpful guidance text rather than appearing blank/broken.

---

### User Story 4 - Anchor/start time for fixed-interval schedules (Priority: P2)

A user creating a fixed-interval recurring task (e.g. "every 15 minutes") can optionally
specify when the first cycle should start, so the interval is aligned to a chosen anchor
instead of being arbitrarily tied to the moment of creation (e.g. an off-looking 6:07 pm). The
Preview reflects the anchored run times.

**Why this priority**: A real scheduling gap — users frequently want "every 15 minutes starting
at the top of the hour." High value but builds on the schedule/preview work above.

**Independent Test**: Create an "every 15 minutes" task with an anchor of 9:00, and confirm the
previewed run times fall on :00/:15/:30/:45 rather than offset from the creation moment.

**Acceptance Scenarios**:

1. **Given** a fixed-interval schedule, **When** the user supplies an anchor start time,
   **Then** the previewed and actual run times align to that anchor plus multiples of the
   interval.
2. **Given** a fixed-interval schedule with an anchor in the past, **When** the schedule is
   evaluated, **Then** the next run is the first anchor-aligned time at or after now.
3. **Given** a fixed-interval schedule with no anchor supplied, **When** the schedule is
   evaluated, **Then** behavior matches today's default (anchored at creation/first evaluation).
4. **Given** a non-interval schedule (daily/weekday/monthly), **When** the user views the form,
   **Then** the interval anchor input is not offered (it does not apply).

---

### User Story 5 - Easier inputs: timezone dropdown, time picker, schedule help (Priority: P2)

A user can pick a **Timezone** from a searchable dropdown of common zones (while still being
able to enter any valid IANA name), choose a **One-off time** with a date/time picker (or get
live confirmation of the parsed local time) instead of hand-typing an RFC 3339 string, and
discover the supported **Schedule** phrasings via an in-form help/examples affordance.

**Why this priority**: Removes guesswork and memorization for the fiddliest inputs. Valuable
but secondary to correctness/clarity items.

**Independent Test**: Open the dialog and confirm the timezone field offers a searchable list,
the one-off time can be set without typing raw RFC 3339, and schedule examples are reachable
from the form.

**Acceptance Scenarios**:

1. **Given** the Timezone field, **When** the user interacts with it, **Then** a searchable
   list of common zones is offered and a typed valid IANA name is also accepted.
2. **Given** an unknown/invalid timezone, **When** the user attempts to Save, **Then** it is
   rejected with clear feedback.
3. **Given** One-off mode, **When** the user sets the time via the picker, **Then** the form
   captures a valid future timestamp without the user typing RFC 3339 by hand.
4. **Given** the Schedule field, **When** the user opens the help/examples affordance, **Then**
   the supported forms (intervals, daily, weekday sets, single weekday, monthly ordinals) are
   shown.

---

### User Story 6 - Cleaner, less error-prone layout (Priority: P3)

The form is visually grouped into sections — **What to run** (Name, Command, Arguments),
**When** (Timezone, Mode, Schedule/One-off, Preview), and **Advanced Settings** (overlap and
catch-up behavior). Advanced Settings is collapsed by default and uses human-readable labels
instead of raw policy codes. A persistent caption under Arguments reminds the user of the
one-argument-per-line rule, and clickable buttons show a hand/pointer cursor on hover.

**Why this priority**: Polish that reduces cognitive load and surface-area for mistakes; depends
on the structural changes above being in place.

**Independent Test**: Open the dialog and confirm the grouped sections, the collapsed Advanced
Settings holding overlap/catch-up with friendly labels, the persistent Arguments caption, and a
pointer cursor when hovering buttons.

**Acceptance Scenarios**:

1. **Given** the dialog opens, **When** the user views it, **Then** fields are grouped into
   labeled sections with visual separation.
2. **Given** the dialog opens, **When** the user views it, **Then** Advanced Settings is
   collapsed and does not show overlap/catch-up until expanded.
3. **Given** Advanced Settings is expanded, **When** the user views the overlap and catch-up
   options, **Then** they read as human-friendly labels (not raw codes), and the saved task
   stores the correct underlying policy values.
4. **Given** the Arguments field, **When** the user has typed into it (placeholder gone),
   **Then** a persistent caption still communicates "one argument per line."
5. **Given** any clickable button, **When** the user hovers over it, **Then** the cursor changes
   to a hand/pointer.

---

### Edge Cases

- **Mode toggle preserves entries**: switching Mode back and forth must not discard text already
  entered in either time field.
- **Editing an existing task**: opening Edit must pre-populate every field (including
  human-readable advanced labels) from stored policy values and reflect any anchored interval.
- **Sub-daily anchor only**: an `at`/anchor time is meaningful only for sub-daily fixed
  intervals; daily-or-coarser schedules continue to use their own time-of-day clause, and the
  existing rejection of `every 15 minutes at 09:00`-style ambiguity must remain coherent with
  the new anchor mechanism.
- **Anchor in a different timezone**: the anchor is interpreted in the task's timezone with
  correct DST handling; internal storage remains UTC.
- **Invalid command line**: empty Command must block Save even if arguments are present.
- **Preview while invalid**: an unparseable schedule shows a clear warning in the Preview rather
  than stale or blank content.
- **Backward compatibility**: tasks created before this change (no anchor) continue to schedule
  exactly as before.

## Requirements *(mandatory)*

### Functional Requirements

#### Field visibility & mode

- **FR-001**: The editor MUST show only the time input relevant to the selected Mode — Schedule
  and Preview for Recurring, One-off time for One-off — and MUST hide or visibly disable the
  other so it does not appear active.
- **FR-002**: Toggling Mode MUST preserve any values already entered in both the Schedule and
  One-off time fields for the duration of the dialog session.

#### Validation

- **FR-003**: The editor MUST mark Name, Command, and the active time field as required.
- **FR-004**: The editor MUST prevent saving while any required field is empty or invalid and
  MUST surface inline feedback identifying the missing/invalid field before save is attempted.
- **FR-005**: One-off time MUST be validated as a parseable, future timestamp before save is
  allowed.
- **FR-006**: Timezone MUST be validated as `Local` or a known IANA zone; unknown zones MUST be
  rejected with clear feedback.

#### Preview

- **FR-007**: The Preview MUST display a plain-language summary of a valid Schedule together with
  the next several run times.
- **FR-008**: The Preview MUST display the fully resolved command line (command plus each
  argument as it will be invoked), updating as Command or Arguments change.
- **FR-009**: Before a valid schedule is entered, the Preview MUST show guidance text rather than
  appearing empty; an invalid schedule MUST show a clear warning.

#### Anchor / start time

- **FR-010**: For fixed-interval (sub-daily) recurring schedules, the system MUST allow an
  optional anchor/start time that determines when the first cycle begins.
- **FR-011**: When an anchor is supplied, run times MUST fall on the anchor plus integer
  multiples of the interval; the next run MUST be the first such time at or after the current
  time.
- **FR-012**: When no anchor is supplied, fixed-interval scheduling MUST behave as it does today
  (no behavioral change for existing tasks).
- **FR-013**: The anchor input MUST only be offered for fixed-interval schedules and MUST be
  interpreted in the task's timezone while internal scheduling remains in UTC.

#### Inputs & help

- **FR-014**: The Timezone field MUST offer a searchable list of common zones while still
  accepting any valid IANA name typed by the user.
- **FR-015**: The One-off time field MUST let the user set a valid future timestamp without
  hand-typing RFC 3339 (via a date/time picker and/or live parsed-time confirmation).
- **FR-016**: The Schedule field MUST provide an in-form, discoverable help/examples affordance
  listing supported phrasing (intervals, daily-with-time, weekday/weekend sets, single weekday,
  monthly ordinals).

#### Layout & advanced settings

- **FR-017**: The editor MUST group fields into visually separated sections: "What to run"
  (Name, Command, Arguments), "When" (Timezone, Mode, Schedule/One-off, Preview), and "Advanced
  Settings".
- **FR-018**: Overlap and Catch-up MUST appear under an "Advanced Settings" section that is
  collapsed by default.
- **FR-019**: Overlap and Catch-up MUST be presented with human-readable labels, and the editor
  MUST translate those labels back to the correct underlying policy values on save and translate
  stored values back to labels when editing.
- **FR-020**: A persistent caption under the Arguments field MUST communicate the
  "one argument per line" rule even after the placeholder disappears.
- **FR-021**: Clickable buttons MUST display a hand/pointer cursor when hovered.

#### Compatibility

- **FR-022**: All existing task fields, defaults (Local timezone, queue-one overlap, catch-up
  one), and validation rules MUST remain functional; the changes are additive/clarifying and
  MUST NOT break creation or editing of existing tasks.

### Key Entities *(include if feature involves data)*

- **Task (editor view)**: the in-dialog representation of a scheduled task — name, command,
  argument list, timezone, mode, schedule phrase (with optional interval anchor) or one-off
  time, overlap policy, catch-up policy.
- **Schedule (interval with anchor)**: a fixed-interval recurrence carrying an interval duration
  and an optional anchor/start instant from which occurrences are computed.
- **Policy label mapping**: the user-facing label ⇄ stored policy-value correspondence for
  overlap and catch-up.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: A first-time user can create a valid recurring task without consulting external
  documentation, because only relevant fields are shown and required fields are clearly marked.
- **SC-002**: Users never reach a failed Save caused by an empty required field — such saves are
  blocked with inline guidance 100% of the time.
- **SC-003**: Before saving, users can confirm both the next run times and the exact command
  line that will execute, directly in the dialog.
- **SC-004**: A user can configure "every 15 minutes starting at a chosen time" and see the
  previewed runs align to that anchor (e.g. :00/:15/:30/:45).
- **SC-005**: Setting timezone and one-off time requires no memorization of IANA names or RFC
  3339 syntax for the common cases.
- **SC-006**: Advanced overlap/catch-up behavior is available but hidden by default, and is
  shown in plain language when expanded, while stored task data remains unchanged in meaning.
- **SC-007**: Existing tasks created before this change continue to run on exactly their prior
  schedule (zero behavioral regression for un-anchored intervals).

## Assumptions

- The change targets the Go-native Fyne desktop GUI (`gosched-gui`) editor dialog and the
  schedule-parsing layer used to evaluate fixed-interval schedules; the CLI and daemon contracts
  are extended only as needed to carry an optional interval anchor.
- "Human-readable labels" for overlap/catch-up are a presentation concern only; persisted policy
  values are unchanged so existing stored tasks and the CLI remain compatible.
- The anchor applies to sub-daily fixed intervals (seconds/minutes/hours); daily-or-coarser
  schedules keep using their existing `at <time>` clause and are unaffected.
- Internal scheduling stays in UTC; the anchor is interpreted in the task's IANA timezone with
  DST handling consistent with the existing engine.
- A "searchable timezone dropdown" is seeded with a curated set of common zones; the full IANA
  database is not enumerated in the UI but any valid name remains acceptable.
- Cursor-on-hover for buttons is achievable within the GUI toolkit's theming/cursor capabilities;
  if a given button type cannot expose a custom cursor, an equivalent affordance is acceptable.
