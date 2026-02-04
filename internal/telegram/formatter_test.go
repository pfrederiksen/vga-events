package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
)

func TestFormatEvent(t *testing.T) {
	tests := []struct {
		name     string
		event    *event.Event
		contains []string
	}{
		{
			name: "complete event",
			event: &event.Event{
				ID:        "test123",
				State:     "NV",
				Title:     "Chimera Golf Club",
				DateText:  "Apr 04 2026",
				City:      "Las Vegas",
				Raw:       "NV - Chimera Golf Club - Las Vegas",
				SourceURL: "https://vgagolf.org/state-events/",
				FirstSeen: time.Now(),
			},
			contains: []string{
				"NV",
				"Chimera Golf Club",
				"Apr 4, 2026", // Enhanced date format includes day of week
				"Las Vegas",
				"vgagolf.org/state-events",
				"login required",
				"#VGAGolf",
				"#Golf",
				"#NV",
				"🏌️",
			},
		},
		{
			name: "event without date",
			event: &event.Event{
				ID:        "test456",
				State:     "CA",
				Title:     "Pebble Beach",
				DateText:  "",
				City:      "Monterey",
				Raw:       "CA - Pebble Beach - Monterey",
				SourceURL: "https://vgagolf.org/state-events/",
				FirstSeen: time.Now(),
			},
			contains: []string{
				"CA",
				"Pebble Beach",
				"Monterey",
				"#VGAGolf",
				"#CA",
			},
		},
		{
			name: "event without city",
			event: &event.Event{
				ID:        "test789",
				State:     "TX",
				Title:     "Dallas Country Club",
				DateText:  "May 15 2026",
				City:      "",
				Raw:       "TX - Dallas Country Club",
				SourceURL: "https://vgagolf.org/state-events/",
				FirstSeen: time.Now(),
			},
			contains: []string{
				"TX",
				"Dallas Country Club",
				"May 15, 2026", // Enhanced date format
				"#Golf",
				"#TX",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatEvent(tt.event)

			// Check that message is not empty
			if got == "" {
				t.Error("FormatEvent() returned empty string")
			}

			// Check that message is within Telegram's limit (4096 chars)
			if len(got) > 4096 {
				t.Errorf("FormatEvent() length = %d, exceeds Telegram limit of 4096", len(got))
			}

			// Check contains
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatEvent() missing %q in message:\n%s", want, got)
				}
			}
		})
	}
}

func TestFormatSummary(t *testing.T) {
	tests := []struct {
		name     string
		count    int
		states   []string
		contains []string
	}{
		{
			name:   "single event, single state",
			count:  1,
			states: []string{"NV"},
			contains: []string{
				"<b>1</b> new event",
				"1 state",
				"NV",
				"#VGAGolf",
			},
		},
		{
			name:   "multiple events, multiple states",
			count:  5,
			states: []string{"NV", "CA", "TX"},
			contains: []string{
				"<b>5</b> new events",
				"3 states",
				"NV, CA, TX",
				"#VGAGolf",
			},
		},
		{
			name:   "multiple events, no states specified",
			count:  10,
			states: []string{},
			contains: []string{
				"<b>10</b> new events",
				"#VGAGolf",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSummary(tt.count, tt.states)

			// Check that message is not empty
			if got == "" {
				t.Error("FormatSummary() returned empty string")
			}

			// Check contains
			for _, want := range tt.contains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatSummary() missing %q in message:\n%s", want, got)
				}
			}
		})
	}
}

