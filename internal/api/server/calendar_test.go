package server

import (
	"bufio"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/go-scheduler/internal/config"
	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/events"
	"github.com/shruggietech/go-scheduler/internal/store"
)

func TestCalendar_PastAndScheduled(t *testing.T) {
	s := newTestServer(t)

	// Create a daily task; a recorded past run sits inside the window.
	create := doJSON(t, s, http.MethodPost, "/v1/tasks", TaskCreateRequest{
		Name: "daily", Command: "/bin/true", Schedule: "every day at 09:00", Timezone: "UTC",
	})
	var resp TaskResponse
	_ = json.Unmarshal(create.Body.Bytes(), &resp)

	past := time.Now().UTC().Add(-2 * time.Hour)
	_ = s.store.CreateRun(&domain.Run{TaskID: resp.Task.ID, ScheduledFor: past, Outcome: domain.OutcomeSuccess, Trigger: domain.TriggerSchedule})

	from := time.Now().UTC().Add(-24 * time.Hour).Format(time.RFC3339)
	to := time.Now().UTC().Add(24 * time.Hour).Format(time.RFC3339)
	rec := doJSON(t, s, http.MethodGet, "/v1/calendar?from="+from+"&to="+to, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var cal CalendarResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &cal); err != nil {
		t.Fatal(err)
	}
	var hasPast, hasScheduled bool
	for _, o := range cal.Occurrences {
		if o.Kind == "past" && o.Outcome == domain.OutcomeSuccess {
			hasPast = true
		}
		if o.Kind == "scheduled" {
			hasScheduled = true
		}
	}
	if !hasPast {
		t.Fatal("expected the past run to appear in the calendar")
	}
	if !hasScheduled {
		t.Fatal("expected at least one future scheduled occurrence")
	}
}

func TestCalendar_BadRange(t *testing.T) {
	s := newTestServer(t)
	from := time.Now().UTC().Format(time.RFC3339)
	to := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	rec := doJSON(t, s, http.MethodGet, "/v1/calendar?from="+from+"&to="+to, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for reversed range, got %d", rec.Code)
	}
}

func TestEvents_StreamsAlert(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	broker := events.NewBroker()
	s := New(st, nil, broker, config.NewLogger(config.Default(), discard{}))

	srv := httptest.NewServer(s.Handler())
	defer srv.Close()

	// Open the SSE stream.
	req, _ := http.NewRequest(http.MethodGet, srv.URL+"/v1/events", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if ct := resp.Header.Get("Content-Type"); ct != "text/event-stream" {
		t.Fatalf("content-type = %q", ct)
	}

	reader := bufio.NewReader(resp.Body)
	// Read the initial ": connected" comment line.
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatal(err)
	}

	// Publish an alert; it should arrive on the stream.
	go func() {
		time.Sleep(50 * time.Millisecond)
		broker.PublishAlert(domain.Alert{ID: "a1", Kind: domain.AlertRunFailed, Severity: domain.SeverityError, Message: "boom"})
	}()

	deadline := time.Now().Add(2 * time.Second)
	var gotData string
	for time.Now().Before(deadline) {
		line, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		if strings.HasPrefix(line, "data: ") {
			gotData = strings.TrimSpace(strings.TrimPrefix(line, "data: "))
			break
		}
	}
	if !strings.Contains(gotData, "\"kind\":\"alert\"") || !strings.Contains(gotData, "boom") {
		t.Fatalf("did not receive the alert event, got: %q", gotData)
	}
}
