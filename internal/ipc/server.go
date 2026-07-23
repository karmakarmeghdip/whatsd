package ipc

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"whatsd/internal/types"
	"whatsd/internal/whatsapp"
)

// Server handles IPC incoming Unix domain socket connections.
type Server struct {
	socketPath string
	waClient   *whatsapp.Client
	listener   net.Listener
	mu         sync.Mutex
	clients    map[net.Conn]struct{}
	stopCh     chan struct{}
}

// NewServer creates a new IPC server instance.
func NewServer(socketPath string, waClient *whatsapp.Client) *Server {
	return &Server{
		socketPath: socketPath,
		waClient:   waClient,
		clients:    make(map[net.Conn]struct{}),
		stopCh:     make(chan struct{}),
	}
}

// Start begins listening on the Unix domain socket.
func (s *Server) Start(ctx context.Context) error {
	// Remove stale socket file if it exists
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove old socket file %s: %w", s.socketPath, err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on unix socket %s: %w", s.socketPath, err)
	}
	s.listener = listener

	// Ensure restricted permissions on socket file
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		slog.Warn("failed to set restrictive permissions on socket file", "path", s.socketPath, "err", err)
	}

	slog.Info("IPC server listening", "socket", s.socketPath)

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				select {
				case <-s.stopCh:
					return
				default:
					slog.Error("failed to accept IPC connection", "err", err)
					continue
				}
			}
			s.registerClient(conn)
			go s.handleConnection(ctx, conn)
		}
	}()

	return nil
}

// Broadcast sends an event notification to all connected IPC clients.
func (s *Server) Broadcast(evt types.EventNotification) {
	data, err := json.Marshal(evt)
	if err != nil {
		slog.Error("failed to marshal event notification", "err", err)
		return
	}
	data = append(data, '\n')

	s.mu.Lock()
	defer s.mu.Unlock()

	for conn := range s.clients {
		_, err := conn.Write(data)
		if err != nil {
			slog.Debug("failed to write event to IPC client, closing connection", "err", err)
			_ = conn.Close()
			delete(s.clients, conn)
		}
	}
}

// Stop closes all connections and the socket listener.
func (s *Server) Stop() {
	s.mu.Lock()
	select {
	case <-s.stopCh:
		s.mu.Unlock()
		return
	default:
		close(s.stopCh)
	}

	if s.listener != nil {
		_ = s.listener.Close()
	}
	for conn := range s.clients {
		_ = conn.Close()
	}
	s.clients = make(map[net.Conn]struct{})
	s.mu.Unlock()

	_ = os.Remove(s.socketPath)
	slog.Info("IPC server stopped")
}

func (s *Server) registerClient(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[conn] = struct{}{}
}

func (s *Server) unregisterClient(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.clients, conn)
	_ = conn.Close()
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer s.unregisterClient(conn)
	scanner := bufio.NewScanner(conn)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req types.Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(conn, "", fmt.Sprintf("invalid request payload: %v", err))
			continue
		}

		s.handleRequest(ctx, conn, req)
	}

	if err := scanner.Err(); err != nil && !errors.Is(err, io.EOF) {
		slog.Debug("IPC scanner error", "err", err)
	}
}

