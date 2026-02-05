package calendar

import (
	"strings"
	"testing"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

func TestGenerateICS(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event-123",
		State:    "NV",
		Title:    "Spring Championship",
		DateText: "Mar 15 2026",
		City:     "Las Vegas",
	}

	ics := GenerateICS(evt)

	// Check required ICS fields
	requiredFields := []string{
		"BEGIN:VCALENDAR",
		"VERSION:2.0",
		"PRODID:-//VGA Events//vga-events//EN",
		"BEGIN:VEVENT",
		"UID:test-event-123@vgagolf.org",
		"DTSTAMP:",
		"DTSTART:",
		"DTEND:",
		"SUMMARY:VGA Golf - Spring Championship",
		"DESCRIPTION:",
		"LOCATION:Spring Championship\\, Las Vegas", // Comma is escaped
		"URL:https://vgagolf.org/state-events",
		"STATUS:CONFIRMED",
		"END:VEVENT",
		"END:VCALENDAR",
	}

	for _, field := range requiredFields {
		if !strings.Contains(ics, field) {
			t.Errorf("ICS missing required field: %s", field)
		}
	}

	// Check that lines end with \r\n
	if !strings.Contains(ics, "\r\n") {
		t.Error("ICS should use \\r\\n line endings")
	}
}

func TestGenerateICS_UnparseableDate(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event",
		State:    "CA",
		Title:    "Test Event",
		DateText: "Invalid Date",
		City:     "San Francisco",
	}

	ics := GenerateICS(evt)

	// Should still generate valid ICS with fallback date
	if !strings.Contains(ics, "BEGIN:VEVENT") {
		t.Error("Should generate ICS even with unparseable date")
	}

	if !strings.Contains(ics, "DTSTART:") {
		t.Error("Should include DTSTART with fallback date")
	}
}

func TestGenerateICS_SpecialCharacters(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event",
		State:    "TX",
		Title:    "Test Event; With, Special\\Characters\nAnd Newlines",
		DateText: "Apr 20 2026",
		City:     "Austin",
	}

	ics := GenerateICS(evt)

	// Check that special characters are escaped
	if strings.Contains(ics, "SUMMARY:VGA Golf - Test Event; With, Special\\Characters\nAnd Newlines") {
		t.Error("Special characters should be escaped in SUMMARY")
	}

	// Should have escaped versions
	if !strings.Contains(ics, "\\;") || !strings.Contains(ics, "\\,") || !strings.Contains(ics, "\\n") {
		t.Error("Special characters should be escaped")
	}
}

func TestGenerateBulkICS(t *testing.T) {
	events := []*event.Event{
		{
			ID:       "event1",
			State:    "NV",
			Title:    "Event 1",
			DateText: "Mar 15 2026",
			City:     "Las Vegas",
		},
		{
			ID:       "event2",
			State:    "CA",
			Title:    "Event 2",
			DateText: "Apr 20 2026",
			City:     "San Francisco",
		},
		{
			ID:       "event3",
			State:    "TX",
			Title:    "Event 3",
			DateText: "May 10 2026",
			City:     "Austin",
		},
	}

	ics := GenerateBulkICS(events, "VGA Golf Events - Test")

	// Check calendar header
	if !strings.Contains(ics, "BEGIN:VCALENDAR") {
		t.Error("Missing calendar BEGIN")
	}
	if !strings.Contains(ics, "END:VCALENDAR") {
		t.Error("Missing calendar END")
	}

	// Check calendar name
	if !strings.Contains(ics, "X-WR-CALNAME:VGA Golf Events - Test") {
		t.Error("Missing calendar name")
	}

	// Count VEVENT entries (should be 3)
	beginCount := strings.Count(ics, "BEGIN:VEVENT")
	endCount := strings.Count(ics, "END:VEVENT")

	if beginCount != 3 {
		t.Errorf("Expected 3 BEGIN:VEVENT, got %d", beginCount)
	}
	if endCount != 3 {
		t.Errorf("Expected 3 END:VEVENT, got %d", endCount)
	}

	// Check that all event UIDs are present
	for _, evt := range events {
		uid := "UID:" + evt.ID + "@vgagolf.org"
		if !strings.Contains(ics, uid) {
			t.Errorf("Missing UID for event: %s", evt.ID)
		}
	}
}

func TestGenerateBulkICS_EmptyEvents(t *testing.T) {
	ics := GenerateBulkICS([]*event.Event{}, "Test Calendar")

	if ics != "" {
		t.Error("Empty events array should return empty string")
	}
}

