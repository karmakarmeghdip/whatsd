# WhatsApp Daemon Implementation Plan

## Goal

Build `whatsd`, a Rust daemon that owns one or more WhatsApp Web sessions and exposes WhatsApp Desktop/Web-like functionality to local Linux clients over a Unix domain socket. Client applications such as Emacs packages, GTK applications, shell tools, or other local integrations should not connect to WhatsApp directly. They should speak a stable local IPC protocol to the daemon.

## Current Repository State

The project is currently a minimal Rust binary crate:

```text
Cargo.toml
Cargo.lock
src/main.rs
```

Initial dependencies have been added only to prepare the daemon foundation. No daemon behavior is implemented yet.

## Added Crates

Core WhatsApp stack:

```toml
whatsapp-rust = "0.6"
```

The docs show that `whatsapp-rust` re-exports most of its stack, including `wacore`, `wacore_binary`, `waproto`, `SqliteStore`, transport, HTTP, and public third-party API types. We are following upstream's default feature set and using a pinned nightly toolchain rather than trying to keep this project stable-compatible at this stage.

Toolchain:

```toml
[toolchain]
channel = "nightly-2026-04-05"
components = ["rustfmt", "clippy", "rust-analyzer", "rust-src"]
```

Runtime and daemon foundation:

```toml
tokio = { version = "1.48", features = ["macros", "rt-multi-thread", "net", "io-util", "signal", "sync", "time", "fs"] }
clap = { version = "4", features = ["derive", "env"] }
directories = "6"
uuid = { version = "1", features = ["v4", "serde"] }
```

Protocol, errors, and observability:

```toml
serde = { version = "1", features = ["derive"] }
serde_json = "1"
anyhow = "1"
thiserror = "2"
tracing = "0.1"
tracing-subscriber = { version = "0.3", features = ["env-filter"] }
```

Note: the generated docs mention a `tracing` feature for `whatsapp-rust`, but the published `0.6.0` crate rejected that feature during `cargo add`. We should use our own `tracing` instrumentation for `whatsd` and revisit upstream crate features if a newer release exposes tracing.

## Documentation Findings

`whatsapp-rust` exposes a high-level `Bot::builder()` API suitable for the first implementation:

```rust
let bot = Bot::builder()
    .with_backend(SqliteStore::new("whatsapp.db").await?)
    .on_qr_code(|code, _timeout| async move { /* publish pairing event */ })
    .on_pair_code(|code, _timeout| async move { /* publish pair code event */ })
    .on_connected(|client| async move { /* mark session online */ })
    .on_logged_out(|info| async move { /* mark session logged out */ })
    .on_message(|ctx| async move { /* persist and broadcast message */ })
    .build()
    .await?;
```

Useful documented behavior:

1. `SqliteStore` persists session data, device keys, app state, and reconnect state.
2. QR pairing is available by default through `on_qr_code`.
3. Pair-code login is available through `PairCodeOptions` and `on_pair_code`.
4. `bot.run().await` runs until logout or disconnect; `bot.spawn()` returns a handle with `client()`, `shutdown()`, and `abort()`.
5. Typed event registrars include `on_message`, `on_qr_code`, `on_pair_code`, `on_connected`, and `on_logged_out`.
6. Catch-all event handling is available via `on_event` / `on_event_for` for receipts, group updates, chat state updates, history sync, raw nodes, and other less common events.
7. Sending text can use `client.send_text(&jid, text).await`.
8. Sending arbitrary messages can use `client.send_message(&jid, wa::Message).await`.
9. Reactions use `client.send_reaction()`.
10. Media upload/download, groups, newsletters, polls, receipts, presence, chat updates, profile, privacy, and app-state sync are represented in the crate docs and should be mapped incrementally into daemon IPC commands.
11. The inbound durability hook provides at-least-once delivery by deferring WhatsApp server ack until the daemon commits messages durably. This is important for a daemon because client applications may be offline while the daemon receives messages.

## Architecture

The daemon should have five main layers:

