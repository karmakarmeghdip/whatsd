package history

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	"whatsd/internal/types"
)

func setupTestDB(t *testing.T) (*Store, func()) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("failed to open in-memory sqlite: %v", err)
	}

	store, err := NewStore(db)
	if err != nil {
		_ = db.Close()
		t.Fatalf("failed to create history store: %v", err)
	}

	cleanup := func() {
		_ = db.Close()
	}
	return store, cleanup
}

func TestSaveMessageAndGetChats(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second)

	msg1 := types.MessageItem{
		ID:         "msg-1",
		Chat:       "15551234567@s.whatsapp.net",
		Sender:     "15551234567@s.whatsapp.net",
		SenderName: "Alice",
		Timestamp:  now,
		IsFromMe:   false,
		Text:       "Hello daemon",
		Status:     "received",
	}

	if err := s.SaveMessage(ctx, msg1); err != nil {
		t.Fatalf("SaveMessage failed: %v", err)
	}

	chats, err := s.GetChats(ctx, 10)
	if err != nil {
		t.Fatalf("GetChats failed: %v", err)
	}

	if len(chats) != 1 {
		t.Fatalf("expected 1 chat, got %d", len(chats))
	}

	c := chats[0]
	if c.JID != "15551234567@s.whatsapp.net" {
		t.Errorf("unexpected chat JID: %s", c.JID)
	}
	if c.UnreadCount != 1 {
		t.Errorf("expected unread count 1, got %d", c.UnreadCount)
	}
	if c.LastMessageText != "Hello daemon" {
		t.Errorf("unexpected last message text: %s", c.LastMessageText)
	}

	// Test MarkRead
	if err := s.MarkRead(ctx, c.JID); err != nil {
		t.Fatalf("MarkRead failed: %v", err)
	}

	chatsAfterRead, err := s.GetChats(ctx, 10)
	if err != nil {
		t.Fatalf("GetChats after read failed: %v", err)
	}

	if chatsAfterRead[0].UnreadCount != 0 {
		t.Errorf("expected unread count 0 after mark read, got %d", chatsAfterRead[0].UnreadCount)
	}
}

func TestGetMessagesPagination(t *testing.T) {
	s, cleanup := setupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	baseTime := time.Now().Add(-10 * time.Minute).Truncate(time.Second)

	msg1 := types.MessageItem{
		ID:        "msg-1",
		Chat:      "user1@s.whatsapp.net",
		Sender:    "user1@s.whatsapp.net",
		Timestamp: baseTime,
		Text:      "First message",
	}
	msg2 := types.MessageItem{
		ID:        "msg-2",
		Chat:      "user1@s.whatsapp.net",
		Sender:    "user1@s.whatsapp.net",
		Timestamp: baseTime.Add(1 * time.Minute),
		Text:      "Second message",
	}

	_ = s.SaveMessage(ctx, msg1)
	_ = s.SaveMessage(ctx, msg2)

	msgs, err := s.GetMessages(ctx, "user1@s.whatsapp.net", 10, "")
	if err != nil {
		t.Fatalf("GetMessages failed: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
	if msgs[0].ID != "msg-2" {
		t.Errorf("expected newest message first, got %s", msgs[0].ID)
	}

	// Test pagination before "msg-2"
	paginated, err := s.GetMessages(ctx, "user1@s.whatsapp.net", 10, "msg-2")
	if err != nil {
		t.Fatalf("GetMessages before msg-2 failed: %v", err)
	}

	if len(paginated) != 1 {
		t.Fatalf("expected 1 paginated message, got %d", len(paginated))
	}
	if paginated[0].ID != "msg-1" {
		t.Errorf("expected msg-1, got %s", paginated[0].ID)
	}
}
