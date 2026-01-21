package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/pfrederiksen/vga-events/internal/calendar"
	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
	"github.com/pfrederiksen/vga-events/internal/scraper"
	"github.com/pfrederiksen/vga-events/internal/telegram"
)

const (
	// AllStatesCode is the special state code to match all states
	AllStatesCode = "ALL"
)

var (
	botToken     = flag.String("bot-token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token (or env: TELEGRAM_BOT_TOKEN)")
	gistID       = flag.String("gist-id", os.Getenv("TELEGRAM_GIST_ID"), "GitHub Gist ID (or env: TELEGRAM_GIST_ID)")
	githubToken  = flag.String("github-token", os.Getenv("TELEGRAM_GITHUB_TOKEN"), "GitHub token (or env: TELEGRAM_GITHUB_TOKEN)")
	dryRun       = flag.Bool("dry-run", false, "Show what would be done without making changes")
	loop         = flag.Bool("loop", false, "Run continuously with long polling (for real-time responses)")
	loopDuration = flag.Duration("loop-duration", 5*time.Hour+50*time.Minute, "Maximum duration for loop mode (default 5h50m)")
)

type Update struct {
	UpdateID      int                     `json:"update_id"`
	Message       *Message                `json:"message,omitempty"`
	CallbackQuery *telegram.CallbackQuery `json:"callback_query,omitempty"`
}

type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Date      int64  `json:"date"`
	Text      string `json:"text"`
}

type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

func main() {
	flag.Parse()

	if *botToken == "" {
		fmt.Fprintf(os.Stderr, "Error: bot token is required (use --bot-token or TELEGRAM_BOT_TOKEN env var)\n")
		os.Exit(1)
	}

	if *gistID == "" {
		fmt.Fprintf(os.Stderr, "Error: gist ID is required (use --gist-id or TELEGRAM_GIST_ID env var)\n")
		os.Exit(1)
	}

	if *githubToken == "" {
		fmt.Fprintf(os.Stderr, "Error: GitHub token is required (use --github-token or TELEGRAM_GITHUB_TOKEN env var)\n")
		os.Exit(1)
	}

	// Initialize storage
	storage, err := preferences.NewGistStorage(*gistID, *githubToken)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}

	// Load preferences
	prefs, err := storage.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading preferences: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Loaded preferences for %d users\n", len(prefs))

	if *loop {
		runLoop(storage, prefs, *botToken, *dryRun, *loopDuration)
	} else {
		runOnce(storage, prefs, *botToken, *dryRun)
	}
}

// sendResponse sends the response message and optional initial events to a chat
func sendResponse(botToken, chatID, response string, initialEvents []*event.Event, dryRun bool) {
	if dryRun {
		fmt.Printf("[DRY RUN] Would send to %s:\n%s\n\n", chatID, response)
		if len(initialEvents) > 0 {
			fmt.Printf("[DRY RUN] Would also send %d initial events\n", len(initialEvents))
		}
		return
	}

	// Send response
	tempClient, err := telegram.NewClient(botToken, chatID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating client for chat %s: %v\n", chatID, err)
		return
	}

	if err := tempClient.SendMessage(response); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending response to %s: %v\n", chatID, err)
	} else {
		fmt.Printf("Sent response to %s\n", chatID)
	}

	// Send initial events if any
	if len(initialEvents) > 0 {
		fmt.Printf("Sending %d initial events to %s...\n", len(initialEvents), chatID)
		for i, evt := range initialEvents {
			msg := telegram.FormatEvent(evt)
			if err := tempClient.SendMessage(msg); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending initial event to %s: %v\n", chatID, err)
			}
			// Rate limiting
			if i < len(initialEvents)-1 {
				time.Sleep(1 * time.Second)
			}
		}
		fmt.Printf("Sent initial events to %s\n", chatID)
	}
}

func runLoop(storage *preferences.GistStorage, prefs preferences.Preferences, botToken string, dryRun bool, duration time.Duration) {
	fmt.Printf("Starting long polling loop (will run for %v)...\n", duration)
	startTime := time.Now()
	offset := 0

	for {
		// Check if we've exceeded our time limit
		if time.Since(startTime) >= duration {
			fmt.Printf("Reached time limit (%v), exiting gracefully...\n", duration)
			break
		}

		// Get updates with long polling (30 second timeout)
		updates, err := getUpdatesWithTimeout(botToken, offset, 30)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting updates: %v\n", err)
			time.Sleep(5 * time.Second) // Brief pause before retrying
			continue
		}

		if len(updates) == 0 {
			// No new messages, continue polling
			continue
		}

		fmt.Printf("Processing %d message(s)...\n", len(updates))

		prefsModified := false

		// Process each update
		for _, update := range updates {
			if update.CallbackQuery != nil {
				// Handle callback query (button press)
				handleCallbackQuery(prefs, update.CallbackQuery, &prefsModified, botToken, dryRun)
			} else if update.Message != nil {
				// Handle text message
				chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
				text := strings.TrimSpace(update.Message.Text)

				fmt.Printf("Message from %s (chat %s): %s\n", update.Message.From.FirstName, chatID, text)

				// Parse command
				response, initialEvents := processCommand(prefs, chatID, text, &prefsModified, botToken, dryRun)

				// Send response and initial events
				sendResponse(botToken, chatID, response, initialEvents, dryRun)
			}

			// Update offset to mark this update as processed
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}
		}

		// Save preferences if modified
		if prefsModified {
			if dryRun {
				fmt.Println("[DRY RUN] Would save updated preferences to Gist")
			} else {
				if err := storage.Save(prefs); err != nil {
					fmt.Fprintf(os.Stderr, "Error saving preferences: %v\n", err)
				} else {
					fmt.Println("Preferences saved successfully")
				}
			}
		}
	}

	fmt.Println("Long polling loop completed")
}