func TestGenerateBulkICS_SingleEvent(t *testing.T) {
	events := []*event.Event{
		{
			ID:       "single-event",
			State:    "NV",
			Title:    "Single Event",
			DateText: "Jun 1 2026",
			City:     "Reno",
		},
	}

	ics := GenerateBulkICS(events, "Single Event Calendar")

	// Should still have proper structure
	if !strings.Contains(ics, "BEGIN:VCALENDAR") {
		t.Error("Missing calendar BEGIN")
	}
	if !strings.Contains(ics, "END:VCALENDAR") {
		t.Error("Missing calendar END")
	}

	// Should have exactly one VEVENT
	beginCount := strings.Count(ics, "BEGIN:VEVENT")
	if beginCount != 1 {
		t.Errorf("Expected 1 BEGIN:VEVENT, got %d", beginCount)
	}
}

func TestGenerateBulkICS_NoCalendarName(t *testing.T) {
	events := []*event.Event{
		{
			ID:       "event1",
			State:    "NV",
			Title:    "Event 1",
			DateText: "Mar 15 2026",
			City:     "Las Vegas",
		},
	}

	ics := GenerateBulkICS(events, "")

	// Should generate ICS without calendar name
	if !strings.Contains(ics, "BEGIN:VCALENDAR") {
		t.Error("Should generate ICS even without calendar name")
	}

	// Should not have X-WR-CALNAME if name is empty
	if strings.Contains(ics, "X-WR-CALNAME:") {
		t.Error("Should not include X-WR-CALNAME when name is empty")
	}
}

func TestFormatICSTime(t *testing.T) {
	// Test time formatting
	testTime := time.Date(2026, 3, 15, 14, 30, 0, 0, time.UTC)
	formatted := formatICSTime(testTime)

	expected := "20260315T143000Z"
	if formatted != expected {
		t.Errorf("formatICSTime() = %q, want %q", formatted, expected)
	}
}