func (s *Server) handleRequest(ctx context.Context, conn net.Conn, req types.Request) {
	switch req.Method {
	case "status":
		status := s.waClient.GetStatus()
		s.sendResult(conn, req.ID, status)

	case "pair_qr":
		err := s.waClient.PairQR(ctx, func(evt string, code string, qrErr error) {
			if evt == "error" && qrErr != nil {
				s.Broadcast(types.EventNotification{
					Event: "qr",
					Data:  types.QREventData{Err: qrErr.Error()},
				})
			} else if evt == "code" {
				s.Broadcast(types.EventNotification{
					Event: "qr",
					Data:  types.QREventData{Code: code},
				})
			} else if evt == "success" {
				s.Broadcast(types.EventNotification{
					Event: "qr",
					Data:  types.QREventData{Success: true},
				})
			}
		})
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"status": "qr_pairing_started"})
		}

	case "pair_phone":
		phone, _ := req.Params["phone"].(string)
		if phone == "" {
			s.sendError(conn, req.ID, "missing required param 'phone'")
			return
		}
		code, err := s.waClient.PairPhone(ctx, phone)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, types.PairPhoneResult{Code: code})
		}

	case "send_message":
		to, _ := req.Params["to"].(string)
		text, _ := req.Params["text"].(string)
		replyToID, _ := req.Params["reply_to_id"].(string)

		if to == "" || text == "" {
			s.sendError(conn, req.ID, "missing required params 'to' or 'text'")
			return
		}
		id, ts, err := s.waClient.SendMessage(ctx, to, text, replyToID)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, types.SendMessageResult{
				ID:        id,
				Timestamp: ts,
			})
		}

	case "send_media":
		to, _ := req.Params["to"].(string)
		mediaType, _ := req.Params["media_type"].(string)
		filePath, _ := req.Params["file_path"].(string)
		caption, _ := req.Params["caption"].(string)
		fileName, _ := req.Params["file_name"].(string)

		if to == "" || mediaType == "" || filePath == "" {
			s.sendError(conn, req.ID, "missing required params 'to', 'media_type', or 'file_path'")
			return
		}
		id, ts, err := s.waClient.SendMedia(ctx, to, mediaType, filePath, caption, fileName)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, types.SendMessageResult{
				ID:        id,
				Timestamp: ts,
			})
		}

	case "edit_message":
		chat, _ := req.Params["chat"].(string)
		id, _ := req.Params["message_id"].(string)
		newText, _ := req.Params["new_text"].(string)

		if chat == "" || id == "" || newText == "" {
			s.sendError(conn, req.ID, "missing required params 'chat', 'message_id', or 'new_text'")
			return
		}
		err := s.waClient.EditMessage(ctx, chat, id, newText)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"status": "ok"})
		}

	case "revoke_message":
		chat, _ := req.Params["chat"].(string)
		id, _ := req.Params["message_id"].(string)

		if chat == "" || id == "" {
			s.sendError(conn, req.ID, "missing required params 'chat' or 'message_id'")
			return
		}
		err := s.waClient.RevokeMessage(ctx, chat, id)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"status": "ok"})
		}

	case "send_presence":
		chat, _ := req.Params["chat"].(string)
		state, _ := req.Params["state"].(string)

		if chat == "" || state == "" {
			s.sendError(conn, req.ID, "missing required params 'chat' or 'state'")
			return
		}
		err := s.waClient.SendPresence(ctx, chat, state)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"status": "ok"})
		}

	case "react_message":
		chat, _ := req.Params["chat"].(string)
		id, _ := req.Params["message_id"].(string)
		emoji, _ := req.Params["emoji"].(string)

		if chat == "" || id == "" || emoji == "" {
			s.sendError(conn, req.ID, "missing required params 'chat', 'message_id', or 'emoji'")
			return
		}
		err := s.waClient.ReactMessage(ctx, chat, id, emoji)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"status": "ok"})
		}

	case "logout":
		err := s.waClient.Logout(ctx)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"status": "logged_out"})
		}

	case "get_chats":
		limit := parseIntParam(req.Params["limit"], 50)
		chats, err := s.waClient.GetChats(ctx, limit)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, chats)
		}

	case "get_contacts":
		query, _ := req.Params["query"].(string)
		contacts, err := s.waClient.GetContacts(ctx, query)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, contacts)
		}

	case "get_messages":
		chat, _ := req.Params["chat"].(string)
		if chat == "" {
			s.sendError(conn, req.ID, "missing required param 'chat'")
			return
		}
		limit := parseIntParam(req.Params["limit"], 50)
		beforeID, _ := req.Params["before_id"].(string)
		messages, err := s.waClient.GetMessages(ctx, chat, limit, beforeID)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, messages)
		}

	case "mark_read":
		chat, _ := req.Params["chat"].(string)
		idsRaw, _ := req.Params["message_ids"].([]any)
		if chat == "" {
			s.sendError(conn, req.ID, "missing required param 'chat'")
			return
		}
		var msgIDs []string
		for _, id := range idsRaw {
			if idStr, ok := id.(string); ok {
				msgIDs = append(msgIDs, idStr)
			}
		}
		err := s.waClient.MarkRead(ctx, chat, msgIDs)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"status": "ok"})
		}

	case "download_media":
		id, _ := req.Params["message_id"].(string)
		if id == "" {
			s.sendError(conn, req.ID, "missing required param 'message_id'")
			return
		}
		path, err := s.waClient.DownloadMedia(ctx, id)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"file_path": path})
		}

	case "get_group_info":
		groupJID, _ := req.Params["group_jid"].(string)
		if groupJID == "" {
			s.sendError(conn, req.ID, "missing required param 'group_jid'")
			return
		}
		info, err := s.waClient.GetGroupInfo(ctx, groupJID)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, info)
		}

	case "create_group":
		title, _ := req.Params["title"].(string)
		participants := parseStringSliceParam(req.Params["participants"])
		if title == "" {
			s.sendError(conn, req.ID, "missing required param 'title'")
			return
		}
		info, err := s.waClient.CreateGroup(ctx, title, participants)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, info)
		}

	case "update_group_members":
		groupJID, _ := req.Params["group_jid"].(string)
		action, _ := req.Params["action"].(string)
		participants := parseStringSliceParam(req.Params["participants"])
		if groupJID == "" || action == "" {
			s.sendError(conn, req.ID, "missing required params 'group_jid' or 'action'")
			return
		}
		members, err := s.waClient.UpdateGroupMembers(ctx, groupJID, action, participants)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, members)
		}

	case "get_contact":
		jid, _ := req.Params["jid"].(string)
		if jid == "" {
			s.sendError(conn, req.ID, "missing required param 'jid'")
			return
		}
		contact, err := s.waClient.GetContact(ctx, jid)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, contact)
		}

	case "get_profile_picture":
		jid, _ := req.Params["jid"].(string)
		if jid == "" {
			s.sendError(conn, req.ID, "missing required param 'jid'")
			return
		}
		preview, _ := req.Params["preview"].(bool)
		pic, err := s.waClient.GetProfilePicture(ctx, jid, preview)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, pic)
		}

	case "set_chat_state":
		chat, _ := req.Params["chat"].(string)
		action, _ := req.Params["action"].(string)
		if chat == "" || action == "" {
			s.sendError(conn, req.ID, "missing required params 'chat' or 'action'")
			return
		}
		duration := parseDurationParam(req.Params["mute_duration"])
		err := s.waClient.SetChatState(ctx, chat, action, duration)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"status": "ok"})
		}

	case "post_status":
		text, _ := req.Params["text"].(string)
		filePath, _ := req.Params["file_path"].(string)
		mediaType, _ := req.Params["media_type"].(string)
		caption, _ := req.Params["caption"].(string)
		res, err := s.waClient.PostStatus(ctx, types.PostStatusParams{
			Text:      text,
			FilePath:  filePath,
			MediaType: mediaType,
			Caption:   caption,
		})
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, res)
		}

	case "get_statuses":
		limit := parseIntParam(req.Params["limit"], 50)
		statuses, err := s.waClient.GetStatuses(ctx, limit)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, statuses)
		}

	case "get_newsletters":
		newsletters, err := s.waClient.GetSubscribedNewsletters(ctx)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, newsletters)
		}

	case "follow_newsletter":
		newsletterJID, _ := req.Params["newsletter_jid"].(string)
		unfollow, _ := req.Params["unfollow"].(bool)
		if newsletterJID == "" {
			s.sendError(conn, req.ID, "missing required param 'newsletter_jid'")
			return
		}
		err := s.waClient.FollowNewsletter(ctx, newsletterJID, unfollow)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, map[string]string{"status": "ok"})
		}

	case "get_newsletter_messages":
		newsletterJID, _ := req.Params["newsletter_jid"].(string)
		if newsletterJID == "" {
			s.sendError(conn, req.ID, "missing required param 'newsletter_jid'")
			return
		}
		limit := parseIntParam(req.Params["limit"], 50)
		beforeID := int64(parseIntParam(req.Params["before_id"], 0))
		msgs, err := s.waClient.GetNewsletterMessages(ctx, newsletterJID, limit, beforeID)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, msgs)
		}

	case "get_blocklist":
		blocklist, err := s.waClient.GetBlocklist(ctx)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, blocklist)
		}

	case "block_contact":
		jid, _ := req.Params["jid"].(string)
		action, _ := req.Params["action"].(string)
		if jid == "" {
			s.sendError(conn, req.ID, "missing required param 'jid'")
			return
		}
		unblock := strings.ToLower(action) == "unblock"
		blocklist, err := s.waClient.BlockContact(ctx, jid, unblock)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, blocklist)
		}

	case "get_privacy_settings":
		settings, err := s.waClient.GetPrivacySettings(ctx)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, settings)
		}

	case "set_privacy_setting":
		setting, _ := req.Params["setting"].(string)
		value, _ := req.Params["value"].(string)
		if setting == "" || value == "" {
			s.sendError(conn, req.ID, "missing required params 'setting' or 'value'")
			return
		}
		settings, err := s.waClient.SetPrivacySetting(ctx, setting, value)
		if err != nil {
			s.sendError(conn, req.ID, err.Error())
		} else {
			s.sendResult(conn, req.ID, settings)
		}

	default:
		s.sendError(conn, req.ID, fmt.Sprintf("unknown method: %s", req.Method))
	}
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

func (s *Server) sendResult(conn net.Conn, id string, result any) {
	resp := types.Response{
		ID:     id,
		Result: result,
	}
	s.sendResponse(conn, resp)
}

func (s *Server) sendError(conn net.Conn, id string, errMsg string) {
	resp := types.Response{
		ID:    id,
		Error: errMsg,
	}
	s.sendResponse(conn, resp)
}

func (s *Server) sendResponse(conn net.Conn, resp types.Response) {
	data, err := json.Marshal(resp)
	if err != nil {
		slog.Error("failed to marshal response", "err", err)
		return
	}
	data = append(data, '\n')
	_, _ = conn.Write(data)
}
