package main

import (
	"flag"
	"fmt"
	"strings"
)

func runChats(client *Client, args []string) error {
	chatsFlags := flag.NewFlagSet("chats", flag.ContinueOnError)
	limit := chatsFlags.Int("limit", 50, "Maximum number of chats to return")
	if err := chatsFlags.Parse(args); err != nil {
		return err
	}
	return client.CallAndPrint("get_chats", map[string]any{"limit": *limit})
}

func runContacts(client *Client, args []string) error {
	contactsFlags := flag.NewFlagSet("contacts", flag.ContinueOnError)
	query := contactsFlags.String("query", "", "Search query for name, push name, or JID")
	if err := contactsFlags.Parse(args); err != nil {
		return err
	}
	return client.CallAndPrint("get_contacts", map[string]any{"query": *query})
}

func runContact(client *Client, args []string) error {
	contactFlags := flag.NewFlagSet("contact", flag.ContinueOnError)
	jid := contactFlags.String("jid", "", "Contact phone number or JID")
	if err := contactFlags.Parse(args); err != nil {
		return err
	}

	if *jid == "" {
		contactFlags.Usage()
		return fmt.Errorf("error: --jid is required")
	}
	return client.CallAndPrint("get_contact", map[string]any{"jid": *jid})
}

func runAvatar(client *Client, args []string) error {
	avatarFlags := flag.NewFlagSet("avatar", flag.ContinueOnError)
	jid := avatarFlags.String("jid", "", "Contact or Group JID")
	preview := avatarFlags.Bool("preview", false, "Fetch low-res preview image")
	if err := avatarFlags.Parse(args); err != nil {
		return err
	}

	if *jid == "" {
		avatarFlags.Usage()
		return fmt.Errorf("error: --jid is required")
	}
	return client.CallAndPrint("get_profile_picture", map[string]any{
		"jid":     *jid,
		"preview": *preview,
	})
}

func runChatState(client *Client, args []string) error {
	chatStateFlags := flag.NewFlagSet("chat-state", flag.ContinueOnError)
	chat := chatStateFlags.String("chat", "", "Chat JID")
	action := chatStateFlags.String("action", "", "Action (mute|unmute|pin|unpin|archive|unarchive)")
	duration := chatStateFlags.String("duration", "", "Optional mute duration (e.g. 8h, 24h)")
	if err := chatStateFlags.Parse(args); err != nil {
		return err
	}

	if *chat == "" || *action == "" {
		chatStateFlags.Usage()
		return fmt.Errorf("error: --chat and --action are required")
	}

	params := map[string]any{
		"chat":   *chat,
		"action": *action,
	}
	if *duration != "" {
		params["mute_duration"] = *duration
	}
	return client.CallAndPrint("set_chat_state", params)
}

func runHistory(client *Client, args []string) error {
	historyFlags := flag.NewFlagSet("history", flag.ContinueOnError)
	chat := historyFlags.String("chat", "", "Chat JID or phone number")
	limit := historyFlags.Int("limit", 50, "Maximum number of messages to return")
	beforeID := historyFlags.String("before", "", "Message ID cursor for pagination")
	if err := historyFlags.Parse(args); err != nil {
		return err
	}

	if *chat == "" {
		historyFlags.Usage()
		return fmt.Errorf("error: --chat JID is required")
	}

	params := map[string]any{
		"chat":  *chat,
		"limit": *limit,
	}
	if *beforeID != "" {
		params["before_id"] = *beforeID
	}
	return client.CallAndPrint("get_messages", params)
}

func runMarkRead(client *Client, args []string) error {
	markReadFlags := flag.NewFlagSet("mark-read", flag.ContinueOnError)
	chat := markReadFlags.String("chat", "", "Chat JID to mark as read")
	ids := markReadFlags.String("ids", "", "Comma-separated list of message IDs to send read receipt for")
	if err := markReadFlags.Parse(args); err != nil {
		return err
	}

	if *chat == "" {
		markReadFlags.Usage()
		return fmt.Errorf("error: --chat is required")
	}

	params := map[string]any{
		"chat": *chat,
	}
	if *ids != "" {
		var idList []string
		for _, id := range strings.Split(*ids, ",") {
			if trimmed := strings.TrimSpace(id); trimmed != "" {
				idList = append(idList, trimmed)
			}
		}
		params["message_ids"] = idList
	}
	return client.CallAndPrint("mark_read", params)
}

func runDownload(client *Client, args []string) error {
	downloadFlags := flag.NewFlagSet("download", flag.ContinueOnError)
	id := downloadFlags.String("id", "", "Message ID to download")
	chat := downloadFlags.String("chat", "", "Chat JID (optional)")
	if err := downloadFlags.Parse(args); err != nil {
		return err
	}

	if *id == "" {
		downloadFlags.Usage()
		return fmt.Errorf("error: --id is required for download")
	}

	params := map[string]any{
		"message_id": *id,
	}
	if *chat != "" {
		params["chat"] = *chat
	}
	return client.CallAndPrint("download_media", params)
}
