package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pfrederiksen/vga-events/internal/course"
	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/telegram"
)

var (
	botToken            = flag.String("bot-token", os.Getenv("TELEGRAM_BOT_TOKEN"), "Telegram bot token (or env: TELEGRAM_BOT_TOKEN)")
	chatID              = flag.String("chat-id", os.Getenv("TELEGRAM_CHAT_ID"), "Telegram chat ID (or env: TELEGRAM_CHAT_ID)")
	golfCourseAPIKey    = flag.String("golf-api-key", os.Getenv("GOLF_COURSE_API_KEY"), "Golf Course API key (or env: GOLF_COURSE_API_KEY)")
	eventsFile          = flag.String("events-file", "", "Path to events JSON file (or read from stdin)")
	dryRun              = flag.Bool("dry-run", false, "Print messages without sending")
	maxMessages         = flag.Int("max-messages", 10, "Maximum number of messages to send")
	stateFilter         = flag.String("state", "", "Only send messages for this state")
	hidePast            = flag.Bool("hide-past", true, "Filter out past events (default: true)")
	daysAhead           = flag.Int("days-ahead", 0, "Only show events within N days (0 = disabled)")
	checkReminders      = flag.Bool("check-reminders", false, "Check if event matches reminder days (exits 0 if match, 1 if no match)")
	reminderDays        = flag.Int("reminder-days", 0, "Number of days before event to send reminder (used with --check-reminders)")
	removalNotification = flag.Bool("removal-notification", false, "Send removal notifications (reads from removed_events field)")
	changeNotification  = flag.Bool("change-notification", false, "Send change notifications (reads from changed_events field)")
	eventStatus         = flag.String("event-status", "", "Event status for removal/change notification (registered/interested/maybe)")
	eventNote           = flag.String("event-note", "", "User's note for the event (for removal/change notifications)")
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
		NewEvents     []*event.Event        `json:"new_events"`
		RemovedEvents []*event.Event        `json:"removed_events"`
		ChangedEvents []*event.EventChange  `json:"changed_events"`
	}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	// Return appropriate events based on mode
	if *removalNotification {
		return result.RemovedEvents, nil
	}
	// For change notifications, we need to return nil here and handle separately
	// since we need the EventChange objects, not Event objects
	return result.NewEvents, nil
}

// readChangedEvents reads changed events from file or stdin
// Returns both the changes and a map of current events for lookup
func readChangedEvents(filePath string) ([]*event.EventChange, map[string]*event.Event, error) {
	var reader io.Reader
	if filePath != "" {
		f, err := os.Open(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("opening events file: %w", err)
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
		NewEvents     []*event.Event       `json:"new_events"`
		ChangedEvents []*event.EventChange `json:"changed_events"`
	}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("parsing JSON: %w", err)
	}

	// Build event map for lookup
	eventsMap := make(map[string]*event.Event)
	for _, evt := range result.NewEvents {
		eventsMap[evt.ID] = evt
	}

	return result.ChangedEvents, eventsMap, nil
}

// handleChangeNotifications handles the change notification flow
func handleChangeNotifications() {
	changes, eventsMap, err := readChangedEvents(*eventsFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading changed events: %v\n", err)
		os.Exit(1)
	}

	if len(changes) == 0 {
		fmt.Println("No changed events to send")
		os.Exit(0)
	}

	// Filter by state if specified
	if *stateFilter != "" {
		filteredChanges := make([]*event.EventChange, 0)
		for _, change := range changes {
			if evt, exists := eventsMap[change.EventID]; exists {
				if evt.State == *stateFilter {
					filteredChanges = append(filteredChanges, change)
				}
			}
		}
		changes = filteredChanges
	}

	// Limit number of messages
	if len(changes) > *maxMessages {
		changes = changes[:*maxMessages]
	}

	if len(changes) == 0 {
		fmt.Println("No changed events match criteria")
		os.Exit(0)
	}

	// Dry run mode
	if *dryRun {
		handleDryRunChanges(changes, eventsMap)
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

	// Send change notifications
	for i, change := range changes {
		evt, exists := eventsMap[change.EventID]
		if !exists {
			fmt.Fprintf(os.Stderr, "Warning: Event %s not found, skipping change notification\n", change.EventID)
			continue
		}

		// Format the change message with status and note
		msg, keyboard := telegram.FormatEventChangeWithNote(evt, change.ChangeType, change.OldValue, change.NewValue, *eventStatus, *eventNote)

		// Send message with keyboard
		if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
			fmt.Fprintf(os.Stderr, "Error sending change notification for event %s: %v\n", evt.ID, err)
			os.Exit(1)
		}

		// Rate limiting: wait between messages
		if i < len(changes)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	fmt.Printf("Successfully sent %d change notification(s)\n", len(changes))
}

// getCourseDetailsForEvent fetches course information for an event
func getCourseDetailsForEvent(client *course.Client, evt *event.Event) *telegram.CourseDetails {
	if client == nil {
		return nil
	}

	courseInfo, err := client.FindBestMatch(evt.Title, evt.City, evt.State)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Error looking up course for %s: %v\n", evt.Title, err)
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
		Name: courseInfo.GetDisplayName(),
		Tees: tees,
	}
}

