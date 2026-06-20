# GUI task editor — field reference

This page explains every field in the desktop GUI's **New Task** / **Edit Task** dialog:
what it accepts, what's required, and what each option means. It's the GUI counterpart to the
CLI contract in [`specs/001-task-scheduler/contracts/cli.md`](../specs/001-task-scheduler/contracts/cli.md).

The dialog is grouped into three sections: **What to run** (Name, Command, Arguments), **When**
(Timezone, Mode, the relevant time field, and the live Preview), and a collapsible **Advanced
Settings** (Overlap, Catch-up) that starts closed. Required fields are marked with a `*` and the
**Save** button stays disabled until every required field is valid.

## At a glance

| Field | Required | Format / options | Default |
|-------|----------|------------------|---------|
| **Name** | yes | any text label | — |
| **Command** | yes | a single executable (name or full path) | — |
| **Arguments** | no | one argument per line | empty |
| **Timezone** | no | searchable list of common zones, or any IANA name / `Local` | `Local` |
| **Mode** | yes | `Recurring` or `One-off` | `Recurring` |
| **Schedule** | when Recurring (create) | human-readable phrase (see below) | — |
| **Start at** | no | anchor time for sub-daily intervals, e.g. `09:00` | — |
| **One-off date / time** | when One-off (create) | date + time picked in the task's zone, must be future | — |
| **Overlap** *(Advanced)* | no | Queue one run · Skip this run · Allow concurrent runs | Queue one run |
| **Catch-up** *(Advanced)* | no | Run once to catch up · Skip missed runs | Run once to catch up |

**Mode decides which time field is shown.** In `Recurring` mode the **Schedule** (and optional
**Start at**) field is shown and the one-off inputs are hidden; in `One-off` mode it's the reverse.
Switching Mode keeps whatever you already typed in either field. When editing an existing task,
leaving the time field blank keeps the task's current schedule.

**Live Preview.** Below the time fields, the Preview shows two things at once: a plain-language
summary of the schedule with the next few run times, and a **"Will run:"** line showing the exact
command and arguments as they will be invoked.

**Overlap and Catch-up** are shown with friendly labels but stored as the same underlying policy
values (`queue_one`/`skip`/`allow_concurrent` and `one`/`none`) used by the CLI and API.

---

## Name

A label for the task — any text. Used only to identify the task in lists and the calendar.

## Command

The program to run — **just the executable, not a full command line.** Put any arguments in the
**Arguments** field below, not here.

- Examples: `cmd`, `python`, `notepad.exe`, `C:\Windows\System32\notepad.exe`, `/usr/bin/make-report`
- Required and must be non-empty.

## Arguments

**One argument per line.** This is the most common point of confusion: don't paste a whole
command line on one line. Each line becomes one separate argument passed to the command. Blank
lines and surrounding whitespace are ignored.

To run the equivalent of `cmd /c echo hello`:

```
/c
echo hello
```

## Timezone

An [IANA time-zone name](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones) or the
literal word `Local`. The field is a searchable dropdown seeded with common zones; you can pick
one from the list or type any other valid IANA name. Schedules are interpreted in this zone, with
correct Daylight Saving Time handling; the backend stores everything in UTC.

- Examples: `Local` (your system clock), `UTC`, `America/New_York`, `Europe/London`, `Asia/Tokyo`
- An empty field is treated as `Local`. An unknown name (e.g. `Mars/Phobos`) is rejected and
  blocks Save.

## Mode

- **Recurring** — the task fires repeatedly on a schedule. Fill in **Schedule**.
- **One-off** — the task fires exactly once at a specific time. Fill in **One-off time**.

## Schedule *(Recurring mode)*

A plain-language phrase — no cron syntax. Parsing is case-insensitive. Supported forms:

| Pattern | Examples |
|---------|----------|
| Fixed interval | `every 15 minutes`, `every 30s`, `every 2 hours`, `every 3 days`, `every week` |
| Daily with a time | `every day at 09:00` |
| Weekday / weekend sets | `weekdays at 9:00 AM`, `weekends at 18:00` |
| A single weekday | `every monday at 9am` |
| Monthly ordinal weekday | `3rd wednesday monthly at 14:00`, `last friday of the month` |

