package main

import (
	"encoding/json"
	"fmt"
	"os"

	"whatsd/internal/types"
)

func handlePostStatus(socketPath string, text string, filePath string, mediaType string, caption string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "post_status",
		Params: map[string]any{
			"text":       text,
			"file_path":  filePath,
			"media_type": mediaType,
			"caption":    caption,
		},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Post Status Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Status Posted Successfully:")
	fmt.Println(string(dataBytes))
}

func handleGetStatuses(socketPath string, limit int) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "get_statuses",
		Params: map[string]any{"limit": limit},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Get Statuses Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Recent Status Updates:")
	fmt.Println(string(dataBytes))
}

func handleGetNewsletters(socketPath string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "get_newsletters",
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Get Newsletters Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Subscribed Channels / Newsletters:")
	fmt.Println(string(dataBytes))
}

func handleFollowNewsletter(socketPath string, newsletterJID string, unfollow bool) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "follow_newsletter",
		Params: map[string]any{
			"newsletter_jid": newsletterJID,
			"unfollow":       unfollow,
		},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Follow/Unfollow Newsletter Error: %s\n", resp.Error)
		os.Exit(1)
	}

	action := "followed"
	if unfollow {
		action = "unfollowed"
	}
	fmt.Printf("Successfully %s channel %s.\n", action, newsletterJID)
}

func handleGetNewsletterMessages(socketPath string, newsletterJID string, limit int, beforeID int64) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	params := map[string]any{
		"newsletter_jid": newsletterJID,
		"limit":          limit,
	}
	if beforeID > 0 {
		params["before_id"] = beforeID
	}

	req := types.Request{
		ID:     "1",
		Method: "get_newsletter_messages",
		Params: params,
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Get Newsletter Messages Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Channel Posts:")
	fmt.Println(string(dataBytes))
}

func handleGetBlocklist(socketPath string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "get_blocklist",
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Get Blocklist Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Blocked Contacts:")
	fmt.Println(string(dataBytes))
}

func handleBlockContact(socketPath string, jid string, action string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "block_contact",
		Params: map[string]any{
			"jid":    jid,
			"action": action,
		},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Block/Unblock Contact Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Printf("Blocklist Updated (action=%s):\n", action)
	fmt.Println(string(dataBytes))
}

func handleGetPrivacySettings(socketPath string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "get_privacy_settings",
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Get Privacy Settings Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Current Privacy Settings:")
	fmt.Println(string(dataBytes))
}

func handleSetPrivacySetting(socketPath string, setting string, value string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer func() { _ = conn.Close() }()

	req := types.Request{
		ID:     "1",
		Method: "set_privacy_setting",
		Params: map[string]any{
			"setting": setting,
			"value":   value,
		},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Set Privacy Setting Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Printf("Privacy Setting %q Updated to %q:\n", setting, value)
	fmt.Println(string(dataBytes))
}
