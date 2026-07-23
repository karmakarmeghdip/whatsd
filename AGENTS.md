
# AGENTS.md

## Project Overview

`whatsd` is a Linux-first WhatsApp daemon written in Go. It keeps WhatsApp Web sessions running in the background using the `whatsmeow` library and exposes local client operations over a Unix domain socket. Clients may include Emacs packages, GTK apps, command-line tools, and other local integrations.

## Current Status

The repository is early-stage. It has the whatsmeow dependency cloned in references directory. Basic mvp implementation is required, after which it'll be tested.


## Development Principles

1. Keep changes small and reviewable.
2. Prefer a stable local IPC contract over exposing upstream `whatsmeow` library types or protobuf internals directly.
3. Treat the Unix socket as a sensitive local control surface.
4. Do not log message bodies or phone numbers unless explicitly adding a debug-only feature.
5. Use durable storage before claiming reliable message delivery.
6. Prefer standard library `context.Context` and goroutines for daemon concurrency and lifecycle management.
7. Keep Linux/XDG behavior first-class.

## Go Guidelines

1. Target Go 1.22 or higher.
2. Run `make fmt` after Go edits.
3. Run `make lint` and `make test` after implementation, then fix errors and warnings in a loop until the codebase is clean.
4. Use standard error wrapping (`fmt.Errorf("...: %w", err)`) and standard `errors.Is` / `errors.As` checks. Avoid naked error returns or unhandled errors.
5. Use standard library `log/slog` for structured daemon logging.
6. Keep IPC request/response structs and shared application DTOs under `internal/types/`.
7. Avoid adding compatibility layers before the IPC protocol has external consumers.
8. Avoid monolithic files. Keep files under 250 lines when practical, and split into subpackages under `internal/` before files become hard to scan.

## AI-Assisted Go Workflow

1. Prefer small, idiomatic changes that preserve compilation (`make build`) at each step.
2. After implementing behavior, run `make fmt`, `make lint`, and `make test`.
3. If `make lint` or `make test` reports errors or warnings, fix them and rerun the Makefile targets in a loop until clean.
4. Propagate critical errors upward with explicit error wrapping via `fmt.Errorf("context message: %w", err)`.
5. Handle non-critical local errors internally with explicit control flow and `slog.Warn(...)` or `slog.Error(...)` instead of panicking.
6. Do not use `panic()` or `log.Fatal()` inside runtime daemon paths, IPC connection loops, or event handlers.
7. Keep protocol and cross-module data structures in `internal/types/`; implementation packages should consume those types rather than defining private wire structs.

## Dependency Notes

This project relies on `whatsmeow` for WhatsApp multi-device Web API communication:

```go
go.mau.fi/whatsmeow v0.0.0-... // Go library for WhatsApp multi-device Web API
