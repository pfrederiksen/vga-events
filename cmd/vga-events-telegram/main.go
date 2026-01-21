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
	botToken    = flag.String("bot-token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token (or env: TELEGRAM_BOT_TOKEN)")
	chatID      = flag.String("chat-id", os.Getenv("TELEGRAM_CHAT_ID"), "Telegram chat ID (or env: TELEGRAM_CHAT_ID)")
	eventsFile  = flag.String("events-file", "", "Path to events JSON file (or read from stdin)")
	dryRun      = flag.Bool("dry-run", false, "Print messages without sending")
	maxMessages = flag.Int("max-messages", 10, "Maximum number of messages to send")
	stateFilter = flag.String("state", "", "Only send messages for this state")
	hidePast    = flag.Bool("hide-past", true, "Filter out past events (default: true)")
	daysAhead   = flag.Int("days-ahead", 0, "Only show events within N days (0 = disabled)")
)

func main() {
	flag.Parse()

	// Read events from file or stdin
	var reader io.Reader
	if *eventsFile != "" {
		f, err := os.Open(*eventsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening events file: %v\n", err)
			os.Exit(1)
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

	// Parse JSON
	var result struct {
		NewEvents []*event.Event `json:"new_events"`
	}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	if len(result.NewEvents) == 0 {
		fmt.Println("No new events to send")
		os.Exit(0)
	}

	// Filter events by state if specified
	events := result.NewEvents
	if *stateFilter != "" {
		filtered := make([]*event.Event, 0)
		for _, evt := range events {
			if evt.State == *stateFilter {
				filtered = append(filtered, evt)
			}
		}
		events = filtered
	}

	// Apply time-based filtering
	if *hidePast || *daysAhead > 0 {
		filtered := make([]*event.Event, 0)
		for _, evt := range events {
			// Filter past events if enabled
			if *hidePast && evt.IsPastEvent() {
				continue
			}
			// Filter events beyond days_ahead window if enabled
			if *daysAhead > 0 && !evt.IsWithinDays(*daysAhead) {
				continue
			}
			filtered = append(filtered, evt)
		}
		events = filtered
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
		msg, keyboard := telegram.FormatEventWithCalendar(evt)

		if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending message for event %s: %v\n", evt.ID, err)
			os.Exit(1)
		}

		// Rate limiting: wait between messages
		if i < len(events)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Printf("Successfully sent %d message(s)\n", len(events))
}
