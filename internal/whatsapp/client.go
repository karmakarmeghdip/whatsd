package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"go.mau.fi/whatsmeow"
	waStore "go.mau.fi/whatsmeow/store"
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