1. `Daemon`: process lifecycle, CLI/env config, logging, shutdown, PID/socket cleanup.
2. `IpcServer`: Unix domain socket listener, client connection tasks, request decoding, response encoding, subscriptions, backpressure.
3. `SessionManager`: owns WhatsApp sessions, pairing state, connection state, client handles, and command dispatch into `whatsapp-rust`.
4. `Store`: durable local daemon state not already covered by `whatsapp-rust` storage, such as normalized message indexes, client-visible chats, event cursor state, and idempotency keys.
5. `Types`: shared protocol and application DTOs used for IPC and internal module communication.

The first version can keep these as Rust modules in one crate:

```text
src/
  lib.rs
  main.rs
  config.rs
  daemon.rs
  types/
    mod.rs
    protocol.rs
    session.rs
    message.rs
  ipc/
    mod.rs
    server.rs
  session/
    mod.rs
    manager.rs
    events.rs
  store/
    mod.rs
```

The `types` module is the home for IPC request/response structs, command/event enums, session DTOs, message payload DTOs, and other cross-module application structs. Implementation modules should use these shared types instead of defining separate private wire structs.

Only expand the library crate as needed for tests, client tooling, or generated protocol bindings.

## Development Hygiene

AI-assisted changes should keep the Rust codebase clean and idiomatic:

1. Run `cargo fmt` after Rust edits.
2. Run `cargo check` and `cargo clippy --all-targets --all-features` after implementation.
3. Fix errors and warnings in a loop until both commands pass cleanly.
4. Use `anyhow::Result` for simple application-boundary error propagation.
5. Add `.context(...)` where extra operational context helps diagnose failures.
6. Handle non-critical local errors internally with explicit branches and `tracing` warnings.
7. Avoid `unwrap`, `expect`, and panics in daemon runtime paths.
8. Keep files under 200 lines where practical. Split into submodules before a file becomes monolithic.

## Socket Location

Default paths should follow XDG conventions:

```text
Socket: $XDG_RUNTIME_DIR/whatsd/whatsd.sock
State:  $XDG_STATE_HOME/whatsd/
Cache:  $XDG_CACHE_HOME/whatsd/
Logs:   stderr initially; systemd/journald later
```

The CLI should allow overrides:

```text
whatsd --socket /path/to.sock --state-dir /path/to/state --database /path/to/whatsapp.db
```

Socket permissions should be user-only by default. Avoid world-writable locations unless the daemon also verifies peer credentials.

## IPC Protocol

Start with newline-delimited JSON over the Unix socket for easy debugging from shells and editors. Each frame is one JSON object followed by `\n`.

Every client request:

```json
{"id":"uuid","type":"command.name","payload":{}}
```

Every daemon response:

```json
{"id":"uuid","ok":true,"payload":{}}
{"id":"uuid","ok":false,"error":{"code":"invalid_request","message":"..."}}
```

Daemon-pushed events for subscribed clients:

```json
{"type":"event.message","payload":{}}
```

JSON is not the final performance ceiling. It is the best first protocol because Emacs, GTK, shell tools, and test clients can all speak it without generated bindings. Once the API stabilizes, add optional MessagePack, CBOR, or bincode framing if needed.

## Initial IPC Commands

Process and health:

```text
daemon.ping
daemon.version
daemon.status
daemon.shutdown
```

Authentication and session:

```text
session.status
session.connect
session.disconnect
session.logout
session.pair_qr
session.pair_code
```

Messages:

```text
message.send_text
message.send
message.react
message.edit
message.revoke
message.mark_read
message.list
message.get
```

Subscriptions:

```text
event.subscribe
event.unsubscribe
```

Chats and contacts:

```text
chat.list
chat.get
chat.archive
chat.pin
chat.mute
contact.list
contact.get
```

Groups and newsletters can be a second milestone after the basic session/message loop is reliable.

## Event Model

The daemon should normalize `whatsapp-rust` events into stable IPC events:

```text
event.connected
event.disconnected
event.logged_out
event.pairing_qr
event.pairing_code
event.message
event.receipt
event.presence
event.chat_update
event.group_update
event.history_sync_progress
event.error
```

Do not expose raw `whatsapp-rust` structs as the public IPC contract. They can change with crate releases and may be too detailed for simple clients. Keep raw-event escape hatches later behind an explicitly unstable command such as `debug.raw_event.subscribe`.

## Durability Strategy

Use `SqliteStore` for WhatsApp session and protocol state. Add daemon-owned persistence for client-visible state only when needed.

