package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pfrederiksen/vga-events/internal/calendar"
	"github.com/pfrederiksen/vga-events/internal/course"
	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
	"github.com/pfrederiksen/vga-events/internal/scraper"
	"github.com/pfrederiksen/vga-events/internal/telegram"
)

const (
	// AllStatesCode is the special state code to match all states
	AllStatesCode = "ALL"

	// Error messages
	errFetchingEvents      = "❌ Error fetching events. Please try again later."
	errSendingCalendarFile = "❌ Error sending calendar file"
	errUserNotFound        = "❌ Error: User not found"
)

var (
	botToken         = flag.String("bot-token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token (or env: TELEGRAM_BOT_TOKEN)")
	gistID           = flag.String("gist-id", os.Getenv("TELEGRAM_GIST_ID"), "GitHub Gist ID (or env: TELEGRAM_GIST_ID)")
	githubToken      = flag.String("github-token", os.Getenv("TELEGRAM_GITHUB_TOKEN"), "GitHub token (or env: TELEGRAM_GITHUB_TOKEN)")
	golfCourseAPIKey = flag.String("golf-api-key", os.Getenv("GOLF_COURSE_API_KEY"), "Golf Course API key (or env: GOLF_COURSE_API_KEY)")
	encryptionKey    = flag.String("encryption-key", os.Getenv("TELEGRAM_ENCRYPTION_KEY"), "Encryption key for sensitive data (or env: TELEGRAM_ENCRYPTION_KEY)")
	dryRun           = flag.Bool("dry-run", false, "Show what would be done without making changes")
	loop             = flag.Bool("loop", false, "Run continuously with long polling (for real-time responses)")
	loopDuration     = flag.Duration("loop-duration", 5*time.Hour+50*time.Minute, "Maximum duration for loop mode (default 5h50m)")
	// Digest mode flags
	digest     = flag.String("digest", "", "Send digest to specific chat ID (used by GitHub Actions)")
	digestFile = flag.String("digest-file", "", "Path to digest events JSON file")
	digestType = flag.String("digest-type", "daily", "Type of digest: daily or weekly")
	// Stats rollover flag
	archiveWeeklyStats = flag.Bool("archive-weekly-stats", false, "Archive current week's stats to history for all users")
)

// Global course API client (initialized if key provided)
var courseClient *course.Client

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

// RateLimiter implements a simple sliding window rate limiter
type RateLimiter struct {
	mu       sync.Mutex
	requests map[string][]time.Time
	limit    int           // max requests
	window   time.Duration // time window
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// Allow checks if a request from the given chatID should be allowed
func (rl *RateLimiter) Allow(chatID string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Get existing requests for this chat ID
	timestamps := rl.requests[chatID]

	// Remove timestamps older than the window
	var validTimestamps []time.Time
	for _, ts := range timestamps {
		if ts.After(cutoff) {
			validTimestamps = append(validTimestamps, ts)
		}
	}

	// Check if limit exceeded
	if len(validTimestamps) >= rl.limit {
		return false
	}

	// Add current timestamp and update
	validTimestamps = append(validTimestamps, now)
	rl.requests[chatID] = validTimestamps

	return true
}

// CleanupOldEntries removes expired entries to prevent memory growth
func (rl *RateLimiter) CleanupOldEntries() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	for chatID, timestamps := range rl.requests {
		var validTimestamps []time.Time
		for _, ts := range timestamps {
			if ts.After(cutoff) {
				validTimestamps = append(validTimestamps, ts)
			}
		}
		if len(validTimestamps) == 0 {
			delete(rl.requests, chatID)
		} else {
			rl.requests[chatID] = validTimestamps
		}
	}
}

