package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	waTypes "go.mau.fi/whatsmeow/types"

	"whatsd/internal/types"
)

// SendMessage sends a text message to the given target JID.
func (c *Client) SendMessage(ctx context.Context, toJID string, text string, replyToID string) (string, time.Time, error) {
	if text == "" {
		return "", time.Time{}, fmt.Errorf("message text cannot be empty")
	}

	recipient, err := waTypes.ParseJID(toJID)
	if err != nil {
		recipient, err = waTypes.ParseJID(toJID + "@s.whatsapp.net")
		if err != nil {
			return "", time.Time{}, fmt.Errorf("invalid target JID %q: %w", toJID, err)
		}
	}

	msg := &waE2E.Message{
		Conversation: proto.String(text),
	}

	if replyToID != "" && c.historyStore != nil {
		senderStr, err := c.historyStore.GetMessageSender(ctx, replyToID)
		if err == nil && senderStr != "" {
			msg = &waE2E.Message{
				ExtendedTextMessage: &waE2E.ExtendedTextMessage{
					Text: proto.String(text),
					ContextInfo: &waE2E.ContextInfo{
						StanzaID:    proto.String(replyToID),
						Participant: proto.String(senderStr),
					},
				},
			}
		}
	}

	resp, err := c.waClient.SendMessage(ctx, recipient, msg)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to send message to %s: %w", recipient.String(), err)
	}

	if c.historyStore != nil {
		myJID := ""
		if c.waClient.Store.ID != nil {
			myJID = c.waClient.Store.ID.String()
		}
		msgItem := types.MessageItem{
			ID:        resp.ID,
			Chat:      recipient.String(),
			Sender:    myJID,
			Timestamp: resp.Timestamp,
			IsFromMe:  true,
			IsGroup:   recipient.Server == waTypes.GroupServer,
			Text:      text,
			Status:    "sent",
		}
		if saveErr := c.historyStore.SaveMessage(ctx, msgItem); saveErr != nil {
			slog.Error("failed to save outgoing message to history", "id", resp.ID, "err", saveErr)
		}
	}

	slog.Info("sent text message", "to", recipient.String(), "id", resp.ID)
	return resp.ID, resp.Timestamp, nil
}

// SendMedia sends a media message to the given target JID.
func (c *Client) SendMedia(ctx context.Context, toJID string, mediaType string, filePath string, caption string, fileName string) (string, time.Time, error) {
	recipient, err := waTypes.ParseJID(toJID)
	if err != nil {
		recipient, err = waTypes.ParseJID(toJID + "@s.whatsapp.net")
		if err != nil {
			return "", time.Time{}, fmt.Errorf("invalid target JID %q: %w", toJID, err)
		}
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read media file: %w", err)
	}

	var waMediaType whatsmeow.MediaType
	var msg waE2E.Message
	mimeType := http.DetectContentType(data)

	switch mediaType {
	case "image":
		waMediaType = whatsmeow.MediaImage
	case "video":
		waMediaType = whatsmeow.MediaVideo
	case "audio":
		waMediaType = whatsmeow.MediaAudio
	case "document":
		waMediaType = whatsmeow.MediaDocument
	default:
		return "", time.Time{}, fmt.Errorf("unsupported media type: %s", mediaType)
	}

	resp, err := c.waClient.Upload(ctx, data, waMediaType)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to upload media: %w", err)
	}

	switch mediaType {
	case "image":
		msg.ImageMessage = &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}
	case "video":
		msg.VideoMessage = &waE2E.VideoMessage{
			Caption:       proto.String(caption),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}
	case "audio":
		msg.AudioMessage = &waE2E.AudioMessage{
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}
	case "document":
		if fileName == "" {
			fileName = filepath.Base(filePath)
		}
		msg.DocumentMessage = &waE2E.DocumentMessage{
			Caption:       proto.String(caption),
			FileName:      proto.String(fileName),
			Mimetype:      proto.String(mimeType),
			URL:           proto.String(resp.URL),
			DirectPath:    proto.String(resp.DirectPath),
			MediaKey:      resp.MediaKey,
			FileEncSHA256: resp.FileEncSHA256,
			FileSHA256:    resp.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(data))),
		}
	}

	sendResp, err := c.waClient.SendMessage(ctx, recipient, &msg)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to send media message: %w", err)
	}

	if c.historyStore != nil {
		myJID := ""
		if c.waClient.Store.ID != nil {
			myJID = c.waClient.Store.ID.String()
		}
		rawMsgBytes, _ := proto.Marshal(&msg)
		msgItem := types.MessageItem{
			ID:         sendResp.ID,
			Chat:       recipient.String(),
			Sender:     myJID,
			Timestamp:  sendResp.Timestamp,
			IsFromMe:   true,
			IsGroup:    recipient.Server == waTypes.GroupServer,
			Text:       caption,
			MediaType:  mediaType,
			MediaPath:  filePath,
			MimeType:   mimeType,
			FileLength: uint64(len(data)),
			FileName:   fileName,
			RawMessage: rawMsgBytes,
			Status:     "sent",
		}
		if saveErr := c.historyStore.SaveMessage(ctx, msgItem); saveErr != nil {
			slog.Error("failed to save outgoing media message to history", "id", sendResp.ID, "err", saveErr)
		}
	}

	slog.Info("sent media message", "to", recipient.String(), "id", sendResp.ID, "media_type", mediaType)
	return sendResp.ID, sendResp.Timestamp, nil
}

