package whatsapp

import (
	"context"
	"fmt"
	"time"

	waTypes "go.mau.fi/whatsmeow/types"
	"whatsd/internal/types"
)

// GetChats returns recent active conversations from the history store.
func (c *Client) GetChats(ctx context.Context, limit int) ([]types.ChatItem, error) {
	c.mu.RLock()
	hStore := c.historyStore
	c.mu.RUnlock()

	if hStore == nil {
		return nil, fmt.Errorf("history store is not initialized")
	}

	return hStore.GetChats(ctx, limit)
}

// GetMessages queries historical messages for a chat JID from the history store.
func (c *Client) GetMessages(ctx context.Context, chatJID string, limit int, beforeID string) ([]types.MessageItem, error) {
	c.mu.RLock()
	hStore := c.historyStore
	c.mu.RUnlock()

	if hStore == nil {
		return nil, fmt.Errorf("history store is not initialized")
	}

	return hStore.GetMessages(ctx, chatJID, limit, beforeID)
}

// MarkRead resets the unread count and marks messages as read for a chat in the history store.
func (c *Client) MarkRead(ctx context.Context, chatJID string, messageIDs []string) error {
	recipient, err := waTypes.ParseJID(chatJID)
	if err != nil {
		recipient, err = waTypes.ParseJID(chatJID + "@s.whatsapp.net")
		if err != nil {
			return fmt.Errorf("invalid target JID %q: %w", chatJID, err)
		}
	}

	c.mu.RLock()
	hStore := c.historyStore
	c.mu.RUnlock()

	if len(messageIDs) > 0 {
		for _, id := range messageIDs {
			senderJID := recipient
			if hStore != nil {
				if senderStr, err := hStore.GetMessageSender(ctx, id); err == nil && senderStr != "" {
					senderJID, _ = waTypes.ParseJID(senderStr)
				}
			}
			_ = c.waClient.MarkRead(ctx, []string{id}, time.Now(), recipient, senderJID)
		}
	}

	if hStore != nil {
		return hStore.MarkRead(ctx, chatJID)
	}

	return nil
}
