// Package viewmodel holds the GUI's application state and the logic that mutates
// it in response to API data and live events. It is deliberately free of any UI
// (Fyne) dependency so the state logic is unit-testable without a display or a C
// toolchain; the Fyne layer renders this state and forwards user actions.
package viewmodel

import (
	"context"
	"sync"

	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/events"
)

// API is the subset of the API client the view-model needs (injectable for tests).
type API interface {
	ListTasks(ctx context.Context, group, state string) ([]domain.Task, error)
	ListGroups(ctx context.Context) ([]domain.Group, error)
	ListAlerts(ctx context.Context, unacked bool) ([]domain.Alert, error)
}

// State is a snapshot of what the GUI displays.
type State struct {
	Tasks      []domain.Task
	Groups     []domain.Group
	Alerts     []domain.Alert
	RecentRuns []domain.Run
}

const maxRecentRuns = 50

// Model owns the GUI state and refreshes it from the API.
type Model struct {
	api API
	mu  sync.RWMutex
	st  State
	// OnChange, if set, is invoked (off the lock) whenever state changes so the
	// UI can refresh.
	OnChange func()
}

// New creates a Model backed by api.
func New(api API) *Model { return &Model{api: api} }

// Snapshot returns a copy-safe view of the current state.
func (m *Model) Snapshot() State {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.st
}

// Refresh reloads tasks, groups, and alerts from the API.
func (m *Model) Refresh(ctx context.Context) error {
	tasks, err := m.api.ListTasks(ctx, "", "")
	if err != nil {
		return err
	}
	groups, err := m.api.ListGroups(ctx)
	if err != nil {
		return err
	}
	alerts, err := m.api.ListAlerts(ctx, false)
	if err != nil {
		return err
	}
	m.mu.Lock()
	m.st.Tasks = tasks
	m.st.Groups = groups
	m.st.Alerts = alerts
	m.mu.Unlock()
	m.notify()
	return nil
}

// ApplyEvent folds a live event into the state: new alerts are prepended
// (deduplicated by ID) and recent runs are tracked (most recent first, capped).
func (m *Model) ApplyEvent(e events.Event) {
	m.mu.Lock()
	switch e.Kind {
	case events.KindAlert:
		if e.Alert != nil && !containsAlert(m.st.Alerts, e.Alert.ID) {
			m.st.Alerts = append([]domain.Alert{*e.Alert}, m.st.Alerts...)
		}
	case events.KindRun:
		if e.Run != nil {
			runs := append([]domain.Run{*e.Run}, m.st.RecentRuns...)
			if len(runs) > maxRecentRuns {
				runs = runs[:maxRecentRuns]
			}
			m.st.RecentRuns = runs
		}
	}
	m.mu.Unlock()
	m.notify()
}

// UnacknowledgedAlerts returns the count of alerts not yet acknowledged.
func (m *Model) UnacknowledgedAlerts() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	n := 0
	for _, a := range m.st.Alerts {
		if !a.Acknowledged {
			n++
		}
	}
	return n
}

func (m *Model) notify() {
	if m.OnChange != nil {
		m.OnChange()
	}
}

func containsAlert(alerts []domain.Alert, id string) bool {
	for _, a := range alerts {
		if a.ID == id {
			return true
		}
	}
	return false
}
