package event

import (
	"strings"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name      string
		dateText  string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantZero  bool
	}{
		{
			name:      "Full date Mar 13 2026",
			dateText:  "Mar 13 2026",
			wantYear:  2026,
			wantMonth: time.March,
			wantDay:   13,
		},
		{
			name:      "Full date single digit day Jan 2 2026",
			dateText:  "Jan 2 2026",
			wantYear:  2026,
			wantMonth: time.January,
			wantDay:   2,
		},
		{
			name:      "Dot format 4.4.26",
			dateText:  "4.4.26",
			wantYear:  2026,
			wantMonth: time.April,
			wantDay:   4,
		},
		{
			name:      "Dot format with leading zeros 04.04.26",
			dateText:  "04.04.26",
			wantYear:  2026,
			wantMonth: time.April,
			wantDay:   4,
		},
		{
			name:      "Slash format 02/15/26",
			dateText:  "02/15/26",
			wantYear:  2026,
			wantMonth: time.February,
			wantDay:   15,
		},
		{
			name:      "Month and day only Jan 24",
			dateText:  "Jan 24",
			wantYear:  time.Now().Year(),
			wantMonth: time.January,
			wantDay:   24,
		},
		{
			name:      "Month and single digit day Jan 2",
			dateText:  "Jan 2",
			wantYear:  time.Now().Year(),
			wantMonth: time.January,
			wantDay:   2,
		},
		{
			name:     "Empty string",
			dateText: "",
			wantZero: true,
		},
		{
			name:     "Invalid format",
			dateText: "Not a date",
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDate(tt.dateText)

			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("ParseDate(%q) = %v, want zero time", tt.dateText, got)
				}
				return
			}

			if got.Year() != tt.wantYear {
				t.Errorf("ParseDate(%q).Year() = %d, want %d", tt.dateText, got.Year(), tt.wantYear)
			}
			if got.Month() != tt.wantMonth {
				t.Errorf("ParseDate(%q).Month() = %v, want %v", tt.dateText, got.Month(), tt.wantMonth)
			}
			if got.Day() != tt.wantDay {
				t.Errorf("ParseDate(%q).Day() = %d, want %d", tt.dateText, got.Day(), tt.wantDay)
			}
		})
	}
}

// testBoolMethod is a helper for testing methods that return bool
func testBoolMethod(t *testing.T, methodName string, tests []struct {
	name     string
	dateText string
	want     bool
}, fn func(*Event) bool) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &Event{DateText: tt.dateText}
			if got := fn(evt); got != tt.want {
				t.Errorf("Event.%s() = %v, want %v", methodName, got, tt.want)
			}
		})
	}
}

func TestEvent_IsPastEvent(t *testing.T) {
	tests := []struct {
		name     string
		dateText string
		want     bool
	}{
		{
			name:     "Past date",
			dateText: "Jan 1 2020",
			want:     true,
		},
		{
			name:     "Future date",
			dateText: "Dec 31 2099",
			want:     false,
		},
		{
			name:     "Unparseable date",
			dateText: "invalid",
			want:     false, // Safe default: don't filter
		},
		{
			name:     "Empty date",
			dateText: "",
			want:     false, // Safe default: don't filter
		},
	}

	testBoolMethod(t, "IsPastEvent", tests, (*Event).IsPastEvent)
}

func TestEvent_IsWithinDays(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1).Format("Jan 2 2006")
	nextWeek := time.Now().AddDate(0, 0, 7).Format("Jan 2 2006")
	nextMonth := time.Now().AddDate(0, 0, 35).Format("Jan 2 2006")

	tests := []struct {
		name     string
		dateText string
		days     int
		want     bool
	}{
		{
			name:     "Within 30 days - tomorrow",
			dateText: tomorrow,
			days:     30,
			want:     true,
		},
		{
			name:     "Within 30 days - next week",
			dateText: nextWeek,
			days:     30,
			want:     true,
		},
		{
			name:     "Beyond 30 days - next month",
			dateText: nextMonth,
			days:     30,
			want:     false,
		},
		{
			name:     "Feature disabled (days=0)",
			dateText: nextMonth,
			days:     0,
			want:     true, // Always include when disabled
		},
		{
			name:     "Feature disabled (days negative)",
			dateText: nextMonth,
			days:     -1,
			want:     true, // Always include when disabled
		},
		{
			name:     "Unparseable date",
			dateText: "invalid",
			days:     30,
			want:     true, // Safe default: include
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &Event{DateText: tt.dateText}
			if got := evt.IsWithinDays(tt.days); got != tt.want {
				t.Errorf("Event.IsWithinDays(%d) = %v, want %v", tt.days, got, tt.want)
			}
		})
	}
}

func TestEvent_IsUpcoming(t *testing.T) {
	tests := []struct {
		name     string
		dateText string
		want     bool
	}{
		{
			name:     "Future date",
			dateText: "Dec 31 2099",
			want:     true,
		},
		{
			name:     "Past date",
			dateText: "Jan 1 2020",
			want:     false,
		},
		{
			name:     "Unparseable date",
			dateText: "invalid",
			want:     true, // Safe default: include
		},
		{
			name:     "Empty date",
			dateText: "",
			want:     true, // Safe default: include
		},
	}

	testBoolMethod(t, "IsUpcoming", tests, (*Event).IsUpcoming)
}