func runOnce(storage *preferences.GistStorage, prefs preferences.Preferences, botToken string, dryRun bool) {
	// Get updates from Telegram
	updates, err := getUpdates(botToken, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting updates: %v\n", err)
		os.Exit(1)
	}

	if len(updates) == 0 {
		fmt.Println("No new messages to process")
		os.Exit(0)
	}

	fmt.Printf("Processing %d message(s)...\n", len(updates))

	prefsModified := false
	maxUpdateID := 0

	// Process each update
	for _, update := range updates {
		// Track the highest update ID for acknowledgment
		if update.UpdateID > maxUpdateID {
			maxUpdateID = update.UpdateID
		}

		if update.CallbackQuery != nil {
			// Handle callback query (button press)
			handleCallbackQuery(prefs, update.CallbackQuery, &prefsModified, botToken, dryRun)
		} else if update.Message != nil {
			// Handle text message
			chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
			text := strings.TrimSpace(update.Message.Text)

			fmt.Printf("Message from %s (chat %s): %s\n", update.Message.From.FirstName, chatID, text)

			// Parse command
			response, initialEvents := processCommand(prefs, chatID, text, &prefsModified, botToken, dryRun)

			// Send response and initial events
			sendResponse(botToken, chatID, response, initialEvents, dryRun)
		}
	}

	// Save preferences if modified
	if prefsModified {
		if dryRun {
			fmt.Println("[DRY RUN] Would save updated preferences to Gist")
			prefsJSON, _ := prefs.ToJSON()
			fmt.Printf("Updated preferences:\n%s\n", string(prefsJSON))
		} else {
			if err := storage.Save(prefs); err != nil {
				fmt.Fprintf(os.Stderr, "Error saving preferences: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Preferences saved successfully")
		}
	} else {
		fmt.Println("No preference changes to save")
	}

	// Acknowledge processed messages by calling getUpdates with next offset
	// This prevents reprocessing the same messages on the next run
	if maxUpdateID > 0 && !dryRun {
		_, err := getUpdates(botToken, maxUpdateID+1)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to acknowledge processed messages: %v\n", err)
		} else {
			fmt.Printf("Acknowledged %d processed message(s)\n", len(updates))
		}
	}
}

func handlePreviewCallback(prefs preferences.Preferences, callback *telegram.CallbackQuery, modified *bool, botToken string, dryRun bool) string {
	// Handle event preview request after subscription
	// Format: preview:STATE:COUNT (e.g., "preview:NV:5" or "preview:CA:all")
	parts := strings.Split(callback.Data, ":")
	if len(parts) != 3 {
		return "❌ Invalid preview request"
	}

	state := parts[1]
	countStr := parts[2]

	// Fetch current events for this state
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return "❌ Error fetching events"
	}

	// Filter events by state and sort by date (soonest first)
	var stateEvents []*event.Event
	for _, evt := range allEvents {
		if state == AllStatesCode || strings.EqualFold(evt.State, state) {
			stateEvents = append(stateEvents, evt)
		}
	}

	// Sort by date (soonest first)
	event.SortByDate(stateEvents)

	// Determine how many to send
	var eventsToSend []*event.Event
	responseText := ""

	if countStr == "0" {
		// User chose not to see events
		responseText = "✅ Got it! You'll only be notified about new events going forward."
	} else if countStr == "all" {
		eventsToSend = stateEvents
	} else {
		count := 0
		if _, err := fmt.Sscanf(countStr, "%d", &count); err == nil {
			if count > 0 && count < len(stateEvents) {
				eventsToSend = stateEvents[:count]
			} else {
				eventsToSend = stateEvents
			}
		} else {
			eventsToSend = stateEvents // Default to all if parsing fails
		}
	}

	// Mark ALL state events as seen (not just the ones we're sending)
	callbackChatID := fmt.Sprintf("%d", callback.From.ID)
	user := prefs.GetUser(callbackChatID)
	for _, evt := range stateEvents {
		user.MarkEventSeen(evt.ID)
	}
	*modified = true

	// Send the requested events
	if len(eventsToSend) > 0 && !dryRun {
		client, err := telegram.NewClient(botToken, callbackChatID)
		if err != nil {
			return fmt.Sprintf("❌ Error sending events: %v", err)
		}

		for i, evt := range eventsToSend {
			msg, keyboard := telegram.FormatEventWithCalendar(evt)
			if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending event %s: %v\n", evt.ID, err)
			}

			// Rate limiting
			if i < len(eventsToSend)-1 {
				time.Sleep(1 * time.Second)
			}
		}

		return fmt.Sprintf("✅ Sent %d event(s)! All events marked as seen.", len(eventsToSend))
	} else if len(eventsToSend) > 0 {
		return fmt.Sprintf("[DRY RUN] Would send %d event(s)", len(eventsToSend))
	}

	return responseText
}

