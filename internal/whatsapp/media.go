package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/protobuf/proto"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
)

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