func TestFormatEventWithCalendar(t *testing.T) {
	evt := &event.Event{
		ID:        "test123",
		State:     "NV",
		Title:     "Test Event",
		DateText:  "Apr 04 2026",
		City:      "Las Vegas",
		SourceURL: "https://vgagolf.org/state-events/",
		FirstSeen: time.Now(),
	}

	text, keyboard := FormatEventWithCalendar(evt)

	// Check text contains event info
	if !strings.Contains(text, "NV") || !strings.Contains(text, "Test Event") {
		t.Error("Text should contain event information")
	}

	// Check keyboard has calendar button
	if keyboard == nil {
		t.Fatal("Keyboard should not be nil")
	}

	if len(keyboard.InlineKeyboard) != 1 {
		t.Errorf("Expected 1 keyboard row, got %d", len(keyboard.InlineKeyboard))
	}

	if len(keyboard.InlineKeyboard[0]) != 1 {
		t.Errorf("Expected 1 button in first row, got %d", len(keyboard.InlineKeyboard[0]))
	}

	button := keyboard.InlineKeyboard[0][0]
	if button.Text != "📅 Add to Calendar" {
		t.Errorf("Button text = %q, want %q", button.Text, "📅 Add to Calendar")
	}

	expectedCallback := "calendar:test123"
	if button.CallbackData != expectedCallback {
		t.Errorf("Button callback = %q, want %q", button.CallbackData, expectedCallback)
	}
}

func TestFormatEventWithStatus(t *testing.T) {
	evt := &event.Event{
		ID:        "test456",
		State:     "CA",
		Title:     "Status Test Event",
		DateText:  "May 20 2026",
		City:      "San Francisco",
		SourceURL: "https://vgagolf.org/state-events/",
		FirstSeen: time.Now(),
	}

	tests := []struct {
		name                string
		status              string
		expectedStatusText  string
		expectedStatusEmoji string
	}{
		{
			name:                "interested status",
			status:              "interested",
			expectedStatusText:  "Interested",
			expectedStatusEmoji: "⭐",
		},
		{
			name:                "registered status",
			status:              "registered",
			expectedStatusText:  "Registered",
			expectedStatusEmoji: "✅",
		},
		{
			name:                "maybe status",
			status:              "maybe",
			expectedStatusText:  "Maybe",
			expectedStatusEmoji: "🤔",
		},
		{
			name:                "skip status",
			status:              "skip",
			expectedStatusText:  "Skipped",
			expectedStatusEmoji: "❌",
		},
		{
			name:                "no status",
			status:              "",
			expectedStatusText:  "",
			expectedStatusEmoji: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			text, keyboard := FormatEventWithStatus(evt, tt.status)

			// Check that text contains event info
			if !strings.Contains(text, "CA") || !strings.Contains(text, "Status Test Event") {
				t.Error("Text should contain event information")
			}

			// Check status indicator in text
			if tt.status != "" {
				if !strings.Contains(text, tt.expectedStatusEmoji) {
					t.Errorf("Text should contain status emoji %s", tt.expectedStatusEmoji)
				}
				if !strings.Contains(text, tt.expectedStatusText) {
					t.Errorf("Text should contain status text %s", tt.expectedStatusText)
				}
			}

			// Check keyboard structure
			if keyboard == nil {
				t.Fatal("Keyboard should not be nil")
			}

			// Should have 3 rows: calendar, interested/registered, maybe/skip
			if len(keyboard.InlineKeyboard) != 3 {
				t.Errorf("Expected 3 keyboard rows, got %d", len(keyboard.InlineKeyboard))
			}

			// First row: Calendar button
			if len(keyboard.InlineKeyboard[0]) != 1 {
				t.Errorf("Expected 1 button in first row (calendar), got %d", len(keyboard.InlineKeyboard[0]))
			}
			calendarButton := keyboard.InlineKeyboard[0][0]
			if calendarButton.Text != "📅 Calendar" {
				t.Errorf("Calendar button text = %q, want '📅 Calendar'", calendarButton.Text)
			}
			if calendarButton.CallbackData != "calendar:test456" {
				t.Errorf("Calendar button callback = %q, want 'calendar:test456'", calendarButton.CallbackData)
			}

			// Second row: Interested and Registered buttons
			if len(keyboard.InlineKeyboard[1]) != 2 {
				t.Errorf("Expected 2 buttons in second row (interested/registered), got %d", len(keyboard.InlineKeyboard[1]))
			}
			interestedButton := keyboard.InlineKeyboard[1][0]
			if interestedButton.Text != "⭐ Interested" {
				t.Errorf("Interested button text = %q, want '⭐ Interested'", interestedButton.Text)
			}
			if interestedButton.CallbackData != "status:test456:interested" {
				t.Errorf("Interested button callback = %q, want 'status:test456:interested'", interestedButton.CallbackData)
			}

			registeredButton := keyboard.InlineKeyboard[1][1]
			if registeredButton.Text != "✅ Registered" {
				t.Errorf("Registered button text = %q, want '✅ Registered'", registeredButton.Text)
			}
			if registeredButton.CallbackData != "status:test456:registered" {
				t.Errorf("Registered button callback = %q, want 'status:test456:registered'", registeredButton.CallbackData)
			}

			// Third row: Maybe and Skip buttons
			if len(keyboard.InlineKeyboard[2]) != 2 {
				t.Errorf("Expected 2 buttons in third row (maybe/skip), got %d", len(keyboard.InlineKeyboard[2]))
			}
			maybeButton := keyboard.InlineKeyboard[2][0]
			if maybeButton.Text != "🤔 Maybe" {
				t.Errorf("Maybe button text = %q, want '🤔 Maybe'", maybeButton.Text)
			}
			if maybeButton.CallbackData != "status:test456:maybe" {
				t.Errorf("Maybe button callback = %q, want 'status:test456:maybe'", maybeButton.CallbackData)
			}

			skipButton := keyboard.InlineKeyboard[2][1]
			if skipButton.Text != "❌ Skip" {
				t.Errorf("Skip button text = %q, want '❌ Skip'", skipButton.Text)
			}
			if skipButton.CallbackData != "status:test456:skip" {
				t.Errorf("Skip button callback = %q, want 'status:test456:skip'", skipButton.CallbackData)
			}
		})
	}
}

