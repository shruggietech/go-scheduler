// Package gui implements the go-scheduler desktop GUI with Fyne. Its widget
// construction is cgo-free (Fyne's headless test driver renders without OpenGL),
// so the UI is unit-tested here; only the real windowed application entry point
// (cmd/gosched-gui) imports the GL driver and requires cgo.
//
// The GUI talks to the daemon exclusively through the Backend interface (the API
// client implements it), so it operates on the same state as the CLI and is
// fully testable with a fake backend.
package gui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/shruggietech/go-scheduler/gui/viewmodel"
	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/events"
)

// Backend is everything the GUI needs from the daemon. The API client satisfies
// it; tests inject a fake.
type Backend interface {
	viewmodel.API // ListTasks, ListGroups, ListAlerts

	CreateTask(ctx context.Context, req server.TaskCreateRequest) (server.TaskResponse, error)
	UpdateTask(ctx context.Context, id string, req server.TaskUpdateRequest) (server.TaskResponse, error)
	DeleteTask(ctx context.Context, id string) error
	SetTaskEnabled(ctx context.Context, id string, enabled bool) error
	RunNow(ctx context.Context, id string) error
	Preview(ctx context.Context, req server.PreviewRequest) (server.PreviewResponse, error)

	CreateGroup(ctx context.Context, req server.GroupCreateRequest) (domain.Group, error)
	SetGroupEnabled(ctx context.Context, id string, enabled bool) error
	DeleteGroup(ctx context.Context, id string) error

	CreateTrigger(ctx context.Context, req server.TriggerCreateRequest) (domain.Trigger, error)
	ListTriggers(ctx context.Context) ([]domain.Trigger, error)
	DeleteTrigger(ctx context.Context, id string) error

	AckAlert(ctx context.Context, id string) error
	GetCalendar(ctx context.Context, from, to time.Time) (server.CalendarResponse, error)
	StreamEvents(ctx context.Context, onEvent func(events.Event)) error
}

// App is the GUI application.
type App struct {
	fyne    fyne.App
	win     fyne.Window
	backend Backend
	model   *viewmodel.Model

	tabs       *container.AppTabs
	alertsTab  *container.TabItem
	refreshers []func()
}

// NewUI builds the GUI against fyneApp (created by the caller with the GL driver)
// and backend. It constructs the window content but does not show it.
func NewUI(fyneApp fyne.App, backend Backend) *App {
	a := &App{
		fyne:    fyneApp,
		backend: backend,
		model:   viewmodel.New(backend),
	}
	a.win = fyneApp.NewWindow("go-scheduler")
	a.win.Resize(fyne.NewSize(960, 640))
	a.win.SetContent(a.buildRoot())
	a.model.OnChange = func() { fyne.Do(a.onModelChange) }
	return a
}

// buildRoot assembles the tabbed layout.
func (a *App) buildRoot() fyne.CanvasObject {
	a.tabs = container.NewAppTabs(
		container.NewTabItem("Tasks", a.buildTasksTab()),
		container.NewTabItem("Schedule", a.buildScheduleTab()),
		container.NewTabItem("Groups", a.buildGroupsTab()),
		container.NewTabItem("Triggers", a.buildTriggersTab()),
	)
	a.alertsTab = container.NewTabItem("Alerts", a.buildAlertsTab())
	a.tabs.Append(a.alertsTab)
	a.tabs.SetTabLocation(container.TabLocationLeading)
	return a.tabs
}

// Run shows the window, kicks off the first data load and the live event stream,
// and blocks until the window is closed.
func (a *App) Run() {
	a.refreshAll()
	go a.streamEvents()
	a.win.ShowAndRun()
}

// refreshAll reloads model state and every tab's local view.
func (a *App) refreshAll() {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := a.model.Refresh(ctx); err != nil {
			fyne.Do(func() { a.showError(err) })
		}
		for _, r := range a.refreshers {
			rr := r
			fyne.Do(rr)
		}
	}()
}

// streamEvents consumes the SSE stream and folds events into the model,
// reconnecting after a short delay if the stream drops.
func (a *App) streamEvents() {
	for {
		ctx, cancel := context.WithCancel(context.Background())
		err := a.backend.StreamEvents(ctx, func(e events.Event) {
			a.model.ApplyEvent(e)
		})
		cancel()
		if err == nil {
			return
		}
		time.Sleep(2 * time.Second) // reconnect backoff
	}
}

// onModelChange refreshes the alert badge and alert list when state changes.
func (a *App) onModelChange() {
	a.updateAlertBadge()
	for _, r := range a.refreshers {
		r()
	}
}

func (a *App) updateAlertBadge() {
	if a.alertsTab == nil {
		return
	}
	n := a.model.UnacknowledgedAlerts()
	if n > 0 {
		a.alertsTab.Text = "Alerts (" + itoa(n) + ")"
	} else {
		a.alertsTab.Text = "Alerts"
	}
	if a.tabs != nil {
		a.tabs.Refresh()
	}
}

func (a *App) registerRefresher(f func()) { a.refreshers = append(a.refreshers, f) }

// bgCtx returns a short-lived context for a backend call.
func (a *App) bgCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// run executes a backend mutation in the background and refreshes on success.
func (a *App) run(fn func(ctx context.Context) error) {
	go func() {
		ctx, cancel := a.bgCtx()
		defer cancel()
		if err := fn(ctx); err != nil {
			fyne.Do(func() { a.showError(err) })
			return
		}
		a.refreshAll()
	}()
}
