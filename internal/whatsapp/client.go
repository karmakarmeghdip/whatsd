package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	waStore "go.mau.fi/whatsmeow/store"
	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	"whatsd/internal/types"
)

// Client wraps whatsmeow.Client with high-level daemon methods.
type Client struct {
	mu           sync.RWMutex
	waClient     *whatsmeow.Client
	eventHandler func(evt types.EventNotification)
}

// NewClient constructs a new whatsmeow client wrapper.
func NewClient(deviceStore *waStore.Device, eventHandler func(evt types.EventNotification)) (*Client, error) {
	waLogger := waLog.Stdout("whatsmeow", "INFO", true)
	waClient := whatsmeow.NewClient(deviceStore, waLogger)

	c := &Client{
		waClient:     waClient,
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

	slog.Info("sent text message", "to", recipient.String(), "id", resp.ID)
	return resp.ID, resp.Timestamp, nil
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

func (c *Client) handleWAEvent(rawEvt any) {
	switch evt := rawEvt.(type) {
	case *events.Message:
		data, ok := c.extractMessageData(evt)
		if ok && c.eventHandler != nil {
			c.eventHandler(types.EventNotification{
				Event: "message",
				Data:  data,
			})
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
	if conv := evt.Message.GetConversation(); conv != "" {
		text = conv
	} else if ext := evt.Message.GetExtendedTextMessage(); ext != nil {
		text = ext.GetText()
	}

	if text == "" {
		return types.MessageEventData{}, false
	}

	return types.MessageEventData{
		ID:        evt.Info.ID,
		Chat:      evt.Info.Chat.String(),
		Sender:    evt.Info.Sender.String(),
		PushName:  evt.Info.PushName,
		Timestamp: evt.Info.Timestamp,
		IsFromMe:  evt.Info.IsFromMe,
		IsGroup:   evt.Info.IsGroup,
		Text:      text,
	}, true
}
