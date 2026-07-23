package ipc

import (
	"context"
	"net"
	"strings"

	"whatsd/internal/types"
)

func handleStatus(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
	status := s.waClient.GetStatus()
	s.sendResult(conn, req.ID, status)
}

func handlePairQR(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handlePairPhone(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleLogout(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
	err := s.waClient.Logout(ctx)
	if err != nil {
		s.sendError(conn, req.ID, err.Error())
	} else {
		s.sendResult(conn, req.ID, map[string]string{"status": "logged_out"})
	}
}

func handleSetChatState(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handlePostStatus(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleGetStatuses(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
	limit := parseIntParam(req.Params["limit"], 50)
	statuses, err := s.waClient.GetStatuses(ctx, limit)
	if err != nil {
		s.sendError(conn, req.ID, err.Error())
	} else {
		s.sendResult(conn, req.ID, statuses)
	}
}

func handleGetNewsletters(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
	newsletters, err := s.waClient.GetSubscribedNewsletters(ctx)
	if err != nil {
		s.sendError(conn, req.ID, err.Error())
	} else {
		s.sendResult(conn, req.ID, newsletters)
	}
}

func handleFollowNewsletter(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleGetNewsletterMessages(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleGetBlocklist(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
	blocklist, err := s.waClient.GetBlocklist(ctx)
	if err != nil {
		s.sendError(conn, req.ID, err.Error())
	} else {
		s.sendResult(conn, req.ID, blocklist)
	}
}

func handleBlockContact(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleGetPrivacySettings(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
	settings, err := s.waClient.GetPrivacySettings(ctx)
	if err != nil {
		s.sendError(conn, req.ID, err.Error())
	} else {
		s.sendResult(conn, req.ID, settings)
	}
}

func handleSetPrivacySetting(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}
