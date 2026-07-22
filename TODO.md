# `whatsd` — WhatsApp Desktop Feature Parity Roadmap

This document outlines the master roadmap and feature checklist for `whatsd` to achieve full feature parity with WhatsApp Desktop / WhatsApp Web.

The goal of `whatsd` is to run as a lightweight, Linux-first background daemon, owning WhatsApp Web sessions and exposing all desktop features over a secure local Unix domain socket IPC protocol.

---

## 📊 Summary Progress Tracker

- [x] **Phase 1: Daemon Foundation & Security Model** *(Milestone 1 - Skeleton)*
- [x] **Phase 2: Data Types & IPC Framing Protocol** *(Commands & Events DTOs)*
- [ ] **Phase 3: Session Lifecycle & Dual-Pairing Engine** *(Milestone 2)*
- [ ] **Phase 4: Core Messaging & Rich Content** *(Milestone 3)*
- [ ] **Phase 5: Media Attachments & CDN Engine** *(Milestone 3.5)*
- [ ] **Phase 6: Durable Local Persistence & History Indexing** *(Milestone 4)*
- [ ] **Phase 7: Chat Threads & Organization** *(Milestone 5.1)*
- [ ] **Phase 8: Group Chats & Community Management** *(Milestone 5.2)*
- [ ] **Phase 9: Channels / Newsletters (Broadcast)** *(Milestone 5.3)*
- [ ] **Phase 10: Contacts, Privacy & Profile Management** *(Milestone 5.4)*
- [ ] **Phase 11: Real-time Presence, Typing & Receipts** *(Milestone 5.5)*
- [ ] **Phase 12: Voice & Video Calling Signaling (WebRTC)** *(Milestone 5.6)*
- [ ] **Phase 13: History Sync & Global Full-Text Search** *(Milestone 5.7)*
- [ ] **Phase 14: Client Ecosystem & Desktop Integrations** *(Milestone 6)*

---

## 🚀 Phase-by-Phase Detailed Checklist

### Phase 1: Daemon Foundation & Security Model
- [x] CLI command-line parser & environment variable handling (`clap` derive mode).
- [x] Configuration structure (`Config`) and safe XDG directory resolver (`directories`).
- [x] Path validation for `account_id` preventing path traversal attacks (`..`, unsafe chars).
- [x] Private directory security validation (`0700` user-only permissions) on state & cache dirs.
- [x] EUID ownership validation (`libc::geteuid()`) to ensure local single-user isolation.
- [x] Unix domain socket listener creation (`$XDG_RUNTIME_DIR/whatsd/whatsd.sock`) with strict `0600` socket permissions.
- [x] Stale socket detection and safe removal after connection probing.
- [x] Async process signal listeners (`SIGTERM`, `SIGINT` / Ctrl-C) with Tokio `watch` broadcast channel shutdown signaling.
- [ ] Linux `SO_PEERCRED` credential checking on incoming socket streams to verify process UID.

### Phase 2: Data Types & IPC Wire Protocol
- [x] Newline-delimited JSON frame encoder/decoder (`LinesCodec` streaming format).
- [x] Core wire protocol structs: `IpcRequest`, `IpcResponse`, `DaemonEvent`, `ErrorBody`.
- [x] `CommandType` enum covering 34 IPC commands (`daemon.*`, `session.*`, `message.*`, `chat.*`, `contact.*`, `presence.*`, `media.*`).
- [x] `EventType` enum covering 12 normalized events (`event.connected`, `event.message`, `event.receipt`, `event.presence`, etc.).
- [x] Comprehensive client subscription registry (`SubscriptionMap`) & predicate filter matcher (`EventFilter`).
- [ ] Protocol version negotiation command (`daemon.version_check`) for forward client compatibility.

### Phase 3: Session Lifecycle & Dual-Pairing Engine
- [ ] `SessionManager` state machine (`Disconnected`, `Connecting`, `Pairing`, `Connected`, `LoggedOut`).
- [ ] QR Code login flow:
  - [ ] Catch `on_qr_code` callback from `whatsapp-rust`.
  - [ ] Normalize QR code data matrix & publish `event.pairing_qr` to IPC subscribers.
