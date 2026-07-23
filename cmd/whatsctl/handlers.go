package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/mdp/qrterminal/v3"

	"whatsd/internal/types"
)

func connectSocket(socketPath string) (net.Conn, error) {
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return nil, fmt.Errorf("could not connect to whatsd socket at %s: %w", socketPath, err)
	}
	return conn, nil
}

func handleStatus(socketPath string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := types.Request{ID: "1", Method: "status"}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Daemon Error: %s\n", resp.Error)
		os.Exit(1)
	}

	resultBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println(string(resultBytes))
}

func handlePairQR(socketPath string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Requesting QR pairing mode...")
	req := types.Request{ID: "1", Method: "pair_qr"}
	sendReq(conn, req)

	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		var resp types.Response
		if err := json.Unmarshal(scanner.Bytes(), &resp); err == nil && resp.Error != "" {
			fmt.Printf("Daemon Error: %s\n", resp.Error)
			return
		}
	}

	for scanner.Scan() {
		line := scanner.Bytes()
		var event types.EventNotification
		if err := json.Unmarshal(line, &event); err == nil && event.Event == "qr" {
			dataBytes, _ := json.Marshal(event.Data)
			var qrData types.QREventData
			_ = json.Unmarshal(dataBytes, &qrData)

			if qrData.Success {
				fmt.Println("\nSuccessfully paired with WhatsApp!")
				return
			}
			if qrData.Err != "" {
				fmt.Println("\nQR Pairing Error:", qrData.Err)
				return
			}
			if qrData.Code != "" {
				fmt.Println("\nScan the QR code below using WhatsApp on your phone:")
				qrterminal.GenerateHalfBlock(qrData.Code, qrterminal.L, os.Stdout)
			}
		}
	}
}

func handlePairPhone(socketPath string, phone string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := types.Request{
		ID:     "1",
		Method: "pair_phone",
		Params: map[string]any{"phone": phone},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Daemon Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.Marshal(resp.Result)
	var result types.PairPhoneResult
	_ = json.Unmarshal(dataBytes, &result)

	fmt.Printf("\nPairing Code for %s: %s\n", phone, result.Code)
	fmt.Println("Enter this code in WhatsApp -> Linked Devices -> Link with Phone Number.")
}

func handleSend(socketPath string, to string, text string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := types.Request{
		ID:     "1",
		Method: "send_message",
		Params: map[string]any{"to": to, "text": text},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Send Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Message Sent Successfully:")
	fmt.Println(string(dataBytes))
}

func handleSendMedia(socketPath string, to string, mediaType string, filePath string, caption string, fileName string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := types.Request{
		ID:     "1",
		Method: "send_media",
		Params: map[string]any{
			"to":         to,
			"media_type": mediaType,
			"file_path":  filePath,
			"caption":    caption,
			"file_name":  fileName,
		},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("SendMedia Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Media Sent Successfully:")
	fmt.Println(string(dataBytes))
}

func handleChats(socketPath string, limit int) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := types.Request{
		ID:     "1",
		Method: "get_chats",
		Params: map[string]any{"limit": limit},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Chats Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.Marshal(resp.Result)
	var chats []types.ChatItem
	_ = json.Unmarshal(dataBytes, &chats)

	if len(chats) == 0 {
		fmt.Println("No stored chats found.")
		return
	}

	fmt.Printf("%-35s %-20s %-8s %s\n", "JID", "NAME", "UNREAD", "LAST MESSAGE")
	fmt.Println("--------------------------------------------------------------------------------")
	for _, c := range chats {
		unreadStr := fmt.Sprintf("%d", c.UnreadCount)
		if c.UnreadCount > 0 {
			unreadStr = fmt.Sprintf("[%d]", c.UnreadCount)
		}
		name := c.Name
		if name == "" {
			name = c.JID
		}
		if len(name) > 20 {
			name = name[:17] + "..."
		}
		lastText := c.LastMessageText
		if len(lastText) > 30 {
			lastText = lastText[:27] + "..."
		}
		fmt.Printf("%-35s %-20s %-8s %s\n", c.JID, name, unreadStr, lastText)
	}
}

func handleHistory(socketPath string, chat string, limit int, beforeID string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	params := map[string]any{"chat": chat, "limit": limit}
	if beforeID != "" {
		params["before_id"] = beforeID
	}

	req := types.Request{
		ID:     "1",
		Method: "get_messages",
		Params: params,
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("History Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.Marshal(resp.Result)
	var msgs []types.MessageItem
	_ = json.Unmarshal(dataBytes, &msgs)

	if len(msgs) == 0 {
		fmt.Printf("No stored messages found for chat %s.\n", chat)
		return
	}

	fmt.Printf("Message History for %s (%d messages):\n\n", chat, len(msgs))
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		direction := "IN"
		if m.IsFromMe {
			direction = "OUT"
		}
		sender := m.SenderName
		if sender == "" {
			sender = m.Sender
		}
		fmt.Printf("[%s] %s (%s) [%s]: %s\n",
			m.Timestamp.Format("2006-01-02 15:04:05"), direction, sender, m.ID, m.Text)
	}
}

func handleMarkRead(socketPath string, chat string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := types.Request{
		ID:     "1",
		Method: "mark_read",
		Params: map[string]any{"chat": chat},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Mark-Read Error: %s\n", resp.Error)
		os.Exit(1)
	}

	fmt.Printf("Successfully marked chat %s as read.\n", chat)
}

func handleDownload(socketPath string, id string, chat string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := types.Request{
		ID:     "1",
		Method: "download_media",
		Params: map[string]any{"message_id": id, "chat_jid": chat},
	}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Download Error: %s\n", resp.Error)
		os.Exit(1)
	}

	dataBytes, _ := json.MarshalIndent(resp.Result, "", "  ")
	fmt.Println("Media downloaded successfully:")
	fmt.Println(string(dataBytes))
}

func handleListen(socketPath string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Println("Listening for WhatsApp daemon events (Press Ctrl+C to exit)...")
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		line := scanner.Bytes()
		var event types.EventNotification
		if err := json.Unmarshal(line, &event); err == nil {
			switch event.Event {
			case "message":
				dataBytes, _ := json.Marshal(event.Data)
				var msg types.MessageEventData
				_ = json.Unmarshal(dataBytes, &msg)
				fmt.Printf("[%s] From %s (%s): %s\n",
					msg.Timestamp.Format("15:04:05"), msg.Sender, msg.PushName, msg.Text)
			case "history_sync_progress":
				fmt.Printf("[HISTORY SYNC PROGRESS] %v\n", event.Data)
			case "status":
				fmt.Printf("[STATUS UPDATE] %v\n", event.Data)
			default:
				fmt.Printf("[%s] %v\n", event.Event, event.Data)
			}
		}
	}
}

func handleLogout(socketPath string) {
	conn, err := connectSocket(socketPath)
	if err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
	defer conn.Close()

	req := types.Request{ID: "1", Method: "logout"}
	sendReq(conn, req)

	resp := readResp(conn)
	if resp.Error != "" {
		fmt.Printf("Logout Error: %s\n", resp.Error)
		os.Exit(1)
	}
	fmt.Println("Successfully logged out session.")
}

func sendReq(conn net.Conn, req types.Request) {
	data, _ := json.Marshal(req)
	data = append(data, '\n')
	_, _ = conn.Write(data)
}

func readResp(conn net.Conn) types.Response {
	scanner := bufio.NewScanner(conn)
	if scanner.Scan() {
		var resp types.Response
		_ = json.Unmarshal(scanner.Bytes(), &resp)
		return resp
	}
	return types.Response{Error: "no response from daemon"}
}
