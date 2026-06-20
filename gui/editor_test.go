package gui

import (
	"strings"
	"testing"
	"time"

	"github.com/shruggietech/go-scheduler/internal/domain"
	"github.com/shruggietech/go-scheduler/internal/timezone"
)

// newTestEditor builds a wired editor (ready == true) against a fake backend.
func newTestEditor(t *testing.T, existing *domain.Task) (*taskEditor, *fakeBackend) {
	t.Helper()
	fb := &fakeBackend{}
	ui := NewUI(testApp, fb)
	e := newTaskEditor(ui, existing)
	e.previewSync = true // deterministic: no cross-test goroutines/fyne.Do
	e.build()            // wires layout, sets ready
	return e, fb
}

func whenLabels(e *taskEditor) []string {
	out := make([]string, len(e.whenForm.Items))
	for i, it := range e.whenForm.Items {
		out[i] = it.Text
	}
	return out
}

func hasLabel(labels []string, want string) bool {
	for _, l := range labels {
		if l == want {
			return true
		}
	}
	return false
}

// --- US1: mode-driven visibility -----------------------------------------

func TestEditor_ModeVisibility(t *testing.T) {
	e, _ := newTestEditor(t, nil)

	labels := whenLabels(e)
	if !hasLabel(labels, "Schedule *") {
		t.Fatalf("Recurring mode missing Schedule row: %v", labels)
	}
	if hasLabel(labels, "Date *") || hasLabel(labels, "Time *") {
		t.Fatalf("Recurring mode should not show one-off rows: %v", labels)
	}

	e.mode.SetSelected(modeOneOff)
	labels = whenLabels(e)
	if !hasLabel(labels, "Date *") || !hasLabel(labels, "Time *") {
		t.Fatalf("One-off mode missing date/time rows: %v", labels)
	}
	if hasLabel(labels, "Schedule *") {
		t.Fatalf("One-off mode should not show Schedule row: %v", labels)
	}
}

func TestEditor_ModeTogglePreservesValues(t *testing.T) {
	e, _ := newTestEditor(t, nil)
	e.schedule.SetText("every 15 minutes")
	e.oneOffDate.SetText("2099-01-02")

	e.mode.SetSelected(modeOneOff)
	e.mode.SetSelected(modeRecurring)

	if e.schedule.Text != "every 15 minutes" {
		t.Fatalf("schedule lost on toggle: %q", e.schedule.Text)
	}
	if e.oneOffDate.Text != "2099-01-02" {
		t.Fatalf("one-off date lost on toggle: %q", e.oneOffDate.Text)
	}
}

// --- US2: validation gating ----------------------------------------------

func TestEditor_SaveGating(t *testing.T) {
	e, _ := newTestEditor(t, nil)

	if !e.save.Disabled() {
		t.Fatal("Save should start disabled (empty form)")
	}

	e.name.SetText("nightly")
	e.command.SetText("cmd")
	if !e.save.Disabled() {
		t.Fatal("Save should stay disabled without a schedule")
	}

	e.schedule.SetText("every 15 minutes")
	if e.save.Disabled() {
		t.Fatal("Save should be enabled with name+command+valid schedule")
	}

	e.schedule.SetText("nonsense gibberish")
	if !e.save.Disabled() {
		t.Fatal("Save should disable for an unparseable schedule")
	}
}

func TestEditor_SaveGating_OneOff(t *testing.T) {
	e, _ := newTestEditor(t, nil)
	e.name.SetText("once")
	e.command.SetText("cmd")
	e.mode.SetSelected(modeOneOff)

	e.oneOffDate.SetText("2000-01-01")
	e.oneOffTime.SetText("09:00")
	if !e.save.Disabled() {
		t.Fatal("Save should disable for a past one-off time")
	}

	e.oneOffDate.SetText("2099-01-01")
	if e.save.Disabled() {
		t.Fatal("Save should enable for a future one-off time")
	}
}

func TestEditor_SaveGating_BadTimezone(t *testing.T) {
	e, _ := newTestEditor(t, nil)
	e.name.SetText("x")
	e.command.SetText("cmd")
	e.schedule.SetText("every 15 minutes")
	if e.save.Disabled() {
		t.Fatal("precondition: Save enabled")
	}
	e.tz.SetText("Mars/Phobos")
	if !e.save.Disabled() {
		t.Fatal("Save should disable for an unknown timezone")
	}
}

