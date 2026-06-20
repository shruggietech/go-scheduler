package gui

import (
	_ "embed"

	"fyne.io/fyne/v2"
)

// iconBytes is the full-resolution application icon (a calendar+clock mark,
// transparent PNG) used for large surfaces like the macOS dock.
//
//go:embed assets/icon.png
var iconBytes []byte

// windowIconBytes is a small, purpose-rendered tile of the same mark (extracted
// from the multi-size .ico). Fyne scales the window icon down to ~16px for the
// title bar; downscaling the 1080px source that far mangles the thin strokes,
// so the title bar uses this crisp 64px tile instead.
//
//go:embed assets/icon-window.png
var windowIconBytes []byte

// appIcon is the full-resolution mark, used for the application-level icon
// (macOS dock and other large surfaces).
var appIcon = fyne.NewStaticResource("icon.png", iconBytes)

// windowIcon is the small crisp mark used for window decorations (title bar and
// the Windows taskbar/alt-tab), where the image is shown at 16–64px.
var windowIcon = fyne.NewStaticResource("icon-window.png", windowIconBytes)
