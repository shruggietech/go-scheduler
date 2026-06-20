package gui

import (
	"strings"

	"github.com/shruggietech/go-scheduler/internal/domain"
)

// This file holds presentation-only data for the task editor: the human-readable
// labels shown for overlap/catch-up policies (the stored wire values are
// unchanged), the curated timezone suggestions, and the command-line preview
// builder. Keeping them here keeps editor.go focused on widget wiring.

// overlapChoice / catchupChoice pair a friendly label with its stored value.
type policyChoice[T ~string] struct {
	label string
	value T
}

var overlapChoices = []policyChoice[domain.OverlapPolicy]{
	{"Queue one run", domain.OverlapQueueOne},
	{"Skip this run", domain.OverlapSkip},
	{"Allow concurrent runs", domain.OverlapAllowConcurrent},
}

var catchupChoices = []policyChoice[domain.CatchupPolicy]{
	{"Run once to catch up", domain.CatchupOne},
	{"Skip missed runs", domain.CatchupNone},
}

// overlapLabels / catchupLabels are the ordered display strings for the selects.
func overlapLabels() []string { return labelsOf(overlapChoices) }
func catchupLabels() []string { return labelsOf(catchupChoices) }

func labelsOf[T ~string](cs []policyChoice[T]) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.label
	}
	return out
}

// overlapValue maps a display label back to its stored value, falling back to the
// default (first choice) for any unknown label so the UI never crashes on legacy
// or empty input.
func overlapValue(label string) domain.OverlapPolicy {
	for _, c := range overlapChoices {
		if c.label == label {
			return c.value
		}
	}
	return overlapChoices[0].value
}

func catchupValue(label string) domain.CatchupPolicy {
	for _, c := range catchupChoices {
		if c.label == label {
			return c.value
		}
	}
	return catchupChoices[0].value
}

// overlapLabel maps a stored value back to its display label (default label for
// unknown values).
func overlapLabel(v domain.OverlapPolicy) string {
	for _, c := range overlapChoices {
		if c.value == v {
			return c.label
		}
	}
	return overlapChoices[0].label
}

func catchupLabel(v domain.CatchupPolicy) string {
	for _, c := range catchupChoices {
		if c.value == v {
			return c.label
		}
	}
	return catchupChoices[0].label
}

// commonZones seeds the timezone SelectEntry. It is a curated, ordered subset of
// the IANA database for quick selection; any other valid IANA name typed by the
// user is still accepted (validated via timezone.Resolve).
var commonZones = []string{
	"Local", "UTC",
	"America/New_York", "America/Chicago", "America/Denver", "America/Los_Angeles",
	"America/Sao_Paulo",
	"Europe/London", "Europe/Paris", "Europe/Berlin", "Europe/Moscow",
	"Asia/Kolkata", "Asia/Shanghai", "Asia/Tokyo",
	"Australia/Sydney", "Pacific/Auckland",
}

// commandLinePreview renders the resolved command line for display only: the
// command followed by each argument, with whitespace-bearing tokens quoted for
// readability. Execution still receives the raw argument slice — this never
// re-parses or shell-splits.
func commandLinePreview(command string, args []string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, quoteForDisplay(command))
	for _, a := range args {
		parts = append(parts, quoteForDisplay(a))
	}
	return strings.Join(parts, " ")
}

func quoteForDisplay(s string) string {
	if s == "" || strings.ContainsAny(s, " \t") {
		return `"` + s + `"`
	}
	return s
}
