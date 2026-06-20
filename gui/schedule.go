package gui

import (
	"context"
	"fmt"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/shruggietech/go-scheduler/internal/api/server"
)

// buildScheduleTab shows an agenda/timeline of past and upcoming runs over a
// window (the calendar/schedule view, FR-023).
func (a *App) buildScheduleTab() fyne.CanvasObject {
	var occ []server.Occurrence
	days := 7

	list := widget.NewList(
		func() int { return len(occ) },
		func() fyne.CanvasObject { return widget.NewLabel("template") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			e := occ[i]
			marker := "▷" // scheduled (future)
			if e.Kind == "past" {
				marker = "✓"
				if e.Outcome != "success" && e.Outcome != "" {
					marker = "✗"
				}
			}
			o.(*widget.Label).SetText(fmt.Sprintf("%s  %s   %s", marker, fmtTime(e.Time), e.TaskName))
		},
	)

	load := func() {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			from := time.Now().Add(-24 * time.Hour)
			to := time.Now().Add(time.Duration(days) * 24 * time.Hour)
			resp, err := a.backend.GetCalendar(ctx, from, to)
			fyne.Do(func() {
				if err != nil {
					a.showError(err)
					return
				}
				occ = sortByTime(resp.Occurrences)
				list.Refresh()
			})
		}()
	}
	a.registerRefresher(load)

	rangeSel := widget.NewSelect([]string{"1 day", "7 days", "30 days"}, func(s string) {
		switch s {
		case "1 day":
			days = 1
		case "30 days":
			days = 30
		default:
			days = 7
		}
		load()
	})
	rangeSel.SetSelected("7 days")
	refreshBtn := widget.NewButtonWithIcon("Refresh", theme.ViewRefreshIcon(), load)

	toolbar := container.NewHBox(widget.NewLabel("Window:"), rangeSel, refreshBtn)
	return container.NewBorder(toolbar, nil, nil, nil, list)
}

// sortByTime orders occurrences ascending by time (simple insertion-free sort).
func sortByTime(in []server.Occurrence) []server.Occurrence {
	out := make([]server.Occurrence, len(in))
	copy(out, in)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].Time.Before(out[j-1].Time); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
}
