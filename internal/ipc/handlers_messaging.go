package ipc

import (
	"context"
	"net"

	"whatsd/internal/types"
)

func handleSendMessage(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleSendMedia(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleEditMessage(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleRevokeMessage(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleSendPresence(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleReactMessage(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}
