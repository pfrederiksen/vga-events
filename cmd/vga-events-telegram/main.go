package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/telegram"
)

var (
	botToken       = flag.String("bot-token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token (or env: TELEGRAM_BOT_TOKEN)")
	chatID         = flag.String("chat-id", os.Getenv("TELEGRAM_CHAT_ID"), "Telegram chat ID (or env: TELEGRAM_CHAT_ID)")
	eventsFile     = flag.String("events-file", "", "Path to events JSON file (or read from stdin)")
	dryRun         = flag.Bool("dry-run", false, "Print messages without sending")
	maxMessages    = flag.Int("max-messages", 10, "Maximum number of messages to send")
	stateFilter    = flag.String("state", "", "Only send messages for this state")
	hidePast       = flag.Bool("hide-past", true, "Filter out past events (default: true)")
	daysAhead      = flag.Int("days-ahead", 0, "Only show events within N days (0 = disabled)")
	checkReminders = flag.Bool("check-reminders", false, "Check if event matches reminder days (exits 0 if match, 1 if no match)")
	reminderDays   = flag.Int("reminder-days", 0, "Number of days before event to send reminder (used with --check-reminders)")
)

// filterByState filters events by state code
func filterByState(events []*event.Event, state string) []*event.Event {
	if state == "" {
		return events
	}
	filtered := make([]*event.Event, 0)
	for _, evt := range events {
		if evt.State == state {
			filtered = append(filtered, evt)
		}
	}
	return filtered
}

// filterByTime applies time-based filtering (past events, days ahead)
func filterByTime(events []*event.Event, hidePastEvents bool, daysAheadFilter int) []*event.Event {
	if !hidePastEvents && daysAheadFilter <= 0 {
		return events
	}

	filtered := make([]*event.Event, 0)
	for _, evt := range events {
		// Filter past events if enabled
		if hidePastEvents && evt.IsPastEvent() {
			continue
		}
		// Filter events beyond days_ahead window if enabled
		if daysAheadFilter > 0 && !evt.IsWithinDays(daysAheadFilter) {
			continue
		}
		filtered = append(filtered, evt)
	}
	return filtered
}

// filterByReminderDays filters events that are exactly X days away
func filterByReminderDays(events []*event.Event, days int) []*event.Event {
	filtered := make([]*event.Event, 0)
	for _, evt := range events {
		parsedDate := event.ParseDate(evt.DateText)
		if parsedDate.IsZero() {
			continue
		}

		// Calculate days until event (at midnight)
		now := time.Now()
		today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		eventDate := time.Date(parsedDate.Year(), parsedDate.Month(), parsedDate.Day(), 0, 0, 0, 0, now.Location())
		daysUntil := int(eventDate.Sub(today).Hours() / 24)

		if daysUntil == days {
			filtered = append(filtered, evt)
		}
	}
	return filtered
}

// readEvents reads events from file or stdin
func readEvents(filePath string) ([]*event.Event, error) {
	var reader io.Reader
	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, fmt.Errorf("opening events file: %w", err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "Error closing file: %v\n", err)
			}
		}()
		reader = f
	} else {
		reader = os.Stdin
	}

	var result struct {
		NewEvents []*event.Event `json:"new_events"`
	}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	return result.NewEvents, nil
}

func main() {
	flag.Parse()

	// Read events from file or stdin
	events, err := readEvents(*eventsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading events: %v\n", err)
		os.Exit(1)
	}

	if len(events) == 0 {
		fmt.Println("No new events to send")
		os.Exit(0)
	}

	// Apply filters
	events = filterByState(events, *stateFilter)
	events = filterByTime(events, *hidePast, *daysAhead)

	// Check reminders mode
	if *checkReminders {
		if *reminderDays <= 0 {
			fmt.Fprintf(os.Stderr, "Error: --reminder-days must be positive when using --check-reminders\n")
			os.Exit(1)
		}

		events = filterByReminderDays(events, *reminderDays)

		// If no events match, exit with code 1 (no reminder to send)
		if len(events) == 0 {
			os.Exit(1)
		}
	}

	// Limit number of messages
	if len(events) > *maxMessages {
		events = events[:*maxMessages]
	}

	if len(events) == 0 {
		fmt.Println("No events match criteria")
		os.Exit(0)
	}

	// Dry run mode
	if *dryRun {
		fmt.Printf("DRY RUN MODE - Would send %d messages:\n\n", len(events))
		for i, evt := range events {
			msg, _ := telegram.FormatEventWithCalendar(evt)
			fmt.Printf("--- Message %d/%d ---\n", i+1, len(events))
			fmt.Println(msg)
			fmt.Printf("\n(Length: %d characters)\n", len(msg))
			fmt.Printf("Calendar button: 📅 Add to Calendar (callback: calendar:%s)\n\n", evt.ID)
		}
		os.Exit(0)
	}

	// Initialize Telegram client
	if *botToken == "" {
		fmt.Fprintf(os.Stderr, "Error: bot token is required (use --bot-token or TELEGRAM_BOT_TOKEN env var)\n")
		os.Exit(1)
	}

	if *chatID == "" {
		fmt.Fprintf(os.Stderr, "Error: chat ID is required (use --chat-id or TELEGRAM_CHAT_ID env var)\n")
		os.Exit(1)
	}

	client, err := telegram.NewClient(*botToken, *chatID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing Telegram client: %v\n", err)
		os.Exit(1)
	}

	// Send messages with calendar buttons
	for i, evt := range events {
		var msg string
		var keyboard *telegram.InlineKeyboardMarkup

		// Use reminder format if in reminder mode
		if *checkReminders {
			msg, keyboard = telegram.FormatReminder(evt, *reminderDays)
		} else {
			msg, keyboard = telegram.FormatEventWithCalendar(evt)
		}

		if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending message for event %s: %v\n", evt.ID, err)
			os.Exit(1)
		}

		// Rate limiting: wait between messages
		if i < len(events)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	messageType := "message"
	if *checkReminders {
		messageType = "reminder"
	}
	fmt.Printf("Successfully sent %d %s(s)\n", len(events), messageType)
}
