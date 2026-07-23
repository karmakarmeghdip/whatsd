package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	waStore "go.mau.fi/whatsmeow/store"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"whatsd/internal/history"
	"whatsd/internal/types"
)

// Client wraps whatsmeow.Client with high-level daemon methods.
type Client struct {
	mu           sync.RWMutex
	waClient     *whatsmeow.Client
	historyStore *history.Store
	eventHandler func(evt types.EventNotification)
}

// NewClient constructs a new whatsmeow client wrapper.
func NewClient(deviceStore *waStore.Device, historyStore *history.Store, eventHandler func(evt types.EventNotification)) (*Client, error) {
	waLogger := waLog.Stdout("whatsmeow", "INFO", true)
	waClient := whatsmeow.NewClient(deviceStore, waLogger)

	c := &Client{
		waClient:     waClient,
		historyStore: historyStore,
		eventHandler: eventHandler,
	}

	waClient.AddEventHandler(c.handleWAEvent)
	return c, nil
}

// Connect connects to the WhatsApp WebSocket servers.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.waClient.IsConnected() {
		return nil
	}

	err := c.waClient.Connect()
	if err != nil {
		return fmt.Errorf("failed to connect to WhatsApp: %w", err)
	}

	slog.Info("connected to WhatsApp WebSocket")
	return nil
}

// Disconnect cleanly disconnects from WhatsApp servers.
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.waClient.IsConnected() {
		c.waClient.Disconnect()
		slog.Info("disconnected from WhatsApp WebSocket")
	}
}

// IsConnected returns whether the client is currently connected.
func (c *Client) IsConnected() bool {
	return c.waClient.IsConnected()
}

// IsLoggedIn returns true if a session identity is stored in the device store.
func (c *Client) IsLoggedIn() bool {
	return c.waClient.Store.ID != nil
}

// GetStatus returns the current daemon connection and identity status.
func (c *Client) GetStatus() types.StatusResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	res := types.StatusResult{
		Connected: c.waClient.IsConnected(),
		LoggedIn:  c.waClient.Store.ID != nil,
	}

	if c.waClient.Store.ID != nil {
		res.JID = c.waClient.Store.ID.String()
	}
	return res
}

// PairQR starts the QR code pairing process.
func (c *Client) PairQR(ctx context.Context, qrCallback func(evt string, code string, err error)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.waClient.Store.ID != nil {
		return fmt.Errorf("already logged in as %s", c.waClient.Store.ID.String())
	}

	qrChan, err := c.waClient.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get QR channel: %w", err)
	}

	if !c.waClient.IsConnected() {
		err := c.waClient.Connect()
		if err != nil {
			return fmt.Errorf("failed to connect for QR pairing: %w", err)
		}
	}

	go func() {
		for item := range qrChan {
			if item.Event == whatsmeow.QRChannelEventCode {
				qrCallback("code", item.Code, nil)
			} else if item.Event == whatsmeow.QRChannelEventError {
				qrCallback("error", "", item.Error)
			} else if item.Event == "success" {
				slog.Info("QR pairing completed successfully")
				qrCallback("success", "", nil)
			}
		}
	}()

	return nil
}

// PairPhone generates a phone pairing code for the specified phone number.
func (c *Client) PairPhone(ctx context.Context, phone string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.waClient.Store.ID != nil {
		return "", fmt.Errorf("already logged in as %s", c.waClient.Store.ID.String())
	}

	if !c.waClient.IsConnected() {
		err := c.waClient.Connect()
		if err != nil {
			return "", fmt.Errorf("failed to connect for phone pairing: %w", err)
		}
	}

	code, err := c.waClient.PairPhone(ctx, phone, true, whatsmeow.PairClientChrome, "Chrome (Linux)")
	if err != nil {
		return "", fmt.Errorf("failed to pair with phone %s: %w", phone, err)
	}

	slog.Info("generated phone pairing code", "phone", phone, "code", code)
	return code, nil
}

// SendMessage sends a text message to the given target JID.
func (c *Client) SendMessage(ctx context.Context, toJID string, text string) (string, time.Time, error) {
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

// Logout logs out the current session and clears state.
func (c *Client) Logout(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	err := c.waClient.Logout(ctx)
	if err != nil {
		return fmt.Errorf("failed to logout: %w", err)
	}
	slog.Info("logged out whatsmeow session")
	return nil
}

// DownloadMedia fetches and decrypts media for a given message ID, saving it locally.
func (c *Client) DownloadMedia(ctx context.Context, messageID string) (string, error) {
	if c.historyStore == nil {
		return "", fmt.Errorf("history store not available")
	}

	rawMsg, err := c.historyStore.GetRawMessage(ctx, messageID)
	if err != nil {
		return "", fmt.Errorf("failed to get raw message: %w", err)
	}

	if len(rawMsg) == 0 {
		return "", fmt.Errorf("no raw message found for id %s", messageID)
	}

	msg := &waE2E.Message{}
	if err := proto.Unmarshal(rawMsg, msg); err != nil {
		return "", fmt.Errorf("failed to unmarshal raw message: %w", err)
	}

	var dlMsg whatsmeow.DownloadableMessage
	mimeType := ""
	if msg.GetImageMessage() != nil {
		dlMsg = msg.GetImageMessage()
		mimeType = msg.GetImageMessage().GetMimetype()
	} else if msg.GetVideoMessage() != nil {
		dlMsg = msg.GetVideoMessage()
		mimeType = msg.GetVideoMessage().GetMimetype()
	} else if msg.GetAudioMessage() != nil {
		dlMsg = msg.GetAudioMessage()
		mimeType = msg.GetAudioMessage().GetMimetype()
	} else if msg.GetDocumentMessage() != nil {
		dlMsg = msg.GetDocumentMessage()
		mimeType = msg.GetDocumentMessage().GetMimetype()
	} else if msg.GetStickerMessage() != nil {
		dlMsg = msg.GetStickerMessage()
		mimeType = msg.GetStickerMessage().GetMimetype()
	} else {
		return "", fmt.Errorf("message %s does not contain a supported downloadable media", messageID)
	}

	data, err := c.waClient.Download(ctx, dlMsg)
	if err != nil {
		return "", fmt.Errorf("failed to download media: %w", err)
	}

	ext := ".bin"
	switch {
	case strings.Contains(mimeType, "image/jpeg"):
		ext = ".jpg"
	case strings.Contains(mimeType, "image/png"):
		ext = ".png"
	case strings.Contains(mimeType, "video/mp4"):
		ext = ".mp4"
	case strings.Contains(mimeType, "audio/ogg") || strings.Contains(mimeType, "audio/opus"):
		ext = ".ogg"
	case strings.Contains(mimeType, "audio/mpeg"):
		ext = ".mp3"
	case strings.Contains(mimeType, "application/pdf"):
		ext = ".pdf"
	case strings.Contains(mimeType, "image/webp"):
		ext = ".webp"
	}

	mediaDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "whatsd", "media")
	if err := os.MkdirAll(mediaDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create media directory: %w", err)
	}

	fileName := fmt.Sprintf("%s%s", messageID, ext)
	filePath := filepath.Join(mediaDir, fileName)

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		return "", fmt.Errorf("failed to save media file: %w", err)
	}

	if err := c.historyStore.UpdateMediaPath(ctx, messageID, filePath); err != nil {
		slog.Warn("failed to update media path in database", "id", messageID, "err", err)
	}

	return filePath, nil
}

func (c *Client) handleWAEvent(rawEvt any) {
	switch evt := rawEvt.(type) {
	case *events.Message:
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