// validateUserInput validates and sanitizes user-provided text input
// Returns (sanitized text, error message)
func validateUserInput(input string, maxLength int, fieldName string) (string, string) {
	// Trim whitespace
	input = strings.TrimSpace(input)

	// Check length
	if len(input) == 0 {
		return "", fmt.Sprintf("❌ %s cannot be empty.", fieldName)
	}

	if len(input) > maxLength {
		return "", fmt.Sprintf("❌ %s is too long (max %d characters, got %d).", fieldName, maxLength, len(input))
	}

	// Remove control characters (except newlines and tabs which are acceptable in notes)
	var sanitized strings.Builder
	for _, r := range input {
		// Allow printable characters, spaces, newlines, tabs
		if r >= 32 || r == '\n' || r == '\t' {
			sanitized.WriteRune(r)
		}
	}

	result := sanitized.String()
	if len(result) == 0 {
		return "", fmt.Sprintf("❌ %s contains only invalid characters.", fieldName)
	}

	return result, ""
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

	// Initialize storage with encryption if key is provided
	storage, err := preferences.NewGistStorageWithEncryption(*gistID, *githubToken, *encryptionKey)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error initializing storage: %v\n", err)
		os.Exit(1)
	}
	if *encryptionKey != "" {
		fmt.Println("Encryption enabled for sensitive data")
	}

	// Initialize Golf Course API client if key is provided
	if *golfCourseAPIKey != "" {
		courseClient = course.NewClient(*golfCourseAPIKey)
		fmt.Println("Golf Course API enabled")
	}

	// Digest mode: send digest and exit
	if *digest != "" {
		if *digestFile == "" {
			fmt.Fprintf(os.Stderr, "Error: --digest-file is required when using --digest\n")
			os.Exit(1)
		}
		sendDigest(*botToken, *digest, *digestFile, *digestType)
		os.Exit(0)
	}

	// Load preferences
	prefs, err := storage.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading preferences: %v\n", err)
		os.Exit(1)
	}

	// Archive weekly stats mode: archive stats and exit
	if *archiveWeeklyStats {
		archiveWeeklyStatsForAllUsers(prefs, storage)
		os.Exit(0)
	}

	fmt.Printf("Loaded preferences for %d users\n", len(prefs))

	// Initialize rate limiter: 10 commands per minute per user
	rateLimiter := NewRateLimiter(10, time.Minute)

	// Start cleanup goroutine to prevent memory growth
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			rateLimiter.CleanupOldEntries()
		}
	}()

	if *loop {
		runLoop(storage, prefs, *botToken, *dryRun, *loopDuration, rateLimiter)
	} else {
		runOnce(storage, prefs, *botToken, *dryRun, rateLimiter)
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

func runLoop(storage *preferences.GistStorage, prefs preferences.Preferences, botToken string, dryRun bool, duration time.Duration, rateLimiter *RateLimiter) {
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
				chatID := fmt.Sprintf("%d", update.CallbackQuery.From.ID)

				// Check rate limit
				if !rateLimiter.Allow(chatID) {
					fmt.Printf("Rate limit exceeded for chat %s\n", chatID)
					sendResponse(botToken, chatID, "⚠️ Too many requests. Please wait a moment before trying again.", nil, dryRun)
					continue
				}

				handleCallbackQuery(prefs, update.CallbackQuery, &prefsModified, botToken, dryRun)
			} else if update.Message != nil {
				// Handle text message
				chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
				text := strings.TrimSpace(update.Message.Text)

				fmt.Printf("Message from %s (chat %s): %s\n", update.Message.From.FirstName, chatID, text)

				// Check rate limit
				if !rateLimiter.Allow(chatID) {
					fmt.Printf("Rate limit exceeded for chat %s\n", chatID)
					sendResponse(botToken, chatID, "⚠️ Too many requests. Please wait a moment before trying again.", nil, dryRun)
					continue
				}

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

func runOnce(storage *preferences.GistStorage, prefs preferences.Preferences, botToken string, dryRun bool, rateLimiter *RateLimiter) {
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
			chatID := fmt.Sprintf("%d", update.CallbackQuery.From.ID)

			// Check rate limit
			if !rateLimiter.Allow(chatID) {
				fmt.Printf("Rate limit exceeded for chat %s\n", chatID)
				sendResponse(botToken, chatID, "⚠️ Too many requests. Please wait a moment before trying again.", nil, dryRun)
				continue
			}

			handleCallbackQuery(prefs, update.CallbackQuery, &prefsModified, botToken, dryRun)
		} else if update.Message != nil {
			// Handle text message
			chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
			text := strings.TrimSpace(update.Message.Text)

			fmt.Printf("Message from %s (chat %s): %s\n", update.Message.From.FirstName, chatID, text)

			// Check rate limit
			if !rateLimiter.Allow(chatID) {
				fmt.Printf("Rate limit exceeded for chat %s\n", chatID)
				sendResponse(botToken, chatID, "⚠️ Too many requests. Please wait a moment before trying again.", nil, dryRun)
				continue
			}

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
			user := prefs.GetUser(callbackChatID)
			currentStatus := user.GetEventStatus(evt.ID)
			note := user.GetEventNote(evt.ID)
			msg, keyboard := telegram.FormatEventWithStatusAndNote(evt, currentStatus, note, callbackChatID, prefs)
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
		return errFetchingEvents
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
			user := prefs.GetUser(chatID)
			currentStatus := user.GetEventStatus(evt.ID)
			note := user.GetEventNote(evt.ID)
			msg, keyboard := telegram.FormatEventWithStatusAndNote(evt, currentStatus, note, chatID, prefs)
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

	case "unsubscribe-all":
		if param == "confirm" {
			user := prefs.GetUser(chatID)
			states := user.States
			count := len(states)
			if count > 0 {
				user.States = []string{}
				*modified = true
				responseText = fmt.Sprintf("✅ <b>Unsubscribed from all %d state(s)</b>\n\nYou will no longer receive event notifications.\n\nUse /subscribe &lt;STATE&gt; to start receiving notifications again.", count)
			} else {
				responseText = "ℹ️ You have no active subscriptions."
			}
		} else {
			responseText = "❌ Invalid action"
		}

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
		case "main":
			responseText, keyboard = showMenuKeyboard()
		case "all-events":
			responseText, _ = handleAllEvents(prefs, chatID, botToken, dryRun, modified)
		case "my-events":
			responseText, _ = handleMyEvents(prefs, chatID, botToken, dryRun, modified)
		case "upcoming":
			responseText = handleUpcomingEventsCallback(prefs, chatID, botToken, dryRun)
		case "reminders":
			responseText, keyboard = showRemindersKeyboard(prefs, chatID)
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

			// Track stats: event marked with status
			user.IncrementEventStatus(status)

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
				responseText = errSendingCalendarFile
				fmt.Fprintf(os.Stderr, "Error creating Telegram client: %v\n", err)
				break
			}

			caption := fmt.Sprintf("📅 <b>%s - %s</b>\n\nTap to add to your calendar!", evt.State, evt.Title)
			if err := client.SendDocument(filename, []byte(icsContent), caption); err != nil {
				responseText = errSendingCalendarFile
				fmt.Fprintf(os.Stderr, "Error sending document: %v\n", err)
				break
			}

			responseText = "✅ Calendar file sent! Tap it to add the event to your calendar."
		} else {
			responseText = fmt.Sprintf("[DRY RUN] Would send .ics file for event: %s - %s", evt.State, evt.Title)
		}

	case "bulk":
		// Handle bulk actions
		// Format: bulk:ACTION (e.g., "bulk:clear-skipped", "bulk:export-registered")
		responseText, keyboard = handleBulkCallback(prefs, chatID, param, modified, botToken, dryRun)

	case "ack-change":
		// Acknowledge event change notification
		// Format: ack-change:EVENT_ID
		responseText = "✅ Change acknowledged. You can update your calendar if needed."

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

//nolint:gocyclo // Command dispatcher naturally has high complexity due to many cases
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
			return "❌ Please specify a state code or 'all'.\n\nUsage: /unsubscribe NV\nUsage: /unsubscribe all", nil
		}
		arg := strings.ToLower(strings.TrimSpace(parts[1]))
		if arg == "all" {
			// Show confirmation keyboard
			return handleUnsubscribeAllWithKeyboard(prefs, chatID, botToken, dryRun)
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

		// Validate input
		keyword, errMsg := validateUserInput(keyword, 100, "Search keyword")
		if errMsg != "" {
			return errMsg, nil
		}

		return handleSearch(prefs, chatID, keyword, botToken, dryRun, modified)

	case "/export-calendar":
		// Optional parameter: state code
		var stateFilter string
		if len(parts) >= 2 {
			stateFilter = strings.ToUpper(strings.TrimSpace(parts[1]))
		}
		return handleExportCalendar(prefs, chatID, stateFilter, botToken, dryRun)

	case "/my-events":
		return handleMyEvents(prefs, chatID, botToken, dryRun, modified)

	case "/events":
		return handleAllEvents(prefs, chatID, botToken, dryRun, modified)

	case "/note":
		if len(parts) < 2 {
			return "❌ Please specify an event ID.\n\nUsage: /note &lt;event_id&gt; &lt;note_text&gt;\nUsage: /note &lt;event_id&gt; clear", nil
		}
		eventID := parts[1]

		// Check if second param is "clear"
		if len(parts) >= 3 && strings.ToLower(parts[2]) == "clear" {
			return handleRemoveNote(prefs, chatID, eventID, modified)
		}

		// Need note text
		if len(parts) < 3 {
			return "❌ Please provide note text.\n\nUsage: /note &lt;event_id&gt; &lt;note_text&gt;", nil
		}

		// Join remaining parts as note text
		noteText := strings.Join(parts[2:], " ")

		// Validate input
		noteText, errMsg := validateUserInput(noteText, 500, "Note text")
		if errMsg != "" {
			return errMsg, nil
		}

		return handleAddNote(prefs, chatID, eventID, noteText, modified)

	case "/notes":
		return handleListNotes(prefs, chatID, botToken, dryRun)

	case "/near":
		if len(parts) < 2 {
			return "❌ Please specify a city name.\n\nUsage: /near &lt;city&gt;\n\nExamples:\n/near Las Vegas\n/near \"San Diego\"", nil
		}
		// Join remaining parts as city name (supports multi-word cities)
		cityName := strings.Join(parts[1:], " ")
		cityName = strings.Trim(cityName, `"'`) // Remove quotes if present

		// Validate input
		cityName, errMsg := validateUserInput(cityName, 100, "City name")
		if errMsg != "" {
			return errMsg, nil
		}

		return handleNear(prefs, chatID, cityName, botToken, dryRun, modified)

	case "/reminders":
		return handleRemindersWithKeyboard(prefs, chatID, botToken, dryRun)

	case "/notify-removals":
		arg := ""
		if len(parts) >= 2 {
			arg = parts[1]
		}
		return handleNotifyRemovals(prefs, chatID, arg, modified)

	case "/bulk":
		return handleBulkWithKeyboard(prefs, chatID, botToken, dryRun)

	case "/stats":
		// Optional parameter: week, month, all
		period := "week"
		if len(parts) >= 2 {
			period = strings.ToLower(parts[1])
		}
		return handleStats(prefs, chatID, period), nil

	case "/check":
		return handleCheck(prefs, chatID, botToken, dryRun, modified)

	case "/invite":
		return handleInvite(prefs, chatID), nil

	case "/friends":
		return handleFriends(prefs, chatID), nil

	case "/join":
		if len(parts) < 2 {
			return "❌ Please provide an invite code.\n\nUsage: /join <invite_code>", nil
		}
		return handleJoin(prefs, chatID, parts[1], modified), nil

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
/near - Find events near a city 📍
/events - View all events for your subscribed states 📅
/my-events - View your tracked events ⭐
/note - Add a note to an event 📝
/notes - List all events with notes 📋
/reminders - Configure event reminders 🔔
/notify-removals - Toggle removal notifications ⚠️
/stats - View your engagement statistics 📊
/bulk - Bulk actions for multiple events 🔧
/export-calendar - Download all events as .ics file 📅
/invite - Get your friend invite code 👥
/friends - View your friend list 👥
/join - Join via friend invite code 👥
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

<b>Bulk Actions:</b>
Manage multiple events at once with /bulk:
• Clear all skipped events
• Export all registered events to calendar

<b>Friends &amp; Sharing:</b>
Connect with golf buddies to coordinate events:
• /invite - Get your invite code to share with friends
• /join &lt;code&gt; - Add a friend using their invite code
• /friends - View your friend list
When both you and a friend enable sharing in /settings, you'll see when they're registered for events.

<b>State Codes:</b>
Use 2-letter state codes like NV, CA, TX, etc.
Use %s to subscribe to all states.

<b>Notifications:</b>
You'll receive messages whenever new events are posted in your subscribed states.
• <b>Immediate mode</b> - Get notified right away (default)
• <b>Daily digest</b> - Receive a daily summary at 9 AM UTC
• <b>Weekly digest</b> - Receive a weekly summary on Mondays

Change your preferences with /settings

Checks run every hour.

━━━━━━━━━━━━━━━━━━━━━━
<b>Support &amp; Info:</b>
Need help or have feedback? Contact @iamdesertpaul

Created by Paul Frederiksen
Open source at github.com/pfrederiksen/vga-events`, AllStatesCode)
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

// handleAddNote adds or updates a note for an event
func handleAddNote(prefs preferences.Preferences, chatID, eventID, noteText string, modified *bool) (string, []*event.Event) {
	user := prefs.GetUser(chatID)
	if user == nil {
		return errUserNotFound, nil
	}

	user.SetEventNote(eventID, noteText)
	*modified = true

	return fmt.Sprintf("📝 Note added for event <code>%s</code>:\n\n<i>%s</i>", eventID, noteText), nil
}

// handleRemoveNote removes a note from an event
func handleRemoveNote(prefs preferences.Preferences, chatID, eventID string, modified *bool) (string, []*event.Event) {
	user := prefs.GetUser(chatID)
	if user == nil {
		return errUserNotFound, nil
	}

	// Check if note exists
	if user.GetEventNote(eventID) == "" {
		return fmt.Sprintf("ℹ️ No note found for event <code>%s</code>", eventID), nil
	}

	user.RemoveEventNote(eventID)
	*modified = true

	return fmt.Sprintf("✅ Note removed for event <code>%s</code>", eventID), nil
}

// handleStats displays user engagement statistics
func handleStats(prefs preferences.Preferences, chatID, period string) string {
	user := prefs.GetUser(chatID)
	if user == nil {
		return errUserNotFound
	}

	if !user.EnableStats {
		return "📊 Statistics tracking is disabled.\n\nStats help you track your VGA Golf engagement!"
	}

	var stats *preferences.WeeklyStats
	var periodName string

	switch period {
	case "week", "":
		stats = user.WeeklyStats
		periodName = "This Week"
	case "all":
		stats = user.GetAllTimeStats()
		periodName = "All Time"
	default:
		return "❌ Invalid period. Use: /stats, /stats week, or /stats all"
	}

	if stats == nil {
		return "📊 No statistics available yet.\n\nStart viewing events to track your activity!"
	}

	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("📊 <b>Your VGA Golf Stats (%s)</b>\n\n", periodName))

	// Events viewed
	msg.WriteString(fmt.Sprintf("📅 <b>Events Viewed:</b> %d\n", stats.EventsViewed))

	// Events marked by status
	if len(stats.EventsMarked) > 0 {
		msg.WriteString("\n<b>Events Marked:</b>\n")
		if count, ok := stats.EventsMarked[preferences.EventStatusInterested]; ok && count > 0 {
			msg.WriteString(fmt.Sprintf("  ⭐ Interested: %d\n", count))
		}
		if count, ok := stats.EventsMarked[preferences.EventStatusRegistered]; ok && count > 0 {
			msg.WriteString(fmt.Sprintf("  ✅ Registered: %d\n", count))
		}
		if count, ok := stats.EventsMarked[preferences.EventStatusMaybe]; ok && count > 0 {
			msg.WriteString(fmt.Sprintf("  🤔 Maybe: %d\n", count))
		}
		if count, ok := stats.EventsMarked[preferences.EventStatusSkip]; ok && count > 0 {
			msg.WriteString(fmt.Sprintf("  ❌ Skipped: %d\n", count))
		}
	}

	// Total marked
	totalMarked := 0
	for _, count := range stats.EventsMarked {
		totalMarked += count
	}
	if totalMarked > 0 {
		msg.WriteString(fmt.Sprintf("\n<b>Total Actions:</b> %d\n", totalMarked))
	}

	// Show history count for week view
	if period == "week" || period == "" {
		msg.WriteString(fmt.Sprintf("\n<i>Week started: %s</i>\n", stats.WeekStart.Format("Jan 2, 2006")))
		if len(user.StatsHistory) > 0 {
			msg.WriteString(fmt.Sprintf("\n📈 <b>%d week(s)</b> of history available\n", len(user.StatsHistory)))
			msg.WriteString("Use /stats all to see all-time stats")
		}
	} else {
		// All-time view
		weekCount := len(user.StatsHistory) + 1 // +1 for current week
		msg.WriteString(fmt.Sprintf("\n<i>Tracking for %d week(s)</i>", weekCount))
	}

	return msg.String()
}

// handleInvite displays the user's invite code
func handleInvite(prefs preferences.Preferences, chatID string) string {
	user := prefs.GetUser(chatID)
	if user == nil {
		return errUserNotFound
	}

	inviteCode := user.GetInviteCode()

	msg := fmt.Sprintf(`👥 <b>Invite Friends</b>

Your invite code: <code>%s</code>

<b>How it works:</b>
1. Share your invite code with friends
2. They send: /join %s
3. You'll be connected as friends!

<b>Privacy Note:</b>
When you're friends with someone and both have sharing enabled (/settings), you can see when they're registered for the same events.

Use /friends to see your current friends.`, inviteCode, inviteCode)

	return msg
}

// handleFriends lists the user's friends
func handleFriends(prefs preferences.Preferences, chatID string) string {
	user := prefs.GetUser(chatID)
	if user == nil {
		return errUserNotFound
	}

	if len(user.FriendChatIDs) == 0 {
		return `👥 <b>Friends</b>

You have no friends added yet.

Use /invite to get your invite code and share it with friends!

<b>Benefits of adding friends:</b>
• See which events your friends are registered for
• Coordinate golf outings together
• Share events with your group

Privacy is built-in: you control what you share via settings.`
	}

	msg := fmt.Sprintf(`👥 <b>Your Friends</b>

You have <b>%d friend(s)</b>:

`, len(user.FriendChatIDs))

	for i, friendChatID := range user.FriendChatIDs {
		msg += fmt.Sprintf("%d. User ID: <code>%s</code>\n", i+1, friendChatID)
	}

	msg += "\n<b>Sharing Status:</b> "
	if user.ShareEvents {
		msg += "✅ Enabled (friends can see your registered events)"
	} else {
		msg += "❌ Disabled (use /settings to enable)"
	}

	msg += "\n\nUse /invite to add more friends!"

	return msg
}

// handleJoin processes a friend invite
func handleJoin(prefs preferences.Preferences, chatID, inviteCode string, modified *bool) string {
	user := prefs.GetUser(chatID)
	if user == nil {
		return errUserNotFound
	}

	// Check if trying to add themselves
	if user.GetInviteCode() == inviteCode {
		return "❌ You can't add yourself as a friend!"
	}

	// Find the user with this invite code
	var friendChatID string
	for _, potentialFriendID := range prefs.GetAllUsers() {
		potentialFriend := prefs.GetUser(potentialFriendID)
		if potentialFriend.GetInviteCode() == inviteCode {
			friendChatID = potentialFriendID
			break
		}
	}

	if friendChatID == "" {
		return fmt.Sprintf("❌ Invalid invite code: <code>%s</code>\n\nMake sure you entered the code correctly.", inviteCode)
	}

	// Check if already friends
	if user.IsFriend(friendChatID) {
		return "ℹ️ You're already friends with this user!"
	}

	// Add friend (bidirectional)
	user.AddFriend(friendChatID)
	friend := prefs.GetUser(friendChatID)
	friend.AddFriend(chatID)
	*modified = true

	return fmt.Sprintf(`✅ <b>Friend Added!</b>

You're now connected with user <code>%s</code>

<b>Next steps:</b>
• Use /friends to see your friend list
• Enable event sharing in /settings to see when they're registered for events
• Use /my-events to coordinate attendance

Both you and your friend need to enable sharing to see each other's event registrations.`, friendChatID)
}

