package ipc

import (
	"context"
	"net"

	"whatsd/internal/types"
)

func handleGetChats(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
	limit := parseIntParam(req.Params["limit"], 50)
	chats, err := s.waClient.GetChats(ctx, limit)
	if err != nil {
		s.sendError(conn, req.ID, err.Error())
	} else {
		s.sendResult(conn, req.ID, chats)
	}
}

func handleGetContacts(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
	query, _ := req.Params["query"].(string)
	contacts, err := s.waClient.GetContacts(ctx, query)
	if err != nil {
		s.sendError(conn, req.ID, err.Error())
	} else {
		s.sendResult(conn, req.ID, contacts)
	}
}

func handleGetMessages(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleMarkRead(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleDownloadMedia(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleGetContact(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleGetProfilePicture(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}
