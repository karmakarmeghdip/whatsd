package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	waTypes "go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"

	"whatsd/internal/types"
)

// GetBlocklist retrieves the user's blocked contact JID list.
func (c *Client) GetBlocklist(ctx context.Context) (*types.BlocklistResult, error) {
	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	blocklist, err := waClient.GetBlocklist(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch blocklist: %w", err)
	}

	var jids []string
	if blocklist != nil {
		for _, j := range blocklist.JIDs {
			jids = append(jids, j.String())
		}
	}

	return &types.BlocklistResult{JIDs: jids}, nil
}

// BlockContact blocks or unblocks a specific contact on WhatsApp.
func (c *Client) BlockContact(ctx context.Context, jidStr string, unblock bool) (*types.BlocklistResult, error) {
	uJID, err := parseUserJID(jidStr)
	if err != nil {
		return nil, err
	}

	action := events.BlocklistChangeActionBlock
	if unblock {
		action = events.BlocklistChangeActionUnblock
	}

	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	if uJID.Server == waTypes.DefaultUserServer && waClient != nil && waClient.Store != nil && waClient.Store.LIDs != nil {
		lid, err := waClient.Store.LIDs.GetLIDForPN(ctx, uJID)
		if err == nil && !lid.IsEmpty() {
			uJID = lid
		}
	}

	blocklist, err := waClient.UpdateBlocklist(ctx, uJID, action)
	if err != nil {
		return nil, fmt.Errorf("failed to update blocklist for %s (action=%s): %w", uJID.String(), action, err)
	}

	slog.Info("updated blocklist successfully", "jid", uJID.String(), "action", action)

	var jids []string
	if blocklist != nil {
		for _, j := range blocklist.JIDs {
			jids = append(jids, j.String())
		}
	}

	return &types.BlocklistResult{JIDs: jids}, nil
}

// GetPrivacySettings fetches the user's current WhatsApp privacy configuration.
func (c *Client) GetPrivacySettings(ctx context.Context) (*types.PrivacySettingsResult, error) {
	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	settings, err := waClient.TryFetchPrivacySettings(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch privacy settings: %w", err)
	}

	return &types.PrivacySettingsResult{
		GroupAdd:     string(settings.GroupAdd),
		LastSeen:     string(settings.LastSeen),
		Status:       string(settings.Status),
		Profile:      string(settings.Profile),
		ReadReceipts: string(settings.ReadReceipts),
		Online:       string(settings.Online),
		CallAdd:      string(settings.CallAdd),
	}, nil
}

// SetPrivacySetting modifies a specific privacy setting (last_seen, profile_photo, read_receipts, etc.).
func (c *Client) SetPrivacySetting(ctx context.Context, settingStr string, valueStr string) (*types.PrivacySettingsResult, error) {
	var name waTypes.PrivacySettingType
	switch strings.ToLower(settingStr) {
	case "group_add", "groupadd":
		name = waTypes.PrivacySettingTypeGroupAdd
	case "last_seen", "lastseen":
		name = waTypes.PrivacySettingTypeLastSeen
	case "status":
		name = waTypes.PrivacySettingTypeStatus
	case "profile", "profile_photo", "profilephoto":
		name = waTypes.PrivacySettingTypeProfile
	case "read_receipts", "readreceipts":
		name = waTypes.PrivacySettingTypeReadReceipts
	case "online":
		name = waTypes.PrivacySettingTypeOnline
	case "call_add", "calladd":
		name = waTypes.PrivacySettingTypeCallAdd
	default:
		return nil, fmt.Errorf("invalid privacy setting %q", settingStr)
	}

	var value waTypes.PrivacySetting
	switch strings.ToLower(valueStr) {
	case "all":
		value = waTypes.PrivacySettingAll
	case "contacts":
		value = waTypes.PrivacySettingContacts
	case "contact_blacklist", "contactblacklist":
		value = waTypes.PrivacySettingContactBlacklist
	case "none":
		value = waTypes.PrivacySettingNone
	case "match_last_seen", "matchlastseen":
		value = waTypes.PrivacySettingMatchLastSeen
	default:
		return nil, fmt.Errorf("invalid privacy value %q", valueStr)
	}

	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	newSettings, err := waClient.SetPrivacySetting(ctx, name, value)
	if err != nil {
		return nil, fmt.Errorf("failed to set privacy setting %s to %s: %w", settingStr, valueStr, err)
	}

	slog.Info("updated privacy setting successfully", "setting", settingStr, "value", valueStr)

	return &types.PrivacySettingsResult{
		GroupAdd:     string(newSettings.GroupAdd),
		LastSeen:     string(newSettings.LastSeen),
		Status:       string(newSettings.Status),
		Profile:      string(newSettings.Profile),
		ReadReceipts: string(newSettings.ReadReceipts),
		Online:       string(newSettings.Online),
		CallAdd:      string(newSettings.CallAdd),
	}, nil
}
