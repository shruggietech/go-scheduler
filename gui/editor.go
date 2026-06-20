package gui

import (
	"context"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"

	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/domain"
)

// showTaskEditor opens the guided create/edit dialog. A live plain-language
// preview of the schedule is shown as the user types (FR-006). existing is nil
// for a new task.
func (a *App) showTaskEditor(existing *domain.Task) {
	name := widget.NewEntry()
	command := widget.NewEntry()
	args := widget.NewMultiLineEntry()
	args.SetPlaceHolder("one argument per line")
	tz := widget.NewEntry()
	tz.SetText("Local")
	mode := widget.NewSelect([]string{"Recurring", "One-off"}, nil)
	schedule := widget.NewEntry()
	schedule.SetPlaceHolder(`e.g. "every 15 minutes" or "3rd wednesday monthly at 14:00"`)
	at := widget.NewEntry()
	at.SetPlaceHolder("2026-08-04T09:00:00Z")
	overlap := widget.NewSelect([]string{"queue_one", "skip", "allow_concurrent"}, nil)
	overlap.SetSelected("queue_one")
	catchup := widget.NewSelect([]string{"one", "none"}, nil)
	catchup.SetSelected("one")

	preview := widget.NewLabel("")
	preview.Wrapping = fyne.TextWrapWord

	updatePreview := func() {
		if mode.Selected != "Recurring" || strings.TrimSpace(schedule.Text) == "" {
			preview.SetText("")
			return
		}
		go func() {
			ctx, cancel := a.bgCtx()
			defer cancel()
			resp, err := a.backend.Preview(ctx, server.PreviewRequest{Schedule: schedule.Text, Timezone: tzOrLocal(tz.Text)})
			fyne.Do(func() {
				if err != nil {
					preview.SetText("⚠ " + err.Error())
					return
				}
				txt := resp.HumanSummary
				for _, r := range resp.NextRuns {
					txt += "\n  • " + fmtTime(r)
				}
				preview.SetText(txt)
			})
		}()
	}
	schedule.OnChanged = func(string) { updatePreview() }
	mode.OnChanged = func(string) { updatePreview() }
	mode.SetSelected("Recurring")

	if existing != nil {
		name.SetText(existing.Name)
		command.SetText(existing.Command)
		args.SetText(strings.Join(existing.Args, "\n"))
		if existing.Timezone != "" {
			tz.SetText(existing.Timezone)
		}
		overlap.SetSelected(string(existing.OverlapPolicy))
		catchup.SetSelected(string(existing.CatchupPolicy))
	}

	items := []*widget.FormItem{
		widget.NewFormItem("Name", name),
		widget.NewFormItem("Command", command),
		widget.NewFormItem("Arguments", args),
		widget.NewFormItem("Timezone", tz),
		widget.NewFormItem("Mode", mode),
		widget.NewFormItem("Schedule", schedule),
		widget.NewFormItem("One-off time", at),
		widget.NewFormItem("Preview", preview),
		widget.NewFormItem("Overlap", overlap),
		widget.NewFormItem("Catch-up", catchup),
	}

	title := "New Task"
	if existing != nil {
		title = "Edit Task"
	}

	d := dialog.NewForm(title, "Save", "Cancel", items, func(ok bool) {
		if !ok {
			return
		}
		a.submitTask(existing, taskForm{
			name: name.Text, command: command.Text, args: splitArgs(args.Text),
			tz: tzOrLocal(tz.Text), mode: mode.Selected, schedule: schedule.Text,
			at: at.Text, overlap: overlap.Selected, catchup: catchup.Selected,
		})
	}, a.win)
	d.Resize(fyne.NewSize(580, 620))
	d.Show()
}

type taskForm struct {
	name, command, tz, mode, schedule, at, overlap, catchup string
	args                                                    []string
}

func (a *App) submitTask(existing *domain.Task, f taskForm) {
	var atPtr *time.Time
	if f.mode == "One-off" {
		ts, err := time.Parse(time.RFC3339, strings.TrimSpace(f.at))
		if err != nil {
			a.showError(errInvalidOneOff)
			return
		}
		atPtr = &ts
	}

	a.run(func(ctx context.Context) error {
		if existing == nil {
			req := server.TaskCreateRequest{
				Name: f.name, Command: f.command, Args: f.args, Timezone: f.tz,
				OverlapPolicy: f.overlap, CatchupPolicy: f.catchup,
			}
			if atPtr != nil {
				req.At = atPtr
			} else {
				req.Schedule = f.schedule
			}
			_, err := a.backend.CreateTask(ctx, req)
			return err
		}
		req := server.TaskUpdateRequest{
			Name: f.name, Command: f.command, Args: f.args, Timezone: f.tz,
			OverlapPolicy: f.overlap, CatchupPolicy: f.catchup,
		}
		if atPtr != nil {
			req.At = atPtr
		} else {
			req.Schedule = f.schedule
		}
		_, err := a.backend.UpdateTask(ctx, existing.ID, req)
		return err
	})
}

func splitArgs(s string) []string {
	var out []string
	for _, line := range strings.Split(s, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			out = append(out, t)
		}
	}
	return out
}

func tzOrLocal(s string) string {
	if strings.TrimSpace(s) == "" {
		return "Local"
	}
	return s
}