// handleNear finds events near a specified city
func handleNear(prefs preferences.Preferences, chatID, cityName, botToken string, dryRun bool, modified *bool) (string, []*event.Event) {
	user := prefs.GetUser(chatID)
	if len(user.States) == 0 {
		return "ℹ️ You need to subscribe to at least one state first.\n\nUse /subscribe &lt;STATE&gt; to get started.", nil
	}

	// Fetch current events
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return "❌ Error fetching events. Please try again later.", nil
	}

	// Filter events by subscribed states first
	var subscribedEvents []*event.Event
	for _, evt := range allEvents {
		for _, state := range user.States {
			if evt.State == state {
				subscribedEvents = append(subscribedEvents, evt)
				break
			}
		}
	}

	// Filter by city (case-insensitive substring match)
	normalizedCity := strings.ToLower(strings.TrimSpace(cityName))
	var matchingEvents []*event.Event
	for _, evt := range subscribedEvents {
		if strings.Contains(strings.ToLower(evt.City), normalizedCity) {
			// Apply user's date filter if enabled
			if user.DaysAhead > 0 && !evt.IsWithinDays(user.DaysAhead) {
				continue
			}
			// Filter past events if enabled
			if user.HidePastEvents && evt.IsPastEvent() {
				continue
			}
			matchingEvents = append(matchingEvents, evt)
		}
	}

	if len(matchingEvents) == 0 {
		return fmt.Sprintf("📍 No events found near <b>%s</b> in your subscribed states.\n\nTry a different city name or check your subscriptions with /list", cityName), nil
	}

	// Sort by date
	event.SortByDate(matchingEvents)

	// Send header
	client, err := telegram.NewClient(botToken, chatID)
	if err != nil {
		return fmt.Sprintf("📍 <b>Events near %s</b>\n\nFound %d event(s)", cityName, len(matchingEvents)), nil
	}

	headerMsg := fmt.Sprintf("📍 <b>Events near %s</b>\n\nFound %d event(s) in your subscribed states:", cityName, len(matchingEvents))
	if err := client.SendMessage(headerMsg); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending header: %v\n", err)
	}

	// Send each event
	for i, evt := range matchingEvents {
		currentStatus := user.GetEventStatus(evt.ID)
		note := user.GetEventNote(evt.ID)
		msg, keyboard := telegram.FormatEventWithStatusAndNote(evt, currentStatus, note, chatID, prefs)

		if !dryRun {
			if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending event %s: %v\n", evt.ID, err)
			}

			// Rate limiting
			if i < len(matchingEvents)-1 {
				time.Sleep(1 * time.Second)
			}
		}
	}

	// Track stats: events viewed
	if !dryRun {
		user.IncrementEventsViewed(len(matchingEvents))
		*modified = true
	}

	return "", nil // Already sent via messages
}

