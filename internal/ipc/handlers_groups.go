package ipc

import (
	"context"
	"net"

	"whatsd/internal/types"
)

func handleGetGroupInfo(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleCreateGroup(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}

func handleUpdateGroupMembers(ctx context.Context, s *Server, conn net.Conn, req types.Request) {
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
}