func handleUpcomingEventsCallback(prefs preferences.Preferences, chatID string, botToken string, dryRun bool) string {
	// Get user's subscribed states
	states := prefs.GetStates(chatID)
	if len(states) == 0 {
		return `📅 <b>No Subscriptions</b>

You're not subscribed to any states yet.

Use /subscribe to start receiving event notifications!`
	}

	// Fetch current events
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return "❌ Error fetching events. Please try again later."
	}

	// Filter events by subscribed states
	var filteredEvents []*event.Event
	for _, evt := range allEvents {
		for _, state := range states {
			if state == AllStatesCode || strings.EqualFold(evt.State, state) {
				filteredEvents = append(filteredEvents, evt)
				break
			}
		}
	}

	if len(filteredEvents) == 0 {
		return fmt.Sprintf(`📅 <b>No Upcoming Events</b>

No events found for your subscribed states: %s

Check back later or subscribe to more states with /subscribe`, strings.Join(states, ", "))
	}

	// Sort by date (soonest first)
	event.SortByDate(filteredEvents)

	// Limit to 10 events
	eventsToSend := filteredEvents
	if len(eventsToSend) > 10 {
		eventsToSend = eventsToSend[:10]
	}

	// Send events with calendar buttons
	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err != nil {
			return "❌ Error sending events"
		}

		// Send header message
		headerMsg := fmt.Sprintf(`📅 <b>Upcoming Events</b>

Showing %d of %d events for %s

Sorted by soonest first:`, len(eventsToSend), len(filteredEvents), strings.Join(states, ", "))

		if err := client.SendMessage(headerMsg); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending header: %v\n", err)
		}

		// Send each event with calendar button
		for i, evt := range eventsToSend {
			msg, keyboard := telegram.FormatEventWithCalendar(evt)
			if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending event %s: %v\n", evt.ID, err)
			}

			// Rate limiting
			if i < len(eventsToSend)-1 {
				time.Sleep(1 * time.Second)
			}
		}

		return "" // Already sent
	}

	return fmt.Sprintf("[DRY RUN] Would send %d upcoming events", len(eventsToSend))
}

//nolint:gocyclo // Callback handler complexity is inherent to handling multiple callback types
func handleCallbackQuery(prefs preferences.Preferences, callback *telegram.CallbackQuery, modified *bool, botToken string, dryRun bool) {
	chatID := fmt.Sprintf("%d", callback.From.ID)
	messageID := 0
	if callback.Message != nil {
		messageID = callback.Message.MessageID
	}

	fmt.Printf("Callback from %s (chat %s): %s\n", callback.From.FirstName, chatID, callback.Data)

	// Parse callback data (format: "action:param")
	parts := strings.Split(callback.Data, ":")
	if len(parts) == 0 {
		return
	}

	action := parts[0]
	var param string
	if len(parts) > 1 {
		param = parts[1]
	}

	var responseText string
	var keyboard *telegram.InlineKeyboardMarkup

	switch action {
	case "subscribe":
		if param != "" {
			responseText, _ = handleSubscribe(prefs, chatID, param, modified, botToken, dryRun)
		} else {
			// Show state selection keyboard
			responseText, keyboard = showStateSelectionKeyboard()
		}

	case "unsubscribe":
		responseText = handleUnsubscribe(prefs, chatID, param, modified)

	case "manage":
		responseText, keyboard = showManageSubscriptionsKeyboard(prefs, chatID)

	case "settings":
		responseText, keyboard = showSettingsKeyboard(prefs, chatID)

	case "digest":
		if user := prefs.GetUser(chatID); user != nil {
			if user.SetDigestFrequency(param) {
				*modified = true
				responseText = fmt.Sprintf("✅ Digest frequency updated to <b>%s</b>", param)
			} else {
				responseText = "❌ Invalid digest frequency"
			}
		}

	case "preview":
		responseText = handlePreviewCallback(prefs, callback, modified, botToken, dryRun)

	case "menu":
		// Handle menu actions
		switch param {
		case "upcoming":
			responseText = handleUpcomingEventsCallback(prefs, chatID, botToken, dryRun)
		case "search":
			responseText = `🔍 <b>Search Events</b>

To search for events, use the command:
/search &lt;keyword&gt;

Example:
/search "Pine Valley"
/search Championship
/search Las Vegas`
		case "help":
			responseText = getHelpMessage()
		default:
			responseText = "Unknown menu action"
		}

	case "status":
		// Handle event status update
		// Format: status:EVENT_ID:STATUS (e.g., "status:abc123:interested")
		parts := strings.Split(callback.Data, ":")
		if len(parts) != 3 {
			responseText = "❌ Invalid status request"
			break
		}
		eventID := parts[1]
		status := parts[2]

		user := prefs.GetUser(chatID)
		if user.SetEventStatus(eventID, status) {
			*modified = true

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

			responseText = fmt.Sprintf("%s Event marked as <b>%s</b>", statusEmoji, statusText)
		} else {
			responseText = "❌ Invalid status"
		}

	case "reminder":
		// Handle reminder configuration
		// Format: reminder:ACTION:DAYS (e.g., "reminder:add:7" or "reminder:done:0")
		parts := strings.Split(callback.Data, ":")
		if len(parts) != 3 {
			responseText = "❌ Invalid reminder request"
			break
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
			responseText, keyboard = showRemindersKeyboard(prefs, chatID)

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
			responseText, keyboard = showRemindersKeyboard(prefs, chatID)

		case "done":
			// Save and close
			responseText = "✅ Reminder settings saved!"
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

		default:
			responseText = "❌ Unknown reminder action"
		}

	case "calendar":
		// Calendar download - fetch event and send .ics file
		eventID := param

		// Fetch fresh events from VGA website to find the event
		sc := scraper.New()
		allEvents, err := sc.FetchEvents()
		if err != nil {
			responseText = "❌ Error fetching event data"
			fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
			break
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
			responseText = "❌ Event not found. This event may have been removed from the VGA website."
			fmt.Fprintf(os.Stderr, "Event %s not found in current events\n", eventID)
			break
		}

		// Generate .ics file
		icsContent := calendar.GenerateICS(evt)
		filename := fmt.Sprintf("vga-event-%s.ics", evt.State)

		// Send the .ics file
		if !dryRun {
			client, err := telegram.NewClient(botToken, chatID)
			if err != nil {
				responseText = "❌ Error sending calendar file"
				fmt.Fprintf(os.Stderr, "Error creating Telegram client: %v\n", err)
				break
			}

			caption := fmt.Sprintf("📅 <b>%s - %s</b>\n\nTap to add to your calendar!", evt.State, evt.Title)
			if err := client.SendDocument(filename, []byte(icsContent), caption); err != nil {
				responseText = "❌ Error sending calendar file"
				fmt.Fprintf(os.Stderr, "Error sending document: %v\n", err)
				break
			}

			responseText = "✅ Calendar file sent! Tap it to add the event to your calendar."
		} else {
			responseText = fmt.Sprintf("[DRY RUN] Would send .ics file for event: %s - %s", evt.State, evt.Title)
		}

	default:
		responseText = "Unknown action"
	}

	// Answer the callback query
	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err == nil {
			if err := client.AnswerCallbackQuery(callback.ID, "", false); err != nil {
				fmt.Fprintf(os.Stderr, "Error answering callback: %v\n", err)
			}

			// Edit the message with new text and keyboard
			if messageID > 0 {
				if err := client.EditMessageText(chatID, messageID, responseText, keyboard); err != nil {
					fmt.Fprintf(os.Stderr, "Error editing message: %v\n", err)
				}
			} else {
				// If no message ID, send new message
				var sendErr error
				if keyboard != nil {
					sendErr = client.SendMessageWithKeyboard(responseText, keyboard)
				} else {
					sendErr = client.SendMessage(responseText)
				}
				if sendErr != nil {
					fmt.Fprintf(os.Stderr, "Error sending message: %v\n", err)
				}
			}
		}
	} else {
		fmt.Printf("[DRY RUN] Would answer callback and update message:\n%s\n\n", responseText)
	}
}

