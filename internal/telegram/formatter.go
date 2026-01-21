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

	// Date (if available) - use enhanced formatting
	if evt.DateText != "" {
		niceDate := event.FormatDateNice(evt.DateText)
		msg.WriteString(fmt.Sprintf("📅 %s\n", niceDate))
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

// FormatEventWithCalendar formats an event message with a calendar download button
func FormatEventWithCalendar(evt *event.Event) (string, *InlineKeyboardMarkup) {
	text := FormatEvent(evt)

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "📅 Add to Calendar", CallbackData: fmt.Sprintf("calendar:%s", evt.ID)},
			},
		},
	}

	return text, keyboard
}

// FormatEventWithStatus formats an event message with status and calendar buttons
func FormatEventWithStatus(evt *event.Event, currentStatus string) (string, *InlineKeyboardMarkup) {
	text := FormatEvent(evt)

	// Add current status indicator to text if status is set
	if currentStatus != "" {
		statusEmoji := ""
		statusText := ""
		switch currentStatus {
		case "interested":
			statusEmoji = "⭐"
			statusText = "Interested"
		case "registered":
			statusEmoji = "✅"
			statusText = "Registered"
		case "maybe":
			statusEmoji = "🤔"
			statusText = "Maybe"
		case "skip":
			statusEmoji = "❌"
			statusText = "Skipped"
		}
		if statusEmoji != "" {
			text = fmt.Sprintf("%s %s <b>%s</b>\n\n%s", statusEmoji, statusEmoji, statusText, text)
		}
	}

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "📅 Calendar", CallbackData: fmt.Sprintf("calendar:%s", evt.ID)},
			},
			{
				{Text: "⭐ Interested", CallbackData: fmt.Sprintf("status:%s:interested", evt.ID)},
				{Text: "✅ Registered", CallbackData: fmt.Sprintf("status:%s:registered", evt.ID)},
			},
			{
				{Text: "🤔 Maybe", CallbackData: fmt.Sprintf("status:%s:maybe", evt.ID)},
				{Text: "❌ Skip", CallbackData: fmt.Sprintf("status:%s:skip", evt.ID)},
			},
		},
	}

	return text, keyboard
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

// FormatReminder formats a reminder message for an upcoming event
func FormatReminder(evt *event.Event, daysUntil int) (string, *InlineKeyboardMarkup) {
	var msg strings.Builder

	// Reminder header with emoji
	msg.WriteString("⏰ <b>Event Reminder!</b>\n\n")

	// Days until message
	if daysUntil == 1 {
		msg.WriteString("📅 <b>Tomorrow!</b>\n\n")
	} else if daysUntil == 7 {
		msg.WriteString("📅 <b>In 1 week</b>\n\n")
	} else if daysUntil == 14 {
		msg.WriteString("📅 <b>In 2 weeks</b>\n\n")
	} else {
		msg.WriteString(fmt.Sprintf("📅 <b>In %d days</b>\n\n", daysUntil))
	}

	// Event details
	msg.WriteString(fmt.Sprintf("🏌️ <b>%s</b> - %s\n", evt.State, evt.Title))

	// Date (if available) - use enhanced formatting
	if evt.DateText != "" {
		niceDate := event.FormatDateNice(evt.DateText)
		msg.WriteString(fmt.Sprintf("📆 %s\n", niceDate))
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
	msg.WriteString(fmt.Sprintf("\n#VGAGolf #Golf %s #Reminder", stateHashtag))

	// Keyboard with calendar and status tracking
	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "📅 Calendar", CallbackData: fmt.Sprintf("calendar:%s", evt.ID)},
			},
			{
				{Text: "⭐ Interested", CallbackData: fmt.Sprintf("status:%s:interested", evt.ID)},
				{Text: "✅ Registered", CallbackData: fmt.Sprintf("status:%s:registered", evt.ID)},
			},
			{
				{Text: "🤔 Maybe", CallbackData: fmt.Sprintf("status:%s:maybe", evt.ID)},
				{Text: "❌ Skip", CallbackData: fmt.Sprintf("status:%s:skip", evt.ID)},
			},
		},
	}

	return msg.String(), keyboard
}
