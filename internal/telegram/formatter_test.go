package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
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
				"Apr 04 2026",
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
				"May 15 2026",
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