func processCommand(prefs preferences.Preferences, chatID, text string, modified *bool, botToken string, dryRun bool) (string, []*event.Event) {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "Please send a command. Use /help to see available commands.", nil
	}

	command := strings.ToLower(parts[0])

	switch command {
	case "/start", "/help":
		return getHelpMessage(), nil

	case "/subscribe":
		if len(parts) < 2 {
			// Show state selection keyboard
			return handleSubscribeWithKeyboard(chatID, botToken, dryRun)
		}
		return handleSubscribe(prefs, chatID, parts[1], modified, botToken, dryRun)

	case "/unsubscribe":
		if len(parts) < 2 {
			return "❌ Please specify a state code.\n\nUsage: /unsubscribe NV", nil
		}
		return handleUnsubscribe(prefs, chatID, parts[1], modified), nil

	case "/list":
		return handleList(prefs, chatID), nil

	case "/manage":
		return handleManageWithKeyboard(prefs, chatID, botToken, dryRun)

	case "/settings":
		return handleSettingsWithKeyboard(prefs, chatID, botToken, dryRun)

	case "/menu":
		return handleMenuWithKeyboard(chatID, botToken, dryRun)

	case "/search":
		if len(parts) < 2 {
			return `🔍 <b>Event Search</b>

Please provide a search keyword.

<b>Usage:</b> /search &lt;keyword&gt;

<b>Examples:</b>
/search "Pine Valley"
/search Championship
/search Las Vegas
/search NV`, nil
		}
		keyword := strings.Join(parts[1:], " ")
		keyword = strings.Trim(keyword, `"'`) // Remove quotes if present
		return handleSearch(chatID, keyword, botToken, dryRun)

	case "/export-calendar":
		// Optional parameter: state code
		var stateFilter string
		if len(parts) >= 2 {
			stateFilter = strings.ToUpper(strings.TrimSpace(parts[1]))
		}
		return handleExportCalendar(prefs, chatID, stateFilter, botToken, dryRun)

	case "/my-events":
		return handleMyEvents(prefs, chatID, botToken, dryRun)

	case "/reminders":
		return handleRemindersWithKeyboard(prefs, chatID, botToken, dryRun)

	case "/check":
		return handleCheck(chatID), nil

	default:
		return fmt.Sprintf("Unknown command: %s\n\nUse /help to see available commands.", command), nil
	}
}

func getHelpMessage() string {
	return fmt.Sprintf(`🤖 <b>VGA Events Bot</b>

I help you track VGA Golf events in your favorite states!

<b>Commands:</b>

/menu - Quick actions menu 🎯
/search - Search for events by keyword 🔍
/my-events - View your tracked events ⭐
/reminders - Configure event reminders 🔔
/export-calendar - Download all events as .ics file 📅
/subscribe - Choose states with buttons (or /subscribe NV)
/manage - Manage your subscriptions with buttons
/settings - Configure notification preferences
/list - Show your current subscriptions
/check - Trigger an immediate check (experimental)
/help - Show this help message

<b>Event Tracking:</b>
Mark events with status buttons:
• ⭐ Interested - Events you want to attend
• ✅ Registered - Events you've signed up for
• 🤔 Maybe - Events you're considering
• ❌ Skip - Events you're not interested in

<b>Reminders:</b>
Get reminded before events you've marked as ⭐ Interested or ✅ Registered.
Configure reminder timing with /reminders (1 day, 3 days, 1 week, or 2 weeks before).

<b>State Codes:</b>
Use 2-letter state codes like NV, CA, TX, etc.
Use %s to subscribe to all states.

<b>Notifications:</b>
You'll receive messages whenever new events are posted in your subscribed states.
• <b>Immediate mode</b> - Get notified right away (default)
• <b>Daily digest</b> - Receive a daily summary at 9 AM UTC
• <b>Weekly digest</b> - Receive a weekly summary on Mondays

Change your preferences with /settings

Checks run every hour.`, AllStatesCode)
}

