package gui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

func (a *App) buildTasksTab() fyne.CanvasObject {
	var tasks []domain.Task
	selected := -1

	list := widget.NewList(
		func() int { return len(tasks) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			t := tasks[i]
			o.(*widget.Label).SetText(fmt.Sprintf("%s   [%s]   %s   %s",
				t.Name, t.State, boolStr(t.Enabled, "enabled", "disabled"), t.Timezone))
		},
	)
	list.OnSelected = func(id widget.ListItemID) { selected = id }
	list.OnUnselected = func(widget.ListItemID) { selected = -1 }

	refresh := func() {
		tasks = a.model.Snapshot().Tasks
		list.Refresh()
	}
	a.registerRefresher(refresh)

	cur := func() (domain.Task, bool) {
		if selected < 0 || selected >= len(tasks) {
			return domain.Task{}, false
		}
		return tasks[selected], true
	}
	withSel := func(fn func(t domain.Task)) {
		if t, ok := cur(); ok {
			fn(t)
		} else {
			dialog.ShowInformation("No selection", "Select a task first.", a.win)
		}
	}

	newBtn := widget.NewButtonWithIcon("New", theme.ContentAddIcon(), func() { a.showTaskEditor(nil) })
	editBtn := widget.NewButtonWithIcon("Edit", theme.DocumentCreateIcon(), func() {
		withSel(func(t domain.Task) { a.showTaskEditor(&t) })
	})
	runBtn := widget.NewButtonWithIcon("Run now", theme.MediaPlayIcon(), func() {
		withSel(func(t domain.Task) { a.run(func(ctx context.Context) error { return a.backend.RunNow(ctx, t.ID) }) })
	})
	toggleBtn := widget.NewButton("Enable/Disable", func() {
		withSel(func(t domain.Task) {
			a.run(func(ctx context.Context) error { return a.backend.SetTaskEnabled(ctx, t.ID, !t.Enabled) })
		})
	})
	delBtn := widget.NewButtonWithIcon("Delete", theme.DeleteIcon(), func() {
		withSel(func(t domain.Task) {
			dialog.ShowConfirm("Delete task", "Delete "+t.Name+"?", func(ok bool) {
				if ok {
					a.run(func(ctx context.Context) error { return a.backend.DeleteTask(ctx, t.ID) })
				}
			}, a.win)
		})
	})
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() { a.refreshAll() })

	toolbar := container.NewHBox(newBtn, editBtn, runBtn, toggleBtn, delBtn, refreshBtn)
	return container.NewBorder(toolbar, nil, nil, nil, list)
}
