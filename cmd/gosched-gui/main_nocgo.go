//go:build !cgo

// Stub entry point built when cgo is disabled (the Fyne GUI requires cgo +
// OpenGL). This keeps `go build ./...` working on cgo-free toolchains; the real
// GUI is built in CI/release with CGO_ENABLED=1.
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(os.Stderr, "gosched-gui was built without GUI support (requires cgo + OpenGL). "+
		"Rebuild with CGO_ENABLED=1 and a C toolchain.")
	os.Exit(1)
}
