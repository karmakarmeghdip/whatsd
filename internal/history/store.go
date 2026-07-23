package history

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"whatsd/internal/types"
)

// Store handles persistent custom storage for messages and chats in SQLite.
type Store struct {
	db *sql.DB
}

// NewStore initializes custom tables and indexes for message history and chat persistence.
func NewStore(db *sql.DB) (*Store, error) {
	s := &Store{db: db}
	if err := s.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize history schema: %w", err)
	}
	return s, nil
}

func (s *Store) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS whatsmeow_contacts (
		our_jid TEXT,
		their_jid TEXT,
		first_name TEXT,
		full_name TEXT,
		push_name TEXT,
		business_name TEXT,
		redacted_phone TEXT,
		PRIMARY KEY (our_jid, their_jid)
	);

	CREATE TABLE IF NOT EXISTS whatsmeow_chat_settings (
		our_jid TEXT,
		chat_jid TEXT,
		muted_until BIGINT NOT NULL DEFAULT 0,
		pinned BOOLEAN NOT NULL DEFAULT false,
		archived BOOLEAN NOT NULL DEFAULT false,
		PRIMARY KEY (our_jid, chat_jid)
	);

	CREATE TABLE IF NOT EXISTS whatsd_chats (
		jid TEXT PRIMARY KEY,
		name TEXT NOT NULL DEFAULT '',
		last_message_id TEXT NOT NULL DEFAULT '',
		last_message_text TEXT NOT NULL DEFAULT '',
		last_message_timestamp DATETIME,
		unread_count INTEGER NOT NULL DEFAULT 0,
		updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS whatsd_messages (
		id TEXT PRIMARY KEY,
		chat_jid TEXT NOT NULL,
		sender_jid TEXT NOT NULL,
		sender_name TEXT NOT NULL DEFAULT '',
		timestamp DATETIME NOT NULL,
		is_from_me BOOLEAN NOT NULL DEFAULT 0,
		is_group BOOLEAN NOT NULL DEFAULT 0,
		text TEXT NOT NULL DEFAULT '',
		media_type TEXT NOT NULL DEFAULT '',
		media_path TEXT NOT NULL DEFAULT '',
		reply_to_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL DEFAULT 'received',
		is_edited BOOLEAN NOT NULL DEFAULT 0,
		is_revoked BOOLEAN NOT NULL DEFAULT 0,
		caption TEXT NOT NULL DEFAULT '',
		file_name TEXT NOT NULL DEFAULT '',
		mime_type TEXT NOT NULL DEFAULT '',
		file_length INTEGER NOT NULL DEFAULT 0,
		raw_message BLOB,
		created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(chat_jid) REFERENCES whatsd_chats(jid) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_whatsd_messages_chat_ts ON whatsd_messages(chat_jid, timestamp DESC);
	CREATE INDEX IF NOT EXISTS idx_whatsd_chats_updated ON whatsd_chats(updated_at DESC);
	`
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("executing history schema sql failed: %w", err)
	}

	// Migrations (ignore errors if columns already exist)
	_, _ = s.db.Exec("ALTER TABLE whatsd_messages ADD COLUMN caption TEXT NOT NULL DEFAULT ''")
	_, _ = s.db.Exec("ALTER TABLE whatsd_messages ADD COLUMN file_name TEXT NOT NULL DEFAULT ''")
	_, _ = s.db.Exec("ALTER TABLE whatsd_messages ADD COLUMN mime_type TEXT NOT NULL DEFAULT ''")
	_, _ = s.db.Exec("ALTER TABLE whatsd_messages ADD COLUMN file_length INTEGER NOT NULL DEFAULT 0")
	_, _ = s.db.Exec("ALTER TABLE whatsd_messages ADD COLUMN raw_message BLOB")

	return nil
}

// SaveMessage stores an incoming or outgoing message and updates the corresponding chat entry.
func (s *Store) SaveMessage(ctx context.Context, msg types.MessageItem) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	upsertChatSQL := `
	INSERT INTO whatsd_chats (jid, name, last_message_id, last_message_text, last_message_timestamp, unread_count, updated_at)
	VALUES (?, ?, ?, ?, ?, CASE WHEN ? = 0 THEN 1 ELSE 0 END, ?)
	ON CONFLICT(jid) DO UPDATE SET
		last_message_id = excluded.last_message_id,
		last_message_text = excluded.last_message_text,
		last_message_timestamp = excluded.last_message_timestamp,
		unread_count = CASE 
			WHEN ? = 0 THEN whatsd_chats.unread_count + 1 
			ELSE whatsd_chats.unread_count 
		END,
		updated_at = excluded.updated_at,
		name = CASE WHEN excluded.name != '' THEN excluded.name ELSE whatsd_chats.name END;
	`
	isFromMeInt := 0
	if msg.IsFromMe {
		isFromMeInt = 1
	}

	_, err = tx.ExecContext(ctx, upsertChatSQL,
		msg.Chat, msg.SenderName, msg.ID, msg.Text, msg.Timestamp, isFromMeInt, msg.Timestamp, isFromMeInt,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert chat record: %w", err)
	}

	insertMsgSQL := `
	INSERT INTO whatsd_messages (
		id, chat_jid, sender_jid, sender_name, timestamp, is_from_me, is_group,
		text, media_type, media_path, reply_to_id, status, is_edited, is_revoked,
		caption, file_name, mime_type, file_length, raw_message
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		text = excluded.text,
		status = excluded.status,
		is_edited = excluded.is_edited,
		is_revoked = excluded.is_revoked,
		media_path = excluded.media_path;
	`
	_, err = tx.ExecContext(ctx, insertMsgSQL,
		msg.ID, msg.Chat, msg.Sender, msg.SenderName, msg.Timestamp,
		msg.IsFromMe, msg.IsGroup, msg.Text, msg.MediaType, msg.MediaPath,
		msg.ReplyToID, msg.Status, msg.IsEdited, msg.IsRevoked,
		msg.Caption, msg.FileName, msg.MimeType, msg.FileLength, msg.RawMessage,
	)
	if err != nil {
		return fmt.Errorf("failed to insert message record: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetChats returns a list of recent active chats joined with whatsmeow metadata.
func (s *Store) GetChats(ctx context.Context, limit int) ([]types.ChatItem, error) {
	if limit <= 0 {
		limit = 50
	}

	query := `
	SELECT 
		c.jid, 
		COALESCE(NULLIF(c.name, ''), NULLIF(con.full_name, ''), NULLIF(con.push_name, ''), NULLIF(con.first_name, ''), c.jid) AS chat_name,
		c.last_message_id, 
		c.last_message_text, 
		c.last_message_timestamp, 
		c.unread_count, 
		c.updated_at,
		COALESCE(cs.muted_until > ?, 0) AS is_muted,
		COALESCE(cs.pinned, 0) AS is_pinned,
		COALESCE(cs.archived, 0) AS is_archived
	FROM whatsd_chats c
	LEFT JOIN whatsmeow_contacts con ON con.their_jid = c.jid
	LEFT JOIN whatsmeow_chat_settings cs ON cs.chat_jid = c.jid
	ORDER BY is_pinned DESC, c.updated_at DESC
	LIMIT ?;
	`

	nowUnix := time.Now().Unix()
	rows, err := s.db.QueryContext(ctx, query, nowUnix, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query chats: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var chats []types.ChatItem
	for rows.Next() {
		var chat types.ChatItem
		var lastMsgTS sql.NullTime
		var isMuted, isPinned, isArchived bool

		err := rows.Scan(
			&chat.JID, &chat.Name, &chat.LastMessageID, &chat.LastMessageText,
			&lastMsgTS, &chat.UnreadCount, &chat.UpdatedAt,
			&isMuted, &isPinned, &isArchived,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan chat row: %w", err)
		}
		if lastMsgTS.Valid {
			chat.LastMessageTimestamp = lastMsgTS.Time
		}
		chat.IsMuted = isMuted
		chat.IsPinned = isPinned
		chat.IsArchived = isArchived

		chats = append(chats, chat)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in GetChats: %w", err)
	}

	return chats, nil
}

// GetMessages queries message history for a specific chat JID.
func (s *Store) GetMessages(ctx context.Context, chatJID string, limit int, beforeID string) ([]types.MessageItem, error) {
	if limit <= 0 {
		limit = 50
	}

	var rows *sql.Rows
	var err error

	if beforeID != "" {
		query := `
		SELECT id, chat_jid, sender_jid, sender_name, timestamp, is_from_me, is_group,
		       text, media_type, media_path, reply_to_id, status, is_edited, is_revoked,
		       caption, file_name, mime_type, file_length, raw_message
		FROM whatsd_messages
		WHERE chat_jid = ? AND timestamp < (SELECT timestamp FROM whatsd_messages WHERE id = ?)
		ORDER BY timestamp DESC
		LIMIT ?;
		`
		rows, err = s.db.QueryContext(ctx, query, chatJID, beforeID, limit)
	} else {
		query := `
		SELECT id, chat_jid, sender_jid, sender_name, timestamp, is_from_me, is_group,
		       text, media_type, media_path, reply_to_id, status, is_edited, is_revoked,
		       caption, file_name, mime_type, file_length, raw_message
		FROM whatsd_messages
		WHERE chat_jid = ?
		ORDER BY timestamp DESC
		LIMIT ?;
		`
		rows, err = s.db.QueryContext(ctx, query, chatJID, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to query messages: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var msgs []types.MessageItem
	for rows.Next() {
		var m types.MessageItem
		err := rows.Scan(
			&m.ID, &m.Chat, &m.Sender, &m.SenderName, &m.Timestamp, &m.IsFromMe, &m.IsGroup,
			&m.Text, &m.MediaType, &m.MediaPath, &m.ReplyToID, &m.Status, &m.IsEdited, &m.IsRevoked,
			&m.Caption, &m.FileName, &m.MimeType, &m.FileLength, &m.RawMessage,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message row: %w", err)
		}
		msgs = append(msgs, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error in GetMessages: %w", err)
	}

	return msgs, nil
}

// MarkRead resets the unread count for a chat and updates unread message status.
func (s *Store) MarkRead(ctx context.Context, chatJID string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start mark read transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `UPDATE whatsd_chats SET unread_count = 0 WHERE jid = ?`, chatJID)
	if err != nil {
		return fmt.Errorf("failed to reset chat unread count: %w", err)
	}

	_, err = tx.ExecContext(ctx, `UPDATE whatsd_messages SET status = 'read' WHERE chat_jid = ? AND is_from_me = 0 AND status != 'read'`, chatJID)
	if err != nil {
		return fmt.Errorf("failed to update message read status: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit mark read transaction: %w", err)
	}

	slog.Info("marked chat as read in history store", "chat", chatJID)
	return nil
}

// GetRawMessage fetches the raw protobuf message bytes for a specific message ID.
func (s *Store) GetRawMessage(ctx context.Context, messageID string) ([]byte, error) {
	var rawMsg []byte
	err := s.db.QueryRowContext(ctx, "SELECT raw_message FROM whatsd_messages WHERE id = ?", messageID).Scan(&rawMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch raw message for id %s: %w", messageID, err)
	}
	return rawMsg, nil
}

// UpdateMediaPath sets the downloaded media path for a message.
func (s *Store) UpdateMediaPath(ctx context.Context, messageID string, mediaPath string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE whatsd_messages SET media_path = ? WHERE id = ?", mediaPath, messageID)
	if err != nil {
		return fmt.Errorf("failed to update media path: %w", err)
	}
	return nil
}

// UpdateMessageText updates a message's text and marks it as edited.
func (s *Store) UpdateMessageText(ctx context.Context, messageID string, newText string, isEdited bool) error {
	isEditedInt := 0
	if isEdited {
		isEditedInt = 1
	}
	_, err := s.db.ExecContext(ctx, "UPDATE whatsd_messages SET text = ?, is_edited = ? WHERE id = ?", newText, isEditedInt, messageID)
	if err != nil {
		return fmt.Errorf("failed to update message text: %w", err)
	}
	return nil
}

// MarkRevoked marks a message as revoked.
func (s *Store) MarkRevoked(ctx context.Context, messageID string) error {
	_, err := s.db.ExecContext(ctx, "UPDATE whatsd_messages SET is_revoked = 1, text = '' WHERE id = ?", messageID)
	if err != nil {
		return fmt.Errorf("failed to mark message revoked: %w", err)
	}
	return nil
}

// GetMessageSender returns the sender JID for a given message ID.
func (s *Store) GetMessageSender(ctx context.Context, messageID string) (string, error) {
	var sender string
	err := s.db.QueryRowContext(ctx, "SELECT sender_jid FROM whatsd_messages WHERE id = ?", messageID).Scan(&sender)
	return sender, err
}

// GetContactsFromChats fetches contacts recorded from chat history.
func (s *Store) GetContactsFromChats(ctx context.Context) ([]types.ContactItem, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT jid, name FROM whatsd_chats WHERE name != ''")
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var items []types.ContactItem
	for rows.Next() {
		var jid, name string
		if err := rows.Scan(&jid, &name); err == nil {
			items = append(items, types.ContactItem{
				JID:      jid,
				FullName: name,
			})
		}
	}
	return items, nil
}

// SetChatState updates local chat settings (muted_until, pinned, archived) in SQLite.
func (s *Store) SetChatState(ctx context.Context, ourJID string, chatJID string, action string, muteDuration time.Duration) error {
	var mutedUntil int64 = 0
	if action == "mute" {
		if muteDuration > 0 {
			mutedUntil = time.Now().Add(muteDuration).Unix()
		} else {
			mutedUntil = time.Now().Add(100 * 365 * 24 * time.Hour).Unix()
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	upsertSQL := `
	INSERT INTO whatsmeow_chat_settings (our_jid, chat_jid, muted_until, pinned, archived)
	VALUES (?, ?, ?, ?, ?)
	ON CONFLICT(our_jid, chat_jid) DO UPDATE SET
		muted_until = CASE WHEN ? = 'mute' THEN excluded.muted_until WHEN ? = 'unmute' THEN 0 ELSE whatsmeow_chat_settings.muted_until END,
		pinned = CASE WHEN ? = 'pin' THEN 1 WHEN ? = 'unpin' THEN 0 ELSE whatsmeow_chat_settings.pinned END,
		archived = CASE WHEN ? = 'archive' THEN 1 WHEN ? = 'unarchive' THEN 0 ELSE whatsmeow_chat_settings.archived END;
	`

	var isPinned, isArchived bool
	if action == "pin" {
		isPinned = true
	}
	if action == "archive" {
		isArchived = true
	}

	_, err = tx.ExecContext(ctx, upsertSQL,
		ourJID, chatJID, mutedUntil, isPinned, isArchived,
		action, action,
		action, action,
		action, action,
	)
	if err != nil {
		return fmt.Errorf("failed to upsert chat settings: %w", err)
	}

	return tx.Commit()
}
