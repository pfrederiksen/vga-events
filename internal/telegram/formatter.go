package telegram

import (
	"fmt"
	"strings"

	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
)

// TeeDetails represents a single tee's information
type TeeDetails struct {
	Name    string
	Par     int
	Yardage int
	Slope   int
	Rating  float64
	Holes   int
}

// CourseDetails represents golf course information for formatting
type CourseDetails struct {
	Name    string
	Tees    []TeeDetails
	Website string
	Phone   string
}

// FormatEvent formats a single event as a Telegram message
func FormatEvent(evt *event.Event) string {
	return FormatEventWithNote(evt, "")
}

// FormatEventWithNote formats an event with an optional note
func FormatEventWithNote(evt *event.Event, note string) string {
	var msg strings.Builder

	// Header with emoji (include 📝 if note exists)
	if note != "" {
		msg.WriteString("🏌️ 📝 <b>New VGA Golf Event!</b>\n\n")
	} else {
		msg.WriteString("🏌️ <b>New VGA Golf Event!</b>\n\n")
	}

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

	// Note (if available)
	if note != "" {
		msg.WriteString(fmt.Sprintf("\n📝 <i>%s</i>\n", note))
	}

	// Registration link
	msg.WriteString("\n🔗 <a href=\"https://vgagolf.org/state-events\">vgagolf.org/state-events</a>\n")
	msg.WriteString("<i>(login required)</i>\n")

	// Hashtags
	stateHashtag := fmt.Sprintf("#%s", strings.ReplaceAll(evt.State, " ", ""))
	msg.WriteString(fmt.Sprintf("\n#VGAGolf #Golf %s", stateHashtag))

	return msg.String()
}

// FormatEventWithCourse formats an event with optional course details
func FormatEventWithCourse(evt *event.Event, course *CourseDetails, note string) string {
	var msg strings.Builder

	// Header with emoji (include 📝 if note exists)
	if note != "" {
		msg.WriteString("🏌️ 📝 <b>New VGA Golf Event!</b>\n\n")
	} else {
		msg.WriteString("🏌️ <b>New VGA Golf Event!</b>\n\n")
	}

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

	// Course details (if available)
	if course != nil && len(course.Tees) > 0 {
		msg.WriteString("\n")
		msg.WriteString(fmt.Sprintf("⛳ <b>%s</b>\n", course.Name))

		// Show all tees
		for _, tee := range course.Tees {
			msg.WriteString(fmt.Sprintf("  <i>%s:</i> ", tee.Name))

			// Par and yardage
			if tee.Par > 0 && tee.Yardage > 0 {
				msg.WriteString(fmt.Sprintf("Par %d, %s yds", tee.Par, formatYardage(tee.Yardage)))
			}

			// Slope and rating
			if tee.Slope > 0 {
				msg.WriteString(fmt.Sprintf(", Slope %d", tee.Slope))
			}
			if tee.Rating > 0 {
				msg.WriteString(fmt.Sprintf(", %.1f", tee.Rating))
			}
			msg.WriteString("\n")
		}

		// Website if available
		if course.Website != "" {
			msg.WriteString(fmt.Sprintf("🌐 %s\n", course.Website))
		}

		// Phone if available
		if course.Phone != "" {
			msg.WriteString(fmt.Sprintf("📞 %s\n", course.Phone))
		}
	}

	// Note (if available)
	if note != "" {
		msg.WriteString(fmt.Sprintf("\n📝 <i>%s</i>\n", note))
	}

	// Registration link
	msg.WriteString("\n🔗 <a href=\"https://vgagolf.org/state-events\">vgagolf.org/state-events</a>\n")
	msg.WriteString("<i>(login required)</i>\n")

	// Hashtags
	stateHashtag := fmt.Sprintf("#%s", strings.ReplaceAll(evt.State, " ", ""))
	msg.WriteString(fmt.Sprintf("\n#VGAGolf #Golf %s", stateHashtag))

	return msg.String()
}

// formatYardage formats yardage with comma for thousands
func formatYardage(yardage int) string {
	if yardage >= 10000 {
		return fmt.Sprintf("%d,%03d", yardage/1000, yardage%1000)
	} else if yardage >= 1000 {
		return fmt.Sprintf("%d,%03d", yardage/1000, yardage%1000)
	}
	return fmt.Sprintf("%d", yardage)
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
	return FormatEventWithStatusAndNote(evt, currentStatus, "", "", nil)
}