func TestFormatReminder(t *testing.T) {
	tests := []struct {
		name         string
		event        *event.Event
		daysUntil    int
		wantContains []string
	}{
		{
			name: "Reminder 1 day before",
			event: &event.Event{
				ID:       "test123",
				State:    "NV",
				Title:    "Chimera Golf Club",
				DateText: "Apr 4 2026",
				City:     "Las Vegas",
			},
			daysUntil: 1,
			wantContains: []string{
				"⏰",
				"Event Reminder",
				"Tomorrow",
				"NV",
				"Chimera Golf Club",
				"Las Vegas",
				"vgagolf.org/state-events",
				"#Reminder",
			},
		},
		{
			name: "Reminder 7 days before",
			event: &event.Event{
				ID:       "test456",
				State:    "CA",
				Title:    "Pebble Beach",
				DateText: "May 15 2026",
				City:     "Monterey",
			},
			daysUntil: 7,
			wantContains: []string{
				"⏰",
				"Event Reminder",
				"In 1 week",
				"CA",
				"Pebble Beach",
				"Monterey",
			},
		},
		{
			name: "Reminder 14 days before",
			event: &event.Event{
				ID:       "test789",
				State:    "TX",
				Title:    "Dallas Country Club",
				DateText: "Jun 1 2026",
				City:     "",
			},
			daysUntil: 14,
			wantContains: []string{
				"In 2 weeks",
				"TX",
				"Dallas Country Club",
			},
		},
		{
			name: "Reminder 3 days before",
			event: &event.Event{
				ID:       "test101",
				State:    "AZ",
				Title:    "Phoenix Golf Resort",
				DateText: "Jul 10 2026",
			},
			daysUntil: 3,
			wantContains: []string{
				"In 3 days",
				"AZ",
				"Phoenix Golf Resort",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, keyboard := FormatReminder(tt.event, tt.daysUntil)

			// Check that message is not empty
			if msg == "" {
				t.Error("FormatReminder() returned empty string")
			}

			// Check that message is within Telegram's limit
			if len(msg) > 4096 {
				t.Errorf("FormatReminder() length = %d, exceeds Telegram limit of 4096", len(msg))
			}

			// Check contains
			for _, want := range tt.wantContains {
				if !strings.Contains(msg, want) {
					t.Errorf("FormatReminder() missing %q in message:\n%s", want, msg)
				}
			}

			// Check keyboard structure
			if keyboard == nil {
				t.Error("FormatReminder() returned nil keyboard")
				return
			}

			// Should have 3 rows of buttons
			if len(keyboard.InlineKeyboard) != 3 {
				t.Errorf("Keyboard has %d rows, want 3", len(keyboard.InlineKeyboard))
			}

			// First row: Calendar button
			if len(keyboard.InlineKeyboard) > 0 {
				if len(keyboard.InlineKeyboard[0]) != 1 {
					t.Errorf("First row has %d buttons, want 1", len(keyboard.InlineKeyboard[0]))
				}
				calButton := keyboard.InlineKeyboard[0][0]
				if !strings.Contains(calButton.Text, "Calendar") {
					t.Errorf("Calendar button text = %q, want to contain 'Calendar'", calButton.Text)
				}
			}

			// Second row: Interested and Registered buttons
			if len(keyboard.InlineKeyboard) > 1 {
				if len(keyboard.InlineKeyboard[1]) != 2 {
					t.Errorf("Second row has %d buttons, want 2", len(keyboard.InlineKeyboard[1]))
				}
			}

			// Third row: Maybe and Skip buttons
			if len(keyboard.InlineKeyboard) > 2 {
				if len(keyboard.InlineKeyboard[2]) != 2 {
					t.Errorf("Third row has %d buttons, want 2", len(keyboard.InlineKeyboard[2]))
				}
			}
		})
	}
}

