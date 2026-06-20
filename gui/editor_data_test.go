package gui

import (
	"testing"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

func TestPolicyLabelRoundTrip(t *testing.T) {
	for _, v := range []domain.OverlapPolicy{domain.OverlapQueueOne, domain.OverlapSkip, domain.OverlapAllowConcurrent} {
		if got := overlapValue(overlapLabel(v)); got != v {
			t.Fatalf("overlap round-trip: %q -> %q -> %q", v, overlapLabel(v), got)
		}
	}
	for _, v := range []domain.CatchupPolicy{domain.CatchupOne, domain.CatchupNone} {
		if got := catchupValue(catchupLabel(v)); got != v {
			t.Fatalf("catchup round-trip: %q -> %q -> %q", v, catchupLabel(v), got)
		}
	}
}

func TestPolicyLabelUnknownFallsBackToDefault(t *testing.T) {
	if got := overlapValue("not a real label"); got != domain.OverlapQueueOne {
		t.Fatalf("unknown overlap label = %q, want default %q", got, domain.OverlapQueueOne)
	}
	if got := catchupValue(""); got != domain.CatchupOne {
		t.Fatalf("unknown catchup label = %q, want default %q", got, domain.CatchupOne)
	}
	if got := overlapLabel(domain.OverlapPolicy("legacy")); got != overlapChoices[0].label {
		t.Fatalf("unknown overlap value label = %q, want default", got)
	}
}

func TestCommandLinePreview(t *testing.T) {
	tests := []struct {
		command string
		args    []string
		want    string
	}{
		{"", nil, ""},
		{"cmd", nil, "cmd"},
		{"cmd", []string{"/c", "echo hello world"}, `cmd /c "echo hello world"`},
		{"python", []string{"-m", "http.server"}, "python -m http.server"},
	}
	for _, tt := range tests {
		if got := commandLinePreview(tt.command, tt.args); got != tt.want {
			t.Fatalf("commandLinePreview(%q, %v) = %q, want %q", tt.command, tt.args, got, tt.want)
		}
	}
}
