package history

import (
	"context"
	"log/slog"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waHistorySync"
	waTypes "go.mau.fi/whatsmeow/types"

	"whatsd/internal/types"
)

// ProcessHistorySync parses and persists a HistorySync data payload from whatsmeow.
func (s *Store) ProcessHistorySync(ctx context.Context, data *waHistorySync.HistorySync, cli *whatsmeow.Client, progressCB func(percent int, syncType string)) error {
	if data == nil {
		return nil
	}

	syncTypeStr := data.GetSyncType().String()
	progressPercent := 0
	if data.Progress != nil {
		progressPercent = int(data.GetProgress())
	}

	slog.Info("processing history sync", "type", syncTypeStr, "conversations", len(data.Conversations), "progress", progressPercent)

	for _, conv := range data.Conversations {
		rawID := conv.GetID()
		if rawID == "" {
			continue
		}

		chatJID, err := waTypes.ParseJID(rawID)
		if err != nil {
			slog.Debug("failed to parse history sync chat JID", "id", rawID, "err", err)
			continue
		}

		chatName := conv.GetName()
		unreadCount := int(conv.GetUnreadCount())

		var lastMsgID, lastMsgText string
		var lastMsgTS int64

		for _, historyMsg := range conv.Messages {
			if historyMsg.GetMessage() == nil {
				continue
			}

			evt, err := cli.ParseWebMessage(chatJID, historyMsg.GetMessage())
			if err != nil || evt == nil || evt.Message == nil {
				continue
			}

			text := ""
			if convText := evt.Message.GetConversation(); convText != "" {
				text = convText
			} else if extText := evt.Message.GetExtendedTextMessage(); extText != nil {
				text = extText.GetText()
			}

			if text == "" {
				continue
			}

			msgItem := types.MessageItem{
				ID:         evt.Info.ID,
				Chat:       evt.Info.Chat.String(),
				Sender:     evt.Info.Sender.String(),
				SenderName: evt.Info.PushName,
				Timestamp:  evt.Info.Timestamp,
				IsFromMe:   evt.Info.IsFromMe,
				IsGroup:    evt.Info.IsGroup,
				Text:       text,
				Status:     "received",
			}

			_, err = s.db.ExecContext(ctx, `
				INSERT INTO whatsd_messages (
					id, chat_jid, sender_jid, sender_name, timestamp, is_from_me, is_group,
					text, status
				) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
				ON CONFLICT(id) DO UPDATE SET
					text = excluded.text,
					status = excluded.status;
			`, msgItem.ID, msgItem.Chat, msgItem.Sender, msgItem.SenderName, msgItem.Timestamp, msgItem.IsFromMe, msgItem.IsGroup, msgItem.Text, msgItem.Status)
			if err != nil {
				slog.Debug("failed to save history sync message", "id", msgItem.ID, "err", err)
			}

			if evt.Info.Timestamp.Unix() > lastMsgTS {
				lastMsgTS = evt.Info.Timestamp.Unix()
				lastMsgID = evt.Info.ID
				lastMsgText = text
			}
		}

		convTS := int64(conv.GetConversationTimestamp())
		if lastMsgTS == 0 {
			lastMsgTS = convTS
		}
		if convTS == 0 {
			convTS = lastMsgTS
		}

		if chatName != "" || unreadCount > 0 || lastMsgID != "" || convTS > 0 {
			_, err = s.db.ExecContext(ctx, `
				INSERT INTO whatsd_chats (jid, name, last_message_id, last_message_text, unread_count, updated_at, last_message_timestamp)
				VALUES (?, ?, ?, ?, ?, datetime(?, 'unixepoch'), datetime(?, 'unixepoch'))
				ON CONFLICT(jid) DO UPDATE SET
					name = CASE WHEN excluded.name != '' THEN excluded.name ELSE whatsd_chats.name END,
					last_message_id = CASE WHEN excluded.last_message_id != '' THEN excluded.last_message_id ELSE whatsd_chats.last_message_id END,
					last_message_text = CASE WHEN excluded.last_message_text != '' THEN excluded.last_message_text ELSE whatsd_chats.last_message_text END,
					unread_count = CASE WHEN excluded.unread_count > 0 THEN excluded.unread_count ELSE whatsd_chats.unread_count END,
					updated_at = excluded.updated_at,
					last_message_timestamp = excluded.last_message_timestamp;
			`, chatJID.String(), chatName, lastMsgID, lastMsgText, unreadCount, convTS, lastMsgTS)

			if err != nil {
				slog.Debug("failed to update history sync chat record", "jid", chatJID.String(), "err", err)
			}
		}
	}

	if progressCB != nil {
		progressCB(progressPercent, syncTypeStr)
	}

	slog.Info("completed history sync batch processing", "type", syncTypeStr)
	return nil
}