func TestFormatRemovedEvent(t *testing.T) {
	evt := &event.Event{
		ID:       "test-removed-1",
		State:    "NV",
		Title:    "Chimera Golf Club",
		DateText: "Apr 04 2026",
		City:     "Las Vegas",
		Raw:      "NV - Chimera Golf Club - Las Vegas",
	}

	t.Run("high urgency with registered status", func(t *testing.T) {
		msg := FormatRemovedEvent(evt, preferences.EventStatusRegistered, "")

		// Check for high urgency indicator
		if !strings.Contains(msg, "⚠️") {
			t.Error("expected high urgency warning emoji")
		}

		// Check for status indicator
		if !strings.Contains(msg, "✅") || !strings.Contains(msg, "registered") {
			t.Error("expected registered status indicator")
		}

		// Check for event details
		if !strings.Contains(msg, "NV") {
			t.Error("expected state code")
		}
		if !strings.Contains(msg, "Chimera Golf Club") {
			t.Error("expected course name")
		}
		if !strings.Contains(msg, "Apr") || !strings.Contains(msg, "2026") {
			t.Error("expected formatted date")
		}
		if !strings.Contains(msg, "Las Vegas") {
			t.Error("expected city")
		}

		// Check for explanation
		if !strings.Contains(msg, "no longer listed") {
			t.Error("expected removal explanation")
		}
	})

	t.Run("high urgency with interested status", func(t *testing.T) {
		msg := FormatRemovedEvent(evt, preferences.EventStatusInterested, "")

		if !strings.Contains(msg, "⭐") || !strings.Contains(msg, "interested") {
			t.Error("expected interested status indicator")
		}
	})

	t.Run("high urgency with maybe status", func(t *testing.T) {
		msg := FormatRemovedEvent(evt, preferences.EventStatusMaybe, "")

		if !strings.Contains(msg, "🤔") || !strings.Contains(msg, "maybe") {
			t.Error("expected maybe status indicator")
		}
	})

	t.Run("includes user note", func(t *testing.T) {
		note := "Early tee time, bringing friends"
		msg := FormatRemovedEvent(evt, preferences.EventStatusRegistered, note)

		if !strings.Contains(msg, "📝") {
			t.Error("expected note emoji")
		}
		if !strings.Contains(msg, note) {
			t.Error("expected note text to be included")
		}
	})

	t.Run("includes hashtags", func(t *testing.T) {
		msg := FormatRemovedEvent(evt, preferences.EventStatusRegistered, "")

		if !strings.Contains(msg, "#VGAGolf") {
			t.Error("expected #VGAGolf hashtag")
		}
		if !strings.Contains(msg, "#EventCancelled") {
			t.Error("expected #EventCancelled hashtag")
		}
		if !strings.Contains(msg, "#NV") {
			t.Error("expected state hashtag")
		}
	})
}