func handleSubscribe(prefs preferences.Preferences, chatID, state string, modified *bool, botToken string, dryRun bool) (string, []*event.Event) {
	state = strings.ToUpper(strings.TrimSpace(state))

	if !preferences.IsValidState(state) {
		return fmt.Sprintf("❌ Invalid state code: %s\n\nPlease use a valid 2-letter state code (e.g., NV, CA, TX) or %s.", state, AllStatesCode), nil
	}

	if prefs.HasState(chatID, state) {
		stateName := preferences.GetStateName(state)
		return fmt.Sprintf("ℹ️ You're already subscribed to %s (%s).\n\nUse /list to see all your subscriptions.", stateName, state), nil
	}

	prefs.AddState(chatID, state)
	*modified = true

	stateName := preferences.GetStateName(state)
	states := prefs.GetStates(chatID)

	response := fmt.Sprintf("✅ <b>Subscribed to %s (%s)!</b>\n\n", stateName, state)
	response += "You'll receive notifications when new events are posted.\n\n"
	response += fmt.Sprintf("<b>Your subscriptions:</b> %s\n\n", strings.Join(states, ", "))

	// Check if there are existing events for this state
	if !dryRun {
		fmt.Printf("Checking for existing events in state %s...\n", state)
		sc := scraper.New()
		allEvents, err := sc.FetchEvents()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to fetch events: %v\n", err)
			response += "⚠️ Unable to check for existing events, but you're subscribed!"
			return response, nil
		}

		// Filter and count events by state
		var stateEvents []*event.Event
		for _, evt := range allEvents {
			if state == AllStatesCode || strings.EqualFold(evt.State, state) {
				stateEvents = append(stateEvents, evt)
			}
		}

		totalEvents := len(stateEvents)
		if totalEvents == 0 {
			response += "ℹ️ No current events found for this state. You'll be notified when new events are posted!"
			return response, nil
		}

		// Ask user how many events they want to see with inline keyboard
		response += fmt.Sprintf("📅 There are currently <b>%d event(s)</b> scheduled in %s.\n\n", totalEvents, stateName)
		response += "Would you like to see existing events, or just be notified about new ones?"

		// Build keyboard with preview options
		keyboard := buildEventPreviewKeyboard(state, totalEvents)

		// Send keyboard message
		client, err := telegram.NewClient(botToken, chatID)
		if err == nil {
			if err := client.SendMessageWithKeyboard(response, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending preview keyboard: %v\n", err)
				return response, nil // Fallback to plain message
			}
			return "", nil // Already sent via keyboard
		}

		// If we couldn't send keyboard, just return the message
		return response, nil
	}

	return response, nil
}

func handleUnsubscribe(prefs preferences.Preferences, chatID, state string, modified *bool) string {
	state = strings.ToUpper(strings.TrimSpace(state))

	if !prefs.RemoveState(chatID, state) {
		return fmt.Sprintf("ℹ️ You weren't subscribed to %s.\n\nUse /list to see your current subscriptions.", state)
	}

	*modified = true
	stateName := preferences.GetStateName(state)
	states := prefs.GetStates(chatID)

	response := fmt.Sprintf("✅ <b>Unsubscribed from %s (%s)</b>\n\n", stateName, state)

	if len(states) > 0 {
		response += fmt.Sprintf("<b>Your remaining subscriptions:</b> %s", strings.Join(states, ", "))
	} else {
		response += "You have no active subscriptions.\n\nUse /subscribe &lt;STATE&gt; to subscribe to a state."
	}

	return response
}

func handleList(prefs preferences.Preferences, chatID string) string {
	states := prefs.GetStates(chatID)

	if len(states) == 0 {
		return `📋 <b>Your Subscriptions</b>

You have no active subscriptions.

Use /subscribe &lt;STATE&gt; to start receiving notifications.
Example: /subscribe NV`
	}

	response := "📋 <b>Your Subscriptions</b>\n\nYou're subscribed to:\n"
	for _, state := range states {
		stateName := preferences.GetStateName(state)
		response += fmt.Sprintf("• %s (%s)\n", stateName, state)
	}

	response += "\nUse /subscribe &lt;STATE&gt; to add more\n"
	response += "Use /unsubscribe &lt;STATE&gt; to remove"

	return response
}

func handleCheck(chatID string) string {
	return `🔍 <b>Manual Check</b>

The bot checks for new events every hour automatically.

If you're subscribed to any states, you'll receive notifications when new events are posted.

Use /list to see your current subscriptions.`
}

func handleSearch(chatID, keyword string, botToken string, dryRun bool) (string, []*event.Event) {
	// Fetch all events
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return "❌ Error fetching events. Please try again later.", nil
	}

	// Filter events by keyword (case-insensitive search in title, city, state)
	keywordLower := strings.ToLower(keyword)
	var matchingEvents []*event.Event
	for _, evt := range allEvents {
		if strings.Contains(strings.ToLower(evt.Title), keywordLower) ||
			strings.Contains(strings.ToLower(evt.City), keywordLower) ||
			strings.Contains(strings.ToLower(evt.State), keywordLower) {
			matchingEvents = append(matchingEvents, evt)
		}
	}

	if len(matchingEvents) == 0 {
		return fmt.Sprintf(`🔍 <b>No Results</b>

No events found matching "%s"

Try a different search term or use /menu to see all upcoming events.`, keyword), nil
	}

	// Sort by date (soonest first)
	event.SortByDate(matchingEvents)

	// Limit to 10 events
	eventsToSend := matchingEvents
	if len(eventsToSend) > 10 {
		eventsToSend = eventsToSend[:10]
	}

	// Send results
	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err != nil {
			return "❌ Error sending results", nil
		}

		// Send header message
		headerMsg := fmt.Sprintf(`🔍 <b>Search Results</b>

Found %d event(s) matching "%s"

Showing first %d results:`, len(matchingEvents), keyword, len(eventsToSend))

		if err := client.SendMessage(headerMsg); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending header: %v\n", err)
		}

		// Send each event with calendar button and subscribe option
		for i, evt := range eventsToSend {
			msg, keyboard := telegram.FormatEventWithCalendar(evt)
			if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending event %s: %v\n", evt.ID, err)
			}

			// Rate limiting
			if i < len(eventsToSend)-1 {
				time.Sleep(1 * time.Second)
			}
		}

		return "", nil // Already sent
	}

	return fmt.Sprintf("[DRY RUN] Would send %d search results for '%s'", len(eventsToSend), keyword), nil
}

