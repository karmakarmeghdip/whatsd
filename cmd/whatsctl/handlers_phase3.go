package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"whatsd/internal/types"
)

func handleGroupInfo(socketPath string, groupJID string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "get_group_info",
		Params: map[string]any{"group_jid": groupJID},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Group Info Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Group Information:")
	fmt.Println(string(dataBytes))
}

func handleCreateGroup(socketPath string, title string, participantsStr string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	var participants []string
	if participantsStr != "" {
		parts := strings.Split(participantsStr, ",")
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				participants = append(participants, p)
			}
		}
	}

	req := types.Request{
		ID:     "1",
		Method: "create_group",
		Params: map[string]any{
			"title":        title,
			"participants": participants,
		},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Create Group Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Group Created Successfully:")
	fmt.Println(string(dataBytes))
}

func handleGroupMembers(socketPath string, groupJID string, action string, participantsStr string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	var participants []string
	if participantsStr != "" {
		parts := strings.Split(participantsStr, ",")
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				participants = append(participants, p)
			}
		}
	}

	req := types.Request{
		ID:     "1",
		Method: "update_group_members",
		Params: map[string]any{
			"group_jid":    groupJID,
			"action":       action,
			"participants": participants,
		},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Update Group Members Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Group Members Updated Successfully:")
	fmt.Println(string(dataBytes))
}

func handleContact(socketPath string, jid string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "get_contact",
		Params: map[string]any{"jid": jid},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Contact Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Contact Details:")
	fmt.Println(string(dataBytes))
}

func handleAvatar(socketPath string, jid string, preview bool) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "get_profile_picture",
		Params: map[string]any{"jid": jid, "preview": preview},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Avatar Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Profile Picture Information:")
	fmt.Println(string(dataBytes))
}

func handleChatState(socketPath string, chat string, action string, duration string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	params := map[string]any{
		"chat":   chat,
		"action": action,
	}
	if duration != "" {
		params["mute_duration"] = duration
	}

	req := types.Request{
		ID:     "1",
		Method: "set_chat_state",
		Params: params,
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Chat-State Error: %s\n", resp.Error)
		os.Exit(1)
	}

	fmt.Printf("Chat state %q applied successfully to %s.\n", action, chat)
}