func TestEscapeICS(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Simple text", "Simple text"},
		{"Text with, comma", "Text with\\, comma"},
		{"Text with; semicolon", "Text with\\; semicolon"},
		{"Text with\\backslash", "Text with\\\\backslash"},
		{"Text with\nnewline", "Text with\\nnewline"},
		{"All, special; chars\\\n", "All\\, special\\; chars\\\\\\n"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := escapeICS(tt.input)
			if got != tt.expected {
				t.Errorf("escapeICS(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestGenerateICS_EventTimes(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Mar 15 2026",
		City:     "Las Vegas",
	}

	ics := GenerateICS(evt)

	// Event should be at 9 AM
	if !strings.Contains(ics, "T090000Z") {
		t.Error("Event should start at 9 AM UTC")
	}

	// Event should end at 1 PM (9 AM + 4 hours)
	if !strings.Contains(ics, "T130000Z") {
		t.Error("Event should end at 1 PM UTC (4 hours duration)")
	}
}

func TestGenerateMultiEventICS(t *testing.T) {
	events := []*event.Event{
		{
			ID:       "event1",
			State:    "NV",
			Title:    "Chimera Golf Club",
			DateText: "Apr 4 2026",
			City:     "Las Vegas",
		},
		{
			ID:       "event2",
			State:    "CA",
			Title:    "Pebble Beach",
			DateText: "May 15 2026",
			City:     "Monterey",
		},
		{
			ID:       "event3",
			State:    "TX",
			Title:    "Dallas Country Club",
			DateText: "Jun 1 2026",
			City:     "",
		},
	}

	ics := GenerateMultiEventICS(events)

	// Should contain calendar header
	if !strings.Contains(ics, "BEGIN:VCALENDAR") {
		t.Error("ICS should contain BEGIN:VCALENDAR")
	}

	if !strings.Contains(ics, "END:VCALENDAR") {
		t.Error("ICS should contain END:VCALENDAR")
	}

	// Should use default calendar name
	if !strings.Contains(ics, "VGA Registered Events") {
		t.Error("ICS should contain default calendar name 'VGA Registered Events'")
	}

	// Should contain all three events
	eventCount := strings.Count(ics, "BEGIN:VEVENT")
	if eventCount != 3 {
		t.Errorf("ICS contains %d events, want 3", eventCount)
	}

	// Check each event is present
	if !strings.Contains(ics, "UID:event1@vgagolf.org") {
		t.Error("ICS should contain event1")
	}
	if !strings.Contains(ics, "UID:event2@vgagolf.org") {
		t.Error("ICS should contain event2")
	}
	if !strings.Contains(ics, "UID:event3@vgagolf.org") {
		t.Error("ICS should contain event3")
	}

	// Check event details
	if !strings.Contains(ics, "Chimera Golf Club") {
		t.Error("ICS should contain first event title")
	}
	if !strings.Contains(ics, "Pebble Beach") {
		t.Error("ICS should contain second event title")
	}
	if !strings.Contains(ics, "Dallas Country Club") {
		t.Error("ICS should contain third event title")
	}
}

func TestGenerateMultiEventICS_EmptyEvents(t *testing.T) {
	ics := GenerateMultiEventICS([]*event.Event{})

	// Should return empty string for no events
	if ics != "" {
		t.Errorf("GenerateMultiEventICS([]) = %q, want empty string", ics)
	}
}

func TestGenerateBulkICS_WithCustomName(t *testing.T) {
	events := []*event.Event{
		{
			ID:       "event1",
			State:    "NV",
			Title:    "Test Event",
			DateText: "Apr 4 2026",
		},
	}

	ics := GenerateBulkICS(events, "My Custom Calendar")

	// Should contain custom calendar name
	if !strings.Contains(ics, "My Custom Calendar") {
		t.Error("ICS should contain custom calendar name 'My Custom Calendar'")
	}

	// Should still be valid ICS format
	if !strings.Contains(ics, "BEGIN:VCALENDAR") {
		t.Error("ICS should contain BEGIN:VCALENDAR")
	}

	if !strings.Contains(ics, "END:VCALENDAR") {
		t.Error("ICS should contain END:VCALENDAR")
	}
}

// TestGenerateICSWithOptions_Alarm tests that alarms are added correctly
func TestGenerateICSWithOptions_Alarm(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	opts := &EventOptions{
		ReminderBefore: 24 * time.Hour, // 1 day before
	}

	ics := GenerateICSWithOptions(evt, opts)

	// Should contain VALARM component
	if !strings.Contains(ics, "BEGIN:VALARM") {
		t.Error("ICS should contain BEGIN:VALARM")
	}
	if !strings.Contains(ics, "END:VALARM") {
		t.Error("ICS should contain END:VALARM")
	}
	if !strings.Contains(ics, "ACTION:DISPLAY") {
		t.Error("ICS should contain ACTION:DISPLAY")
	}
	if !strings.Contains(ics, "TRIGGER:-PT24H") {
		t.Error("ICS should contain TRIGGER:-PT24H for 24 hour reminder")
	}
}

// TestGenerateICSWithOptions_Status tests status-based properties
func TestGenerateICSWithOptions_Status(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	tests := []struct {
		status       string
		wantEmoji    string
		wantCategory string
		wantColor    string
	}{
		{"registered", "✅", "CATEGORIES:Registered", "COLOR:green"},
		{"interested", "⭐", "CATEGORIES:Interested", "COLOR:yellow"},
		{"maybe", "🤔", "CATEGORIES:Maybe", "COLOR:gray"},
		{"skip", "❌", "CATEGORIES:Skipped", "COLOR:black"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			opts := &EventOptions{
				Status: tt.status,
			}

			ics := GenerateICSWithOptions(evt, opts)

			// Check emoji in summary
			if !strings.Contains(ics, tt.wantEmoji) {
				t.Errorf("ICS should contain emoji %s for status %s", tt.wantEmoji, tt.status)
			}

			// Check category
			if !strings.Contains(ics, tt.wantCategory) {
				t.Errorf("ICS should contain %s", tt.wantCategory)
			}

			// Check color
			if !strings.Contains(ics, tt.wantColor) {
				t.Errorf("ICS should contain %s", tt.wantColor)
			}
		})
	}
}

// TestGenerateICSWithOptions_CourseDetails tests course details in description
func TestGenerateICSWithOptions_CourseDetails(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	opts := &EventOptions{
		CourseDetails: &CourseInfo{
			Name: "Chimera Golf Club",
			Tees: []TeeInfo{
				{Name: "Championship", Par: 72, Yardage: 7041},
				{Name: "Blue", Par: 72, Yardage: 6500},
			},
		},
	}

	ics := GenerateICSWithOptions(evt, opts)

	// Should contain course name in description
	if !strings.Contains(ics, "Chimera Golf Club") {
		t.Error("ICS description should contain course name")
	}

	// Should contain tee details
	if !strings.Contains(ics, "Championship") {
		t.Error("ICS description should contain tee name")
	}
	if !strings.Contains(ics, "Par 72") {
		t.Error("ICS description should contain par")
	}
	if !strings.Contains(ics, "7041 yds") {
		t.Error("ICS description should contain yardage")
	}
}

// TestGenerateICSWithOptions_Note tests user notes in description
func TestGenerateICSWithOptions_Note(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	opts := &EventOptions{
		Note: "Bring extra balls",
	}

	ics := GenerateICSWithOptions(evt, opts)

	// Should contain note in description
	if !strings.Contains(ics, "Note: Bring extra balls") {
		t.Error("ICS description should contain user note")
	}
}

// TestGenerateBulkICSWithOptions tests bulk export with per-event options
func TestGenerateBulkICSWithOptions(t *testing.T) {
	events := []*event.Event{
		{
			ID:       "event1",
			State:    "NV",
			Title:    "Event 1",
			DateText: "Apr 4 2026",
			City:     "Las Vegas",
		},
		{
			ID:       "event2",
			State:    "CA",
			Title:    "Event 2",
			DateText: "May 15 2026",
			City:     "Monterey",
		},
	}

	optsMap := map[string]*EventOptions{
		"event1": {
			Status: "registered",
			Note:   "First event",
		},
		"event2": {
			Status: "interested",
		},
	}

	ics := GenerateBulkICSWithOptions(events, "Test Calendar", optsMap)

	// Should contain both events
	if !strings.Contains(ics, "UID:event1@vgagolf.org") {
		t.Error("ICS should contain event1")
	}
	if !strings.Contains(ics, "UID:event2@vgagolf.org") {
		t.Error("ICS should contain event2")
	}

	// Should contain status-specific properties
	if !strings.Contains(ics, "✅") {
		t.Error("ICS should contain registered emoji for event1")
	}
	if !strings.Contains(ics, "⭐") {
		t.Error("ICS should contain interested emoji for event2")
	}

	// Should contain note
	if !strings.Contains(ics, "Note: First event") {
		t.Error("ICS should contain note for event1")
	}

	// Should contain alarms (default 24h)
	alarmCount := strings.Count(ics, "BEGIN:VALARM")
	if alarmCount != 2 {
		t.Errorf("ICS should contain 2 alarms, got %d", alarmCount)
	}
}

// TestGetStatusEmoji tests status emoji mapping
func TestGetStatusEmoji(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"registered", "✅"},
		{"interested", "⭐"},
		{"maybe", "🤔"},
		{"skip", "❌"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := getStatusEmoji(tt.status)
			if got != tt.want {
				t.Errorf("getStatusEmoji(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

// TestGetStatusCategory tests status category mapping
func TestGetStatusCategory(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"registered", "Registered"},
		{"interested", "Interested"},
		{"maybe", "Maybe"},
		{"skip", "Skipped"},
		{"unknown", "VGA Golf"},
		{"", "VGA Golf"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := getStatusCategory(tt.status)
			if got != tt.want {
				t.Errorf("getStatusCategory(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

// TestGetStatusColor tests status color mapping
func TestGetStatusColor(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"registered", "green"},
		{"interested", "yellow"},
		{"maybe", "gray"},
		{"skip", "black"},
		{"unknown", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := getStatusColor(tt.status)
			if got != tt.want {
				t.Errorf("getStatusColor(%q) = %q, want %q", tt.status, got, tt.want)
			}
		})
	}
}

// TestGenerateICSWithOptions_DefaultAlarm tests default alarm behavior
func TestGenerateICSWithOptions_DefaultAlarm(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	// No options provided - should still get default alarm
	ics := GenerateICS(evt)

	// Should contain VALARM with default 24h reminder
	if !strings.Contains(ics, "BEGIN:VALARM") {
		t.Error("ICS should contain BEGIN:VALARM with default options")
	}
	if !strings.Contains(ics, "TRIGGER:-PT24H") {
		t.Error("ICS should contain default TRIGGER:-PT24H")
	}
}

// TestGenerateICSWithOptions_CustomReminder tests custom reminder timing
func TestGenerateICSWithOptions_CustomReminder(t *testing.T) {
	evt := &event.Event{
		ID:       "test-event",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	tests := []struct {
		name           string
		reminderBefore time.Duration
		wantTrigger    string
	}{
		{"1 day", 24 * time.Hour, "-PT24H"},
		{"2 days", 48 * time.Hour, "-PT48H"},
		{"1 week", 168 * time.Hour, "-PT168H"},
		{"12 hours", 12 * time.Hour, "-PT12H"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := &EventOptions{
				ReminderBefore: tt.reminderBefore,
			}

			ics := GenerateICSWithOptions(evt, opts)

			if !strings.Contains(ics, "TRIGGER:"+tt.wantTrigger) {
				t.Errorf("ICS should contain TRIGGER:%s", tt.wantTrigger)
			}
		})
	}
}
