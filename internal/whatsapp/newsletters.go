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

// GetSubscribedNewsletters returns the list of channels/newsletters the user is following.
func (c *Client) GetSubscribedNewsletters(ctx context.Context) ([]types.NewsletterItem, error) {
	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	newsletters, err := waClient.GetSubscribedNewsletters(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch subscribed newsletters: %w", err)
	}

	var results []types.NewsletterItem
	for _, n := range newsletters {
		if n == nil {
			continue
		}
		item := types.NewsletterItem{
			JID:         n.ID.String(),
			Name:        n.ThreadMeta.Name.Text,
			Description: n.ThreadMeta.Description.Text,
		}
		if n.ThreadMeta.SubscriberCount > 0 {
			item.SubscribersCount = n.ThreadMeta.SubscriberCount
		}
		item.State = string(n.State.Type)
		if n.ViewerMeta != nil {
			item.Role = string(n.ViewerMeta.Role)
			item.Muted = n.ViewerMeta.Mute == waTypes.NewsletterMuteOn
		}
		results = append(results, item)
	}

	return results, nil
}

// FollowNewsletter follows or unfollows a WhatsApp channel.
func (c *Client) FollowNewsletter(ctx context.Context, newsletterJID string, unfollow bool) error {
	nJID, err := parseNewsletterJID(newsletterJID)
	if err != nil {
		return err
	}

	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	if unfollow {
		err = waClient.UnfollowNewsletter(ctx, nJID)
		if err != nil {
			return fmt.Errorf("failed to unfollow newsletter %s: %w", nJID.String(), err)
		}
		slog.Info("unfollowed newsletter successfully", "jid", nJID.String())
	} else {
		err = waClient.FollowNewsletter(ctx, nJID)
		if err != nil {
			return fmt.Errorf("failed to follow newsletter %s: %w", nJID.String(), err)
		}
		slog.Info("followed newsletter successfully", "jid", nJID.String())
	}

	return nil
}

// GetNewsletterMessages fetches recent channel posts.
func (c *Client) GetNewsletterMessages(ctx context.Context, newsletterJID string, limit int, beforeID int64) ([]types.NewsletterMessageItem, error) {
	nJID, err := parseNewsletterJID(newsletterJID)
	if err != nil {
		return nil, err
	}

	if limit <= 0 {
		limit = 50
	}

	c.mu.RLock()
	waClient := c.waClient
	c.mu.RUnlock()

	params := &whatsmeow.GetNewsletterMessagesParams{
		Count: limit,
	}
	if beforeID > 0 {
		params.Before = waTypes.MessageServerID(beforeID)
	}

	msgs, err := waClient.GetNewsletterMessages(ctx, nJID, params)
	if err != nil {
		return nil, fmt.Errorf("failed to get newsletter messages for %s: %w", nJID.String(), err)
	}

	var results []types.NewsletterMessageItem
	for _, m := range msgs {
		if m == nil {
			continue
		}
		text := ""
		if m.Message != nil {
			if conv := m.Message.GetConversation(); conv != "" {
				text = conv
			} else if ext := m.Message.GetExtendedTextMessage(); ext != nil {
				text = ext.GetText()
			} else if img := m.Message.GetImageMessage(); img != nil {
				text = img.GetCaption()
			} else if vid := m.Message.GetVideoMessage(); vid != nil {
				text = vid.GetCaption()
			}
		}
		results = append(results, types.NewsletterMessageItem{
			ID:        fmt.Sprintf("%d", m.MessageServerID),
			ServerID:  int64(m.MessageServerID),
			Views:     m.ViewsCount,
			Text:      text,
			Timestamp: m.Timestamp,
		})
	}

	return results, nil
}

func parseNewsletterJID(jidStr string) (waTypes.JID, error) {
	jid, err := waTypes.ParseJID(jidStr)
	if err == nil && jid.Server == waTypes.NewsletterServer {
		return jid, nil
	}
	if !strings.Contains(jidStr, "@") {
		jid, err = waTypes.ParseJID(jidStr + "@newsletter")
		if err == nil {
			return jid, nil
		}
	}
	if err != nil {
		return waTypes.JID{}, fmt.Errorf("invalid newsletter JID %q: %w", jidStr, err)
	}
	return jid, nil
}
