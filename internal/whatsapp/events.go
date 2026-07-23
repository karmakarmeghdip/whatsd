package whatsapp

import (
	"context"
	"log/slog"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow/proto/waE2E"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"whatsd/internal/types"
)

func (c *Client) handleWAEvent(rawEvt any) {
	switch evt := rawEvt.(type) {
	case *events.Message:
		if pm := evt.Message.GetProtocolMessage(); pm != nil {
			if pm.GetType() == waE2E.ProtocolMessage_REVOKE {
				revokedID := pm.GetKey().GetID()
				if c.historyStore != nil {
					_ = c.historyStore.MarkRevoked(context.Background(), revokedID)
				}
				if c.eventHandler != nil {
					c.eventHandler(types.EventNotification{
						Event: "message_revoke",
						Data:  map[string]string{"message_id": revokedID, "chat": evt.Info.Chat.String()},
					})
				}
				return
			} else if pm.GetType() == waE2E.ProtocolMessage_MESSAGE_EDIT {
				editedID := pm.GetKey().GetID()
				editedText := ""
				if pm.GetEditedMessage() != nil {
					if conv := pm.GetEditedMessage().GetConversation(); conv != "" {
						editedText = conv
					} else if ext := pm.GetEditedMessage().GetExtendedTextMessage(); ext != nil {
						editedText = ext.GetText()
					}
				}
				if c.historyStore != nil {
					_ = c.historyStore.UpdateMessageText(context.Background(), editedID, editedText, true)
				}
				if c.eventHandler != nil {
					c.eventHandler(types.EventNotification{
						Event: "message_edit",
						Data:  map[string]string{"message_id": editedID, "chat": evt.Info.Chat.String(), "new_text": editedText},
					})
				}
				return
			}
		}

		if evt.Message.GetReactionMessage() != nil {
			reaction := evt.Message.GetReactionMessage()
			if c.eventHandler != nil {
				c.eventHandler(types.EventNotification{
					Event: "reaction",
					Data: map[string]any{
						"message_id": reaction.GetKey().GetID(),
						"sender":     evt.Info.Sender.String(),
						"emoji":      reaction.GetText(),
					},
				})
			}
			return
		}

		data, ok := c.extractMessageData(evt)
		if ok {
			if c.historyStore != nil {
				rawMsg, _ := proto.Marshal(evt.Message)
				msgItem := types.MessageItem{
					ID:         data.ID,
					Chat:       data.Chat,
					Sender:     data.Sender,
					SenderName: data.PushName,
					Timestamp:  data.Timestamp,
					IsFromMe:   data.IsFromMe,
					IsGroup:    data.IsGroup,
					Text:       data.Text,
					MediaType:  data.MediaType,
					Caption:    data.Caption,
					FileName:   data.FileName,
					MimeType:   data.MimeType,
					FileLength: data.FileLength,
					RawMessage: rawMsg,
					Status:     "received",
				}
				if err := c.historyStore.SaveMessage(context.Background(), msgItem); err != nil {
					slog.Error("failed to persist incoming message", "id", data.ID, "err", err)
				}
			}
			if c.eventHandler != nil {
				c.eventHandler(types.EventNotification{
					Event: "message",
					Data:  data,
				})
			}
		}
	case *events.Receipt:
		if c.historyStore != nil {
			if evt.Type == waTypes.ReceiptTypeRead || evt.Type == waTypes.ReceiptTypeReadSelf {
				if evt.IsFromMe || evt.Type == waTypes.ReceiptTypeReadSelf {
					if err := c.historyStore.MarkRead(context.Background(), evt.Chat.String()); err != nil {
						slog.Error("failed to mark chat read from receipt", "chat", evt.Chat.String(), "err", err)
					}
				}
			}
		}
		receiptType := "delivered"
		switch evt.Type {
		case waTypes.ReceiptTypeRead, waTypes.ReceiptTypeReadSelf:
			receiptType = "read"
		case waTypes.ReceiptTypePlayed:
			receiptType = "played"
		}
		for _, id := range evt.MessageIDs {
			if c.eventHandler != nil {
				c.eventHandler(types.EventNotification{
					Event: "receipt",
					Data: map[string]any{
						"message_id": string(id),
						"chat":       evt.Chat.String(),
						"sender":     evt.Sender.String(),
						"type":       receiptType,
						"timestamp":  evt.Timestamp,
					},
				})
			}
		}
	case *events.Presence:
		if c.eventHandler != nil {
			state := "available"
			if evt.Unavailable {
				state = "unavailable"
			}
			c.eventHandler(types.EventNotification{
				Event: "presence",
				Data: map[string]any{
					"sender":    evt.From.String(),
					"state":     state,
					"last_seen": evt.LastSeen,
				},
			})
		}
	case *events.ChatPresence:
		if c.eventHandler != nil {
			state := string(evt.State)
			if evt.State == waTypes.ChatPresenceComposing && evt.Media == waTypes.ChatPresenceMediaAudio {
				state = "recording"
			}
			c.eventHandler(types.EventNotification{
				Event: "presence",
				Data: map[string]any{
					"sender": evt.Sender.String(),
					"chat":   evt.Chat.String(),
					"state":  state,
				},
			})
		}
	case *events.HistorySync:
		if c.historyStore != nil {
			go func(hsEvt *events.HistorySync) {
				err := c.historyStore.ProcessHistorySync(context.Background(), hsEvt.Data, c.waClient, func(percent int, syncType string) {
					if c.eventHandler != nil {
						c.eventHandler(types.EventNotification{
							Event: "history_sync_progress",
							Data: types.HistorySyncProgressEventData{
								ProgressPercent: percent,
								SyncType:        syncType,
							},
						})
					}
				})
				if err != nil {
					slog.Error("failed to process history sync", "err", err)
				}
			}(evt)
		}
	case *events.Connected:
		slog.Info("WhatsApp connection established")
		if c.eventHandler != nil {
			c.eventHandler(types.EventNotification{
				Event: "status",
				Data:  c.GetStatus(),
			})
		}
	case *events.LoggedOut:
		slog.Warn("WhatsApp session logged out from server")
		if c.eventHandler != nil {
			c.eventHandler(types.EventNotification{
				Event: "status",
				Data:  c.GetStatus(),
			})
		}
	}
}