func handleExportCalendar(prefs preferences.Preferences, chatID, stateFilter string, botToken string, dryRun bool) (string, []*event.Event) {
	// Get user's subscribed states
	states := prefs.GetStates(chatID)

	// If no state filter provided, use all subscribed states
	var filterStates []string
	if stateFilter != "" {
		// Validate state code
		if !preferences.IsValidState(stateFilter) {
			return fmt.Sprintf(`❌ Invalid state code: %s

<b>Usage:</b>
/export-calendar - Export all your subscribed events
/export-calendar NV - Export events from Nevada
/export-calendar %s - Export events from all states`, stateFilter, AllStatesCode), nil
		}
		filterStates = []string{stateFilter}
	} else {
		if len(states) == 0 {
			return `📅 <b>No Subscriptions</b>

You're not subscribed to any states yet.

Use /subscribe to start receiving event notifications, then use /export-calendar to download events.

Or use /export-calendar &lt;STATE&gt; to export events from a specific state.`, nil
		}
		filterStates = states
	}

	// Fetch all events
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return "❌ Error fetching events. Please try again later.", nil
	}

	// Filter events by states
	var filteredEvents []*event.Event
	for _, evt := range allEvents {
		for _, state := range filterStates {
			if state == AllStatesCode || strings.EqualFold(evt.State, state) {
				filteredEvents = append(filteredEvents, evt)
				break
			}
		}
	}

	if len(filteredEvents) == 0 {
		return fmt.Sprintf(`📅 <b>No Events Found</b>

No events found for %s

Try /export-calendar with a different state, or check back later.`, strings.Join(filterStates, ", ")), nil
	}

	// Sort by date (soonest first)
	event.SortByDate(filteredEvents)

	// Generate bulk ICS file
	calendarName := fmt.Sprintf("VGA Golf Events - %s", strings.Join(filterStates, ", "))
	icsContent := calendar.GenerateBulkICS(filteredEvents, calendarName)

	if len(icsContent) == 0 {
		return "❌ Error generating calendar file", nil
	}

	// Send the .ics file
	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err != nil {
			return "❌ Error sending calendar file", nil
		}

		filename := "vga-events.ics"
		if len(filterStates) == 1 && filterStates[0] != AllStatesCode {
			filename = fmt.Sprintf("vga-events-%s.ics", filterStates[0])
		}

		caption := fmt.Sprintf(`📅 <b>VGA Events Calendar</b>

✅ Exported %d event(s) from %s

Tap the file to import all events into your calendar app!`, len(filteredEvents), strings.Join(filterStates, ", "))

		if err := client.SendDocument(filename, []byte(icsContent), caption); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending document: %v\n", err)
			return "❌ Error sending calendar file", nil
		}

		fmt.Printf("Sent bulk calendar file to %s (%d events)\n", chatID, len(filteredEvents))
		return "", nil // Already sent
	}

	return fmt.Sprintf("[DRY RUN] Would send bulk calendar file with %d events for %s", len(filteredEvents), strings.Join(filterStates, ", ")), nil
}

func handleMyEvents(prefs preferences.Preferences, chatID string, botToken string, dryRun bool) (string, []*event.Event) {
	user := prefs.GetUser(chatID)

	if len(user.EventStatuses) == 0 {
		return `⭐ <b>My Events</b>

You haven't marked any events yet.

When you see an event notification, use the status buttons to mark it as:
• ⭐ Interested
• ✅ Registered
• 🤔 Maybe
• ❌ Skip

Then use /my-events to see all your tracked events!`, nil
	}

	// Fetch current events from VGA website
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return "❌ Error fetching events. Please try again later.", nil
	}

	// Create a map of event IDs to events for quick lookup
	eventsByID := make(map[string]*event.Event)
	for _, evt := range allEvents {
		eventsByID[evt.ID] = evt
	}

	// Group events by status (excluding "skip")
	statusGroups := map[string][]*event.Event{
		preferences.EventStatusRegistered: {},
		preferences.EventStatusInterested: {},
		preferences.EventStatusMaybe:      {},
	}

	for eventID, status := range user.EventStatuses {
		if status == preferences.EventStatusSkip {
			continue // Don't show skipped events
		}

		evt, exists := eventsByID[eventID]
		if !exists {
			// Event no longer exists on VGA website
			continue
		}

		if group, ok := statusGroups[status]; ok {
			statusGroups[status] = append(group, evt)
		}
	}

	// Count total events
	totalEvents := 0
	for _, group := range statusGroups {
		totalEvents += len(group)
	}

	if totalEvents == 0 {
		return `⭐ <b>My Events</b>

None of your tracked events are currently scheduled.

They may have been removed from the VGA website or marked as "Skip".`, nil
	}

	// Sort each group by date
	for _, group := range statusGroups {
		event.SortByDate(group)
	}

	// Send events
	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err != nil {
			return "❌ Error sending events", nil
		}

		// Send header
		headerMsg := fmt.Sprintf(`⭐ <b>My Events</b>

You have %d tracked event(s):`, totalEvents)

		if err := client.SendMessage(headerMsg); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending header: %v\n", err)
		}

		// Send each group
		statusOrder := []string{
			preferences.EventStatusRegistered,
			preferences.EventStatusInterested,
			preferences.EventStatusMaybe,
		}

		statusNames := map[string]string{
			preferences.EventStatusRegistered: "✅ Registered",
			preferences.EventStatusInterested: "⭐ Interested",
			preferences.EventStatusMaybe:      "🤔 Maybe",
		}

		for _, status := range statusOrder {
			group := statusGroups[status]
			if len(group) == 0 {
				continue
			}

			// Send group header
			groupHeader := fmt.Sprintf("\n<b>%s (%d)</b>", statusNames[status], len(group))
			if err := client.SendMessage(groupHeader); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending group header: %v\n", err)
			}

			// Send each event with status buttons
			for i, evt := range group {
				msg, keyboard := telegram.FormatEventWithStatus(evt, status)
				if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
					fmt.Fprintf(os.Stderr, "Error sending event %s: %v\n", evt.ID, err)
				}

				// Rate limiting
				if i < len(group)-1 {
					time.Sleep(1 * time.Second)
				}
			}
		}

		return "", nil // Already sent
	}

	return fmt.Sprintf("[DRY RUN] Would send %d tracked events", totalEvents), nil
}

