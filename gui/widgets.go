package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

// cursorButton is a widget.Button that shows the pointer (hand) cursor on hover.
// Fyne's stock button keeps the default arrow cursor, giving no visual hint that
// it is clickable; implementing desktop.Cursorable opts into the link-style
// pointer so buttons read as interactive (FR-021).
type cursorButton struct {
	widget.Button
}

// newCursorButton builds a pointer-cursor button with the given label and tap
// handler. importance controls the visual emphasis (e.g. widget.HighImportance
// for the primary Save action).
func newCursorButton(label string, icon fyne.Resource, importance widget.Importance, tapped func()) *cursorButton {
	b := &cursorButton{}
	b.ExtendBaseWidget(b)
	b.Text = label
	b.Icon = icon
	b.Importance = importance
	b.OnTapped = tapped
	return b
}

// Cursor implements desktop.Cursorable.
func (b *cursorButton) Cursor() desktop.Cursor { return desktop.PointerCursor }
