package whatsapp

import (
	"context"
	"fmt"
	"log/slog"

	"go.mau.fi/whatsmeow"
)

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
			switch item.Event {
			case whatsmeow.QRChannelEventCode:
				qrCallback("code", item.Code, nil)
			case whatsmeow.QRChannelEventError:
				qrCallback("error", "", item.Error)
			case "success":
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