// handleDryRun handles dry run mode output for events
func handleDryRun(events []*event.Event) {
	notificationType := "new event"
	if *removalNotification {
		notificationType = "removal"
	} else if *checkReminders {
		notificationType = "reminder"
	}

	fmt.Printf("DRY RUN MODE - Would send %d %s notification(s):\n\n", len(events), notificationType)

	// Initialize Golf Course API client once if key is provided
	var courseClient *course.Client
	if *golfCourseAPIKey != "" {
		courseClient = course.NewClient(*golfCourseAPIKey)
		fmt.Printf("Golf Course API enabled for dry run\n\n")
	}

	for i, evt := range events {
		var msg string
		var hasKeyboard bool

		if *removalNotification {
			if *eventStatus != "" {
				msg = telegram.FormatRemovedEvent(evt, *eventStatus, *eventNote)
				fmt.Printf("--- Removal Message %d/%d (High Urgency) ---\n", i+1, len(events))
			} else {
				msg = telegram.FormatRemovedEventGeneral(evt)
				fmt.Printf("--- Removal Message %d/%d (Low Urgency) ---\n", i+1, len(events))
			}
			hasKeyboard = false
		} else {
			courseDetails := getCourseDetailsForEvent(courseClient, evt)
			msg, _ = telegram.FormatEventWithStatusAndCourse(evt, courseDetails, "", "", "", nil)
			fmt.Printf("--- Message %d/%d ---\n", i+1, len(events))
			if courseDetails != nil {
				fmt.Printf("Course info: %s (%d tee options)\n", courseDetails.Name, len(courseDetails.Tees))
			}
			hasKeyboard = true
		}

		fmt.Println(msg)
		fmt.Printf("\n(Length: %d characters)\n", len(msg))
		if hasKeyboard {
			fmt.Printf("Buttons: 📅 Calendar, ⭐ Interested, ✅ Registered, 🤔 Maybe, ❌ Skip\n\n")
		} else {
			fmt.Printf("No interactive buttons\n\n")
		}
	}
}

// handleDryRunChanges handles dry run mode output for change notifications
func handleDryRunChanges(changes []*event.EventChange, eventsMap map[string]*event.Event) {
	fmt.Printf("DRY RUN MODE - Would send %d change notification(s):\n\n", len(changes))

	for i, change := range changes {
		evt, exists := eventsMap[change.EventID]
		if !exists {
			fmt.Printf("--- Change Message %d/%d ---\n", i+1, len(changes))
			fmt.Printf("WARNING: Event %s not found in current events\n\n", change.EventID)
			continue
		}

		msg, _ := telegram.FormatEventChangeWithNote(evt, change.ChangeType, change.OldValue, change.NewValue, *eventStatus, *eventNote)

		fmt.Printf("--- Change Message %d/%d ---\n", i+1, len(changes))
		if *eventStatus != "" {
			fmt.Printf("User status: %s\n", *eventStatus)
		}
		if *eventNote != "" {
			fmt.Printf("User note: %s\n", *eventNote)
		}
		fmt.Println(msg)
		fmt.Printf("\n(Length: %d characters)\n", len(msg))
		fmt.Printf("Buttons: 📅 Update Calendar, ✅ Acknowledged\n\n")
	}
}

func main() {
	flag.Parse()

	// Handle change notifications separately
	if *changeNotification {
		handleChangeNotifications()
		return
	}

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
		handleDryRun(events)
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

	// Initialize Golf Course API client if key is provided
	var courseClient *course.Client
	if *golfCourseAPIKey != "" {
		courseClient = course.NewClient(*golfCourseAPIKey)
		fmt.Printf("Golf Course API enabled\n")
	}

	// Send messages with interactive buttons
	for i, evt := range events {
		var msg string
		var keyboard *telegram.InlineKeyboardMarkup

		// Format message based on notification type
		if *removalNotification {
			// Removal notification - check if user had event tracked
			if *eventStatus != "" {
				// High urgency: user had this event tracked
				msg = telegram.FormatRemovedEvent(evt, *eventStatus, *eventNote)
			} else {
				// Low urgency: user just subscribed to state
				msg = telegram.FormatRemovedEventGeneral(evt)
			}
			// No keyboard for removal notifications
			keyboard = nil
		} else if *checkReminders {
			// Reminder notification
			msg, keyboard = telegram.FormatReminder(evt, *reminderDays)
		} else {
			// New event notification
			// Look up course information if Golf Course API is enabled
			courseDetails := getCourseDetailsForEvent(courseClient, evt)
			if courseDetails != nil {
				fmt.Printf("Found course info for %s: %s (%d tee options)\n",
					evt.Title, courseDetails.Name, len(courseDetails.Tees))
			}
			// Use status keyboard with course info for new events
			msg, keyboard = telegram.FormatEventWithStatusAndCourse(evt, courseDetails, "", "", "", nil)
		}

		// Send message
		if keyboard != nil {
			if err := client.SendMessageWithKeyboard(msg, keyboard); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending message for event %s: %v\n", evt.ID, err)
				os.Exit(1)
			}
		} else {
			if err := client.SendMessage(msg); err != nil {
				fmt.Fprintf(os.Stderr, "Error sending message for event %s: %v\n", evt.ID, err)
				os.Exit(1)
			}
		}

		// Rate limiting: wait between messages
		if i < len(events)-1 {
			time.Sleep(1 * time.Second)
		}
	}

	messageType := "message"
	if *removalNotification {
		messageType = "removal notification"
	} else if *checkReminders {
		messageType = "reminder"
	}
	fmt.Printf("Successfully sent %d %s(s)\n", len(events), messageType)
}
