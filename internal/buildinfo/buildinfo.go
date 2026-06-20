// Package buildinfo exposes build-time identification for the binaries.
package buildinfo

// Version is the scheduler version. It can be overridden at build time with
// -ldflags "-X github.com/shruggietech/go-scheduler/internal/buildinfo.Version=...".
var Version = "0.0.1-dev"
