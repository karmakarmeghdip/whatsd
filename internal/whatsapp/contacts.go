package whatsapp

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"go.mau.fi/whatsmeow"
	waTypes "go.mau.fi/whatsmeow/types"

	"whatsd/internal/types"
)

// GetContacts queries synced device contacts and history chats matching an optional query string.
func (c *Client) GetContacts(ctx context.Context, query string) ([]types.ContactItem, error) {
	c.mu.RLock()
	waClient := c.waClient
	hStore := c.historyStore
	c.mu.RUnlock()

	contactMap := make(map[string]types.ContactItem)

	if waClient != nil && waClient.Store != nil && waClient.Store.Contacts != nil {
		contacts, err := waClient.Store.Contacts.GetAllContacts(ctx)
		if err == nil {
			for jid, info := range contacts {
				contactMap[jid.String()] = types.ContactItem{
					JID:          jid.String(),
					FirstName:    info.FirstName,
					FullName:     info.FullName,
					PushName:     info.PushName,
					BusinessName: info.BusinessName,
				}
			}
		}
	}

	if hStore != nil {
		chatContacts, err := hStore.GetContactsFromChats(ctx)
		if err == nil {
			for _, item := range chatContacts {
				if existing, found := contactMap[item.JID]; found {
					if existing.FullName == "" {
						existing.FullName = item.FullName
						contactMap[item.JID] = existing
					}
				} else {
					contactMap[item.JID] = item
				}
			}
		}
	}

	queryLower := strings.ToLower(query)
	var result []types.ContactItem
	for _, item := range contactMap {
		if queryLower != "" {
			match := strings.Contains(strings.ToLower(item.JID), queryLower) ||
				strings.Contains(strings.ToLower(item.FullName), queryLower) ||
				strings.Contains(strings.ToLower(item.FirstName), queryLower) ||
				strings.Contains(strings.ToLower(item.PushName), queryLower) ||
				strings.Contains(strings.ToLower(item.BusinessName), queryLower)
			if !match {
				continue
			}
		}
		result = append(result, item)
	}

	return result, nil
}

// GetContact queries contact details, WhatsApp registration, status text, and verified business info.
func (c *Client) GetContact(ctx context.Context, jidStr string) (*types.ContactInfoResult, error) {
	uJID, err := parseUserJID(jidStr)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	result := &types.ContactInfoResult{
		JID: uJID.String(),
	}

	if uJID.Server == waTypes.DefaultUserServer {
		onWA, err := waClient.IsOnWhatsApp(ctx, []string{"+" + uJID.User})
		if err == nil && len(onWA) > 0 {
			result.IsOnWhatsApp = onWA[0].IsIn
			if onWA[0].JID.String() != "" {
				result.JID = onWA[0].JID.String()
				uJID = onWA[0].JID
			}
		} else {
			result.IsOnWhatsApp = true
		}
	} else {
		result.IsOnWhatsApp = true
	}

	infoMap, err := waClient.GetUserInfo(ctx, []waTypes.JID{uJID})
	if err != nil {
		slog.Debug("failed to fetch user info for contact", "jid", uJID.String(), "err", err)
	} else if info, found := infoMap[uJID]; found {
		result.Status = info.Status
		result.PictureID = info.PictureID
		if info.VerifiedName != nil && info.VerifiedName.Details != nil {
			result.VerifiedName = info.VerifiedName.Details.GetVerifiedName()
		}
	}

	return result, nil
}

// GetProfilePicture downloads and returns the profile picture or group photo for a given JID.
func (c *Client) GetProfilePicture(ctx context.Context, jidStr string, preview bool) (*types.ProfilePictureResult, error) {
	targetJID, err := waTypes.ParseJID(jidStr)
	if err != nil {
		targetJID, err = parseUserJID(jidStr)
		if err != nil {
			return nil, err
		}
	}

	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	params := &whatsmeow.GetProfilePictureParams{
		Preview:     preview,
		IsCommunity: targetJID.Server == waTypes.GroupServer,
	}

	picInfo, err := waClient.GetProfilePictureInfo(ctx, targetJID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile picture info for %s: %w", targetJID.String(), err)
	}
	if picInfo == nil || picInfo.URL == "" {
		return nil, fmt.Errorf("no profile picture available for %s", targetJID.String())
	}

	result := &types.ProfilePictureResult{
		JID: targetJID.String(),
		URL: picInfo.URL,
		ID:  picInfo.ID,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, picInfo.URL, nil)
	if err != nil {
		slog.Warn("failed to create http request for profile picture download", "url", picInfo.URL, "err", err)
		return result, nil
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		slog.Warn("failed to download profile picture bytes", "url", picInfo.URL, "err", err)
		return result, nil
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		slog.Warn("unexpected HTTP status when downloading profile picture", "status", resp.Status)
		return result, nil
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("failed to read profile picture response body", "err", err)
		return result, nil
	}

	ext := ".jpg"
	contentType := resp.Header.Get("Content-Type")
	if strings.Contains(contentType, "image/png") {
		ext = ".png"
	}

	avatarDir := filepath.Join(os.Getenv("HOME"), ".local", "share", "whatsd", "avatars")
	if err := os.MkdirAll(avatarDir, 0700); err != nil {
		slog.Warn("failed to create avatar storage directory", "path", avatarDir, "err", err)
		return result, nil
	}

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(targetJID.String())))[:16]
	fileName := fmt.Sprintf("%s_%s%s", hash, picInfo.ID, ext)
	filePath := filepath.Join(avatarDir, fileName)

	if err := os.WriteFile(filePath, data, 0600); err != nil {
		slog.Warn("failed to write avatar file to disk", "path", filePath, "err", err)
		return result, nil
	}

	result.FilePath = filePath
	slog.Info("downloaded profile picture successfully", "jid", targetJID.String(), "path", filePath)
	return result, nil
}