// buildEventPreviewKeyboard returns a keyboard asking how many events to preview
func buildEventPreviewKeyboard(state string, totalEvents int) *telegram.InlineKeyboardMarkup {
	buttons := [][]telegram.InlineKeyboardButton{
		{
			{Text: "Don't show events", CallbackData: fmt.Sprintf("preview:%s:0", state)},
		},
	}

	// Offer to show different amounts based on total
	if totalEvents >= 5 {
		buttons = append(buttons, []telegram.InlineKeyboardButton{
			{Text: "Show 5 soonest events", CallbackData: fmt.Sprintf("preview:%s:5", state)},
		})
	}
	if totalEvents >= 10 {
		buttons = append(buttons, []telegram.InlineKeyboardButton{
			{Text: "Show 10 soonest events", CallbackData: fmt.Sprintf("preview:%s:10", state)},
		})
	}
	if totalEvents > 10 {
		buttons = append(buttons, []telegram.InlineKeyboardButton{
			{Text: fmt.Sprintf("Show all %d events", totalEvents), CallbackData: fmt.Sprintf("preview:%s:all", state)},
		})
	} else {
		buttons = append(buttons, []telegram.InlineKeyboardButton{
			{Text: fmt.Sprintf("Show all %d events", totalEvents), CallbackData: fmt.Sprintf("preview:%s:all", state)},
		})
	}

	return &telegram.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

// showStateSelectionKeyboard returns a keyboard with popular states
func showStateSelectionKeyboard() (string, *telegram.InlineKeyboardMarkup) {
	keyboard := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "🌴 California (CA)", CallbackData: "subscribe:CA"},
				{Text: "🎰 Nevada (NV)", CallbackData: "subscribe:NV"},
			},
			{
				{Text: "🤠 Texas (TX)", CallbackData: "subscribe:TX"},
				{Text: "🌵 Arizona (AZ)", CallbackData: "subscribe:AZ"},
			},
			{
				{Text: "🏖️ Florida (FL)", CallbackData: "subscribe:FL"},
				{Text: "🗽 New York (NY)", CallbackData: "subscribe:NY"},
			},
			{
				{Text: "🇺🇸 All States", CallbackData: fmt.Sprintf("subscribe:%s", AllStatesCode)},
			},
		},
	}
	return "📍 <b>Select a state to subscribe:</b>\n\nOr type: /subscribe STATE", keyboard
}

// showManageSubscriptionsKeyboard shows current subscriptions with unsubscribe buttons
func showManageSubscriptionsKeyboard(prefs preferences.Preferences, chatID string) (string, *telegram.InlineKeyboardMarkup) {
	states := prefs.GetStates(chatID)

	if len(states) == 0 {
		return `📋 <b>No Subscriptions</b>

You have no active subscriptions.

Use /subscribe to get started!`, nil
	}

	keyboard := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{},
	}

	text := "📋 <b>Manage Subscriptions</b>\n\nTap to unsubscribe:\n"

	for _, state := range states {
		stateName := preferences.GetStateName(state)
		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []telegram.InlineKeyboardButton{
			{Text: fmt.Sprintf("✅ %s (%s)", stateName, state), CallbackData: fmt.Sprintf("unsubscribe:%s", state)},
		})
	}

	// Add "Subscribe to more" button
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []telegram.InlineKeyboardButton{
		{Text: "➕ Subscribe to more", CallbackData: "subscribe:"},
	})

	return text, keyboard
}

// showSettingsKeyboard shows user settings with digest options
func showSettingsKeyboard(prefs preferences.Preferences, chatID string) (string, *telegram.InlineKeyboardMarkup) {
	user := prefs.GetUser(chatID)

	keyboard := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "📨 Immediate", CallbackData: "digest:immediate"},
			},
			{
				{Text: "📅 Daily Digest", CallbackData: "digest:daily"},
			},
			{
				{Text: "📆 Weekly Digest", CallbackData: "digest:weekly"},
			},
		},
	}

	text := fmt.Sprintf(`⚙️ <b>Settings</b>

<b>Current digest frequency:</b> %s

<b>Notification Mode:</b>
• <b>Immediate</b> - Get notified as soon as new events are posted
• <b>Daily</b> - Receive a daily digest at 9 AM UTC
• <b>Weekly</b> - Receive a weekly digest on Mondays at 9 AM UTC

Select your preferred mode:`, user.DigestFrequency)

	return text, keyboard
}

