//go:build cgo

// Command gosched-gui is the Go-native desktop GUI (Fyne). It is built with cgo
// (the OpenGL driver). On Windows it is linked with -H windowsgui so no console
// window appears. It connects to the running daemon over the local API.
package main

import (
	"context"

	"fyne.io/fyne/v2/app"

	"github.com/shruggietech/go-scheduler/gui"
	"github.com/shruggietech/go-scheduler/internal/api/client"
	"github.com/shruggietech/go-scheduler/internal/autostart"
	"github.com/shruggietech/go-scheduler/internal/config"
	"github.com/shruggietech/go-scheduler/internal/ipc"
)

// Compile-time check that the API client satisfies the GUI's Backend.
var _ gui.Backend = (*client.Client)(nil)

func main() {
	cfg, _ := config.Load("")
	c := client.New(ipc.Endpoint(cfg))

	// Zero-config: if no daemon is reachable, start one in the background
	// (windowless, detached) so opening the GUI just works. A running daemon
	// (e.g. the installed service) is detected and reused.
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	_ = autostart.EnsureRunning(ctx,
		func(ctx context.Context) error { _, err := c.Health(ctx); return err },
		autostart.SpawnDaemon,
	)
	cancel()

	a := app.NewWithID("tech.shruggie.goscheduler")
	gui.NewUI(a, c).Run()
}