// handleListNotes lists all events with notes
func handleListNotes(prefs preferences.Preferences, chatID, botToken string, dryRun bool) (string, []*event.Event) {
	user := prefs.GetUser(chatID)
	if user == nil {
		return errUserNotFound, nil
	}

	if len(user.EventNotes) == 0 {
		return "📝 You have no notes.\n\nUse /note &lt;event_id&gt; &lt;text&gt; to add a note to an event.", nil
	}

	response := fmt.Sprintf("📝 <b>Your Event Notes</b> (%d)\n\n", len(user.EventNotes))

	// List each event with note
	for eventID, noteText := range user.EventNotes {
		response += fmt.Sprintf("Event ID: <code>%s</code>\n", eventID)
		response += fmt.Sprintf("📝 <i>%s</i>\n\n", noteText)
	}

	response += "Use /note &lt;event_id&gt; clear to remove a note.\n"
	response += "Use /events or /my-events to see event details."

	return response, nil
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

func handleCheck(prefs preferences.Preferences, chatID, botToken string, dryRun bool, modified *bool) (string, []*event.Event) {
	user := prefs.GetUser(chatID)

	// Check if user is subscribed to any states
	if len(user.States) == 0 {
		return `🔍 <b>Manual Check</b>

You're not subscribed to any states yet!

Use /subscribe to choose states you want to follow.`, nil
	}

	// Fetch all events
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return errFetchingEvents, nil
	}

	// Filter by subscribed states and exclude already seen events
	var unseenEvents []*event.Event
	for _, evt := range allEvents {
		// Check if event is in a subscribed state
		isSubscribed := false
		for _, state := range user.States {
			if strings.EqualFold(evt.State, state) {
				isSubscribed = true
				break
			}
		}

		if !isSubscribed {
			continue
		}

		// Check if already seen
		if !user.HasSeenEvent(evt.ID) {
			unseenEvents = append(unseenEvents, evt)
		}
	}

	if len(unseenEvents) == 0 {
		statesText := strings.Join(user.States, ", ")
		return fmt.Sprintf(`🔍 <b>Manual Check</b>

No new events found!

You're all caught up with events in: %s

The bot will notify you when new events are posted.`, statesText), nil
	}

	// Sort by date
	event.SortByDate(unseenEvents)

	// Send events
	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err != nil {
			return "❌ Error sending events", nil
		}

		// Send header
		statesText := strings.Join(user.States, ", ")
		headerMsg := fmt.Sprintf(`🔍 <b>Manual Check</b>

Found %d new event(s) in %s:`, len(unseenEvents), statesText)

		if err := client.SendMessage(headerMsg); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending header: %v\n", err)
		}

		// Send each event with course info and note
		for i, evt := range unseenEvents {
			note := user.GetEventNote(evt.ID)
			courseDetails := getCourseDetails(evt)
			msg := telegram.FormatEventWithCourse(evt, courseDetails, note)
			if err := client.SendMessage(msg); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending event %s: %v\n", evt.ID, err)
			}

			// Mark as seen
			user.MarkEventSeen(evt.ID)

			// Rate limiting
			if i < len(unseenEvents)-1 {
				if courseClient != nil {
					time.Sleep(2 * time.Second)
				} else {
					time.Sleep(1 * time.Second)
				}
			}
		}

		*modified = true

		return "", nil // Already sent
	}

	return fmt.Sprintf("[DRY RUN] Would send %d new events", len(unseenEvents)), nil
}

