package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	waTypes "go.mau.fi/whatsmeow/types"

	"whatsd/internal/types"
)

// PostStatus posts a new text or media status update to status@broadcast.
func (c *Client) PostStatus(ctx context.Context, params types.PostStatusParams) (*types.SendMessageResult, error) {
	statusJID := waTypes.StatusBroadcastJID

	c.mu.RLock()
	hStore := c.historyStore
	c.mu.RUnlock()

	var msgID string
	var timestamp time.Time
	var err error

	if params.FilePath != "" {
		mediaType := params.MediaType
		if mediaType == "" {
			mediaType = "image"
		}
		msgID, timestamp, err = c.SendMedia(ctx, statusJID.String(), mediaType, params.FilePath, params.Caption, "")
	} else if params.Text != "" {
		msgID, timestamp, err = c.SendMessage(ctx, statusJID.String(), params.Text, "")
	} else {
		return nil, fmt.Errorf("must specify either text or file_path for status update")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to post status update: %w", err)
	}

	slog.Info("posted status update successfully", "id", msgID, "timestamp", timestamp)
	_ = hStore

	return &types.SendMessageResult{
		ID:        msgID,
		Timestamp: timestamp,
	}, nil
}

// GetStatuses queries recent status updates from contacts stored in message history.
func (c *Client) GetStatuses(ctx context.Context, limit int) ([]types.StatusItem, error) {
	c.mu.RLock()
	hStore := c.historyStore
	c.mu.RUnlock()

	if hStore == nil {
		return nil, fmt.Errorf("history store unavailable")
	}

	msgs, err := hStore.GetMessages(ctx, waTypes.StatusBroadcastJID.String(), limit, "")
	if err != nil {
		return nil, fmt.Errorf("failed to query status updates from store: %w", err)
	}

	var results []types.StatusItem
	for _, m := range msgs {
		results = append(results, types.StatusItem{
			ID:        m.ID,
			Sender:    m.Sender,
			PushName:  m.SenderName,
			Text:      m.Text,
			MediaType: m.MediaType,
			MediaPath: m.MediaPath,
			Timestamp: m.Timestamp,
		})
	}

	return results, nil
}