// --- US3: combined preview -----------------------------------------------

func TestEditor_CommandPreview(t *testing.T) {
	e, _ := newTestEditor(t, nil)
	e.command.SetText("cmd")
	e.args.SetText("/c\necho hello world")
	if got := e.cmdPreview.Text; !strings.Contains(got, `cmd /c "echo hello world"`) {
		t.Fatalf("cmd preview = %q", got)
	}
}

func TestEditor_EmptyScheduleShowsGuidance(t *testing.T) {
	e, _ := newTestEditor(t, nil)
	if got := e.schedPreview.Text; !strings.Contains(strings.ToLower(got), "type a schedule") {
		t.Fatalf("empty schedule preview = %q, want guidance", got)
	}
}

// --- US4: interval anchor ------------------------------------------------

func TestEditor_StartAtVisibilityAndPhrase(t *testing.T) {
	e, _ := newTestEditor(t, nil)

	e.schedule.SetText("every 15 minutes")
	if !hasLabel(whenLabels(e), "Start at") {
		t.Fatalf("Start at should appear for sub-daily interval: %v", whenLabels(e))
	}

	e.schedule.SetText("every day at 09:00")
	if hasLabel(whenLabels(e), "Start at") {
		t.Fatalf("Start at should be hidden for daily schedule: %v", whenLabels(e))
	}

	e.schedule.SetText("every 15 minutes")
	e.startAt.SetText("09:00")
	if got := e.effectiveSchedule(); got != "every 15 minutes starting at 09:00" {
		t.Fatalf("effectiveSchedule = %q", got)
	}
	e.name.SetText("x")
	e.command.SetText("cmd")
	if got := e.buildForm().schedule; got != "every 15 minutes starting at 09:00" {
		t.Fatalf("submitted schedule = %q", got)
	}
}

// --- US5: timezone combo + one-off assembly ------------------------------

func TestEditor_TimezoneComboAndOneOffAssembly(t *testing.T) {
	e, _ := newTestEditor(t, nil)

	e.tz.SetText("UTC")
	if _, err := timezone.Resolve("UTC"); err != nil {
		t.Fatalf("UTC should resolve: %v", err)
	}
	e.mode.SetSelected(modeOneOff)
	e.oneOffDate.SetText("2099-08-04")
	e.oneOffTime.SetText("09:00")
	got, err := e.oneOffInstant()
	if err != nil {
		t.Fatalf("oneOffInstant: %v", err)
	}
	if got.UTC() != time.Date(2099, 8, 4, 9, 0, 0, 0, time.UTC) {
		t.Fatalf("assembled instant = %v", got.UTC())
	}
}

// --- US6: advanced labels submit correct wire values ---------------------

func TestEditor_AdvancedLabelsMapToWire(t *testing.T) {
	e, _ := newTestEditor(t, nil)
	e.name.SetText("x")
	e.command.SetText("cmd")
	e.schedule.SetText("every 15 minutes")
	e.overlap.SetSelected("Allow concurrent runs")
	e.catchup.SetSelected("Skip missed runs")

	f := e.buildForm()
	if f.overlap != string(domain.OverlapAllowConcurrent) {
		t.Fatalf("overlap wire = %q, want %q", f.overlap, domain.OverlapAllowConcurrent)
	}
	if f.catchup != string(domain.CatchupNone) {
		t.Fatalf("catchup wire = %q, want %q", f.catchup, domain.CatchupNone)
	}
}

func TestEditor_EditPrefillMapsPolicyLabels(t *testing.T) {
	task := &domain.Task{
		ID: "t1", Name: "nightly", Command: "cmd", Timezone: "UTC",
		OverlapPolicy: domain.OverlapSkip, CatchupPolicy: domain.CatchupNone,
	}
	e, _ := newTestEditor(t, task)
	if e.overlap.Selected != "Skip this run" {
		t.Fatalf("overlap label = %q, want 'Skip this run'", e.overlap.Selected)
	}
	if e.catchup.Selected != "Skip missed runs" {
		t.Fatalf("catchup label = %q", e.catchup.Selected)
	}
	// On edit, a blank schedule is allowed (keeps the existing one).
	if !e.valid() {
		t.Fatal("edit with name+command should be valid even with blank schedule")
	}
}