// EditMessage edits a previously sent text message.
func (c *Client) EditMessage(ctx context.Context, chatJID string, msgID string, newText string) error {
	recipient, err := waTypes.ParseJID(chatJID)
	if err != nil {
		recipient, err = waTypes.ParseJID(chatJID + "@s.whatsapp.net")
		if err != nil {
			return fmt.Errorf("invalid target JID %q: %w", chatJID, err)
		}
	}

	newMsg := &waE2E.Message{
		Conversation: proto.String(newText),
	}

	editMsg := c.waClient.BuildEdit(recipient, msgID, newMsg)

	_, err = c.waClient.SendMessage(ctx, recipient, editMsg)
	if err != nil {
		return fmt.Errorf("failed to send edit message: %w", err)
	}

	if c.historyStore != nil {
		if err := c.historyStore.UpdateMessageText(ctx, msgID, newText, true); err != nil {
			slog.Error("failed to update edited message in history", "id", msgID, "err", err)
		}
	}
	return nil
}

// RevokeMessage deletes a message for everyone.
func (c *Client) RevokeMessage(ctx context.Context, chatJID string, msgID string) error {
	recipient, err := waTypes.ParseJID(chatJID)
	if err != nil {
		recipient, err = waTypes.ParseJID(chatJID + "@s.whatsapp.net")
		if err != nil {
			return fmt.Errorf("invalid target JID %q: %w", chatJID, err)
		}
	}

	var senderJID waTypes.JID
	if c.waClient.Store.ID != nil {
		senderJID = *c.waClient.Store.ID
	} else {
		return fmt.Errorf("not logged in")
	}

	revokeMsg := c.waClient.BuildRevoke(recipient, senderJID, msgID)

	_, err = c.waClient.SendMessage(ctx, recipient, revokeMsg)
	if err != nil {
		return fmt.Errorf("failed to send revoke message: %w", err)
	}

	if c.historyStore != nil {
		if err := c.historyStore.MarkRevoked(ctx, msgID); err != nil {
			slog.Error("failed to mark message revoked in history", "id", msgID, "err", err)
		}
	}
	return nil
}

// ReactMessage sends an emoji reaction to a specific message.
func (c *Client) ReactMessage(ctx context.Context, chatJID string, msgID string, emoji string) error {
	recipient, err := waTypes.ParseJID(chatJID)
	if err != nil {
		recipient, err = waTypes.ParseJID(chatJID + "@s.whatsapp.net")
		if err != nil {
			return fmt.Errorf("invalid target JID %q: %w", chatJID, err)
		}
	}

	senderJID := recipient
	if c.historyStore != nil {
		senderStr, err := c.historyStore.GetMessageSender(ctx, msgID)
		if err == nil && senderStr != "" {
			senderJID, _ = waTypes.ParseJID(senderStr)
		}
	}

	msg := c.waClient.BuildReaction(recipient, senderJID, msgID, emoji)
	_, err = c.waClient.SendMessage(ctx, recipient, msg)
	return err
}

// SendPresence sends typing or recording presence to a chat.
func (c *Client) SendPresence(ctx context.Context, chatJID string, state string) error {
	recipient, err := waTypes.ParseJID(chatJID)
	if err != nil {
		recipient, err = waTypes.ParseJID(chatJID + "@s.whatsapp.net")
		if err != nil {
			return fmt.Errorf("invalid target JID %q: %w", chatJID, err)
		}
	}

	var waState waTypes.ChatPresence
	var waMedia = waTypes.ChatPresenceMediaText

	switch state {
	case "composing":
		waState = waTypes.ChatPresenceComposing
	case "recording":
		waState = waTypes.ChatPresenceComposing
		waMedia = waTypes.ChatPresenceMediaAudio
	case "paused":
		waState = waTypes.ChatPresencePaused
	default:
		return fmt.Errorf("invalid presence state: %s", state)
	}

	return c.waClient.SendChatPresence(ctx, recipient, waState, waMedia)
}
