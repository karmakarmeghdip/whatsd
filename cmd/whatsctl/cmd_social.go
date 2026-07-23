package main

import (
	"flag"
	"fmt"
)

func runPostStatus(client *Client, args []string) error {
	postStatusFlags := flag.NewFlagSet("post-status", flag.ContinueOnError)
	text := postStatusFlags.String("text", "", "Status update text")
	filePath := postStatusFlags.String("file", "", "Local media file path")
	mediaType := postStatusFlags.String("type", "", "Media type (image|video)")
	caption := postStatusFlags.String("caption", "", "Optional caption")
	if err := postStatusFlags.Parse(args); err != nil {
		return err
	}

	if *text == "" && *filePath == "" {
		postStatusFlags.Usage()
		return fmt.Errorf("error: must specify either --text or --file")
	}

	return client.CallAndPrint("post_status", map[string]any{
		"text":       *text,
		"file_path":  *filePath,
		"media_type": *mediaType,
		"caption":    *caption,
	})
}

func runStatuses(client *Client, args []string) error {
	statusesFlags := flag.NewFlagSet("statuses", flag.ContinueOnError)
	limit := statusesFlags.Int("limit", 50, "Limit number of status updates")
	if err := statusesFlags.Parse(args); err != nil {
		return err
	}

	return client.CallAndPrint("get_statuses", map[string]any{"limit": *limit})
}

func runNewsletters(client *Client, args []string) error {
	return client.CallAndPrint("get_newsletters", nil)
}

func runFollowNewsletter(client *Client, args []string) error {
	followFlags := flag.NewFlagSet("follow-newsletter", flag.ContinueOnError)
	jid := followFlags.String("jid", "", "Newsletter / Channel JID")
	unfollow := followFlags.Bool("unfollow", false, "Unfollow channel instead of follow")
	if err := followFlags.Parse(args); err != nil {
		return err
	}

	if *jid == "" {
		followFlags.Usage()
		return fmt.Errorf("error: --jid is required")
	}

	return client.CallAndPrint("follow_newsletter", map[string]any{
		"newsletter_jid": *jid,
		"unfollow":       *unfollow,
	})
}

func runNewsletterMessages(client *Client, args []string) error {
	msgsFlags := flag.NewFlagSet("newsletter-messages", flag.ContinueOnError)
	jid := msgsFlags.String("jid", "", "Newsletter / Channel JID")
	limit := msgsFlags.Int("limit", 50, "Maximum posts to fetch")
	beforeID := msgsFlags.Int64("before", 0, "Server message ID for pagination")
	if err := msgsFlags.Parse(args); err != nil {
		return err
	}

	if *jid == "" {
		msgsFlags.Usage()
		return fmt.Errorf("error: --jid is required")
	}

	params := map[string]any{
		"newsletter_jid": *jid,
		"limit":          *limit,
	}
	if *beforeID > 0 {
		params["before_id"] = *beforeID
	}
	return client.CallAndPrint("get_newsletter_messages", params)
}

func runBlocklist(client *Client, args []string) error {
	return client.CallAndPrint("get_blocklist", nil)
}

func runBlock(client *Client, args []string) error {
	blockFlags := flag.NewFlagSet("block", flag.ContinueOnError)
	jid := blockFlags.String("jid", "", "Contact JID or phone number")
	unblock := blockFlags.Bool("unblock", false, "Unblock contact")
	if err := blockFlags.Parse(args); err != nil {
		return err
	}

	if *jid == "" {
		blockFlags.Usage()
		return fmt.Errorf("error: --jid is required")
	}

	action := "block"
	if *unblock {
		action = "unblock"
	}
	return client.CallAndPrint("block_contact", map[string]any{
		"jid":    *jid,
		"action": action,
	})
}

func runPrivacy(client *Client, args []string) error {
	return client.CallAndPrint("get_privacy_settings", nil)
}

func runSetPrivacy(client *Client, args []string) error {
	setPrivacyFlags := flag.NewFlagSet("set-privacy", flag.ContinueOnError)
	setting := setPrivacyFlags.String("setting", "", "Setting name (group_add|last_seen|status|profile_photo|read_receipts|online)")
	value := setPrivacyFlags.String("value", "", "Setting value (all|contacts|contact_blacklist|none|match_last_seen)")
	if err := setPrivacyFlags.Parse(args); err != nil {
		return err
	}

	if *setting == "" || *value == "" {
		setPrivacyFlags.Usage()
		return fmt.Errorf("error: --setting and --value are required")
	}

	return client.CallAndPrint("set_privacy_setting", map[string]any{
		"setting": *setting,
		"value":   *value,
	})
}
