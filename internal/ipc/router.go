package ipc

import (
	"context"
	"fmt"
	"net"
	"time"

	"whatsd/internal/types"
)

type HandlerFunc func(ctx context.Context, s *Server, conn net.Conn, req types.Request)

type Router struct {
	handlers map[string]HandlerFunc
}

func NewRouter() *Router {
	r := &Router{
		handlers: make(map[string]HandlerFunc),
	}
	r.registerHandlers()
	return r
}

func (r *Router) registerHandlers() {
	// System / Auth / Status
	r.Register("status", handleStatus)
	r.Register("pair_qr", handlePairQR)
	r.Register("pair_phone", handlePairPhone)
	r.Register("logout", handleLogout)

	// Messaging
	r.Register("send_message", handleSendMessage)
	r.Register("send_media", handleSendMedia)
	r.Register("edit_message", handleEditMessage)
	r.Register("revoke_message", handleRevokeMessage)
	r.Register("send_presence", handleSendPresence)
	r.Register("react_message", handleReactMessage)

	// Chats & Media
	r.Register("get_chats", handleGetChats)
	r.Register("get_contacts", handleGetContacts)
	r.Register("get_messages", handleGetMessages)
	r.Register("mark_read", handleMarkRead)
	r.Register("download_media", handleDownloadMedia)
	r.Register("get_contact", handleGetContact)
	r.Register("get_profile_picture", handleGetProfilePicture)
	r.Register("set_chat_state", handleSetChatState)

	// Groups
	r.Register("get_group_info", handleGetGroupInfo)
	r.Register("create_group", handleCreateGroup)
	r.Register("update_group_members", handleUpdateGroupMembers)

	// Statuses / Newsletters / Privacy
	r.Register("post_status", handlePostStatus)
	r.Register("get_statuses", handleGetStatuses)
	r.Register("get_newsletters", handleGetNewsletters)
	r.Register("follow_newsletter", handleFollowNewsletter)
	r.Register("get_newsletter_messages", handleGetNewsletterMessages)
	r.Register("get_blocklist", handleGetBlocklist)
	r.Register("block_contact", handleBlockContact)
	r.Register("get_privacy_settings", handleGetPrivacySettings)
	r.Register("set_privacy_setting", handleSetPrivacySetting)
}

func (r *Router) Register(method string, handler HandlerFunc) {
	r.handlers[method] = handler
}

func (r *Router) Dispatch(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
	handler, ok := r.handlers[req.Method]
	if !ok {
		s.sendError(conn, req.ID, fmt.Sprintf("unknown method: %s", req.Method))
		return
	}
	handler(ctx, s, conn, req)
}

func parseDurationParam(v any) time.Duration {
	if v == nil {
		return 0
	}
	switch val := v.(type) {
	case string:
		d, err := time.ParseDuration(val)
		if err == nil {
			return d
		}
		return 0
	case float64:
		return time.Duration(val) * time.Second
	case int:
		return time.Duration(val) * time.Second
	default:
		return 0
	}
}

func parseStringSliceParam(v any) []string {
	if v == nil {
		return nil
	}
	var res []string
	if slice, ok := v.([]any); ok {
		for _, item := range slice {
			if s, ok := item.(string); ok && s != "" {
				res = append(res, s)
			}
		}
	} else if sliceStr, ok := v.([]string); ok {
		return sliceStr
	}
	return res
}

func parseIntParam(v any, defaultVal int) int {
	if v == nil {
		return defaultVal
	}
	switch val := v.(type) {
	case float64:
		return int(val)
	case int:
		return val
	default:
		return defaultVal
	}
}
