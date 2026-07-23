package main

import (
	"flag"
	"fmt"
	"os"

	"whatsd/internal/config"
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

	case "send-media":
		sendMediaFlags := flag.NewFlagSet("send-media", flag.ExitOnError)
		to := sendMediaFlags.String("to", "", "Recipient phone number or JID")
		mediaType := sendMediaFlags.String("type", "", "Media type (image|video|audio|document)")
		filePath := sendMediaFlags.String("file", "", "Local file path")
		caption := sendMediaFlags.String("caption", "", "Optional caption")
		fileName := sendMediaFlags.String("name", "", "Optional file name")
		_ = sendMediaFlags.Parse(os.Args[2:])

		if *to == "" || *mediaType == "" || *filePath == "" {
			fmt.Println("Error: --to, --type, and --file are required")
			sendMediaFlags.Usage()
			os.Exit(1)
		}
		handleSendMedia(cfg.SocketPath, *to, *mediaType, *filePath, *caption, *fileName)

	case "chats":
		chatsFlags := flag.NewFlagSet("chats", flag.ExitOnError)
		limit := chatsFlags.Int("limit", 50, "Maximum number of chats to return")
		_ = chatsFlags.Parse(os.Args[2:])
		handleChats(cfg.SocketPath, *limit)

	case "history":
		historyFlags := flag.NewFlagSet("history", flag.ExitOnError)
		chat := historyFlags.String("chat", "", "Chat JID or phone number")
		limit := historyFlags.Int("limit", 50, "Maximum number of messages to return")
		beforeID := historyFlags.String("before", "", "Message ID cursor for pagination")
		_ = historyFlags.Parse(os.Args[2:])

		if *chat == "" {
			fmt.Println("Error: --chat JID is required")
			historyFlags.Usage()
			os.Exit(1)
		}
		handleHistory(cfg.SocketPath, *chat, *limit, *beforeID)

	case "mark-read":
		markReadFlags := flag.NewFlagSet("mark-read", flag.ExitOnError)
		chat := markReadFlags.String("chat", "", "Chat JID or phone number to mark as read")
		_ = markReadFlags.Parse(os.Args[2:])

		if *chat == "" {
			fmt.Println("Error: --chat JID is required")
			markReadFlags.Usage()
			os.Exit(1)
		}
		handleMarkRead(cfg.SocketPath, *chat)

	case "download":
		downloadFlags := flag.NewFlagSet("download", flag.ExitOnError)
		id := downloadFlags.String("id", "", "Message ID to download")
		chat := downloadFlags.String("chat", "", "Chat JID (optional)")
		_ = downloadFlags.Parse(os.Args[2:])

		if *id == "" {
			fmt.Println("Error: --id is required for download")
			downloadFlags.Usage()
			os.Exit(1)
		}
		handleDownload(cfg.SocketPath, *id, *chat)

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
	fmt.Println("  send-media --to <jid> --type <type> --file <path> Send media message")
	fmt.Println("  chats [--limit N]     List active chats with last message preview and unread count")
	fmt.Println("  history --chat <jid>  Query message history for a chat")
	fmt.Println("  mark-read --chat <jid> Reset unread count and mark messages read")
	fmt.Println("  download --id <msg_id> [--chat <jid>] Download media for a message")
	fmt.Println("  listen                Listen for incoming messages and daemon events")
	fmt.Println("  logout                Logout current session")
}