Inbound messages should eventually use `InboundDurabilityHook` so the daemon can commit messages before WhatsApp server ack. The hook must be idempotent using the documented key:

```text
(chat, sender, message_id)
```

The first milestone can receive and broadcast live messages without full daemon indexing, but before claiming reliability we should persist inbound messages and event cursors.

## Concurrency Model

Run these tasks under Tokio:

1. Main shutdown signal watcher.
2. Unix socket accept loop.
3. One read/write task pair per IPC client, or one task with split halves if simpler.
4. WhatsApp bot/session task.
5. Event fanout task that receives normalized events and broadcasts to subscribed IPC clients.

Use bounded channels for client event streams. If a client cannot keep up, prefer disconnecting that client or dropping non-critical events according to a documented policy instead of allowing unbounded memory growth.

## Security Model

The Unix socket is a local control surface for a logged-in WhatsApp account. Treat it as sensitive.

Initial controls:

1. Bind only in a user-owned runtime directory.
2. Set socket directory permissions to `0700`.
3. Refuse to start if the socket path exists and is not a socket owned by the current user.
4. Remove stale socket files only after a failed connection probe confirms no daemon is alive.
5. Avoid logging message bodies by default.

Later controls:

1. Peer credential checks with `SO_PEERCRED` on Linux.
2. Per-client permissions for read-only clients versus send-capable clients.
3. Optional confirmation policy for destructive operations.

## Milestones

### Milestone 1: Daemon Skeleton

Deliverables:

1. CLI config for socket path, state directory, database path, log filter.
2. Tokio main with graceful shutdown.
3. Unix socket listener.
4. JSON line request/response protocol.
5. `daemon.ping`, `daemon.version`, and `daemon.status`.
6. Basic integration test that connects to a temporary Unix socket and calls `daemon.ping`.

### Milestone 2: WhatsApp Session Lifecycle

Deliverables:

1. `SessionManager` wrapping `Bot::builder()` and `SqliteStore`.
2. `session.connect`, `session.disconnect`, `session.status`.
3. QR pairing event publishing.
4. Pair-code login command and pairing-code event publishing.
5. Connected, disconnected, and logged-out event publishing.

### Milestone 3: Basic Messaging

Deliverables:

1. Normalize inbound message events.
2. `event.subscribe` for live message events.
3. `message.send_text` using `client.send_text`.
4. `message.send` for structured `wa::Message` text payloads.
5. Basic receipt event mapping.

### Milestone 4: Durable Local State

Deliverables:

1. Daemon-owned SQLite tables for normalized messages and event cursors.
2. Inbound durability hook with idempotent `(chat, sender, id)` upserts.
3. `message.list` and `message.get` from local state.
4. Reconnect behavior verified with offline batches.

### Milestone 5: Desktop/Web Feature Coverage

Deliverables:

1. Reactions, edits, revokes, read receipts.
2. Media upload/download and thumbnails.
3. Chat list, archive, mute, pin, mark read.
4. Contact list and profile pictures.
5. Group management.
6. Newsletters/channels.
7. Presence and typing state.
8. Privacy/profile operations.

### Milestone 6: Client Developer Experience

Deliverables:

1. Protocol reference in `docs/`.
2. Example CLI client.
3. Emacs proof-of-concept client.
4. GTK proof-of-concept client.
5. Versioned IPC protocol with compatibility notes.

## Open Questions For Review

1. Should the daemon support exactly one WhatsApp account initially, or should the IPC protocol include an `account_id` from day one?
2. Should the first IPC protocol be newline-delimited JSON, or do you prefer a binary format immediately?
3. Should clients be allowed to send messages by default if they can connect to the socket, or should there be a permission model from the first milestone?
4. Should local message history be a daemon feature immediately, or should the first build only proxy live events and sends?
5. Should `whatsd` include a small CLI client in the same repository for testing, for example `whatsctl`?
6. If we later need wider distro packaging, should we revisit a stable-compatible dependency setup or vendor/patch upstream?

## Recommended First Implementation After Approval

Start with Milestone 1 and keep it intentionally small. The first useful merge should create a daemon that starts, binds a secure Unix socket, accepts JSON requests, answers `daemon.ping`, reports status, and shuts down cleanly. This gives every later WhatsApp feature a stable local transport and a testable process model.