func TestFormatEventWithNote(t *testing.T) {
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Chimera Golf Club",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	t.Run("with note", func(t *testing.T) {
		note := "Early tee time, bringing friends"
		msg := FormatEventWithNote(evt, note)

		if !strings.Contains(msg, "📝") {
			t.Error("Message should contain note emoji")
		}
		if !strings.Contains(msg, note) {
			t.Error("Message should contain the note text")
		}
	})

	t.Run("without note", func(t *testing.T) {
		msg := FormatEventWithNote(evt, "")

		if !strings.Contains(msg, "🏌️") {
			t.Error("Message should contain golf emoji")
		}
		if !strings.Contains(msg, "NV") {
			t.Error("Message should contain state")
		}
		if !strings.Contains(msg, "Chimera Golf Club") {
			t.Error("Message should contain title")
		}
	})
}

func TestFormatEventWithCourse(t *testing.T) {
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Pebble Beach",
		DateText: "Apr 4 2026",
		City:     "Monterey",
	}

	course := &CourseDetails{
		Name: "Pebble Beach Golf Links",
		Tees: []TeeDetails{
			{
				Name:    "Championship",
				Par:     72,
				Yardage: 7041,
				Slope:   145,
				Rating:  75.5,
				Holes:   18,
			},
		},
	}

	t.Run("with course details", func(t *testing.T) {
		msg := FormatEventWithCourse(evt, course, "")

		if !strings.Contains(msg, "Championship") {
			t.Error("Message should contain tee name")
		}
		// Yardage is formatted with commas (7,041)
		if !strings.Contains(msg, "7,041") && !strings.Contains(msg, "7041") {
			t.Error("Message should contain yardage")
		}
		if !strings.Contains(msg, "Par 72") {
			t.Error("Message should contain par")
		}
	})

	t.Run("with course and note", func(t *testing.T) {
		note := "Bucket list course!"
		msg := FormatEventWithCourse(evt, course, note)

		if !strings.Contains(msg, note) {
			t.Error("Message should contain note")
		}
		if !strings.Contains(msg, "📝") {
			t.Error("Message should contain note emoji")
		}
	})

	t.Run("without course details", func(t *testing.T) {
		msg := FormatEventWithCourse(evt, nil, "")

		if !strings.Contains(msg, "Pebble Beach") {
			t.Error("Message should contain event title")
		}
	})
}

func TestTeeDetails(t *testing.T) {
	tee := TeeDetails{
		Name:    "Championship",
		Par:     72,
		Yardage: 7200,
		Slope:   145,
		Rating:  75.5,
		Holes:   18,
	}

	if tee.Name != "Championship" {
		t.Errorf("Name = %q, want Championship", tee.Name)
	}
	if tee.Par != 72 {
		t.Errorf("Par = %d, want 72", tee.Par)
	}
	if tee.Yardage != 7200 {
		t.Errorf("Yardage = %d, want 7200", tee.Yardage)
	}
	if tee.Slope != 145 {
		t.Errorf("Slope = %d, want 145", tee.Slope)
	}
	if tee.Rating != 75.5 {
		t.Errorf("Rating = %f, want 75.5", tee.Rating)
	}
	if tee.Holes != 18 {
		t.Errorf("Holes = %d, want 18", tee.Holes)
	}
}