**Units** (any spelling): `second`/`sec`/`s`, `minute`/`min`/`m`, `hour`/`hr`/`h`, `day`/`d`,
`week`/`w`.

**Ordinals:** `1st`–`5th`, `first`–`fifth`, or `last`. The monthly clause can be written as
`monthly`, `of the month`, `of each month`, or `of every month`.

**Time-of-day** accepts: `14:00`, `9:00`, `9:00 AM`, `9am`, or a bare hour like `9` (= 09:00).
Hours are 0–23, minutes 0–59.

> ⚠️ **Sub-daily intervals can't take an `at` time.** Seconds/minutes/hours fire on a rolling
> interval, so `every 15 minutes at 09:00` is **rejected**. The `at <time>` clause is only valid
> for daily-or-coarser schedules (`every day`, `weekdays`, `every monday`, monthly ordinals).

As you type a valid Schedule, the **Preview** row fills in with a plain-English summary plus the
next few run times — a quick way to confirm your phrase parsed the way you meant. A **Examples**
button next to the field opens the full list of supported phrasings.

### Start at *(sub-daily intervals only)*

By default a fixed interval like `every 15 minutes` is anchored to the moment you create the task,
so it might fire at an awkward phase (6:07, 6:22, 6:37 …). To align the cycle, set a **Start at**
time — a separate field that appears only when the Schedule is a sub-daily interval. With
`every 15 minutes` and a Start at of `09:00`, runs fall on `:00 / :15 / :30 / :45`.

Equivalently, you can type the anchor directly in the Schedule using a `starting at` (or `from`)
clause — the GUI and CLI both understand it:

| Phrase | Effect |
|--------|--------|
| `every 15 minutes starting at 09:00` | aligns to `:00/:15/:30/:45` |
| `every 30 minutes from 9am` | aligns to `:00/:30` relative to 09:00 |
| `every 2 hours starting at 08:00` | fires at 08:00, 10:00, 12:00 … |

The anchor is interpreted in the task's **Timezone**. It applies only to sub-daily intervals;
`every day starting at 09:00` is rejected (use `every day at 09:00`).

## One-off date / time *(One-off mode)*

Pick the **Date** (`2026-08-04`) and **Time** (`09:00`) in two fields — no hand-typed RFC 3339
required. The instant is interpreted in the task's **Timezone**, and a line under the fields
echoes the resolved run time so you can confirm it.

- Must be in the **future**, or Save stays disabled.
- The backend still stores the moment as a UTC instant, exactly as before.

## Advanced Settings

The **Overlap** and **Catch-up** controls live in a collapsible **Advanced Settings** section that
starts closed. They are shown with human-readable labels; the stored policy values (used by the
CLI and API) are unchanged.

### Overlap

What to do when a task is still running at the moment its next run would start:

- **Queue one run** (`queue_one`, *default*) — queue exactly one pending run; drop any further
  triggers until the current run finishes. A warning is logged and surfaced as a GUI alert.
- **Skip this run** (`skip`) — skip the new trigger entirely; do nothing until the next scheduled
  time.
- **Allow concurrent runs** (`allow_concurrent`) — let multiple runs of the same task execute at
  the same time.

### Catch-up

What to do after downtime (the daemon was stopped) when one or more scheduled runs were missed:

- **Run once to catch up** (`one`, *default*) — run once to catch up, then resume the normal
  schedule.
- **Skip missed runs** (`none`) — skip all missed runs and resume the normal schedule.

---

## A known-good example

A "heartbeat" task you can watch succeed within a couple of minutes:

| Field | Value |
|-------|-------|
| Name | `heartbeat` |
| Command | `cmd` |
| Arguments | `/c` (line 1) · `echo %DATE% %TIME% >> C:\Users\you\gosched-test.txt` (line 2) |
| Timezone | `Local` |
| Mode | `Recurring` |
| Schedule | `every 1 minute` |
| Overlap *(Advanced)* | Queue one run |
| Catch-up *(Advanced)* | Run once to catch up |

After saving, a new timestamp line should appear in the file about once a minute.
