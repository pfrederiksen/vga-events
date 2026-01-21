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
