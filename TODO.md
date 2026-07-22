# whatsd - Feature Parity Roadmap & TODO List

This document outlines the phased roadmap for `whatsd` to achieve feature parity with WhatsApp Desktop. Each task is self-contained, designed to be implemented by an AI agent in a single turn, and independently testable using `whatsctl`.

Tasks are strictly prioritized: **Phase 1 & 2** cover daily critical messaging features, while **Phases 3 to 5** add rich group management, persistent history, and extended capabilities.

---

## Phase 1: Essential Daily Messaging

- [ ] **1. Media Message Receiving & Downloading**
  - **Goal**: Support receiving images, audio/voice notes, videos, documents, and stickers over IPC, with optional local file downloading.
  - **IPC Protocol**: Extend `message` event payload in `internal/types/types.go` to include `media_type` (`image`, `video`, `audio`, `document`, `sticker`), `caption`, `file_name`, `mime_type`, and `file_length`. Add IPC method `download_media` taking `message_id` and `chat_jid`, saving the decrypted payload to `~/.local/share/whatsd/media/` and returning the local file path.
  - **whatsmeow API**: Use `cli.Download(...)` or `cli.DownloadToFile(...)`.
  - **Verification**: Send an image or audio note to the account from a phone, verify `whatsctl listen` prints media metadata, and run `whatsctl download --id <msg_id> --chat <jid>` to verify local file creation.

- [ ] **2. Media Message Sending**
  - **Goal**: Allow sending images, audio, videos, and documents to contacts/groups.
  - **IPC Protocol**: Add `send_media` method to IPC. Params: `to` (JID), `media_type` (`image`|`video`|`audio`|`document`), `file_path` (local path), `caption` (optional string), `file_name` (optional string).
  - **whatsmeow API**: Upload file using `cli.Upload(...)`, construct appropriate `*waE2E.ImageMessage` / `*waE2E.DocumentMessage` etc., and dispatch with `cli.SendMessage(...)`.
  - **Verification**: `whatsctl send-media --to <jid> --file /path/to/img.png --caption "Test image"` sends the media successfully.

- [ ] **3. Read Receipts & Delivery Receipts**
  - **Goal**: Send read receipts when a client views a chat, and emit receipt events (`delivered`, `read`, `played`) when sent messages are read by recipients.
  - **IPC Protocol**: Add IPC method `mark_read` (params: `chat` JID, `message_ids` array). Add `receipt` push event notification containing `message_id`, `chat`, `sender`, `type` (`read`|`delivered`|`played`), and `timestamp`.
  - **whatsmeow API**: Use `cli.MarkRead(...)` and listen for `*events.Receipt`.
  - **Verification**: Call `whatsctl mark-read --chat <jid> --ids <msg_id>`, observe blue ticks on sender's WhatsApp app; send a message from `whatsctl send` and observe `receipt` event in `whatsctl listen` when recipient opens it.

- [ ] **4. Typing & Presence Indicators**
  - **Goal**: Send presence updates ("composing", "recording", "paused") and emit incoming presence events from contacts.
  - **IPC Protocol**: Add `send_presence` method (params: `chat` JID, `state` (`composing`|`recording`|`paused`)). Broadcast `presence` event (data: `sender` JID, `state`, `last_seen`).
  - **whatsmeow API**: `cli.SendChatPresence(chatJID, waTypes.PresenceComposing, waTypes.ChatPresenceMediaText)` and listen for `*events.Presence`.
  - **Verification**: `whatsctl presence --chat <jid> --state composing` shows "typing..." on phone; typing on phone emits presence event in `whatsctl listen`.

- [ ] **5. Message Quoting / Replies & Emoji Reactions**
  - **Goal**: Support replying to specific messages and sending/receiving emoji reactions.
  - **IPC Protocol**: Extend `send_message` with `reply_to_id` parameter. Add IPC method `react_message` (params: `chat` JID, `message_id`, `emoji` string). Broadcast `reaction` event (data: `message_id`, `sender`, `emoji`).
  - **whatsmeow API**: Construct `waE2E.ContextInfo` with `StanzaID` & `Participant` for replies; use `cli.SendMessage` with `waE2E.ReactionMessage` for reactions. Listen for `*events.Message` containing `ReactionMessage`.
  - **Verification**: `whatsctl react --chat <jid> --id <msg_id> --emoji "👍"` adds reaction to target message; reacting on phone broadcasts reaction event in `whatsctl listen`.

---

## Phase 2: Message & Chat Lifecycle Management

- [ ] **6. Message Editing & Revocation (Delete for Everyone / Delete for Me)**
  - **Goal**: Support editing sent text messages and revoking/deleting messages.
  - **IPC Protocol**: Add `edit_message` method (params: `chat` JID, `message_id`, `new_text`). Add `revoke_message` method (params: `chat` JID, `message_id`). Broadcast `message_edit` and `message_revoke` push events.
  - **whatsmeow API**: Use `cli.BuildEdit(chatJID, msgID, newTextMsg)` and `cli.BuildRevoke(chatJID, senderJID, msgID)`.
  - **Verification**: `whatsctl edit --chat <jid> --id <msg_id> --text "Edited text"` updates message on recipient's device; `whatsctl revoke --chat <jid> --id <msg_id>` deletes message for everyone.