- [ ] 8-Digit Phone Pair Code login flow:
  - [ ] `session.pair_code` IPC command taking target phone number.
  - [ ] Catch `on_pair_code` callback & publish `event.pairing_code` with 8-digit alphanumeric string.
- [ ] Session connection & disconnection handlers (`session.connect`, `session.disconnect`).
- [ ] Remote account logout (`session.logout`) & purging of Signal authentication keys in `whatsapp.db`.
- [ ] Auto-reconnect engine with exponential backoff on transient network dropouts.
- [ ] Active session status query (`session.status`) reporting state, paths, and account metadata.

### Phase 4: Core Messaging & Rich Content
- [ ] Plain text messaging (`message.send_text`).
- [ ] Structured message sending (`message.send`) with support for rich text formatting.
- [ ] Reply-to / Quoted message context parsing and sending (`context_info` with stanza ID & participant).
- [ ] Emoji Reactions (`message.react`):
  - [ ] Add reaction emoji to target message.
  - [ ] Change or remove existing reaction emoji.
  - [ ] Normalize & broadcast `event.reaction` to IPC clients.
- [ ] Message Editing (`message.edit`):
  - [ ] Protocol edit message request within WhatsApp time window.
  - [ ] Update local database index and push `event.message_edited`.
- [ ] Message Revoking / Deleting (`message.revoke`):
  - [ ] "Delete for Everyone" protocol revoke request.
  - [ ] "Delete for Me" local hiding/deletion.
- [ ] Message Forwarding (`message.send_forwarded`):
  - [ ] Forward single or multiple messages to destination chats.
  - [ ] Maintain forwarding score / "Forwarded Many Times" badge metadata.
- [ ] Starred Messages (`message.star`, `message.unstar`, `message.list_starred`):
  - [ ] Toggle starred state on messages.
  - [ ] Fetch all starred messages across chats.
- [ ] Ephemeral / Disappearing Messages (`chat.set_disappearing_timer`):
  - [ ] Configure disappearing message timers (24 hours, 7 days, 90 days, Off).
  - [ ] Honor expiration timestamps for local auto-deletion.
- [ ] Poll Creation & Voting (`poll.create`, `poll.vote`):
  - [ ] Send single-choice and multiple-choice polls.
  - [ ] Cast votes and update live poll tally metrics.
- [ ] Location Sharing (`message.send_location`):
  - [ ] Send static location pins (latitude, longitude, name, address).
  - [ ] Live location updates (stream location updates over time).
- [ ] Contact Card Sharing (`message.send_vcard`):
  - [ ] Send single and multi-contact vCards.

### Phase 5: Media Attachments & CDN Engine
- [ ] Sandbox-validated media downloader (`media.download`):
  - [ ] Strict relative path checks preventing directory traversal outside `$XDG_CACHE_HOME/whatsd/`.
  - [ ] Fetch encrypted media payloads from WhatsApp CDN servers.
  - [ ] Decrypt media streams using Signal media keys (image, video, audio, document, sticker).
  - [ ] Save decrypted files safely into cache directory.
- [ ] Image Sending (`media.send_image`):
  - [ ] Upload image to WhatsApp CDN, generate low-res JPEG thumbnail, and send message.
  - [ ] High Definition (HD) image quality toggle support.
- [ ] Video Sending (`media.send_video`):
  - [ ] Upload video to CDN, extract video duration & thumbnail, and send message.
- [ ] Audio & Voice Notes (`media.send_audio`):
  - [ ] Voice note recording indicator integration (`chatstate.send` - `recording`).
  - [ ] Send voice notes in Opus/OGG format with audio waveform visualization array.
- [ ] Document & File Sending (`media.send_document`):
  - [ ] Upload arbitrary files (PDF, ZIP, DOCX) with filename & byte length metadata.
- [ ] Stickers (`sticker.send`, `sticker.pack_list`):
  - [ ] Send WebP static and animated stickers.
- [ ] View Once Media (`media.send_view_once`):
  - [ ] Send View Once photos/videos.
  - [ ] Enforce view-once single playback constraint in IPC event delivery.

