package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.mau.fi/whatsmeow/appstate"
	waTypes "go.mau.fi/whatsmeow/types"
)

// SetChatState applies app state synchronization patches to mute/unmute, pin/unpin, or archive/unarchive a chat.
func (c *Client) SetChatState(ctx context.Context, chatJID string, action string, muteDuration time.Duration) error {
	targetJID, err := waTypes.ParseJID(chatJID)
	if err != nil {
		targetJID, err = parseUserJID(chatJID)
		if err != nil {
			return err
		}
	}

	var patch appstate.PatchInfo
	actionLower := strings.ToLower(action)

	switch actionLower {
	case "mute":
		patch = appstate.BuildMute(targetJID, true, muteDuration)
	case "unmute":
		patch = appstate.BuildMute(targetJID, false, 0)
	case "pin":
		patch = appstate.BuildPin(targetJID, true)
	case "unpin":
		patch = appstate.BuildPin(targetJID, false)
	case "archive":
		patch = appstate.BuildArchive(targetJID, true, time.Time{}, nil)
	case "unarchive":
		patch = appstate.BuildArchive(targetJID, false, time.Time{}, nil)
	default:
		return fmt.Errorf("invalid chat state action %q (must be mute, unmute, pin, unpin, archive, or unarchive)", action)
	}

	c.mu.RLock()
	waClient := c.waClient
	hStore := c.historyStore
	c.mu.RUnlock()

	if err := waClient.SendAppState(ctx, patch); err != nil {
		return fmt.Errorf("failed to send app state patch for action %s on %s: %w", action, targetJID.String(), err)
	}

	if hStore != nil {
		myJID := ""
		if waClient.Store.ID != nil {
			myJID = waClient.Store.ID.String()
		}
		if err := hStore.SetChatState(ctx, myJID, targetJID.String(), actionLower, muteDuration); err != nil {
			slog.Warn("failed to update local chat settings in database", "chat", targetJID.String(), "err", err)
		}
	}

	slog.Info("applied chat state patch successfully", "chat", targetJID.String(), "action", actionLower)
	return nil
}