func handleSearch(prefs preferences.Preferences, chatID, keyword string, botToken string, dryRun bool, modified *bool) (string, []*event.Event) {
	// Fetch all events
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return errFetchingEvents, nil
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

	// Apply past event filtering (default: hide past events)
	user := prefs.GetUser(chatID)
	if user.HidePastEvents {
		var upcomingEvents []*event.Event
		for _, evt := range matchingEvents {
			if !evt.IsPastEvent() {
				upcomingEvents = append(upcomingEvents, evt)
			}
		}
		matchingEvents = upcomingEvents
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
			user := prefs.GetUser(chatID)
			currentStatus := user.GetEventStatus(evt.ID)
			note := user.GetEventNote(evt.ID)
			msg, keyboard := telegram.FormatEventWithStatusAndNote(evt, currentStatus, note, chatID, prefs)
			if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending event %s: %v\n", evt.ID, err)
			}

			// Rate limiting
			if i < len(eventsToSend)-1 {
				time.Sleep(1 * time.Second)
			}
		}

		// Track stats: events viewed
		user := prefs.GetUser(chatID)
		user.IncrementEventsViewed(len(eventsToSend))
		*modified = true

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
		return errFetchingEvents, nil
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
			return errSendingCalendarFile, nil
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
			return errSendingCalendarFile, nil
		}

		fmt.Printf("Sent bulk calendar file to %s (%d events)\n", chatID, len(filteredEvents))
		return "", nil // Already sent
	}

	return fmt.Sprintf("[DRY RUN] Would send bulk calendar file with %d events for %s", len(filteredEvents), strings.Join(filterStates, ", ")), nil
}

