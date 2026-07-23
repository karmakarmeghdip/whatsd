package main

import (
	"fmt"
	"os"

	"whatsd/internal/config"
)

type CommandFunc func(client *Client, args []string) error

var commands = map[string]CommandFunc{
	"status":              runStatus,
	"pair":                runPair,
	"send":                runSend,
	"send-media":          runSendMedia,
	"edit":                runEdit,
	"revoke":              runRevoke,
	"presence":            runPresence,
	"react":               runReact,
	"chats":               runChats,
	"contacts":            runContacts,
	"contact":             runContact,
	"avatar":              runAvatar,
	"chat-state":          runChatState,
	"group-info":          runGroupInfo,
	"create-group":        runCreateGroup,
	"group-members":       runGroupMembers,
	"post-status":         runPostStatus,
	"statuses":            runStatuses,
	"newsletters":         runNewsletters,
	"follow-newsletter":   runFollowNewsletter,
	"newsletter-messages": runNewsletterMessages,
	"blocklist":           runBlocklist,
	"block":               runBlock,
	"privacy":             runPrivacy,
	"set-privacy":         runSetPrivacy,
	"history":             runHistory,
	"mark-read":           runMarkRead,
	"download":            runDownload,
	"listen":              runListen,
	"logout":              runLogout,
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cfg := config.LoadConfig()
	client := NewClient(cfg.SocketPath)
	cmdName := os.Args[1]

	handler, exists := commands[cmdName]
	if !exists {
		fmt.Printf("Unknown command: %s\n\n", cmdName)
		printUsage()
		os.Exit(1)
	}

	if err := handler(client, os.Args[2:]); err != nil {
		fmt.Printf("Error: %v\n", err)
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
	fmt.Println("  post-status [--text msg] [--file path] Post text or media status update")
	fmt.Println("  statuses [--limit N]  Query recent status / story updates")
	fmt.Println("  newsletters           List followed channels / newsletters")
	fmt.Println("  follow-newsletter --jid <jid> [--unfollow] Follow/unfollow channel")
	fmt.Println("  newsletter-messages --jid <jid> [--limit N] Query channel posts")
	fmt.Println("  blocklist             List blocked contacts")
	fmt.Println("  block --jid <jid> [--unblock] Block or unblock a contact")
	fmt.Println("  privacy               View current privacy settings")
	fmt.Println("  set-privacy --setting <setting> --value <value> Set privacy configuration")
	fmt.Println("  history --chat <jid>  Query message history for a chat")
	fmt.Println("  mark-read --chat <jid> [--ids <id1,id2>] Reset unread count and mark messages read")
	fmt.Println("  download --id <msg_id> [--chat <jid>] Download media for a message")
	fmt.Println("  listen                Listen for incoming messages and daemon events")
	fmt.Println("  logout                Logout current session")
}
