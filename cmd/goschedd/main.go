// Command goschedd is the scheduler daemon. It hosts the engine, persistence,
// and executor and serves the local API over IPC. In this foundational form it
// loads config, opens the store, and serves health; the scheduling engine is
// wired in during User Story 1.
package main

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/config"
	"github.com/shruggietech/go-scheduler/internal/ipc"
	"github.com/shruggietech/go-scheduler/internal/store"
)

func main() {
	configPath := flag.String("config", "", "path to config file (optional)")
	flag.Parse()

	if err := run(*configPath); err != nil {
		// Daemon errors go to stderr; structured logs go to the configured sink.
		os.Stderr.WriteString("goschedd: " + err.Error() + "\n")
		os.Exit(1)
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return err
	}

	log := config.NewLogger(cfg, os.Stdout)

	st, err := store.Open(cfg.DBPath())
	if err != nil {
		return err
	}
	defer st.Close()

	endpoint := ipc.Endpoint(cfg)
	ln, err := ipc.Listen(endpoint)
	if err != nil {
		return err
	}
	defer ln.Close()

	srv := &http.Server{
		Handler:           server.New(st, log).Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	// Graceful shutdown on SIGINT/SIGTERM.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		log.Info("daemon listening", "endpoint", endpoint, "db", cfg.DBPath())
		if err := srv.Serve(ln); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	select {
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-serveErr:
		return err
	}
}
