// Package main contains the VGA Events Telegram bot command processor.
// This file provides helper functions for bulk operations on multiple events.
package main

import (
	"fmt"
	"strings"

	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
)

const (
	errNoEventIDs = "❌ No event IDs provided."
)

// parseBulkEventIDs parses event IDs from command parts, supporting both
// space-separated and comma-separated formats.
//
// Examples:
//   - Input: ["abc", "def", "ghi"] -> Output: ["abc", "def", "ghi"]
//   - Input: ["abc,def,ghi"] -> Output: ["abc", "def", "ghi"]
//   - Input: ["abc,def", "ghi"] -> Output: ["abc", "def", "ghi"]
//
// Whitespace around IDs is automatically trimmed.
func parseBulkEventIDs(parts []string) []string {
	var eventIDs []string
	for _, part := range parts {
		// Split by comma in case they're comma-separated
		ids := strings.Split(part, ",")
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id != "" {
				eventIDs = append(eventIDs, id)
			}
		}
	}
	return eventIDs
}

// parseEventIDList parses a comma-separated or space-separated list of event IDs.
// Comma-separated format takes precedence if commas are detected.
//
// Examples:
//   - Input: "abc,def,ghi" -> Output: ["abc", "def", "ghi"]
//   - Input: "abc def ghi" -> Output: ["abc", "def", "ghi"]
//
// Whitespace around IDs is automatically trimmed.
func parseEventIDList(input string) []string {
	var eventIDs []string

	// Try comma-separated first
	if strings.Contains(input, ",") {
		parts := strings.Split(input, ",")
		for _, part := range parts {
			id := strings.TrimSpace(part)
			if id != "" {
				eventIDs = append(eventIDs, id)
			}
		}
		return eventIDs
	}

	// Otherwise treat as space-separated or single ID
	parts := strings.Fields(input)
	for _, part := range parts {
		id := strings.TrimSpace(part)
		if id != "" {
			eventIDs = append(eventIDs, id)
		}
	}

	return eventIDs
}

// handleBulkRegister marks multiple events as registered for a user.
//
// For each event ID:
//   - Updates the event status to "registered" in user preferences
//   - Increments the user's registered events counter
//   - Sets the modified flag if any events were successfully updated
//
// Returns a response message summarizing the operation (success count and failures)
// and nil for the events list (bulk operations don't return event data).
func handleBulkRegister(prefs preferences.Preferences, chatID string, eventIDs []string, modified *bool) (string, []*event.Event) {
	if len(eventIDs) == 0 {
		return errNoEventIDs, nil
	}

	user := prefs.GetUser(chatID)
	successCount := 0
	var failedIDs []string

	for _, eventID := range eventIDs {
		if user.SetEventStatus(eventID, preferences.EventStatusRegistered) {
			successCount++
			*modified = true
			// Track stats
			user.IncrementEventStatus(preferences.EventStatusRegistered)
		} else {
			failedIDs = append(failedIDs, eventID)
		}
	}

	// Build response message
	var response strings.Builder
	if successCount > 0 {
		response.WriteString(fmt.Sprintf("✅ Marked <b>%d</b> event(s) as <b>Registered</b>\n", successCount))
	}

	if len(failedIDs) > 0 {
		response.WriteString(fmt.Sprintf("\n❌ Failed to process %d event(s): %s", len(failedIDs), strings.Join(failedIDs, ", ")))
	}

	if successCount == 0 {
		return "❌ No events were marked as registered.", nil
	}

	return response.String(), nil
}

// handleBulkNote adds the same note text to multiple events.
//
// The note is added to each event in the user's EventNotes map.
// Sets the modified flag if any notes were added.
//
// Returns a response message showing the note text and count of events updated,
// and nil for the events list.
func handleBulkNote(prefs preferences.Preferences, chatID string, eventIDs []string, noteText string, modified *bool) (string, []*event.Event) {
	if len(eventIDs) == 0 {
		return errNoEventIDs, nil
	}

	user := prefs.GetUser(chatID)
	successCount := 0

	if user.EventNotes == nil {
		user.EventNotes = make(map[string]string)
	}

	for _, eventID := range eventIDs {
		user.EventNotes[eventID] = noteText
		successCount++
		*modified = true
	}

	return fmt.Sprintf("📝 Added note to <b>%d</b> event(s):\n\n<i>%s</i>", successCount, noteText), nil
}

// handleBulkStatus sets the same status for multiple events.
//
// Validates the status first (must be: interested, registered, maybe, or skip).
// For each event ID:
//   - Updates the event status in user preferences
//   - Increments the appropriate status counter
//   - Sets the modified flag if successful
//
// Returns a response message with status emoji, count of successful updates,
// and any failed event IDs, along with nil for the events list.
func handleBulkStatus(prefs preferences.Preferences, chatID, status string, eventIDs []string, modified *bool) (string, []*event.Event) {
	if len(eventIDs) == 0 {
		return errNoEventIDs, nil
	}

	// Validate status first
	status = strings.ToLower(strings.TrimSpace(status))
	if status != preferences.EventStatusInterested &&
		status != preferences.EventStatusRegistered &&
		status != preferences.EventStatusMaybe &&
		status != preferences.EventStatusSkip {
		return "❌ Invalid status. Must be one of: interested, registered, maybe, skip", nil
	}

	user := prefs.GetUser(chatID)
	successCount := 0
	var failedIDs []string

	for _, eventID := range eventIDs {
		if user.SetEventStatus(eventID, status) {
			successCount++
			*modified = true
			// Track stats
			user.IncrementEventStatus(status)
		} else {
			failedIDs = append(failedIDs, eventID)
		}
	}

	// Get status emoji and text
	statusEmoji := ""
	statusText := ""
	switch status {
	case preferences.EventStatusInterested:
		statusEmoji = "⭐"
		statusText = "Interested"
	case preferences.EventStatusRegistered:
		statusEmoji = "✅"
		statusText = "Registered"
	case preferences.EventStatusMaybe:
		statusEmoji = "🤔"
		statusText = "Maybe"
	case preferences.EventStatusSkip:
		statusEmoji = "❌"
		statusText = "Skipped"
	}

	// Build response message
	var response strings.Builder
	if successCount > 0 {
		response.WriteString(fmt.Sprintf("%s Marked <b>%d</b> event(s) as <b>%s</b>\n", statusEmoji, successCount, statusText))
	}

	if len(failedIDs) > 0 {
		response.WriteString(fmt.Sprintf("\n❌ Failed to process %d event(s): %s", len(failedIDs), strings.Join(failedIDs, ", ")))
	}

	if successCount == 0 {
		return "❌ No events were updated.", nil
	}

	return response.String(), nil
}
