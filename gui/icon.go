package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

// iconBytes is the application icon (a calendar+clock mark, transparent PNG).
//
//go:embed assets/icon.png
var iconBytes []byte

// appIcon is the embedded icon as a Fyne resource, used for the application and
// window icon (taskbar, title bar, and macOS dock).
var appIcon = fyne.NewStaticResource("icon.png", iconBytes)