### Phase 6: Durable Local Persistence & History Indexing
- [ ] SQLite local store manager (`sqlite.rs`) for `whatsd.db`.
- [ ] Embedded database schema migrations (`messages`, `chats`, `event_cursors`, `account_state`).
- [ ] Inbound Durability Hook (`InboundDurabilityHook`):
  - [ ] Defer WhatsApp server ACK until daemon durably writes message to `whatsd.db`.
  - [ ] Idempotent message upserts using composite key: `(chat_jid, sender_jid, message_id)`.
- [ ] Message history pagination query (`message.list`):
  - [ ] Query messages by `chat_jid`, pagination limit, and `before_id` cursor.
- [ ] Fetch single message by ID (`message.get`).
- [ ] Durable event backlog replaying for IPC clients reconnecting after being offline.

### Phase 7: Chat Threads & Organization
- [ ] Fetch chat list (`chat.list`) sorted by last message activity.
- [ ] Single chat metadata query (`chat.get`).
- [ ] Unread message counter tracking per chat thread.
- [ ] Mark Chat as Read / Unread (`message.mark_read`, `chat.mark_unread`).
- [ ] Pin & Unpin Chats (`chat.pin`, `chat.unpin`).
- [ ] Archive & Unarchive Chats (`chat.archive`, `chat.unarchive`).
- [ ] Mute & Unmute Chat Notifications (`chat.mute`, `chat.unmute`):
  - [ ] Mute durations: 8 hours, 1 week, Always.
- [ ] Delete Chat / Clear Chat History (`chat.delete`, `chat.clear`).

### Phase 8: Group Chats & Community Management
- [ ] Create Group (`group.create`) with name, optional description, and initial participants.
- [ ] Get Group Info & Metadata (`group.get`):
  - [ ] Retrieve group subject, description, creation timestamp, owner, and participant list.
- [ ] Participant Management:
  - [ ] Add participants (`group.add_participants`).
  - [ ] Remove participants (`group.remove_participants`).
  - [ ] Promote to Admin (`group.promote_admin`).
  - [ ] Demote from Admin (`group.demote_admin`).
- [ ] Group Permissions & Settings (`group.update_settings`):
  - [ ] Restrict message sending (All Participants vs Admins Only).
  - [ ] Restrict editing group info (All Participants vs Admins Only).
  - [ ] Toggle "Require Admin Approval for New Members".
- [ ] Group Invite Links:
  - [ ] Generate / Get invite link (`group.get_invite_link`).
  - [ ] Revoke invite link (`group.revoke_invite_link`).
  - [ ] Inspect group info via invite code (`group.inspect_invite_code`).
  - [ ] Join group via invite code (`group.join_via_invite`).
- [ ] Mentions & Tags (`@user` tagging parsing and notification routing).
- [ ] Communities (`community.create`, `community.get`, `community.list_groups`):
  - [ ] Link groups to parent Community.
  - [ ] Community announcement channel messaging.

### Phase 9: Channels / Newsletters (Broadcast)
- [ ] Search & Browse Public Channels (`channel.search`).
- [ ] Follow & Unfollow Channels (`channel.follow`, `channel.unfollow`).
- [ ] Channel Updates Feed (`channel.list_messages`):
  - [ ] Receive broadcast updates, images, polls, and video posts.
- [ ] Channel Emoji Reactions (`channel.react`).
- [ ] Channel Admin Tools (create channel, post updates, view follower counts).

### Phase 10: Contacts, Privacy & Profile Management
- [ ] Sync Contact List (`contact.list`) from phone address book & WhatsApp app state.
- [ ] Single Contact Query (`contact.get`).
- [ ] Contact Profile Picture Fetcher (`contact.profile_picture`):
  - [ ] Retrieve high-resolution or thumbnail profile photo URL / binary bytes.
- [ ] User Profile Management:
  - [ ] Update display name (`profile.set_name`).
  - [ ] Update About bio status (`profile.set_about`).
  - [ ] Update profile photo (`profile.set_picture`).