func TestSortByDate(t *testing.T) {
	tests := []struct {
		name      string
		events    []*Event
		wantOrder []string // Expected order of DateText values
	}{
		{
			name:      "Empty slice",
			events:    []*Event{},
			wantOrder: []string{},
		},
		{
			name: "Single event",
			events: []*Event{
				{DateText: "Mar 15 2026"},
			},
			wantOrder: []string{"Mar 15 2026"},
		},
		{
			name: "Already sorted",
			events: []*Event{
				{DateText: "Jan 1 2026"},
				{DateText: "Feb 1 2026"},
				{DateText: "Mar 1 2026"},
			},
			wantOrder: []string{"Jan 1 2026", "Feb 1 2026", "Mar 1 2026"},
		},
		{
			name: "Reverse order",
			events: []*Event{
				{DateText: "Dec 31 2026"},
				{DateText: "Jun 15 2026"},
				{DateText: "Jan 1 2026"},
			},
			wantOrder: []string{"Jan 1 2026", "Jun 15 2026", "Dec 31 2026"},
		},
		{
			name: "Mixed formats",
			events: []*Event{
				{DateText: "4.4.26"},     // Apr 4, 2026
				{DateText: "Jan 1 2026"}, // Jan 1, 2026
				{DateText: "02/15/26"},   // Feb 15, 2026
			},
			wantOrder: []string{"Jan 1 2026", "02/15/26", "4.4.26"},
		},
		{
			name: "Unparseable dates at end",
			events: []*Event{
				{DateText: "Mar 1 2026"},
				{DateText: "not a date"},
				{DateText: "Jan 1 2026"},
				{DateText: "invalid"},
			},
			wantOrder: []string{"Jan 1 2026", "Mar 1 2026", "not a date", "invalid"},
		},
		{
			name: "All unparseable dates",
			events: []*Event{
				{DateText: "invalid1"},
				{DateText: "invalid2"},
				{DateText: "invalid3"},
			},
			wantOrder: []string{"invalid1", "invalid2", "invalid3"}, // Maintain original order
		},
		{
			name: "Empty dates mixed with valid",
			events: []*Event{
				{DateText: ""},
				{DateText: "Mar 1 2026"},
				{DateText: "Jan 1 2026"},
				{DateText: ""},
			},
			wantOrder: []string{"Jan 1 2026", "Mar 1 2026", "", ""},
		},
		{
			name: "Same dates",
			events: []*Event{
				{DateText: "Mar 15 2026"},
				{DateText: "Jan 1 2026"},
				{DateText: "Mar 15 2026"},
			},
			wantOrder: []string{"Jan 1 2026", "Mar 15 2026", "Mar 15 2026"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a copy to avoid modifying the test data
			events := make([]*Event, len(tt.events))
			copy(events, tt.events)

			// Sort the events
			SortByDate(events)

			// Check the order
			if len(events) != len(tt.wantOrder) {
				t.Fatalf("SortByDate() resulted in %d events, want %d", len(events), len(tt.wantOrder))
			}

			for i, event := range events {
				if event.DateText != tt.wantOrder[i] {
					t.Errorf("SortByDate() at position %d = %q, want %q", i, event.DateText, tt.wantOrder[i])
				}
			}
		})
	}
}

func TestFormatDateNice(t *testing.T) {
	tests := []struct {
		name         string
		dateText     string
		wantContains []string // Strings that should be in the result
	}{
		{
			name:     "Empty date",
			dateText: "",
			wantContains: []string{
				"", // Should return empty string
			},
		},
		{
			name:     "Unparseable date",
			dateText: "invalid date",
			wantContains: []string{
				"invalid date", // Should return original text
			},
		},
		{
			name:     "Valid date with full year",
			dateText: "Apr 4 2026",
			wantContains: []string{
				"Apr 4, 2026", // Should include formatted date
				"2026",        // Should include year
			},
		},
		{
			name:     "Valid date with two-digit year",
			dateText: "May 15 2026",
			wantContains: []string{
				"May 15, 2026", // Should include formatted date
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDateNice(tt.dateText)

			// For empty expected result, check exact match
			if len(tt.wantContains) == 1 && tt.wantContains[0] == "" {
				if got != "" {
					t.Errorf("FormatDateNice(%q) = %q, want empty string", tt.dateText, got)
				}
				return
			}

			// For non-empty results, check that all expected strings are present
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatDateNice(%q) = %q, missing %q", tt.dateText, got, want)
				}
			}

			// Check length is reasonable (not too long for Telegram messages)
			if len(got) > 100 {
				t.Errorf("FormatDateNice(%q) length = %d, exceeds reasonable limit of 100", tt.dateText, len(got))
			}
		})
	}
}

func TestFormatDateNice_RelativeTime(t *testing.T) {
	// Test relative time indicators (today, tomorrow, in X days)
	// These tests use fixed dates to ensure predictable results

	tests := []struct {
		name         string
		daysFromNow  int // Offset from current date
		wantContains string
	}{
		{
			name:         "Event in 1 day",
			daysFromNow:  1,
			wantContains: "(tomorrow)",
		},
		{
			name:         "Event in 7 days",
			daysFromNow:  7,
			wantContains: "(in 1 week)",
		},
		{
			name:         "Event in 14 days",
			daysFromNow:  14,
			wantContains: "(in 2 weeks)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a date N days from now
			futureDate := time.Now().AddDate(0, 0, tt.daysFromNow)
			dateText := futureDate.Format("Jan 2 2006")

			got := FormatDateNice(dateText)

			if !strings.Contains(got, tt.wantContains) {
				t.Errorf("FormatDateNice(%q) = %q, want to contain %q", dateText, got, tt.wantContains)
			}
		})
	}
}
