package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"net"
	"os"

	"github.com/mdp/qrterminal/v3"

	"whatsd/internal/config"
	"whatsd/internal/types"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg := config.LoadConfig()
	command := os.Args[1]

	switch command {
	case "status":
		handleStatus(cfg.SocketPath)

	case "pair":
		pairFlags := flag.NewFlagSet("pair", flag.ExitOnError)
		useQR := pairFlags.Bool("qr", false, "Pair using QR code in terminal")
		phone := pairFlags.String("phone", "", "Pair using phone number (e.g., 15551234567)")
		_ = pairFlags.Parse(os.Args[2:])

		if *useQR {
			handlePairQR(cfg.SocketPath)
		} else if *phone != "" {
			handlePairPhone(cfg.SocketPath, *phone)
		} else {
			fmt.Println("Error: must specify either --qr or --phone <number>")
			pairFlags.Usage()
			os.Exit(1)
		}

	case "send":
		sendFlags := flag.NewFlagSet("send", flag.ExitOnError)
		to := sendFlags.String("to", "", "Recipient phone number or JID")
		text := sendFlags.String("text", "", "Message text to send")
		_ = sendFlags.Parse(os.Args[2:])

		if *to == "" || *text == "" {
			fmt.Println("Error: both --to and --text are required")
			sendFlags.Usage()
			os.Exit(1)
		}
		handleSend(cfg.SocketPath, *to, *text)

	case "listen":
		handleListen(cfg.SocketPath)

	case "logout":
		handleLogout(cfg.SocketPath)

	default:
		fmt.Printf("Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("Usage: whatsctl <command> [options]")
	fmt.Println("\nCommands:")
	fmt.Println("  status                Check daemon connection and identity status")
	fmt.Println("  pair --qr             Start QR code pairing (renders QR in terminal)")
	fmt.Println("  pair --phone <num>    Start phone pairing code generation")
	fmt.Println("  send --to <jid> --text <msg> Send a text message")
	fmt.Println("  listen                Listen for incoming messages and daemon events")
	fmt.Println("  logout                Logout current session")
}

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