func (c *Client) extractMessageData(evt *events.Message) (types.MessageEventData, bool) {
	if evt.Message == nil {
		return types.MessageEventData{}, false
	}

	text := ""
	mediaType := ""
	caption := ""
	fileName := ""
	mimeType := ""
	var fileLength uint64 = 0

	if conv := evt.Message.GetConversation(); conv != "" {
		text = conv
	} else if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		text = ext.GetText()
	} else if img := evt.Message.GetImageMessage(); img != nil {
		mediaType = "image"
		caption = img.GetCaption()
		mimeType = img.GetMimetype()
		fileLength = img.GetFileLength()
		text = caption
	} else if vid := evt.Message.GetVideoMessage(); vid != nil {
		mediaType = "video"
		if vid.GetGifPlayback() {
			mediaType = "gif"
		}
		caption = vid.GetCaption()
		mimeType = vid.GetMimetype()
		fileLength = vid.GetFileLength()
		text = caption
	} else if aud := evt.Message.GetAudioMessage(); aud != nil {
		mediaType = "audio"
		mimeType = aud.GetMimetype()
		fileLength = aud.GetFileLength()
	} else if doc := evt.Message.GetDocumentMessage(); doc != nil {
		mediaType = "document"
		caption = doc.GetCaption()
		fileName = doc.GetFileName()
		mimeType = doc.GetMimetype()
		fileLength = doc.GetFileLength()
		text = caption
	} else if sticker := evt.Message.GetStickerMessage(); sticker != nil {
		mediaType = "sticker"
		mimeType = sticker.GetMimetype()
		fileLength = sticker.GetFileLength()
	}

	if text == "" && mediaType == "" {
		return types.MessageEventData{}, false
	}

	return types.MessageEventData{
		ID:         evt.Info.ID,
		Chat:       evt.Info.Chat.String(),
		Sender:     evt.Info.Sender.String(),
		PushName:   evt.Info.PushName,
		Timestamp:  evt.Info.Timestamp,
		IsFromMe:   evt.Info.IsFromMe,
		IsGroup:    evt.Info.IsGroup,
		Text:       text,
		MediaType:  mediaType,
		Caption:    caption,
		FileName:   fileName,
		MimeType:   mimeType,
		FileLength: fileLength,
	}, true
}