// getCourseDetails fetches course information for an event
func getCourseDetails(evt *event.Event) *telegram.CourseDetails {
	if courseClient == nil {
		return nil
	}

	courseInfo, err := courseClient.FindBestMatch(evt.Title, evt.City, evt.State)
	if err != nil {
		// Silently ignore API errors
		return nil
	}

	if courseInfo == nil {
		return nil
	}

	// Collect all tees (combined, no distinction between gender)
	// Deduplicate by tee name - keep first occurrence (male tees come first)
	seenTees := make(map[string]bool)
	var tees []telegram.TeeDetails
	for _, tee := range append(courseInfo.Tees.Male, courseInfo.Tees.Female...) {
		if !seenTees[tee.TeeName] {
			seenTees[tee.TeeName] = true
			tees = append(tees, telegram.TeeDetails{
				Name:    tee.TeeName,
				Par:     tee.ParTotal,
				Yardage: tee.TotalYards,
				Slope:   tee.SlopeRating,
				Rating:  tee.CourseRating,
				Holes:   tee.NumberOfHoles,
			})
		}
	}

	if len(tees) == 0 {
		return nil
	}

	return &telegram.CourseDetails{
		Name:    courseInfo.GetDisplayName(),
		Tees:    tees,
		Website: "",
		Phone:   "",
	}
}

func handleMyEvents(prefs preferences.Preferences, chatID string, botToken string, dryRun bool, modified *bool) (string, []*event.Event) {
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
		return errFetchingEvents, nil
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
				note := user.GetEventNote(evt.ID)
				courseDetails := getCourseDetails(evt)
				msg, keyboard := telegram.FormatEventWithStatusAndCourse(evt, courseDetails, status, note, chatID, prefs)
				if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
					fmt.Fprintf(os.Stderr, "Error sending event %s: %v\n", evt.ID, err)
				}

				// Rate limiting (longer if using course API)
				if i < len(group)-1 {
					if courseClient != nil {
						time.Sleep(2 * time.Second)
					} else {
						time.Sleep(1 * time.Second)
					}
				}
			}
		}

		// Track stats: events viewed
		user.IncrementEventsViewed(totalEvents)
		*modified = true

		return "", nil // Already sent
	}

	return fmt.Sprintf("[DRY RUN] Would send %d tracked events", totalEvents), nil
}

