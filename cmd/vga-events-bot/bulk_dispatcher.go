package main

import (
	"fmt"
	"strings"

	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
)

// processBulkCommand handles all /bulk subcommands
func processBulkCommand(parts []string, prefs preferences.Preferences, chatID string, modified *bool, botToken string, dryRun bool) (string, []*event.Event) {
	// No subcommand - show keyboard menu
	if len(parts) < 2 {
		return handleBulkWithKeyboard(prefs, chatID, botToken, dryRun)
	}

	subcommand := strings.ToLower(parts[1])

	switch subcommand {
	case "register":
		// /bulk register <event_ids>
		if len(parts) < 3 {
			return "❌ Please specify event IDs.\n\nUsage: /bulk register &lt;event_id1&gt; &lt;event_id2&gt; ...\nUsage: /bulk register &lt;event_id1,event_id2,...&gt;", nil
		}
		eventIDs := parseBulkEventIDs(parts[2:])
		return handleBulkRegister(prefs, chatID, eventIDs, modified)

	case "note":
		// /bulk note <event_ids> <note_text>
		if len(parts) < 4 {
			return "❌ Please specify event IDs and note text.\n\nUsage: /bulk note &lt;event_id1,event_id2&gt; &lt;note_text&gt;", nil
		}
		// First part after "note" is event IDs (can be comma-separated or space-separated if quoted)
		eventIDsPart := parts[2]
		eventIDs := parseEventIDList(eventIDsPart)

		// Rest is note text
		noteText := strings.Join(parts[3:], " ")
		noteText, errMsg := validateUserInput(noteText, 500, "Note text")
		if errMsg != "" {
			return errMsg, nil
		}
		return handleBulkNote(prefs, chatID, eventIDs, noteText, modified)

	case "status":
		// /bulk status <status> <event_ids>
		if len(parts) < 4 {
			return "❌ Please specify status and event IDs.\n\nUsage: /bulk status &lt;interested|registered|maybe|skip&gt; &lt;event_id1,event_id2&gt;", nil
		}
		status := strings.ToLower(parts[2])
		eventIDsPart := parts[3]
		eventIDs := parseEventIDList(eventIDsPart)
		return handleBulkStatus(prefs, chatID, status, eventIDs, modified)

	default:
		return fmt.Sprintf("❌ Unknown bulk subcommand: %s\n\nAvailable: register, note, status\nOr use /bulk without parameters for menu.", subcommand), nil
	}
}