- [ ] Block & Unblock Contacts (`contact.block`, `contact.unblock`, `contact.list_blocked`).
- [ ] Privacy Settings Management (`privacy.get`, `privacy.set`):
  - [ ] Last Seen & Online status visibility (Everyone, Contacts, Except, Nobody).
  - [ ] Profile Photo visibility.
  - [ ] About bio visibility.
  - [ ] Read Receipts toggle (blue checkmarks on/off).
  - [ ] Default Disappearing Message timer settings.
  - [ ] Group Add Permission settings.

### Phase 11: Real-time Presence, Typing & Receipts
- [ ] Presence Setting (`presence.set`):
  - [ ] Broadcast `available` or `unavailable` presence to peers.
- [ ] Peer Presence Subscriptions (`presence.subscribe`, `presence.unsubscribe`):
  - [ ] Track online status and last seen timestamps of specific contacts.
  - [ ] Broadcast `event.presence` updates to IPC clients.
- [ ] Chat State / Typing Indicators (`chatstate.send`):
  - [ ] Send `composing` (typing text), `paused`, and `recording` (voice note) states.
  - [ ] Normalize and broadcast peer chat states (`event.chatstate`).
- [ ] Read Receipts & Delivery Status:
  - [ ] Delivery receipt tracking (Sent, Delivered to Device, Read/Opened).
  - [ ] Broadcast `event.receipt` to IPC clients.

### Phase 12: Voice & Video Calling Signaling (WebRTC Abstraction)
- [ ] Call Offer Notification (`event.call_offer`):
  - [ ] Catch incoming voice and video call signaling events from WhatsApp Web.
- [ ] Call Response Operations:
  - [ ] Accept call signaling (`call.accept`).
  - [ ] Reject call signaling (`call.reject`).
  - [ ] Terminate call (`call.terminate`).
- [ ] WebRTC SDP & ICE Candidate Signaling Proxy:
  - [ ] Expose WebRTC session parameters over IPC for external client audio/video renderers (e.g. GTK / WebRTC pipelines).

### Phase 13: History Sync & Global Full-Text Search
- [ ] Initial History Sync Engine:
  - [ ] Process historical chat threads, recent messages, and pushnames on new login.
  - [ ] Emit `event.history_sync_progress` with percent completion metrics.
- [ ] Global Full-Text Search (`search.query`):
  - [ ] SQLite FTS5 (Full-Text Search 5) index on message content, contact names, and chat titles.
  - [ ] Search query API with filtering by sender, date range, chat, and media type.

### Phase 14: Client Ecosystem & Desktop Integrations
- [ ] `whatsctl` Command-Line Interface Client:
  - [ ] Rust CLI tool in repository for interacting with `whatsd` over Unix socket.
  - [ ] Commands: `whatsctl status`, `whatsctl send`, `whatsctl listen`, `whatsctl pairing`.
- [ ] Emacs Lisp Client (`whatsd.el`):
  - [ ] Emacs package providing a full WhatsApp chat buffer, contact list, and notification mode.
- [ ] GTK Native Desktop Client (`whatsd-gtk`):
  - [ ] Native Linux GTK4 / Libadwaita user interface powered by `whatsd` socket IPC.
- [ ] Systemd Service & Desktop Integration:
  - [ ] Systemd user unit template (`whatsd.service`).
  - [ ] Desktop notifications (`libnotify` / `org.freedesktop.Notifications` bridge).
  - [ ] XDG Autostart support.

---

## 🛠️ IPC Command Specification Quick Reference

