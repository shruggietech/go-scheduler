# go-scheduler — developer tasks
# The GUI binary (gosched-gui) requires a C toolchain + OpenGL (Fyne); the daemon
# and CLI are cgo-free. Targets below operate on the whole module.

GO        ?= go
LDFLAGS   ?=
GUI_LDFLAGS_WINDOWS = -H windowsgui

.PHONY: all fmt vet lint test test-race cover bench build build-daemon build-cli build-gui tidy clean

all: fmt vet test build

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	golangci-lint run

test:
	$(GO) test ./...

test-race:
	$(GO) test -race ./...

cover:
	$(GO) test -race -covermode=atomic -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out | tail -1

bench:
	$(GO) test -bench=. -benchmem ./internal/engine/...

build: build-daemon build-cli

build-daemon:
	$(GO) build -o bin/goschedd ./cmd/goschedd

build-cli:
	$(GO) build -o bin/gosched ./cmd/gosched

# GUI: on Windows add $(GUI_LDFLAGS_WINDOWS) so no console window appears.
build-gui:
	$(GO) build -o bin/gosched-gui ./cmd/gosched-gui

tidy:
	$(GO) mod tidy

clean:
	rm -rf bin coverage.out
