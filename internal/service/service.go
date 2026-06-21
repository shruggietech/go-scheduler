// Package service wraps cross-platform system-service management (systemd /
// launchd / Windows Service) via github.com/kardianos/service. The daemon runs
// under the OS service manager so it starts on boot, system-wide; the CLI uses
// Control to install/start/stop/uninstall it.
package service

import (
	"context"
	"fmt"
	"os"

	"github.com/kardianos/service"
)

const (
	svcName        = "goschedd"
	svcDisplayName = "go-schedule"
	svcDescription = "Cross-platform task scheduler daemon"
)

// program adapts a daemon run-function to the kardianos service interface.
type program struct {
	runFn  func(context.Context) error
	cancel context.CancelFunc
}

func (p *program) Start(service.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel
	go func() {
		// A non-nil error means a fatal startup/runtime failure (e.g. the IPC pipe
		// is already in use by another daemon). Exit non-zero so the service
		// manager reflects the failure instead of reporting a zombie "running"
		// service that serves nothing. Normal shutdown (ctx cancel) returns nil.
		if err := p.runFn(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "goschedd: fatal:", err)
			os.Exit(1)
		}
	}()
	return nil
}

func (p *program) Stop(service.Service) error {
	if p.cancel != nil {
		p.cancel()
	}
	return nil
}

func baseConfig(execPath string, args []string) *service.Config {
	return &service.Config{
		Name:        svcName,
		DisplayName: svcDisplayName,
		Description: svcDescription,
		Executable:  execPath, // empty => current executable (used when running)
		Arguments:   args,
	}
}

// Run executes the daemon under the service manager. When launched
// interactively it runs in the foreground until interrupted; under a service
// manager it follows the manager's lifecycle. runFn must honor ctx cancellation.
func Run(runFn func(context.Context) error) error {
	prog := &program{runFn: runFn}
	svc, err := service.New(prog, baseConfig("", nil))
	if err != nil {
		return fmt.Errorf("service: %w", err)
	}
	return svc.Run()
}

// Control performs an install/uninstall/start/stop/restart/status action against
// the system service. For install, execPath/args record how the manager should
// launch the daemon binary.
func Control(action, execPath string, args []string) (string, error) {
	prog := &program{}
	svc, err := service.New(prog, baseConfig(execPath, args))
	if err != nil {
		return "", fmt.Errorf("service: %w", err)
	}

	if action == "status" {
		st, err := svc.Status()
		if err != nil {
			return "", fmt.Errorf("service: status: %w", err)
		}
		return statusString(st), nil
	}

	if err := service.Control(svc, action); err != nil {
		return "", fmt.Errorf("service: %s: %w", action, err)
	}
	return action + " ok", nil
}

func statusString(st service.Status) string {
	switch st {
	case service.StatusRunning:
		return "running"
	case service.StatusStopped:
		return "stopped"
	default:
		return "unknown (is the service installed?)"
	}
}

// Actions lists the supported control verbs.
func Actions() []string {
	return []string{"install", "uninstall", "start", "stop", "restart", "status"}
}
