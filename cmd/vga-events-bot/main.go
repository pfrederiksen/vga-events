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
		if state == "ALL" || strings.EqualFold(evt.State, state) {
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

	case "/check":
		return handleCheck(chatID), nil

	default:
		return fmt.Sprintf("Unknown command: %s\n\nUse /help to see available commands.", command), nil
	}
}

func getHelpMessage() string {
	return `🤖 <b>VGA Events Bot</b>

I help you track VGA Golf events in your favorite states!

<b>Commands:</b>

/subscribe - Choose states with buttons (or /subscribe NV)
/manage - Manage your subscriptions with buttons
/settings - Configure notification preferences
/list - Show your current subscriptions
/check - Trigger an immediate check (experimental)
/help - Show this help message

<b>State Codes:</b>
Use 2-letter state codes like NV, CA, TX, etc.
Use ALL to subscribe to all states.

<b>Notifications:</b>
You'll receive messages whenever new events are posted in your subscribed states.
• <b>Immediate mode</b> - Get notified right away (default)
• <b>Daily digest</b> - Receive a daily summary at 9 AM UTC
• <b>Weekly digest</b> - Receive a weekly summary on Mondays

Change your preferences with /settings

Checks run every hour.`
}

func handleSubscribe(prefs preferences.Preferences, chatID, state string, modified *bool, botToken string, dryRun bool) (string, []*event.Event) {
	state = strings.ToUpper(strings.TrimSpace(state))

	if !preferences.IsValidState(state) {
		return fmt.Sprintf("❌ Invalid state code: %s\n\nPlease use a valid 2-letter state code (e.g., NV, CA, TX) or ALL.", state), nil
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
			if state == "ALL" || strings.EqualFold(evt.State, state) {
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
				{Text: "🇺🇸 All States", CallbackData: "subscribe:ALL"},
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
