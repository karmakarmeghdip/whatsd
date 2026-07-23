package main

import (
	"flag"
	"fmt"
	"strings"
)

func parseCommaList(s string) []string {
	if s == "" {
		return nil
	}
	var res []string
	for _, p := range strings.Split(s, ",") {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			res = append(res, trimmed)
		}
	}
	return res
}

func runGroupInfo(client *Client, args []string) error {
	groupInfoFlags := flag.NewFlagSet("group-info", flag.ContinueOnError)
	jid := groupInfoFlags.String("jid", "", "Group JID")
	if err := groupInfoFlags.Parse(args); err != nil {
		return err
	}

	if *jid == "" {
		groupInfoFlags.Usage()
		return fmt.Errorf("error: --jid is required")
	}
	return client.CallAndPrint("get_group_info", map[string]any{"group_jid": *jid})
}

func runCreateGroup(client *Client, args []string) error {
	createGroupFlags := flag.NewFlagSet("create-group", flag.ContinueOnError)
	title := createGroupFlags.String("title", "", "Group title/name")
	participants := createGroupFlags.String("participants", "", "Comma-separated list of participant phone numbers or JIDs")
	if err := createGroupFlags.Parse(args); err != nil {
		return err
	}

	if *title == "" {
		createGroupFlags.Usage()
		return fmt.Errorf("error: --title is required")
	}

	params := map[string]any{
		"title": *title,
	}
	if *participants != "" {
		params["participants"] = parseCommaList(*participants)
	}
	return client.CallAndPrint("create_group", params)
}

func runGroupMembers(client *Client, args []string) error {
	groupMembersFlags := flag.NewFlagSet("group-members", flag.ContinueOnError)
	jid := groupMembersFlags.String("jid", "", "Group JID")
	action := groupMembersFlags.String("action", "", "Member action (add|remove|promote|demote)")
	participants := groupMembersFlags.String("participants", "", "Comma-separated list of participant phone numbers or JIDs")
	if err := groupMembersFlags.Parse(args); err != nil {
		return err
	}

	if *jid == "" || *action == "" || *participants == "" {
		groupMembersFlags.Usage()
		return fmt.Errorf("error: --jid, --action, and --participants are required")
	}

	return client.CallAndPrint("update_group_members", map[string]any{
		"group_jid":    *jid,
		"action":       *action,
		"participants": parseCommaList(*participants),
	})
}
