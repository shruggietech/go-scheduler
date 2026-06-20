package gui

import (
	"context"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

// buildAlertsTab shows alerts (overlap, failures, missed runs) and lets the user
// acknowledge them (FR-024). The list updates live from the SSE stream.
func (a *App) buildAlertsTab() fyne.CanvasObject {
	var alerts []domain.Alert
	selected := -1

	list := widget.NewList(
		func() int { return len(alerts) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			al := alerts[i]
			ackMark := " "
			if al.Acknowledged {
				ackMark = "✓"
			}
			o.(*widget.Label).SetText(fmt.Sprintf("[%s] %s  %s  — %s", ackMark, al.Severity, al.Kind, al.Message))
		},
	)
	list.OnSelected = func(id widget.ListItemID) { selected = id }
	list.OnUnselected = func(widget.ListItemID) { selected = -1 }

	refresh := func() {
		alerts = a.model.Snapshot().Alerts
		list.Refresh()
	}
	a.registerRefresher(refresh)

	ackBtn := widget.NewButtonWithIcon("Acknowledge", theme.ConfirmIcon(), func() {
		if selected >= 0 && selected < len(alerts) {
			id := alerts[selected].ID
			a.run(func(ctx context.Context) error { return a.backend.AckAlert(ctx, id) })
		}
	})
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), func() { a.refreshAll() })

	toolbar := container.NewHBox(ackBtn, refreshBtn)
	return container.NewBorder(toolbar, nil, nil, nil, list)
}