func handleAllEvents(prefs preferences.Preferences, chatID string, botToken string, dryRun bool, modified *bool) (string, []*event.Event) {
	// Get user's subscribed states
	states := prefs.GetStates(chatID)
	if len(states) == 0 {
		return `📅 <b>All Events</b>

You're not subscribed to any states yet.

Use /subscribe to start receiving events!`, nil
	}

	// Fetch all events
	sc := scraper.New()
	allEvents, err := sc.FetchEvents()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
		return errFetchingEvents, nil
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

	// Apply past event filtering (default: hide past events)
	user := prefs.GetUser(chatID)
	if user.HidePastEvents {
		var upcomingEvents []*event.Event
		for _, evt := range filteredEvents {
			if !evt.IsPastEvent() {
				upcomingEvents = append(upcomingEvents, evt)
			}
		}
		filteredEvents = upcomingEvents
	}

	if len(filteredEvents) == 0 {
		return fmt.Sprintf(`📅 <b>All Events</b>

No events found for your subscribed states: %s

Check back later or subscribe to more states with /subscribe`, strings.Join(states, ", ")), nil
	}

	// Sort by date (soonest first)
	event.SortByDate(filteredEvents)

	// Limit to 50 events to avoid overwhelming the user
	eventsToSend := filteredEvents
	if len(eventsToSend) > 50 {
		eventsToSend = eventsToSend[:50]
	}

	// Send events with status tracking buttons
	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err != nil {
			return "❌ Error sending events", nil
		}

		// Send header message
		headerMsg := fmt.Sprintf(`📅 <b>All Events</b>

Found %d event(s) for %s

Showing %d event(s), sorted by date:`, len(filteredEvents), strings.Join(states, ", "), len(eventsToSend))

		if err := client.SendMessage(headerMsg); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending header: %v\n", err)
		}

		// Send each event with status buttons
		user := prefs.GetUser(chatID)
		for i, evt := range eventsToSend {
			currentStatus := user.GetEventStatus(evt.ID)
			note := user.GetEventNote(evt.ID)
			msg, keyboard := telegram.FormatEventWithStatusAndNote(evt, currentStatus, note, chatID, prefs)
			if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending event %s: %v\n", evt.ID, err)
			}

			// Rate limiting
			if i < len(eventsToSend)-1 {
				time.Sleep(1 * time.Second)
			}
		}

		// Track stats: events viewed
		user.IncrementEventsViewed(len(eventsToSend))
		*modified = true

		return "", nil // Already sent
	}

	return fmt.Sprintf("[DRY RUN] Would send %d events for %s", len(eventsToSend), strings.Join(states, ", ")), nil
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

// handleUnsubscribeAllWithKeyboard shows the unsubscribe all confirmation keyboard
func handleUnsubscribeAllWithKeyboard(prefs preferences.Preferences, chatID, botToken string, dryRun bool) (string, []*event.Event) {
	states := prefs.GetStates(chatID)
	if len(states) == 0 {
		return "ℹ️ You have no active subscriptions.", nil
	}

	text := fmt.Sprintf("⚠️ Are you sure you want to unsubscribe from <b>all %d state(s)</b>?\n\nThis will remove: %s", len(states), strings.Join(states, ", "))
	keyboard := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: "✅ Yes, unsubscribe all", CallbackData: "unsubscribe-all:confirm"},
				{Text: "❌ Cancel", CallbackData: "menu:main"},
			},
		},
	}

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
				{Text: "📅 All Events", CallbackData: "menu:all-events"},
				{Text: "📊 My Events", CallbackData: "menu:my-events"},
			},
			{
				{Text: "🔍 Search", CallbackData: "menu:search"},
				{Text: "📆 Upcoming", CallbackData: "menu:upcoming"},
			},
			{
				{Text: "⭐ Subscriptions", CallbackData: "manage"},
				{Text: "🔔 Reminders", CallbackData: "menu:reminders"},
			},
			{
				{Text: "⚙️ Settings", CallbackData: "settings"},
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

// sendDigest sends a digest message to a specific user
func sendDigest(botToken, chatID, digestFile, digestType string) {
	// Read digest events from file
	f, err := os.Open(digestFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening digest file: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			fmt.Fprintf(os.Stderr, "Error closing digest file: %v\n", closeErr)
		}
	}()

	var result struct {
		NewEvents []*event.Event `json:"new_events"`
	}

	decoder := json.NewDecoder(f)
	if err := decoder.Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing digest JSON: %v\n", err)
		os.Exit(1)
	}

	if len(result.NewEvents) == 0 {
		fmt.Println("No events in digest")
		return
	}

	// Create Telegram client
	client, err := telegram.NewClient(botToken, chatID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Telegram client: %v\n", err)
		os.Exit(1)
	}

	// Format and send digest message
	digestMsg := telegram.FormatDigest(result.NewEvents, digestType)

	if err := client.SendMessage(digestMsg); err != nil {
		fmt.Fprintf(os.Stderr, "Error sending digest: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Successfully sent %s digest with %d event(s) to %s\n", digestType, len(result.NewEvents), chatID)
}

// handleBulkWithKeyboard shows the bulk actions menu
func handleBulkWithKeyboard(prefs preferences.Preferences, chatID, botToken string, dryRun bool) (string, []*event.Event) {
	text, keyboard := showBulkActionsKeyboard(prefs, chatID)

	if !dryRun {
		client, err := telegram.NewClient(botToken, chatID)
		if err != nil {
			return "❌ Error displaying bulk actions menu", nil
		}

		if err := client.SendMessageWithKeyboard(text, keyboard); err != nil {
			return "❌ Error sending bulk actions menu", nil
		}
		return "", nil // Message sent via keyboard
	}

	return text, nil
}