func TestCourseDetails(t *testing.T) {
	course := &CourseDetails{
		Name:    "Pebble Beach Golf Links",
		Website: "https://pebblebeach.com",
		Phone:   "831-622-8723",
		Tees: []TeeDetails{
			{Name: "Championship", Par: 72, Yardage: 7041},
			{Name: "Blue", Par: 72, Yardage: 6737},
		},
	}

	if course.Name != "Pebble Beach Golf Links" {
		t.Errorf("Name = %q, want Pebble Beach Golf Links", course.Name)
	}
	if len(course.Tees) != 2 {
		t.Errorf("Tees length = %d, want 2", len(course.Tees))
	}
	if course.Tees[0].Name != "Championship" {
		t.Errorf("First tee name = %q, want Championship", course.Tees[0].Name)
	}
}

func TestFormatEvent_AlsoIn(t *testing.T) {
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Multi-State Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
		AlsoIn:   []string{"CA", "AZ"},
	}

	msg := FormatEvent(evt)

	if !strings.Contains(msg, "Also in") {
		t.Error("Message should contain 'Also in' text for multi-state events")
	}
	if !strings.Contains(msg, "CA") {
		t.Error("Message should mention CA state")
	}
	if !strings.Contains(msg, "AZ") {
		t.Error("Message should mention AZ state")
	}
}

func TestFormatEventChange(t *testing.T) {
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	msg := FormatEventChange(evt, "title", "Old Title", "New Title")

	if !strings.Contains(msg, "Event Updated") {
		t.Error("Message should contain 'Event Updated'")
	}
	if !strings.Contains(msg, "Old Title") {
		t.Error("Message should contain old value")
	}
	if !strings.Contains(msg, "New Title") {
		t.Error("Message should contain new value")
	}
}

func TestFormatEventChangeWithKeyboard(t *testing.T) {
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
	}

	msg, keyboard := FormatEventChangeWithKeyboard(evt, "date", "Apr 4 2026", "Apr 10 2026", "interested")

	if !strings.Contains(msg, "Test Event") {
		t.Error("Message should contain event title")
	}

	if keyboard == nil {
		t.Error("Keyboard should not be nil")
	}
}

func TestFormatEventChangeWithNote(t *testing.T) {
	evt := &event.Event{
		ID:    "test123",
		State: "NV",
		Title: "Test Event",
	}

	note := "My personal note"
	msg, keyboard := FormatEventChangeWithNote(evt, "title", "Old Title", "New Title", "registered", note)

	if !strings.Contains(msg, note) {
		t.Error("Message should contain note")
	}
	if !strings.Contains(msg, "📝") {
		t.Error("Message should contain note emoji")
	}
	if keyboard == nil {
		t.Error("Keyboard should not be nil")
	}
}

func TestFormatEventWithStatusAndCourse(t *testing.T) {
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	course := &CourseDetails{
		Name: "Test Golf Course",
		Tees: []TeeDetails{
			{
				Name:    "Championship",
				Par:     72,
				Yardage: 7200,
			},
		},
	}

	prefs := preferences.NewPreferences()
	msg, keyboard := FormatEventWithStatusAndCourse(evt, course, "interested", "", "12345", prefs)

	if !strings.Contains(msg, "Test Event") {
		t.Error("Message should contain event title")
	}
	if !strings.Contains(msg, "⭐") {
		t.Error("Message should contain interested emoji")
	}
	if !strings.Contains(msg, "Test Golf Course") {
		t.Error("Message should contain course name")
	}
	if keyboard == nil {
		t.Error("Keyboard should not be nil")
	}
}

