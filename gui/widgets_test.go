package gui

import (
	"testing"

	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

func TestCursorButton_PointerCursorAndTap(t *testing.T) {
	tapped := false
	b := newCursorButton("Go", nil, widget.HighImportance, func() { tapped = true })

	if b.Cursor() != desktop.PointerCursor {
		t.Fatalf("Cursor() = %v, want PointerCursor", b.Cursor())
	}
	b.OnTapped()
	if !tapped {
		t.Fatal("tap handler not invoked")
	}
}
