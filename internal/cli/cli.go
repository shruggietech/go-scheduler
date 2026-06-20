// Package cli implements the gosched command-line interface (cobra). Commands
// operate on the daemon through the shared API client, so the CLI and GUI act on
// identical state. Results go to stdout; diagnostics/errors go to stderr; exit
// codes follow the contract (0 ok, 1 runtime error, 2 usage/validation).
package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"github.com/shruggietech/go-scheduler/internal/api/client"
	"github.com/shruggietech/go-scheduler/internal/api/server"
	"github.com/shruggietech/go-scheduler/internal/buildinfo"
	"github.com/shruggietech/go-scheduler/internal/config"
	"github.com/shruggietech/go-scheduler/internal/ipc"
)

// errUsage marks validation/usage failures so Execute can return exit code 2.
var errUsage = errors.New("usage")

var jsonOut bool

// Execute runs the root command and returns a process exit code.
func Execute() int {
	root := newRoot()
	err := root.Execute()
	if err == nil {
		return 0
	}
	fmt.Fprintln(os.Stderr, "gosched: "+err.Error())
	if errors.Is(err, errUsage) {
		return 2
	}
	// Server-side validation failures map to the usage/validation exit code.
	var se *client.StatusError
	if errors.As(err, &se) && se.Code == server.CodeValidation {
		return 2
	}
	return 1
}

func newRoot() *cobra.Command {
	root := &cobra.Command{
		Use:           "gosched",
		Short:         "Cross-platform task scheduler — cron power without the cryptic syntax",
		SilenceUsage:  true,
		SilenceErrors: true,
		Version:       buildinfo.Version,
	}
	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "machine-readable JSON output")
	root.AddCommand(
		newTaskCmd(),
		newGroupCmd(),
		newRunsCmd(),
		newAlertsCmd(),
		newServiceCmd(),
		newGUICmd(),
		newHealthCmd(),
	)
	return root
}

func newClient() *client.Client {
	cfg, _ := config.Load("")
	return client.New(ipc.Endpoint(cfg))
}

func reqCtx() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// printJSON writes v as indented JSON to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func newHealthCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "Check that the daemon is running",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, cancel := reqCtx()
			defer cancel()
			h, err := newClient().Health(ctx)
			if err != nil {
				return err
			}
			if jsonOut {
				return printJSON(h)
			}
			fmt.Fprintf(os.Stdout, "daemon ok (version %s)\n", h.Version)
			return nil
		},
	}
}
