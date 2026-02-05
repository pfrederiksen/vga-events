package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/pfrederiksen/vga-events/internal/calendar"
	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
	"github.com/pfrederiksen/vga-events/internal/scraper"
	"github.com/pfrederiksen/vga-events/internal/telegram"
)

// handleMenuCallback handles menu navigation callbacks
func handleMenuCallback(param string, prefs preferences.Preferences, chatID string, botToken string, dryRun bool, modified *bool) (string, *telegram.InlineKeyboardMarkup) {
	switch param {
	case "main":
		return showMenuKeyboard()
	case "all-events":
		responseText, _ := handleAllEvents(prefs, chatID, botToken, dryRun, modified)
		return responseText, nil
	case "my-events":
		responseText, _ := handleMyEvents(prefs, chatID, botToken, dryRun, modified)
		return responseText, nil
	case "upcoming":
		return handleUpcomingEventsCallback(prefs, chatID, botToken, dryRun), nil
	case "reminders":
		return showRemindersKeyboard(prefs, chatID)
	case "search":
		return `🔍 <b>Search Events</b>

To search for events, use the command:
/search &lt;keyword&gt;

Example:
/search "Pine Valley"
/search Championship
/search Las Vegas`, nil
	case "help":
		return getHelpMessage(), nil
	default:
		return "Unknown menu action", nil
	}
}

// handleStatusCallback handles event status update callbacks
func handleStatusCallback(callbackData string, prefs preferences.Preferences, chatID string, modified *bool) string {
	// Format: status:EVENT_ID:STATUS (e.g., "status:abc123:interested")
	parts := strings.Split(callbackData, ":")
	if len(parts) != 3 {
		return "❌ Invalid status request"
	}
	eventID := parts[1]
	status := parts[2]

	user := prefs.GetUser(chatID)
	if user.SetEventStatus(eventID, status) {
		*modified = true

		// Track stats: event marked with status
		user.IncrementEventStatus(status)

		// Get status emoji and text
		statusEmoji, statusText := getStatusDisplay(status)

		return fmt.Sprintf("%s Event marked as <b>%s</b>", statusEmoji, statusText)
	}
	return "❌ Invalid status"
}

// handleReminderCallback handles reminder configuration callbacks
func handleReminderCallback(callbackData string, prefs preferences.Preferences, chatID string, modified *bool) (string, *telegram.InlineKeyboardMarkup) {
	// Format: reminder:ACTION:DAYS (e.g., "reminder:add:7" or "reminder:done:0")
	parts := strings.Split(callbackData, ":")
	if len(parts) != 3 {
		return "❌ Invalid reminder request", nil
	}
	action := parts[1]
	days := 0
	_, _ = fmt.Sscanf(parts[2], "%d", &days) // Error ignored, days defaults to 0

	user := prefs.GetUser(chatID)

	switch action {
	case "add":
		// Add reminder day if not already present
		if !user.HasReminderDay(days) {
			user.ReminderDays = append(user.ReminderDays, days)
			*modified = true
		}
		// Update keyboard to show new selection
		return showRemindersKeyboard(prefs, chatID)

	case "remove":
		// Remove reminder day
		newDays := []int{}
		for _, d := range user.ReminderDays {
			if d != days {
				newDays = append(newDays, d)
			}
		}
		user.ReminderDays = newDays
		*modified = true
		// Update keyboard to show new selection
		return showRemindersKeyboard(prefs, chatID)

	case "done":
		// Save and close
		responseText := "✅ Reminder settings saved!"
		if len(user.ReminderDays) > 0 {
			reminders := []string{}
			for _, day := range user.ReminderDays {
				switch day {
				case 1:
					reminders = append(reminders, "1 day")
				case 3:
					reminders = append(reminders, "3 days")
				case 7:
					reminders = append(reminders, "1 week")
				case 14:
					reminders = append(reminders, "2 weeks")
				}
			}
			responseText += fmt.Sprintf("\n\nYou'll be reminded <b>%s</b> before events you've marked as ⭐ Interested or ✅ Registered.", strings.Join(reminders, ", "))
		} else {
			responseText += "\n\nNo reminders configured. Use /reminders to set them up."
		}
		return responseText, nil

	default:
		return "❌ Unknown reminder action", nil
	}
}

// handleCalendarCallback handles calendar download callbacks
func handleCalendarCallback(eventID string, chatID string, botToken string, dryRun bool) string {
	// Fetch fresh events from VGA website to find the event
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return "❌ Error fetching event data"
	}

	// Find the event by ID
	var evt *event.Event
	for _, e := range allEvents {
		if e.ID == eventID {
			evt = e
			break
		}
	}

	if evt == nil {
		fmt.Fprintf(os.Stderr, "Event %s not found in current events\n", eventID)
		return "❌ Event not found. This event may have been removed from the VGA website."
	}

	// Generate .ics file
	icsContent := calendar.GenerateICS(evt)
	filename := fmt.Sprintf("vga-event-%s.ics", evt.State)

	// Send the .ics file
	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating Telegram client: %v\n", err)
			return errSendingCalendarFile
		}

		caption := fmt.Sprintf("📅 <b>%s - %s</b>\n\nTap to add to your calendar!", evt.State, evt.Title)
		if err := client.SendDocument(filename, []byte(icsContent), caption); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending document: %v\n", err)
			return errSendingCalendarFile
		}

		return "✅ Calendar file sent! Tap it to add the event to your calendar."
	}

	return fmt.Sprintf("[DRY RUN] Would send .ics file for event: %s - %s", evt.State, evt.Title)
}