- [ ] **7. Local Message History & Chat Persistence in SQLite**
  - **Goal**: Maintain local message and chat state in SQLite so clients can query message history without relying solely on real-time WebSocket events.
  - **IPC Protocol**: Add IPC method `get_chats` (returns list of chats with last message, unread count) and `get_messages` (params: `chat` JID, `limit` int, `before_id` string).
  - **Storage Architecture**: Expand `internal/store` schema or add `history_store` package to record incoming/outgoing messages and update chat metadata transactionally.
  - **Verification**: Send/receive messages, restart `whatsd`, run `whatsctl chats` and `whatsctl history --chat <jid>` to verify past messages persist across daemon restarts.

- [ ] **8. History Sync Processing**
  - **Goal**: Handle initial WhatsApp history synchronization data (recent chats, contact names, past messages) when pairing a new device.
  - **IPC Protocol**: Emit `history_sync_progress` event (data: `progress_percent`, `sync_type`).
  - **whatsmeow API**: Handle `*events.HistorySync` events from `whatsmeow` and ingest initial message batches into local SQLite store.
  - **Verification**: Pair daemon via QR, verify incoming `history_sync_progress` events, and confirm historical chats are populated in `whatsctl chats`.

---

## Phase 3: Group & Contact Management

- [ ] **9. Group Info, Member List & Group Management**
  - **Goal**: Inspect group metadata, list participants, create groups, and manage members (add/remove/promote/demote).
  - **IPC Protocol**:
    - `get_group_info` (params: `group_jid` -> returns title, topic, owner, members, roles).
    - `create_group` (params: `title`, `participants` array -> returns `group_jid`).
    - `update_group_members` (params: `group_jid`, `action` (`add`|`remove`|`promote`|`demote`), `participants` array).
  - **whatsmeow API**: `cli.GetGroupInfo(...)`, `cli.CreateGroup(...)`, `cli.UpdateGroupParticipants(...)`.
  - **Verification**: `whatsctl group-info --jid <group_jid>` prints member list and roles; test adding/removing members via `whatsctl`.

- [ ] **10. Contact Info & Profile Picture Fetching**
  - **Goal**: Fetch contact details, status text, and profile picture URLs or binary images.
  - **IPC Protocol**: Add `get_contact` (params: `jid`) and `get_profile_picture` (params: `jid`, `preview` bool -> returns image file path or URL).
  - **whatsmeow API**: `cli.IsOnWhatsApp(...)`, `cli.GetUserInfo(...)`, `cli.GetProfilePictureInfo(...)`.
  - **Verification**: `whatsctl contact --jid <jid>` returns contact status; `whatsctl avatar --jid <jid>` downloads and prints path to avatar image.

- [ ] **11. App State Sync (Mute, Pin, Archive Chats)**
  - **Goal**: Support muting/unmuting chats, pinning/unpinning chats, and archiving/unarchiving chats synced across devices.
  - **IPC Protocol**: Add `set_chat_state` (params: `chat` JID, `action` (`mute`|`unmute`|`pin`|`unpin`|`archive`|`unarchive`), `mute_duration` int).
  - **whatsmeow API**: Use `cli.SendAppState(...)` / appstate patches for `Mute`, `Pin`, `Archive`.
  - **Verification**: `whatsctl chat-state --chat <jid> --action mute --duration 8h` mutes chat on both local daemon and mobile app.

---

## Phase 4: Extended WhatsApp Features

- [ ] **12. Status / Stories Viewing & Posting**
  - **Goal**: Receive status/story updates from contacts and post text/media status updates.
  - **IPC Protocol**: Add `get_statuses` (returns recent status posts) and `post_status` (params: `text` or `file_path`). Broadcast `status_update` event.
  - **whatsmeow API**: Send messages to `status@broadcast` JID.
  - **Verification**: Posting status via `whatsctl post-status --text "Hello"` displays status on phone contacts' feeds.

- [ ] **13. Newsletter / Channels Support**
  - **Goal**: Follow channels, read channel updates, and list subscribed newsletters.
  - **IPC Protocol**: `get_newsletters`, `follow_newsletter` (params: `newsletter_jid`), `get_newsletter_messages` (params: `newsletter_jid`).
  - **whatsmeow API**: `cli.GetNewsletterInfo(...)`, `cli.FollowNewsletter(...)`, `cli.GetNewsletterMessages(...)`.
  - **Verification**: `whatsctl newsletters` lists followed channels and recent posts.

- [ ] **14. Blocklist & Privacy Settings Management**
  - **Goal**: View blocked contacts, block/unblock contacts, and configure privacy settings (last seen, profile photo, read receipts).
  - **IPC Protocol**: `get_blocklist`, `block_contact` (params: `jid`, `action` (`block`|`unblock`)), `get_privacy_settings`, `set_privacy_setting`.
  - **whatsmeow API**: `cli.GetBlocklist()`, `cli.UpdateBlocklist()`, `cli.GetPrivacySettings()`, `cli.SetPrivacySetting()`.
  - **Verification**: `whatsctl blocklist` lists blocked users; blocking/unblocking updates server state immediately.

---

## Phase 5: Reliability & Performance Optimization

- [ ] **15. Outbox Queue & Offline Message Resilience**
  - **Goal**: Queue outbound messages in SQLite when disconnected or offline, automatically retrying delivery upon reconnection.
  - **IPC Protocol**: Add status field `queued` for `send_message` when offline, emit `message_delivered_from_outbox` event when flushed.
  - **Storage Architecture**: Add `outbox` table in `internal/store` to track pending message payloads and retry attempts.
  - **Verification**: Disconnect internet, run `whatsctl send`, reconnect internet, verify message is delivered automatically.
