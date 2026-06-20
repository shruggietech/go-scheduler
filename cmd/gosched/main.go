// Command gosched is the scheduler CLI. In this foundational form it supports a
// health check against the running daemon; the full cobra command tree (task,
// group, trigger, service, gui) is added in User Story 1.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/shruggietech/go-scheduler/internal/api/client"
	"github.com/shruggietech/go-scheduler/internal/buildinfo"
	"github.com/shruggietech/go-scheduler/internal/config"
	"github.com/shruggietech/go-scheduler/internal/ipc"
)

func main() {
	args := os.Args[1:]
	cmd := "health"
	if len(args) > 0 {
		cmd = args[0]
	}

	switch cmd {
	case "version", "--version", "-v":
		fmt.Fprintln(os.Stdout, "gosched "+buildinfo.Version)
	case "health":
		if err := health(); err != nil {
			fmt.Fprintln(os.Stderr, "gosched: "+err.Error())
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "gosched: unknown command "+cmd+" (try: health, version)")
		os.Exit(2)
	}
}

func health() error {
	cfg, err := config.Load("")
	if err != nil {
		return err
	}
	c := client.New(ipc.Endpoint(cfg))
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	h, err := c.Health(ctx)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "daemon ok (version %s)\n", h.Version)
	return nil
}