// handleSubscribeWithKeyboard shows the subscription keyboard when /subscribe is called without args
func handleSubscribeWithKeyboard(chatID, botToken string, dryRun bool) (string, []*event.Event) {
	text, keyboard := showStateSelectionKeyboard()

	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err == nil {
			if err := client.SendMessageWithKeyboard(text, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending keyboard: %v\n", err)
			}
			return "", nil // Already sent via keyboard
		}
	}

	return text, nil
}

// handleManageWithKeyboard shows the manage subscriptions keyboard
func handleManageWithKeyboard(prefs preferences.Preferences, chatID, botToken string, dryRun bool) (string, []*event.Event) {
	text, keyboard := showManageSubscriptionsKeyboard(prefs, chatID)

	if keyboard != nil && !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err == nil {
			if err := client.SendMessageWithKeyboard(text, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending keyboard: %v\n", err)
			}
			return "", nil // Already sent via keyboard
		}
	}

	return text, nil
}

// handleSettingsWithKeyboard shows the settings keyboard
func handleSettingsWithKeyboard(prefs preferences.Preferences, chatID, botToken string, dryRun bool) (string, []*event.Event) {
	text, keyboard := showSettingsKeyboard(prefs, chatID)

	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err == nil {
			if err := client.SendMessageWithKeyboard(text, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending keyboard: %v\n", err)
			}
			return "", nil // Already sent via keyboard
		}
	}

	return text, nil
}

// handleMenuWithKeyboard shows the main menu keyboard
func handleMenuWithKeyboard(chatID, botToken string, dryRun bool) (string, []*event.Event) {
	text, keyboard := showMenuKeyboard()

	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err == nil {
			if err := client.SendMessageWithKeyboard(text, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending keyboard: %v\n", err)
			}
			return "", nil // Already sent via keyboard
		}
	}

	return text, nil
}

// handleRemindersWithKeyboard shows the reminders configuration keyboard
func handleRemindersWithKeyboard(prefs preferences.Preferences, chatID, botToken string, dryRun bool) (string, []*event.Event) {
	text, keyboard := showRemindersKeyboard(prefs, chatID)

	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err == nil {
			if err := client.SendMessageWithKeyboard(text, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending keyboard: %v\n", err)
			}
			return "", nil // Already sent via keyboard
		}
	}

	return text, nil
}

// showRemindersKeyboard returns the reminders configuration keyboard
func showRemindersKeyboard(prefs preferences.Preferences, chatID string) (string, *telegram.InlineKeyboardMarkup) {
	user := prefs.GetUser(chatID)

	// Build checkboxes for each reminder option
	keyboard := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{},
	}

	reminderOptions := []struct {
		days  int
		label string
	}{
		{1, "1 day before"},
		{3, "3 days before"},
		{7, "1 week before"},
		{14, "2 weeks before"},
	}

	for _, opt := range reminderOptions {
		checkbox := "☐"
		action := "add"
		if user.HasReminderDay(opt.days) {
			checkbox = "☑"
			action = "remove"
		}

		keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []telegram.InlineKeyboardButton{
			{
				Text:         fmt.Sprintf("%s %s", checkbox, opt.label),
				CallbackData: fmt.Sprintf("reminder:%s:%d", action, opt.days),
			},
		})
	}

	// Add "Done" button
	keyboard.InlineKeyboard = append(keyboard.InlineKeyboard, []telegram.InlineKeyboardButton{
		{Text: "✅ Done", CallbackData: "reminder:done:0"},
	})

	// Build status text
	var statusText string
	if len(user.ReminderDays) == 0 {
		statusText = "No reminders configured"
	} else {
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
		statusText = fmt.Sprintf("Active: %s before events", strings.Join(reminders, ", "))
	}

	text := fmt.Sprintf(`🔔 <b>Event Reminders</b>

%s

Select when you want to be reminded about events you've marked as ⭐ Interested or ✅ Registered.

Tap to toggle reminders:`, statusText)

	return text, keyboard
}

// showMenuKeyboard returns the main menu keyboard
func showMenuKeyboard() (string, *telegram.InlineKeyboardMarkup) {
	keyboard := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "📅 Upcoming Events", CallbackData: "menu:upcoming"},
			},
			{
				{Text: "🔍 Search Events", CallbackData: "menu:search"},
			},
			{
				{Text: "⭐ My Subscriptions", CallbackData: "manage"},
			},
			{
				{Text: "⚙️ Settings", CallbackData: "settings"},
			},
			{
				{Text: "❓ Help", CallbackData: "menu:help"},
			},
		},
	}

	text := `🎯 <b>Quick Actions Menu</b>

Select an action below:`

	return text, keyboard
}

func getUpdates(botToken string, offset int) ([]Update, error) {
	return getUpdatesWithTimeout(botToken, offset, 0)
}

func getUpdatesWithTimeout(botToken string, offset int, timeoutSeconds int) ([]Update, error) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getUpdates", botToken)
	params := []string{}

	if offset > 0 {
		params = append(params, fmt.Sprintf("offset=%d", offset))
	}

	if timeoutSeconds > 0 {
		params = append(params, fmt.Sprintf("timeout=%d", timeoutSeconds))
	}

	if len(params) > 0 {
		url += "?" + strings.Join(params, "&")
	}

	// Add extra time to HTTP client timeout to account for Telegram's long polling
	clientTimeout := time.Duration(timeoutSeconds+10) * time.Second
	if clientTimeout < 15*time.Second {
		clientTimeout = 15 * time.Second
	}

	client := &http.Client{Timeout: clientTimeout}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching updates: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Telegram API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		OK     bool     `json:"ok"`
		Result []Update `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if !result.OK {
		return nil, fmt.Errorf("Telegram API returned ok=false")
	}

	return result.Result, nil
}
