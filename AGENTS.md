# AGENTS.md

## Project Overview

`whatsd` is intended to be a Linux-first WhatsApp daemon written in Rust. It will keep WhatsApp Web sessions running in the background and expose local client operations over a Unix domain socket. Clients may include Emacs packages, GTK apps, command-line tools, and other local integrations.

## Current Status

The repository is early-stage. Dependencies and planning docs exist, but the daemon is not implemented yet. Before making architectural changes, read `docs/daemon-implementation-plan.md`.

## Development Principles

1. Keep changes small and reviewable.
2. Prefer a stable local IPC contract over exposing upstream crate internals directly.
3. Treat the Unix socket as a sensitive local control surface.
4. Do not log message bodies or phone numbers unless explicitly adding a debug-only feature.
5. Use durable storage before claiming reliable message delivery.
6. Prefer Tokio-native async code for daemon internals.
7. Keep Linux/XDG behavior first-class.

## Rust Guidelines

1. Use edition 2024.
2. Run `cargo fmt` after Rust edits.
3. Run `cargo check` and `cargo clippy --all-targets --all-features` after implementation, then fix errors and warnings in a loop until the codebase is clean.
4. Use `anyhow` at application boundaries and for simple upward error propagation. Use `thiserror` only when a module needs typed errors.
5. Use `tracing` for daemon logs.
6. Keep IPC request/response structs and shared application DTOs under `src/types/`.
7. Avoid adding compatibility layers before the IPC protocol has external consumers.
8. Avoid monolithic files. Keep modules under 200 lines when practical and split into submodules before files become hard to scan.

## AI-Assisted Rust Workflow

1. Prefer small, idiomatic changes that preserve compileability at each step.
2. After implementing behavior, run `cargo fmt`, `cargo check`, and `cargo clippy --all-targets --all-features`.
3. If check or Clippy reports errors or warnings, fix them and rerun the same command until it is clean.
4. Propagate critical errors upward with `anyhow::Result` and `.context(...)` at application boundaries.
5. Handle uncritical local errors internally with explicit control flow and `tracing` warnings instead of panics.
6. Do not use `unwrap`, `expect`, or panics in daemon runtime paths unless the invariant is local, obvious, and unrecoverable.
7. Keep protocol and cross-module data structures in `src/types/`; implementation modules should consume those types rather than each defining private wire structs.

## Dependency Notes

This project follows upstream `whatsapp-rust` defaults and pins nightly Rust in `rust-toolchain.toml`:

```toml
whatsapp-rust = "0.6"
```

The pinned toolchain should include `rustfmt`, `clippy`, `rust-analyzer`, and `rust-src` so command-line checks and editor integration use the same nightly compiler context.

The generated upstream docs mention a `tracing` feature for `whatsapp-rust`, but the published `0.6.0` crate did not accept it. Do not add that feature unless a future crate version supports it.

Do not remove the nightly toolchain pin without checking whether upstream `wacore` and the default SIMD feature build on stable.

## IPC Direction

The planned initial protocol is newline-delimited JSON over a Unix domain socket:

```json
{"id":"uuid","type":"daemon.ping","payload":{}}
```

Keep protocol changes documented in `docs/` and prefer versioned, explicit command names such as `message.send_text` and `session.status`.

## Security Notes

Default socket placement should follow XDG runtime conventions, preferably under `$XDG_RUNTIME_DIR/whatsd/`. Socket directories should be user-only. Be careful with stale socket cleanup and never remove arbitrary user files at a configured path.

## Useful Commands

```sh
cargo fmt
cargo check
cargo clippy --all-targets --all-features
cargo test
rust-analyzer --version
```
