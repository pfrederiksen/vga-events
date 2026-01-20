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
	UpdateID int     `json:"update_id"`
	Message  Message `json:"message"`
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
			chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
			text := strings.TrimSpace(update.Message.Text)

			fmt.Printf("Message from %s (chat %s): %s\n", update.Message.From.FirstName, chatID, text)

			// Parse command
			response, initialEvents := processCommand(prefs, chatID, text, &prefsModified, botToken, dryRun)

			// Send response and initial events
			sendResponse(botToken, chatID, response, initialEvents, dryRun)

			// Update offset to mark this message as processed
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
		chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
		text := strings.TrimSpace(update.Message.Text)

		fmt.Printf("Message from %s (chat %s): %s\n", update.Message.From.FirstName, chatID, text)

		// Parse command
		response, initialEvents := processCommand(prefs, chatID, text, &prefsModified, botToken, dryRun)

		// Send response and initial events
		sendResponse(botToken, chatID, response, initialEvents, dryRun)
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
			return "❌ Please specify a state code.\n\nUsage: /subscribe NV", nil
		}
		return handleSubscribe(prefs, chatID, parts[1], modified, dryRun)

	case "/unsubscribe":
		if len(parts) < 2 {
			return "❌ Please specify a state code.\n\nUsage: /unsubscribe NV", nil
		}
		return handleUnsubscribe(prefs, chatID, parts[1], modified), nil

	case "/list":
		return handleList(prefs, chatID), nil

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

/subscribe &lt;STATE&gt; - Subscribe to a state
   Example: /subscribe NV

/unsubscribe &lt;STATE&gt; - Unsubscribe from a state
   Example: /unsubscribe CA

/list - Show your current subscriptions

/check - Trigger an immediate check (experimental)

/help - Show this help message

<b>State Codes:</b>
Use 2-letter state codes like NV, CA, TX, etc.
Use ALL to subscribe to all states.

<b>Notifications:</b>
You'll receive a message whenever new events are posted in your subscribed states. Checks run every hour.`
}

func handleSubscribe(prefs preferences.Preferences, chatID, state string, modified *bool, dryRun bool) (string, []*event.Event) {
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

	// Fetch current events for this state to send as initial sync
	var initialEvents []*event.Event
	if !dryRun {
		fmt.Printf("Fetching initial events for state %s...\n", state)
		sc := scraper.New()
		allEvents, err := sc.FetchEvents()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Failed to fetch initial events: %v\n", err)
			response += "⚠️ Unable to fetch current events, but you're subscribed!"
		} else {
			// Filter events by the subscribed state
			for _, evt := range allEvents {
				if state == "ALL" || strings.EqualFold(evt.State, state) {
					initialEvents = append(initialEvents, evt)
				}
			}

			if len(initialEvents) > 0 {
				// Limit to 10 initial events
				totalEvents := len(initialEvents)
				if totalEvents > 10 {
					initialEvents = initialEvents[:10]
					response += fmt.Sprintf("📨 Sending you the first 10 of %d current events...", totalEvents)
				} else {
					response += fmt.Sprintf("📨 Sending you %d current event(s)...", totalEvents)
				}
			} else {
				response += "ℹ️ No current events found for this state."
			}
		}
	}

	return response, initialEvents
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
