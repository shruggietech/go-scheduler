package gui

import (
	"errors"
	"strconv"
	"time"

	"fyne.io/fyne/v2/dialog"
)

var errInvalidOneOff = errors.New("one-off time must be RFC3339, e.g. 2026-08-04T09:00:00Z")

func (a *App) showError(err error) {
	if err == nil {
		return
	}
	dialog.ShowError(err, a.win)
}

func itoa(n int) string { return strconv.Itoa(n) }

// fmtTime renders an instant for display in the user's local time.
func fmtTime(t time.Time) string {
	return t.Local().Format("Mon 2006-01-02 15:04")
}

func boolStr(b bool, yes, no string) string {
	if b {
		return yes
	}
	return no
}
