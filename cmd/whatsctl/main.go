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
		text := sendFlags.String("text", "", "Message text")
		replyToID := sendFlags.String("reply", "", "Message ID to reply to")
		_ = sendFlags.Parse(os.Args[2:])

		if *to == "" || *text == "" {
			fmt.Println("Error: --to and --text are required")
			sendFlags.Usage()
			os.Exit(1)
		}
		handleSend(cfg.SocketPath, *to, *text, *replyToID)

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

	case "edit":
		editFlags := flag.NewFlagSet("edit", flag.ExitOnError)
		chat := editFlags.String("chat", "", "Chat JID")
		id := editFlags.String("id", "", "Message ID to edit")
		text := editFlags.String("text", "", "New message text")
		_ = editFlags.Parse(os.Args[2:])

		if *chat == "" || *id == "" || *text == "" {
			fmt.Println("Error: --chat, --id, and --text are required")
			editFlags.Usage()
			os.Exit(1)
		}
		handleEdit(cfg.SocketPath, *chat, *id, *text)

	case "revoke":
		revokeFlags := flag.NewFlagSet("revoke", flag.ExitOnError)
		chat := revokeFlags.String("chat", "", "Chat JID")
		id := revokeFlags.String("id", "", "Message ID to revoke")
		_ = revokeFlags.Parse(os.Args[2:])

		if *chat == "" || *id == "" {
			fmt.Println("Error: --chat and --id are required")
			revokeFlags.Usage()
			os.Exit(1)
		}
		handleRevoke(cfg.SocketPath, *chat, *id)

	case "presence":
		presenceFlags := flag.NewFlagSet("presence", flag.ExitOnError)
		chat := presenceFlags.String("chat", "", "Chat JID")
		state := presenceFlags.String("state", "", "Presence state (composing|recording|paused)")
		_ = presenceFlags.Parse(os.Args[2:])

		if *chat == "" || *state == "" {
			fmt.Println("Error: --chat and --state are required")
			presenceFlags.Usage()
			os.Exit(1)
		}
		handleSendPresence(cfg.SocketPath, *chat, *state)

	case "react":
		reactFlags := flag.NewFlagSet("react", flag.ExitOnError)
		chat := reactFlags.String("chat", "", "Chat JID")
		id := reactFlags.String("id", "", "Message ID to react to")
		emoji := reactFlags.String("emoji", "", "Emoji string")
		_ = reactFlags.Parse(os.Args[2:])

		if *chat == "" || *id == "" || *emoji == "" {
			fmt.Println("Error: --chat, --id, and --emoji are required")
			reactFlags.Usage()
			os.Exit(1)
		}
		handleReact(cfg.SocketPath, *chat, *id, *emoji)

	case "chats":
		chatsFlags := flag.NewFlagSet("chats", flag.ExitOnError)
		limit := chatsFlags.Int("limit", 50, "Maximum number of chats to return")
		_ = chatsFlags.Parse(os.Args[2:])
		handleChats(cfg.SocketPath, *limit)

	case "contacts":
		contactsFlags := flag.NewFlagSet("contacts", flag.ExitOnError)
		query := contactsFlags.String("query", "", "Search query for name, push name, or JID")
		_ = contactsFlags.Parse(os.Args[2:])
		handleContacts(cfg.SocketPath, *query)

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
		chat := markReadFlags.String("chat", "", "Chat JID to mark as read")
		ids := markReadFlags.String("ids", "", "Comma-separated list of message IDs to send read receipt for")
		_ = markReadFlags.Parse(os.Args[2:])

		if *chat == "" {
			fmt.Println("Error: --chat is required")
			markReadFlags.Usage()
			os.Exit(1)
		}
		handleMarkRead(cfg.SocketPath, *chat, *ids)

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

	case "group-info":
		groupInfoFlags := flag.NewFlagSet("group-info", flag.ExitOnError)
		jid := groupInfoFlags.String("jid", "", "Group JID")
		_ = groupInfoFlags.Parse(os.Args[2:])

		if *jid == "" {
			fmt.Println("Error: --jid is required")
			groupInfoFlags.Usage()
			os.Exit(1)
		}
		handleGroupInfo(cfg.SocketPath, *jid)

	case "create-group":
		createGroupFlags := flag.NewFlagSet("create-group", flag.ExitOnError)
		title := createGroupFlags.String("title", "", "Group title/name")
		participants := createGroupFlags.String("participants", "", "Comma-separated list of participant phone numbers or JIDs")
		_ = createGroupFlags.Parse(os.Args[2:])

		if *title == "" {
			fmt.Println("Error: --title is required")
			createGroupFlags.Usage()
			os.Exit(1)
		}
		handleCreateGroup(cfg.SocketPath, *title, *participants)

	case "group-members":
		groupMembersFlags := flag.NewFlagSet("group-members", flag.ExitOnError)
		jid := groupMembersFlags.String("jid", "", "Group JID")
		action := groupMembersFlags.String("action", "", "Member action (add|remove|promote|demote)")
		participants := groupMembersFlags.String("participants", "", "Comma-separated list of participant phone numbers or JIDs")
		_ = groupMembersFlags.Parse(os.Args[2:])

		if *jid == "" || *action == "" || *participants == "" {
			fmt.Println("Error: --jid, --action, and --participants are required")
			groupMembersFlags.Usage()
			os.Exit(1)
		}
		handleGroupMembers(cfg.SocketPath, *jid, *action, *participants)

	case "contact":
		contactFlags := flag.NewFlagSet("contact", flag.ExitOnError)
		jid := contactFlags.String("jid", "", "Contact phone number or JID")
		_ = contactFlags.Parse(os.Args[2:])

		if *jid == "" {
			fmt.Println("Error: --jid is required")
			contactFlags.Usage()
			os.Exit(1)
		}
		handleContact(cfg.SocketPath, *jid)

	case "avatar":
		avatarFlags := flag.NewFlagSet("avatar", flag.ExitOnError)
		jid := avatarFlags.String("jid", "", "Contact or Group JID")
		preview := avatarFlags.Bool("preview", false, "Fetch low-res preview image")
		_ = avatarFlags.Parse(os.Args[2:])

		if *jid == "" {
			fmt.Println("Error: --jid is required")
			avatarFlags.Usage()
			os.Exit(1)
		}
		handleAvatar(cfg.SocketPath, *jid, *preview)

	case "chat-state":
		chatStateFlags := flag.NewFlagSet("chat-state", flag.ExitOnError)
		chat := chatStateFlags.String("chat", "", "Chat JID")
		action := chatStateFlags.String("action", "", "Action (mute|unmute|pin|unpin|archive|unarchive)")
		duration := chatStateFlags.String("duration", "", "Optional mute duration (e.g. 8h, 24h)")
		_ = chatStateFlags.Parse(os.Args[2:])

		if *chat == "" || *action == "" {
			fmt.Println("Error: --chat and --action are required")
			chatStateFlags.Usage()
			os.Exit(1)
		}
		handleChatState(cfg.SocketPath, *chat, *action, *duration)

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
	fmt.Println("  send --to <jid> --text <msg> [--reply <id>] Send a text message")
	fmt.Println("  send-media --to <jid> --type <type> --file <path> Send media message")
	fmt.Println("  edit --chat <jid> --id <msg_id> --text <new_text> Edit a message")
	fmt.Println("  revoke --chat <jid> --id <msg_id> Revoke a message")
	fmt.Println("  react --chat <jid> --id <msg_id> --emoji <emoji> React to a message")
	fmt.Println("  presence --chat <jid> --state <state> Send presence state")
	fmt.Println("  chats [--limit N]     List active chats with last message preview and unread count")
	fmt.Println("  contacts [--query Q]  Search contacts directory by name, push name, or JID")
	fmt.Println("  contact --jid <jid>   Fetch contact info, status, and verified business status")
	fmt.Println("  avatar --jid <jid> [--preview] Download profile picture or group photo")
	fmt.Println("  chat-state --chat <jid> --action <mute|unmute|pin|unpin|archive|unarchive> [--duration <dur>] Set chat state")
	fmt.Println("  group-info --jid <jid> Get group metadata and member list")
	fmt.Println("  create-group --title <title> [--participants <jids>] Create new group chat")
	fmt.Println("  group-members --jid <jid> --action <add|remove|promote|demote> --participants <jids> Manage group members")
	fmt.Println("  history --chat <jid>  Query message history for a chat")
	fmt.Println("  mark-read --chat <jid> [--ids <id1,id2>] Reset unread count and mark messages read")
	fmt.Println("  download --id <msg_id> [--chat <jid>] Download media for a message")
	fmt.Println("  listen                Listen for incoming messages and daemon events")
	fmt.Println("  logout                Logout current session")
}