| Command String | Required Payload Fields | Description | Status |
| :--- | :--- | :--- | :--- |
| `daemon.ping` | `{}` | Verify daemon health | [x] Implemented |
| `daemon.version` | `{}` | Get daemon version & toolchain info | [x] Implemented |
| `daemon.status` | `{}` | Query uptime, state, and paths | [x] Implemented |
| `daemon.shutdown` | `{}` | Trigger graceful daemon shutdown | [x] Implemented |
| `session.status` | `{}` | Query WhatsApp session connection state | [x] Implemented |
| `session.connect` | `{}` | Connect session to WhatsApp servers | [ ] Planned |
| `session.disconnect` | `{}` | Disconnect active WebSocket session | [ ] Planned |
| `session.logout` | `{}` | Log out account & purge Signal keys | [ ] Planned |
| `session.pair_qr` | `{}` | Request QR code pairing event stream | [ ] Planned |
| `session.pair_code` | `{"phone_number": "15550199"}` | Request 8-digit pair code for login | [ ] Planned |
| `message.send_text` | `{"recipient": "...", "text": "..."}` | Send plain text message | [ ] Planned |
| `message.send` | `{"recipient": "...", "message": {...}}` | Send structured WA message | [ ] Planned |
| `message.react` | `{"chat": "...", "id": "...", "emoji": "👍"}` | Send message reaction | [ ] Planned |
| `message.edit` | `{"chat": "...", "id": "...", "text": "..."}` | Edit sent message | [ ] Planned |
| `message.revoke` | `{"chat": "...", "id": "...", "everyone": true}` | Revoke / delete message | [ ] Planned |
| `message.mark_read` | `{"chat": "...", "ids": ["..."]}` | Send read receipts | [ ] Planned |
| `message.list` | `{"chat": "...", "limit": 50}` | List paginated local messages | [ ] Planned |
| `message.get` | `{"id": "..."}` | Fetch single message by ID | [ ] Planned |
| `media.download` | `{"message_id": "...", "target_path": "..."}` | Download & decrypt media CDN file | [ ] Planned |
| `event.subscribe` | `{"events": ["..."]}` | Subscribe IPC client to live event stream | [x] Implemented |
| `event.unsubscribe` | `{"events": ["..."]}` | Unsubscribe IPC client from events | [x] Implemented |
| `chat.list` | `{"limit": 50}` | List active chat threads | [ ] Planned |
| `chat.get` | `{"chat": "..."}` | Fetch single chat thread metadata | [ ] Planned |
| `chat.archive` | `{"chat": "...", "archive": true}` | Archive / unarchive chat thread | [ ] Planned |
| `chat.pin` | `{"chat": "...", "pin": true}` | Pin / unpin chat thread | [ ] Planned |
| `chat.mute` | `{"chat": "...", "mute_until": 0}` | Mute / unmute chat notifications | [ ] Planned |
| `contact.list` | `{}` | List synced WhatsApp contacts | [ ] Planned |
| `contact.get` | `{"jid": "..."}` | Get contact profile details | [ ] Planned |
| `contact.profile_picture` | `{"jid": "..."}` | Get contact profile photo URL / binary | [ ] Planned |
| `presence.set` | `{"presence": "available"}` | Set user active presence | [ ] Planned |
| `presence.subscribe` | `{"jid": "..."}` | Track peer online status | [ ] Planned |
| `chatstate.send` | `{"chat": "...", "state": "composing"}` | Send typing / recording indicator | [ ] Planned |
| `group.create` | `{"name": "...", "participants": ["..."]}` | Create new group chat | [ ] Planned |
| `group.get` | `{"jid": "..."}` | Fetch group metadata & participants | [ ] Planned |
| `group.add_participants` | `{"jid": "...", "participants": ["..."]}` | Add members to group | [ ] Planned |
| `group.remove_participants` | `{"jid": "...", "participants": ["..."]}` | Remove members from group | [ ] Planned |

---

## 📌 Development Principles for Parity Contributions

1. **Keep Linux/XDG Behavior First-Class**: Default socket placement must strictly follow `$XDG_RUNTIME_DIR/whatsd/whatsd.sock` and state under `$XDG_STATE_HOME/whatsd/`.
2. **Stable Local IPC Contract**: Never expose upstream `whatsapp-rust` internals directly to IPC wire formats. All requests and responses must conform to the versioned `types/protocol.rs` structures.
3. **No Message Logging**: Do not log raw message contents or phone numbers in `tracing` logs unless executing under explicit debug flags.
4. **Durable Local Storage**: Always commit incoming messages to `whatsd.db` before acknowledging WhatsApp server delivery to ensure at-least-once message delivery to IPC clients.
5. **No Panics in Daemon Paths**: Use `anyhow::Result` at boundary APIs and explicit error handling in background runtime paths. Avoid `unwrap()` and `expect()`.
