package gui

import (
	"context"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"

	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/events"
)

// fakeBackend implements Backend with in-memory data for headless UI tests.
type fakeBackend struct {
	tasks    []domain.Task
	groups   []domain.Group
	alerts   []domain.Alert
	triggers []domain.Trigger
	created  int
}

func (f *fakeBackend) ListTasks(context.Context, string, string) ([]domain.Task, error) {
	return f.tasks, nil
}
func (f *fakeBackend) ListGroups(context.Context) ([]domain.Group, error) { return f.groups, nil }
func (f *fakeBackend) ListAlerts(context.Context, bool) ([]domain.Alert, error) {
	return f.alerts, nil
}
func (f *fakeBackend) CreateTask(context.Context, server.TaskCreateRequest) (server.TaskResponse, error) {
	f.created++
	return server.TaskResponse{}, nil
}
func (f *fakeBackend) UpdateTask(context.Context, string, server.TaskUpdateRequest) (server.TaskResponse, error) {
	return server.TaskResponse{}, nil
}
func (f *fakeBackend) DeleteTask(context.Context, string) error           { return nil }
func (f *fakeBackend) SetTaskEnabled(context.Context, string, bool) error { return nil }
func (f *fakeBackend) RunNow(context.Context, string) error               { return nil }
func (f *fakeBackend) Preview(context.Context, server.PreviewRequest) (server.PreviewResponse, error) {
	return server.PreviewResponse{HumanSummary: "Every day at 09:00"}, nil
}
func (f *fakeBackend) CreateGroup(context.Context, server.GroupCreateRequest) (domain.Group, error) {
	return domain.Group{}, nil
}
func (f *fakeBackend) SetGroupEnabled(context.Context, string, bool) error { return nil }
func (f *fakeBackend) DeleteGroup(context.Context, string) error           { return nil }
func (f *fakeBackend) CreateTrigger(context.Context, server.TriggerCreateRequest) (domain.Trigger, error) {
	return domain.Trigger{}, nil
}
func (f *fakeBackend) ListTriggers(context.Context) ([]domain.Trigger, error) { return f.triggers, nil }
func (f *fakeBackend) DeleteTrigger(context.Context, string) error            { return nil }
func (f *fakeBackend) AckAlert(context.Context, string) error                 { return nil }
func (f *fakeBackend) GetCalendar(context.Context, time.Time, time.Time) (server.CalendarResponse, error) {
	return server.CalendarResponse{}, nil
}
func (f *fakeBackend) StreamEvents(ctx context.Context, _ func(events.Event)) error {
	<-ctx.Done()
	return ctx.Err()
}

func TestUI_BuildsAllTabs(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	ui := NewUI(a, &fakeBackend{
		tasks:  []domain.Task{{ID: "t1", Name: "nightly", State: domain.TaskActive, Enabled: true, Timezone: "UTC"}},
		groups: []domain.Group{{ID: "g1", Name: "Backups", Enabled: true}},
		alerts: []domain.Alert{{ID: "a1", Kind: domain.AlertRunFailed, Message: "boom"}},
	})

	want := []string{"Tasks", "Schedule", "Groups", "Triggers", "Alerts"}
	if len(ui.tabs.Items) != len(want) {
		t.Fatalf("want %d tabs, got %d", len(want), len(ui.tabs.Items))
	}
	for i, w := range want {
		if ui.tabs.Items[i].Text != w {
			t.Fatalf("tab %d = %q, want %q", i, ui.tabs.Items[i].Text, w)
		}
	}
}

func TestUI_TaskEditorBuilds(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	ui := NewUI(a, &fakeBackend{})
	// Opening the editor must not panic and the window keeps a canvas.
	ui.showTaskEditor(nil)
	if ui.win.Canvas() == nil {
		t.Fatal("window canvas missing")
	}
}

func TestUI_AlertBadgeReflectsUnacked(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	ui := NewUI(a, &fakeBackend{})
	ui.model.ApplyEvent(events.Event{Kind: events.KindAlert, Alert: &domain.Alert{ID: "x", Acknowledged: false}})
	ui.updateAlertBadge()
	if ui.alertsTab.Text != "Alerts (1)" {
		t.Fatalf("alert badge = %q, want Alerts (1)", ui.alertsTab.Text)
	}
}
