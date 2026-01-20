package telegram

import (
	"fmt"
	"strings"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// FormatEvent formats a single event as a Telegram message
func FormatEvent(evt *event.Event) string {
	var msg strings.Builder

	// Header with emoji
	msg.WriteString("🏌️ <b>New VGA Golf Event!</b>\n\n")

	// State and course
	msg.WriteString(fmt.Sprintf("📍 <b>%s</b> - %s\n", evt.State, evt.Title))

	// Date (if available)
	if evt.DateText != "" {
		msg.WriteString(fmt.Sprintf("📅 %s\n", evt.DateText))
	}

	// City (if available)
	if evt.City != "" {
		msg.WriteString(fmt.Sprintf("🏢 %s\n", evt.City))
	}

	// Registration link
	msg.WriteString("\n🔗 <a href=\"https://vgagolf.org/state-events\">vgagolf.org/state-events</a>\n")
	msg.WriteString("<i>(login required)</i>\n")

	// Hashtags
	stateHashtag := fmt.Sprintf("#%s", strings.ReplaceAll(evt.State, " ", ""))
	msg.WriteString(fmt.Sprintf("\n#VGAGolf #Golf %s", stateHashtag))

	return msg.String()
}

// FormatSummary creates a summary message for multiple events
func FormatSummary(count int, states []string) string {
	var msg strings.Builder

	msg.WriteString("🏌️ <b>VGA Events Update</b>\n\n")
	msg.WriteString(fmt.Sprintf("Found <b>%d</b> new event", count))
	if count != 1 {
		msg.WriteString("s")
	}

	if len(states) > 0 {
		msg.WriteString(fmt.Sprintf(" in %d state", len(states)))
		if len(states) != 1 {
			msg.WriteString("s")
		}
		msg.WriteString(fmt.Sprintf(": %s", strings.Join(states, ", ")))
	}

	msg.WriteString("\n\n#VGAGolf")

	return msg.String()
}
