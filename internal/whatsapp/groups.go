package whatsapp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"go.mau.fi/whatsmeow"
	waTypes "go.mau.fi/whatsmeow/types"

	"whatsd/internal/types"
)

// GetGroupInfo fetches detailed metadata and participant list for a group chat.
func (c *Client) GetGroupInfo(ctx context.Context, groupJID string) (*types.GroupInfoResult, error) {
	gJID, err := parseGroupJID(groupJID)
	if err != nil {
		return nil, err
	}

	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	info, err := waClient.GetGroupInfo(ctx, gJID)
	if err != nil {
		return nil, fmt.Errorf("failed to get group info for %s: %w", gJID.String(), err)
	}

	result := &types.GroupInfoResult{
		JID:         info.JID.String(),
		Owner:       info.OwnerJID.String(),
		Title:       info.Name,
		Topic:       info.Topic,
		TopicSetBy:  info.TopicSetBy.String(),
		TopicSetAt:  info.TopicSetAt,
		IsLocked:    info.IsLocked,
		IsAnnounce:  info.IsAnnounce,
		MemberCount: info.ParticipantCount,
	}

	for _, p := range info.Participants {
		result.Members = append(result.Members, types.GroupParticipant{
			JID:          p.JID.String(),
			IsAdmin:      p.IsAdmin,
			IsSuperAdmin: p.IsSuperAdmin,
			Error:        p.Error,
		})
	}

	return result, nil
}

// CreateGroup creates a new WhatsApp group chat with the specified title and participants.
func (c *Client) CreateGroup(ctx context.Context, title string, participants []string) (*types.GroupInfoResult, error) {
	if title == "" {
		return nil, fmt.Errorf("group title cannot be empty")
	}

	var participantJIDs []waTypes.JID
	for _, pStr := range participants {
		pStr = strings.TrimSpace(pStr)
		if pStr == "" {
			continue
		}
		pJID, err := parseUserJID(pStr)
		if err != nil {
			return nil, fmt.Errorf("invalid participant JID %q: %w", pStr, err)
		}
		participantJIDs = append(participantJIDs, pJID)
	}

	req := whatsmeow.ReqCreateGroup{
		Name:         title,
		Participants: participantJIDs,
	}

	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	info, err := waClient.CreateGroup(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create group %q: %w", title, err)
	}

	slog.Info("created group successfully", "jid", info.JID.String(), "title", title)

	result := &types.GroupInfoResult{
		JID:         info.JID.String(),
		Owner:       info.OwnerJID.String(),
		Title:       info.Name,
		Topic:       info.Topic,
		IsLocked:    info.IsLocked,
		IsAnnounce:  info.IsAnnounce,
		MemberCount: info.ParticipantCount,
	}

	for _, p := range info.Participants {
		result.Members = append(result.Members, types.GroupParticipant{
			JID:          p.JID.String(),
			IsAdmin:      p.IsAdmin,
			IsSuperAdmin: p.IsSuperAdmin,
			Error:        p.Error,
		})
	}

	return result, nil
}

// UpdateGroupMembers adds, removes, promotes, or demotes members in a group chat.
func (c *Client) UpdateGroupMembers(ctx context.Context, groupJID string, action string, participants []string) ([]types.GroupParticipant, error) {
	gJID, err := parseGroupJID(groupJID)
	if err != nil {
		return nil, err
	}

	var pJIDs []waTypes.JID
	for _, pStr := range participants {
		pStr = strings.TrimSpace(pStr)
		if pStr == "" {
			continue
		}
		pJID, err := parseUserJID(pStr)
		if err != nil {
			return nil, fmt.Errorf("invalid participant JID %q: %w", pStr, err)
		}
		pJIDs = append(pJIDs, pJID)
	}

	if len(pJIDs) == 0 {
		return nil, fmt.Errorf("no valid participants provided for action %s", action)
	}

	var changeAction whatsmeow.ParticipantChange
	switch strings.ToLower(action) {
	case "add":
		changeAction = whatsmeow.ParticipantChangeAdd
	case "remove":
		changeAction = whatsmeow.ParticipantChangeRemove
	case "promote":
		changeAction = whatsmeow.ParticipantChangePromote
	case "demote":
		changeAction = whatsmeow.ParticipantChangeDemote
	default:
		return nil, fmt.Errorf("invalid member action %q (must be add, remove, promote, or demote)", action)
	}

	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	resp, err := waClient.UpdateGroupParticipants(ctx, gJID, pJIDs, changeAction)
	if err != nil {
		return nil, fmt.Errorf("failed to update group participants for %s: %w", gJID.String(), err)
	}

	var results []types.GroupParticipant
	for _, p := range resp {
		results = append(results, types.GroupParticipant{
			JID:          p.JID.String(),
			IsAdmin:      p.IsAdmin,
			IsSuperAdmin: p.IsSuperAdmin,
			Error:        p.Error,
		})
	}

	slog.Info("updated group members successfully", "group", gJID.String(), "action", action, "count", len(results))
	return results, nil
}

func parseGroupJID(jidStr string) (waTypes.JID, error) {
	jid, err := waTypes.ParseJID(jidStr)
	if err == nil && jid.Server == waTypes.GroupServer {
		return jid, nil
	}
	if !strings.Contains(jidStr, "@") {
		jid, err = waTypes.ParseJID(jidStr + "@g.us")
		if err == nil {
			return jid, nil
		}
	}
	if err != nil {
		return waTypes.JID{}, fmt.Errorf("invalid group JID %q: %w", jidStr, err)
	}
	return jid, nil
}

func parseUserJID(jidStr string) (waTypes.JID, error) {
	jid, err := waTypes.ParseJID(jidStr)
	if err == nil {
		return jid, nil
	}
	if !strings.Contains(jidStr, "@") {
		jid, err = waTypes.ParseJID(jidStr + "@s.whatsapp.net")
		if err == nil {
			return jid, nil
		}
	}
	return waTypes.JID{}, fmt.Errorf("invalid user JID %q: %w", jidStr, err)
}
