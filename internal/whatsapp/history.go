package whatsapp

import (
	"context"
	"fmt"

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
func (c *Client) MarkRead(ctx context.Context, chatJID string) error {
	c.mu.RLock()
	hStore := c.historyStore
	c.mu.RUnlock()

	if hStore == nil {
		return fmt.Errorf("history store is not initialized")
	}

	return hStore.MarkRead(ctx, chatJID)
}