// FormatEventWithStatusAndNote formats an event message with status, note, friend count, and calendar buttons
func FormatEventWithStatusAndNote(evt *event.Event, currentStatus, note, chatID string, prefs preferences.Preferences) (string, *InlineKeyboardMarkup) {
	text := FormatEventWithNote(evt, note)

	// Add friend count if user has friends registered/interested in this event
	if chatID != "" && prefs != nil {
		friendIDs := prefs.GetFriendsForEvent(chatID, evt.ID)
		if len(friendIDs) > 0 {
			friendText := ""
			if len(friendIDs) == 1 {
				friendText = "\n👥 <b>1 friend</b> registered for this event\n"
			} else {
				friendText = fmt.Sprintf("\n👥 <b>%d friends</b> registered for this event\n", len(friendIDs))
			}
			// Insert friend info after the course info and before the registration link
			text = strings.Replace(text, "\n🔗 <a href=", friendText+"\n🔗 <a href=", 1)
		}
	}

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

// FormatEventWithStatusAndCourse formats an event with course info, status, note, and interactive buttons
func FormatEventWithStatusAndCourse(evt *event.Event, course *CourseDetails, currentStatus, note, chatID string, prefs preferences.Preferences) (string, *InlineKeyboardMarkup) {
	text := FormatEventWithCourse(evt, course, note)

	// Add friend count if user has friends registered/interested in this event
	if chatID != "" && prefs != nil {
		friendIDs := prefs.GetFriendsForEvent(chatID, evt.ID)
		if len(friendIDs) > 0 {
			friendText := ""
			if len(friendIDs) == 1 {
				friendText = "\n👥 <b>1 friend</b> registered for this event\n"
			} else {
				friendText = fmt.Sprintf("\n👥 <b>%d friends</b> registered for this event\n", len(friendIDs))
			}
			// Insert friend info after the course info and before the registration link
			text = strings.Replace(text, "\n🔗 <a href=", friendText+"\n🔗 <a href=", 1)
		}
	}

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

// FormatEventChange formats an event change notification
func FormatEventChange(evt *event.Event, changeType, oldValue, newValue string) string {
	var msg strings.Builder

	msg.WriteString("⚠️ <b>Event Updated!</b>\n\n")
	msg.WriteString(fmt.Sprintf("📍 <b>%s</b> - %s\n\n", evt.State, evt.Title))

	// Show what changed
	switch changeType {
	case "date":
		msg.WriteString("📅 <b>Date Changed:</b>\n")
		if oldValue != "" {
			msg.WriteString(fmt.Sprintf("  ❌ <s>%s</s>\n", oldValue))
		} else {
			msg.WriteString("  ❌ <s>No date</s>\n")
		}
		if newValue != "" {
			niceDate := event.FormatDateNice(newValue)
			msg.WriteString(fmt.Sprintf("  ✅ %s\n", niceDate))
		} else {
			msg.WriteString("  ✅ No date\n")
		}
	case "title":
		msg.WriteString("🏌️ <b>Title Changed:</b>\n")
		msg.WriteString(fmt.Sprintf("  ❌ <s>%s</s>\n", oldValue))
		msg.WriteString(fmt.Sprintf("  ✅ %s\n", newValue))
	case "city":
		msg.WriteString("🏢 <b>City Changed:</b>\n")
		if oldValue != "" {
			msg.WriteString(fmt.Sprintf("  ❌ <s>%s</s>\n", oldValue))
		} else {
			msg.WriteString("  ❌ <s>No city</s>\n")
		}
		if newValue != "" {
			msg.WriteString(fmt.Sprintf("  ✅ %s\n", newValue))
		} else {
			msg.WriteString("  ✅ No city\n")
		}
	}

	msg.WriteString("\n🔗 <a href=\"https://vgagolf.org/state-events\">vgagolf.org/state-events</a>\n")
	msg.WriteString("<i>(login required)</i>\n")

	return msg.String()
}

// FormatEventChangeWithKeyboard formats an event change with interactive buttons
func FormatEventChangeWithKeyboard(evt *event.Event, changeType, oldValue, newValue string, currentStatus string) (string, *InlineKeyboardMarkup) {
	text := FormatEventChange(evt, changeType, oldValue, newValue)

	// Add current status indicator if set
	if currentStatus != "" {
		statusEmoji := ""
		statusText := ""
		switch currentStatus {
		case "interested":
			statusEmoji = "⭐"
			statusText = "You marked this as Interested"
		case "registered":
			statusEmoji = "✅"
			statusText = "You're Registered for this event"
		case "maybe":
			statusEmoji = "🤔"
			statusText = "You marked this as Maybe"
		}
		if statusEmoji != "" {
			text = fmt.Sprintf("%s\n\n%s <i>%s</i>", text, statusEmoji, statusText)
		}
	}

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "📅 Update Calendar", CallbackData: fmt.Sprintf("calendar:%s", evt.ID)},
			},
			{
				{Text: "✅ Acknowledged", CallbackData: fmt.Sprintf("ack-change:%s", evt.ID)},
			},
		},
	}

	return text, keyboard
}
