package main

import (
	"flag"
	"fmt"
)

func runSend(client *Client, args []string) error {
	sendFlags := flag.NewFlagSet("send", flag.ContinueOnError)
	to := sendFlags.String("to", "", "Recipient phone number or JID")
	text := sendFlags.String("text", "", "Message text")
	replyToID := sendFlags.String("reply", "", "Message ID to reply to")
	if err := sendFlags.Parse(args); err != nil {
		return err
	}

	if *to == "" || *text == "" {
		sendFlags.Usage()
		return fmt.Errorf("error: --to and --text are required")
	}

	params := map[string]any{
		"to":   *to,
		"text": *text,
	}
	if *replyToID != "" {
		params["reply_to_id"] = *replyToID
	}
	return client.CallAndPrint("send_message", params)
}

func runSendMedia(client *Client, args []string) error {
	sendMediaFlags := flag.NewFlagSet("send-media", flag.ContinueOnError)
	to := sendMediaFlags.String("to", "", "Recipient phone number or JID")
	mediaType := sendMediaFlags.String("type", "", "Media type (image|video|audio|document)")
	filePath := sendMediaFlags.String("file", "", "Local file path")
	caption := sendMediaFlags.String("caption", "", "Optional caption")
	fileName := sendMediaFlags.String("name", "", "Optional file name")
	if err := sendMediaFlags.Parse(args); err != nil {
		return err
	}

	if *to == "" || *mediaType == "" || *filePath == "" {
		sendMediaFlags.Usage()
		return fmt.Errorf("error: --to, --type, and --file are required")
	}

	params := map[string]any{
		"to":         *to,
		"media_type": *mediaType,
		"file_path":  *filePath,
		"caption":    *caption,
		"file_name":  *fileName,
	}
	return client.CallAndPrint("send_media", params)
}

func runEdit(client *Client, args []string) error {
	editFlags := flag.NewFlagSet("edit", flag.ContinueOnError)
	chat := editFlags.String("chat", "", "Chat JID")
	id := editFlags.String("id", "", "Message ID to edit")
	text := editFlags.String("text", "", "New message text")
	if err := editFlags.Parse(args); err != nil {
		return err
	}

	if *chat == "" || *id == "" || *text == "" {
		editFlags.Usage()
		return fmt.Errorf("error: --chat, --id, and --text are required")
	}

	return client.CallAndPrint("edit_message", map[string]any{
		"chat":       *chat,
		"message_id": *id,
		"new_text":   *text,
	})
}

func runRevoke(client *Client, args []string) error {
	revokeFlags := flag.NewFlagSet("revoke", flag.ContinueOnError)
	chat := revokeFlags.String("chat", "", "Chat JID")
	id := revokeFlags.String("id", "", "Message ID to revoke")
	if err := revokeFlags.Parse(args); err != nil {
		return err
	}

	if *chat == "" || *id == "" {
		revokeFlags.Usage()
		return fmt.Errorf("error: --chat and --id are required")
	}

	return client.CallAndPrint("revoke_message", map[string]any{
		"chat":       *chat,
		"message_id": *id,
	})
}

func runPresence(client *Client, args []string) error {
	presenceFlags := flag.NewFlagSet("presence", flag.ContinueOnError)
	chat := presenceFlags.String("chat", "", "Chat JID")
	state := presenceFlags.String("state", "", "Presence state (composing|recording|paused)")
	if err := presenceFlags.Parse(args); err != nil {
		return err
	}

	if *chat == "" || *state == "" {
		presenceFlags.Usage()
		return fmt.Errorf("error: --chat and --state are required")
	}

	return client.CallAndPrint("send_presence", map[string]any{
		"chat":  *chat,
		"state": *state,
	})
}

func runReact(client *Client, args []string) error {
	reactFlags := flag.NewFlagSet("react", flag.ContinueOnError)
	chat := reactFlags.String("chat", "", "Chat JID")
	id := reactFlags.String("id", "", "Message ID to react to")
	emoji := reactFlags.String("emoji", "", "Emoji string")
	if err := reactFlags.Parse(args); err != nil {
		return err
	}

	if *chat == "" || *id == "" || *emoji == "" {
		reactFlags.Usage()
		return fmt.Errorf("error: --chat, --id, and --emoji are required")
	}

	return client.CallAndPrint("react_message", map[string]any{
		"chat":       *chat,
		"message_id": *id,
		"emoji":      *emoji,
	})
}
