# Contract: Task Editor Dialog UI

Defines the observable behavior of the rebuilt New Task / Edit Task dialog
([gui/editor.go](../../../gui/editor.go)). Items map to FR-001…FR-021.

## Layout (sections, top to bottom)

1. **What to run**
   - Name (required *)
   - Command (required *)
   - Arguments (multiline) + persistent caption "One argument per line" (FR-020)
2. **When**
   - Timezone — searchable combo (`SelectEntry`), default `Local` (FR-014)
   - Mode — Recurring | One-off
   - Schedule (Recurring only) + "Examples" help affordance (FR-016)
   - Start at (Recurring + sub-daily interval only) — optional anchor time (FR-010)
   - One-off date + time (One-off only), with live parsed-local-time echo (FR-015)
   - Preview — schedule summary + next runs, and resolved command line (FR-007/008/009)
3. **Advanced Settings** — collapsed `Accordion` by default (FR-018)
   - Overlap — human labels (FR-019)
   - Catch-up — human labels (FR-019)

Footer: **Cancel** and **Save** buttons (custom pointer-cursor widget, FR-021). Save is
disabled until valid (FR-004).

## State machine: Mode

| Mode | Visible time inputs | Hidden |
|------|---------------------|--------|
| Recurring | Schedule, (Start at when sub-daily), Preview | One-off date/time |
| One-off | One-off date/time, (parsed-time echo) | Schedule, Start at, Preview-schedule block |

- Switching Mode preserves entered values in both branches (FR-002).
- "Start at" appears only when the current Schedule text parses as a sub-daily interval;
  otherwise it is hidden (FR-013 / US4 scenario 4).

## Validation & Save gating

| Condition | Save |
|-----------|------|
| Name empty | disabled, Name marked invalid |
| Command empty | disabled, Command marked invalid |
| Recurring & Schedule empty/unparseable | disabled, Schedule marked invalid |
| One-off & time missing/unparseable/past | disabled, field shows reason |
| Timezone not `Local`/known IANA | disabled, Timezone marked invalid |
| All relevant rules satisfied | enabled |

Feedback is inline and appears **before** Save is attempted (FR-004). Messages name the field and
the fix, consistent with constitution UX rules.

## Preview content

- **Empty schedule**: guidance text (e.g. "Type a schedule above to see upcoming runs"), not blank
  (FR-009).
- **Valid schedule**: plain-language summary + next several run times (existing backend Preview).
- **Invalid schedule**: "⚠ <reason>".
- **Command line** (always, independent of schedule): "Will run: `command arg1 "arg with space"`"
  assembled from Command + split Args using the same split rules as submit (FR-008). Display-only;
  quoting is cosmetic.

## Advanced Settings labels (display ⇄ wire)

| Control | Display options | Stored value |
|---------|-----------------|--------------|
| Overlap | Queue one run / Skip this run / Allow concurrent runs | queue_one / skip / allow_concurrent |
| Catch-up | Run once to catch up / Skip missed runs | one / none |

- Default selections: "Queue one run", "Run once to catch up".
- Edit-open maps stored value → label; unknown legacy value → default label (no crash).

## Cursor

- All footer buttons (and the Examples affordance if a button) return `desktop.PointerCursor`
  on hover (FR-021).

## Backward compatibility

- Submitting maps display labels back to existing wire values; the create/update requests are
  byte-for-byte compatible with today's API (FR-022).
- Editing a pre-existing task populates every field, including reconstructing an anchored phrase
  and mapping policy values to labels.
