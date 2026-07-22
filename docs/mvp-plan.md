# whatsd - Basic MVP Implementation Plan

## 1. Overview & Objectives

`whatsd` is a Linux-first WhatsApp daemon written in Go. It keeps a WhatsApp Web session running continuously in the background using the `whatsmeow` library and exposes daemon operations over a local Unix domain socket IPC.

The main goals of this MVP implementation are:
1. **Verify `whatsmeow` viability**: Confirm that `whatsmeow` works cleanly with Go 1.22+ and SQLite storage on Linux.
2. **Implement Pairing Functionality**: Support pairing via QR code (ASCII output or string emitted over IPC) and/or phone number pairing code.
3. **Implement Message Receiving**: Capture incoming text messages from chats/groups and broadcast them to connected IPC clients.
4. **Implement Message Sending**: Allow IPC clients to send text messages to a target WhatsApp JID.
5. **Provide a CLI Tool (`whatsctl`)**: Offer a command-line tool to test pairing, check daemon status, send messages, and listen to incoming events.

---

## 2. Architecture & Component Design

```
+-------------------------------------------------------------------+
|                            whatsctl                             |
|       (CLI client for pairing, status, sending, listening)        |
+---------------------------------+---------------------------------+
                                  | Unix Domain Socket
                                  v
+-------------------------------------------------------------------+
|                           whatsd daemon                           |
|                                                                   |
|   +-----------------------+     +-----------------------------+   |
|   |   IPC Server (Unix)   |<--->|     Daemon Controller       |   |
|   |  (JSON-RPC protocol)  |     |   (Lifecycle & Events)      |   |
|   +-----------------------+     +--------------+--------------+   |
|                                                |                  |
|                                                v                  |
|                                 +-----------------------------+   |
|                                 |   whatsmeow Manager         |   |
|                                 |   - Connection Manager      |   |
|                                 |   - QR/Pair Code Handler    |   |
|                                 |   - Event Dispatcher        |   |
|                                 +--------------+--------------+   |
|                                                |                  |
|                                                v                  |
|                                 +-----------------------------+   |
|                                 | SQLite Store (sqlstore)     |   |
|                                 | (~/.local/share/whatsd/...) |   |
|                                 +-----------------------------+   |
+-------------------------------------------------------------------+
```

### 2.1 Package Breakdown (`internal/` & `cmd/`)

- `internal/types/`:
  - IPC request/response structs, method names, event payload DTOs.
  - Data transfer objects representing WhatsApp messages and status.
- `internal/config/`:
  - XDG path resolution (`XDG_DATA_HOME`, `XDG_STATE_HOME`, `XDG_RUNTIME_DIR`).
  - Configuration defaults for database path (`~/.local/share/whatsd/session.db`) and socket path (`${XDG_RUNTIME_DIR}/whatsd.sock` or `~/.local/state/whatsd/whatsd.sock`).
- `internal/store/`:
  - Initializer for `whatsmeow/store/sqlstore` with `modernc.org/sqlite` (CGO-free SQLite driver).
- `internal/whatsapp/`:
  - Wrapper around `whatsmeow.Client`.
  - Handles `Connect()`, `Disconnect()`, QR code channel, pair code generation, event handling for `*events.Message` and `*events.Connected`.
- `internal/ipc/`:
  - Unix domain socket server with concurrent connection handling.
  - Line-delimited JSON RPC reader/writer.
  - Event broadcaster for push notifications (incoming messages, connection status changes).
- `cmd/whatsd/`:
  - Main daemon binary entry point. Signal handling (SIGINT, SIGTERM), graceful shutdown.
- `cmd/whatsctl/`:
  - Command-line utility for manual verification: `status`, `pair`, `send`, `listen`.

---

## 3. IPC Wire Protocol Contract

Communication occurs over a Unix domain socket using line-delimited JSON.

### 3.1 Request / Response Format

**Request:**
```json
{
  "id": "1",
  "method": "status",
  "params": {}
}
```

**Response (Success):**
```json
{
  "id": "1",
  "result": {
    "connected": true,
    "logged_in": true,
    "jid": "1234567890@s.whatsapp.net",
    "push_name": "Alice"
  },
  "error": null
}
```

**Response (Error):**
```json
{
  "id": "1",
  "result": null,
  "error": "not connected to WhatsApp"
}
```

### 3.2 Supported IPC Methods

1. `status`
   - Returns daemon connection state, login status, JID, and push name.
2. `pair_qr`
   - Initiates QR code pairing mode if not logged in.
   - Emits QR events over the IPC stream.
3. `pair_phone`
   - Params: `{"phone": "15551234567"}`
   - Returns phone pairing code string.
4. `send_message`
   - Params: `{"to": "15551234567@s.whatsapp.net", "text": "Hello world"}`
   - Returns sent message ID and timestamp.
5. `logout`
   - Logs out the current session and clears store.

### 3.3 Daemon Event Notifications (Push)

When a client subscribes or is connected to the socket stream, the daemon broadcasts events:

```json
{
  "event": "message",
  "data": {
    "id": "3EB0...",
    "chat": "15551234567@s.whatsapp.net",
    "sender": "15551234567@s.whatsapp.net",
    "sender_name": "Bob",
    "timestamp": "2026-07-22T23:21:00Z",
    "is_from_me": false,
    "text": "Hey there!"
  }
}
```

---

## 4. Implementation Phasing

### Phase 1: Dependency Setup & Core Types
- Configure `go.mod` with `go.mau.fi/whatsmeow` and `modernc.org/sqlite`.
- Define IPC and WhatsApp types in `internal/types/`.
- Setup XDG path resolution in `internal/config/`.

### Phase 2: Database Store & WhatsApp Client Wrapper
- Implement `internal/store` using `sqlstore.New`.
- Implement `internal/whatsapp` to manage client state, connection lifecycle, and event handlers.

### Phase 3: IPC Unix Socket Server
- Implement `internal/ipc` Unix domain socket server with request routing and push event broadcasting.

### Phase 4: Daemon Entry Point (`cmd/whatsd`)
- Assemble store, WhatsApp client, and IPC server into `cmd/whatsd/main.go`.
- Implement clean shutdown handling with `context.Context` and `slog`.

### Phase 5: CLI Test Client (`cmd/whatsctl`) & Empirical Verification
- Build `cmd/whatsctl` supporting `status`, `pair`, `send`, `listen`.
- Test QR pairing and phone pairing.
- Test sending and receiving messages live.

---

## 5. Guidelines & Verification Checklist

- [ ] Target Go 1.22+ clean compilation (`go build ./...`).
- [ ] Strictly follow standard error wrapping (`fmt.Errorf("...: %w", err)`).
- [ ] Structured logging using standard `log/slog`.
- [ ] Run `go fmt ./...` and `go vet ./...` until zero errors.
- [ ] Validate Unix socket file cleanup on daemon termination.
- [ ] Empirical runtime verification of message sending and receiving.