// showBulkActionsKeyboard creates the bulk actions keyboard
func showBulkActionsKeyboard(prefs preferences.Preferences, chatID string) (string, *telegram.InlineKeyboardMarkup) {
	user := prefs.GetUser(chatID)

	// Count events by status
	skippedCount := len(user.GetEventsByStatus(preferences.EventStatusSkip))
	registeredCount := len(user.GetEventsByStatus(preferences.EventStatusRegistered))

	text := `🔧 <b>Bulk Actions</b>

Manage multiple events at once:

<b>Event Status Cleanup:</b>
• Clear all skipped events

<b>Calendar Export:</b>
• Export all registered events to calendar`

	if skippedCount > 0 {
		text += fmt.Sprintf("\n\nYou have <b>%d</b> skipped event(s)", skippedCount)
	}
	if registeredCount > 0 {
		text += fmt.Sprintf("\n\nYou have <b>%d</b> registered event(s)", registeredCount)
	}

	keyboard := &telegram.InlineKeyboardMarkup{
		InlineKeyboard: [][]telegram.InlineKeyboardButton{
			{
				{Text: fmt.Sprintf("🗑 Clear Skipped (%d)", skippedCount), CallbackData: "bulk:clear-skipped"},
			},
			{
				{Text: fmt.Sprintf("📥 Export Registered (%d)", registeredCount), CallbackData: "bulk:export-registered"},
			},
			{
				{Text: "🔙 Back to Menu", CallbackData: "menu:main"},
			},
		},
	}

	return text, keyboard
}

// handleBulkCallback handles bulk action callbacks
func handleBulkCallback(prefs preferences.Preferences, chatID, action string, modified *bool, botToken string, dryRun bool) (string, *telegram.InlineKeyboardMarkup) {
	user := prefs.GetUser(chatID)

	switch action {
	case "clear-skipped":
		// Clear all skipped events
		skippedEvents := user.GetEventsByStatus(preferences.EventStatusSkip)
		if len(skippedEvents) == 0 {
			return "ℹ️ No skipped events to clear", nil
		}

		for _, eventID := range skippedEvents {
			user.RemoveEventStatus(eventID)
		}
		*modified = true

		return fmt.Sprintf("✅ Cleared <b>%d</b> skipped event(s)", len(skippedEvents)), nil

	case "export-registered":
		// Export all registered events to a single calendar file
		registeredEventIDs := user.GetEventsByStatus(preferences.EventStatusRegistered)
		if len(registeredEventIDs) == 0 {
			return "ℹ️ No registered events to export", nil
		}

		// Fetch current events to get full event data
		sc := scraper.New()
		allEvents, err := sc.FetchEvents()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching events: %v\n", err)
			return "❌ Error fetching event data", nil
		}

		// Find registered events
		var registeredEvents []*event.Event
		for _, evt := range allEvents {
			for _, regID := range registeredEventIDs {
				if evt.ID == regID {
					registeredEvents = append(registeredEvents, evt)
					break
				}
			}
		}

		if len(registeredEvents) == 0 {
			return "ℹ️ None of your registered events are currently available on the VGA website", nil
		}

		// Generate combined .ics file
		icsContent := calendar.GenerateMultiEventICS(registeredEvents)
		filename := "vga-registered-events.ics"

		if !dryRun {
			client, err := telegram.NewClient(botToken, chatID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating client: %v\n", err)
				return errSendingCalendarFile, nil
			}

			caption := fmt.Sprintf("📅 <b>Your Registered Events</b>\n\n%d event(s) ready to import to your calendar!", len(registeredEvents))
			if err := client.SendDocument(filename, []byte(icsContent), caption); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending document: %v\n", err)
				return errSendingCalendarFile, nil
			}

			return fmt.Sprintf("✅ Calendar file sent with <b>%d</b> registered event(s)!", len(registeredEvents)), nil
		}

		return fmt.Sprintf("[DRY RUN] Would export %d registered event(s) to calendar", len(registeredEvents)), nil

	default:
		return "❌ Unknown bulk action", nil
	}
}

// archiveWeeklyStatsForAllUsers archives the current week's stats to history for all users
func archiveWeeklyStatsForAllUsers(prefs preferences.Preferences, storage *preferences.GistStorage) {
	fmt.Println("📊 Archiving weekly stats for all users...")

	chatIDs := prefs.GetAllUsers()
	if len(chatIDs) == 0 {
		fmt.Println("ℹ️ No users found")
		return
	}

	archivedCount := 0
	for _, chatID := range chatIDs {
		user := prefs.GetUser(chatID)
		if !user.EnableStats {
			continue
		}

		// Archive current week to history
		user.ArchiveCurrentWeek()
		archivedCount++

		fmt.Printf("✅ Archived stats for user %s\n", chatID)
	}

	// Save updated preferences
	if err := storage.Save(prefs); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving preferences: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Successfully archived stats for %d user(s)\n", archivedCount)
}

// handleNotifyRemovals toggles the removal notification setting
func handleNotifyRemovals(prefs preferences.Preferences, chatID, arg string, modified *bool) (string, []*event.Event) {
	user := prefs.GetUser(chatID)

	if arg == "" {
		// Show current status
		status := "disabled"
		emoji := "❌"
		if user.NotifyOnRemoval {
			status = "enabled"
			emoji = "✅"
		}
		return fmt.Sprintf(`🔔 <b>Removal Notifications</b>

%s Currently: <b>%s</b>

When events are removed or cancelled from the VGA website, you can be notified.

• Events you're <b>registered</b> for → High priority notification ⚠️
• Events in your subscribed states → Low priority notification ℹ️

<b>Usage:</b>
• <code>/notify-removals on</code> - Enable notifications
• <code>/notify-removals off</code> - Disable notifications`, emoji, status), nil
	}

	// Toggle setting
	switch strings.ToLower(arg) {
	case "on", "enable", "yes":
		user.NotifyOnRemoval = true
		*modified = true
		return "✅ Removal notifications <b>enabled</b>\n\nYou'll be notified when events are removed from the VGA website.", nil

	case "off", "disable", "no":
		user.NotifyOnRemoval = false
		*modified = true
		return "❌ Removal notifications <b>disabled</b>\n\nYou won't be notified about removed events.", nil

	default:
		return "❌ Invalid option. Use:\n• <code>/notify-removals on</code>\n• <code>/notify-removals off</code>", nil
	}
}