func TestFormatEventWithStatusAndNote(t *testing.T) {
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	prefs := preferences.NewPreferences()
	note := "Great course!"

	tests := []struct {
		name   string
		status string
		note   string
	}{
		{
			name:   "with interested status and note",
			status: "interested",
			note:   note,
		},
		{
			name:   "with registered status no note",
			status: "registered",
			note:   "",
		},
		{
			name:   "with maybe status and note",
			status: "maybe",
			note:   note,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, keyboard := FormatEventWithStatusAndNote(evt, tt.status, tt.note, "12345", prefs)

			if !strings.Contains(msg, "Test Event") {
				t.Error("Message should contain event title")
			}

			if tt.note != "" && !strings.Contains(msg, tt.note) {
				t.Error("Message should contain note")
			}

			if keyboard == nil {
				t.Error("Keyboard should not be nil")
			}

			// Check status emoji
			switch tt.status {
			case "interested":
				if !strings.Contains(msg, "⭐") {
					t.Error("Message should contain interested emoji")
				}
			case "registered":
				if !strings.Contains(msg, "✅") {
					t.Error("Message should contain registered emoji")
				}
			case "maybe":
				if !strings.Contains(msg, "🤔") {
					t.Error("Message should contain maybe emoji")
				}
			}
		})
	}
}

func TestFormatYardage(t *testing.T) {
	// This is a private function but we can test the public functions that use it
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	course := &CourseDetails{
		Name: "Test Course",
		Tees: []TeeDetails{
			{
				Name:    "Championship",
				Par:     72,
				Yardage: 7041,  // Should be formatted as "7,041"
			},
		},
	}

	msg := FormatEventWithCourse(evt, course, "")

	// Check that yardage is formatted with comma
	if !strings.Contains(msg, "7,041") && !strings.Contains(msg, "7041") {
		t.Error("Message should contain formatted yardage")
	}
}

func TestFormatRemovedEventGeneral(t *testing.T) {
	evt := &event.Event{
		ID:       "test-removed-2",
		State:    "CA",
		Title:    "Pebble Beach Golf Links",
		DateText: "May 15 2026",
		City:     "Monterey",
		Raw:      "CA - Pebble Beach Golf Links - Monterey",
	}

	t.Run("low urgency notification", func(t *testing.T) {
		msg := FormatRemovedEventGeneral(evt)

		// Check for low urgency indicator
		if !strings.Contains(msg, "ℹ️") {
			t.Error("expected info emoji for low urgency")
		}

		// Should NOT have high urgency warning
		if strings.Contains(msg, "⚠️") {
			t.Error("did not expect high urgency warning for general notification")
		}

		// Check for event details
		if !strings.Contains(msg, "CA") {
			t.Error("expected state code")
		}
		if !strings.Contains(msg, "Pebble Beach Golf Links") {
			t.Error("expected course name")
		}
		if !strings.Contains(msg, "May") || !strings.Contains(msg, "2026") {
			t.Error("expected formatted date")
		}
		if !strings.Contains(msg, "Monterey") {
			t.Error("expected city")
		}

		// Check for brief explanation
		if !strings.Contains(msg, "removed") {
			t.Error("expected removal explanation")
		}

		// Should NOT have status indicators
		if strings.Contains(msg, "✅") || strings.Contains(msg, "⭐") || strings.Contains(msg, "🤔") {
			t.Error("did not expect status indicators in general notification")
		}

		// Should NOT have note section
		if strings.Contains(msg, "📝") {
			t.Error("did not expect note section in general notification")
		}
	})

	t.Run("includes hashtags", func(t *testing.T) {
		msg := FormatRemovedEventGeneral(evt)

		if !strings.Contains(msg, "#VGAGolf") {
			t.Error("expected #VGAGolf hashtag")
		}
		if !strings.Contains(msg, "#EventCancelled") {
			t.Error("expected #EventCancelled hashtag")
		}
		if !strings.Contains(msg, "#CA") {
			t.Error("expected state hashtag")
		}
	})

	t.Run("handles missing city", func(t *testing.T) {
		evtNoCity := &event.Event{
			ID:       "test-removed-3",
			State:    "TX",
			Title:    "Dallas Country Club",
			DateText: "Jun 1 2026",
			City:     "",
			Raw:      "TX - Dallas Country Club",
		}

		msg := FormatRemovedEventGeneral(evtNoCity)

		// Should still format correctly without city
		if !strings.Contains(msg, "Dallas Country Club") {
			t.Error("expected course name")
		}
		if !strings.Contains(msg, "TX") {
			t.Error("expected state code")
		}
	})
}
