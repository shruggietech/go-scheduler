package gui

import (
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

// testApp is a single Fyne test app shared by every test in the package. Fyne's
// test driver tracks the "main" goroutine via package-global state, so creating a
// fresh app per test races under -race; one shared app keeps that state stable.
var testApp fyne.App

func TestMain(m *testing.M) {
	testApp = test.NewApp()
	code := m.Run()
	testApp.Quit()
	os.Exit(code)
}
