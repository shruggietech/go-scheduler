package gui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/domain"
)

// buildTriggersTab lists event triggers and lets the user chain tasks on
// completion (FR-007; the GUI trigger config, T062).
func (a *App) buildTriggersTab() fyne.CanvasObject {
	var triggers []domain.Trigger
	taskName := map[string]string{}
	selected := -1

	list := widget.NewList(
		func() int { return len(triggers) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			tr := triggers[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%s → %s   (on %s, window %s)",
				name(taskName, tr.SourceTaskID), name(taskName, tr.TargetTaskID), tr.OnOutcome, tr.DedupWindow))
		},
	)
	list.OnSelected = func(id widget.ListItemID) { selected = id }
	list.OnUnselected = func(widget.ListItemID) { selected = -1 }

	load := func() {
		taskName = map[string]string{}
		for _, t := range a.model.Snapshot().Tasks {
			taskName[t.ID] = t.Name
		}
		go func() {
			ctx, cancel := a.bgCtx()
			defer cancel()
			tr, err := a.backend.ListTriggers(ctx)
			fyne.Do(func() {
				if err != nil {
					a.showError(err)
					return
				}
				triggers = tr
				list.Refresh()
			})
		}()
	}
	a.registerRefresher(load)

	addBtn := widget.NewButtonWithIcon("New Trigger", theme.ContentAddIcon(), func() { a.showTriggerEditor() })
	delBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		if selected >= 0 && selected < len(triggers) {
			id := triggers[selected].ID
			a.run(func(ctx context.Context) error { return a.backend.DeleteTrigger(ctx, id) })
		}
	})
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), load)

	toolbar := container.NewHBox(addBtn, delBtn, refreshBtn)
	return container.NewBorder(toolbar, nil, nil, nil, list)
}

func (a *App) showTriggerEditor() {
	tasks := a.model.Snapshot().Tasks
	if len(tasks) < 1 {
		dialog.ShowInformation("No tasks", "Create tasks before adding a trigger.", a.win)
		return
	}
	options := make([]string, len(tasks))
	idByLabel := map[string]string{}
	for i, t := range tasks {
		label := t.Name + " [" + t.ID[:8] + "]"
		options[i] = label
		idByLabel[label] = t.ID
	}
	source := widget.NewSelect(options, nil)
	target := widget.NewSelect(options, nil)
	on := widget.NewSelect([]string{"success", "failure", "any"}, nil)
	on.SetSelected("success")
	window := widget.NewEntry()
	window.SetPlaceHolder("e.g. 5m (optional)")

	items := []*widget.FormItem{
		widget.NewFormItem("When this task", source),
		widget.NewFormItem("completes with", on),
		widget.NewFormItem("run this task", target),
		widget.NewFormItem("Dedup window", window),
	}
	dialog.NewForm("New Trigger", "Create", "Cancel", items, func(ok bool) {
		if !ok || source.Selected == "" || target.Selected == "" {
			return
		}
		a.run(func(ctx context.Context) error {
			_, err := a.backend.CreateTrigger(ctx, server.TriggerCreateRequest{
				SourceTaskID: idByLabel[source.Selected], TargetTaskID: idByLabel[target.Selected],
				OnOutcome: on.Selected, DedupWindow: window.Text,
			})
			return err
		})
	}, a.win).Show()
}

func name(m map[string]string, id string) string {
	if n, ok := m[id]; ok {
		return n
	}
	if len(id) > 8 {
		return id[:8]
	}
	return id
}
