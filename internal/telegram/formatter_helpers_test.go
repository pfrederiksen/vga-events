package telegram

import (
	"strings"
	"testing"

	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
)

// TestFormatChangeValue tests the formatChangeValue helper function
func TestFormatChangeValue(t *testing.T) {
	tests := []struct {
		name      string
		oldValue  string
		newValue  string
		fieldName string
		wantOld   string
		wantNew   string
	}{
		{
			name:      "both values present",
			oldValue:  "Old Title",
			newValue:  "New Title",
			fieldName: "title",
			wantOld:   "<s>Old Title</s>",
			wantNew:   "New Title",
		},
		{
			name:      "old value empty",
			oldValue:  "",
			newValue:  "Las Vegas",
			fieldName: "city",
			wantOld:   "<s>No city</s>",
			wantNew:   "Las Vegas",
		},
		{
			name:      "new value empty",
			oldValue:  "San Francisco",
			newValue:  "",
			fieldName: "city",
			wantOld:   "<s>San Francisco</s>",
			wantNew:   "No city",
		},
		{
			name:      "both values empty",
			oldValue:  "",
			newValue:  "",
			fieldName: "date",
			wantOld:   "<s>No date</s>",
			wantNew:   "No date",
		},
		{
			name:      "date change",
			oldValue:  "Apr 4 2026",
			newValue:  "Apr 10 2026",
			fieldName: "date",
			wantOld:   "<s>Apr 4 2026</s>",
			wantNew:   "Apr 10 2026",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg strings.Builder
			formatChangeValue(&msg, tt.oldValue, tt.newValue, tt.fieldName)
			result := msg.String()

			if !strings.Contains(result, tt.wantOld) {
				t.Errorf("formatChangeValue() missing old value display: want %q in:\n%s", tt.wantOld, result)
			}
			if !strings.Contains(result, tt.wantNew) {
				t.Errorf("formatChangeValue() missing new value display: want %q in:\n%s", tt.wantNew, result)
			}

			// Verify it contains emoji indicators
			if !strings.Contains(result, "❌") {
				t.Error("formatChangeValue() should contain ❌ emoji for old value")
			}
			if !strings.Contains(result, "✅") {
				t.Error("formatChangeValue() should contain ✅ emoji for new value")
			}
		})
	}
}

// TestFormatEventHeader tests the formatEventHeader helper function
func TestFormatEventHeader(t *testing.T) {
	tests := []struct {
		name       string
		event      *event.Event
		hasNote    bool
		wantEmojis []string
		wantText   []string
	}{
		{
			name: "event with note",
			event: &event.Event{
				State:    "NV",
				Title:    "Chimera Golf Club",
				DateText: "Apr 4 2026",
				City:     "Las Vegas",
				AlsoIn:   []string{},
			},
			hasNote:    true,
			wantEmojis: []string{"🏌️", "📝"},
			wantText: []string{
				"New VGA Golf Event!",
				"NV",
				"Chimera Golf Club",
				"Apr 4, 2026",
				"Las Vegas",
			},
		},
		{
			name: "event without note",
			event: &event.Event{
				State:    "CA",
				Title:    "Pebble Beach",
				DateText: "May 15 2026",
				City:     "Monterey",
				AlsoIn:   []string{},
			},
			hasNote:    false,
			wantEmojis: []string{"🏌️"},
			wantText: []string{
				"New VGA Golf Event!",
				"CA",
				"Pebble Beach",
				"May 15, 2026",
				"Monterey",
			},
		},
		{
			name: "event with AlsoIn states",
			event: &event.Event{
				State:    "NV",
				Title:    "Multi-State Event",
				DateText: "Jun 1 2026",
				City:     "Las Vegas",
				AlsoIn:   []string{"CA", "AZ"},
			},
			hasNote:    false,
			wantEmojis: []string{"🏌️"},
			wantText: []string{
				"New VGA Golf Event!",
				"NV",
				"Multi-State Event",
				"Also in: CA, AZ",
			},
		},
		{
			name: "event without date",
			event: &event.Event{
				State:    "TX",
				Title:    "Dallas Country Club",
				DateText: "",
				City:     "Dallas",
				AlsoIn:   []string{},
			},
			hasNote:    false,
			wantEmojis: []string{"🏌️"},
			wantText: []string{
				"TX",
				"Dallas Country Club",
				"Dallas",
			},
		},
		{
			name: "event without city",
			event: &event.Event{
				State:    "AZ",
				Title:    "Phoenix Golf Resort",
				DateText: "Jul 10 2026",
				City:     "",
				AlsoIn:   []string{},
			},
			hasNote:    false,
			wantEmojis: []string{"🏌️"},
			wantText: []string{
				"AZ",
				"Phoenix Golf Resort",
				"Jul 10, 2026",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg strings.Builder
			formatEventHeader(&msg, tt.event, tt.hasNote)
			result := msg.String()

			// Check for required emojis
			for _, emoji := range tt.wantEmojis {
				if !strings.Contains(result, emoji) {
					t.Errorf("formatEventHeader() missing emoji %q in:\n%s", emoji, result)
				}
			}

			// Check for required text
			for _, text := range tt.wantText {
				if !strings.Contains(result, text) {
					t.Errorf("formatEventHeader() missing text %q in:\n%s", text, result)
				}
			}
		})
	}
}

// TestFormatYardageEdgeCases tests yardage formatting edge cases through FormatEventWithCourse
func TestFormatYardageEdgeCases(t *testing.T) {
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	tests := []struct {
		name          string
		yardage       int
		wantFormatted string
	}{
		{
			name:          "yardage with comma (4 digits)",
			yardage:       7041,
			wantFormatted: "7,041",
		},
		{
			name:          "yardage with comma (5 digits)",
			yardage:       10234,
			wantFormatted: "10,234",
		},
		{
			name:          "yardage without comma (3 digits)",
			yardage:       950,
			wantFormatted: "950",
		},
		{
			name:          "yardage without comma (2 digits)",
			yardage:       50,
			wantFormatted: "50",
		},
		{
			name:          "yardage without comma (1 digit)",
			yardage:       5,
			wantFormatted: "5",
		},
		{
			name:          "zero yardage",
			yardage:       0,
			wantFormatted: "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			course := &CourseDetails{
				Name: "Test Course",
				Tees: []TeeDetails{
					{
						Name:    "Test Tee",
						Par:     72,
						Yardage: tt.yardage,
					},
				},
			}

			msg := FormatEventWithCourse(evt, course, "")

			if !strings.Contains(msg, tt.wantFormatted) {
				t.Errorf("Expected yardage format %q not found in:\n%s", tt.wantFormatted, msg)
			}
		})
	}
}

// TestFormatEventWithStatusAndCourseEdgeCases tests edge cases for FormatEventWithStatusAndCourse
func TestFormatEventWithStatusAndCourseEdgeCases(t *testing.T) {
	evt := &event.Event{
		ID:       "test123",
		State:    "NV",
		Title:    "Test Event",
		DateText: "Apr 4 2026",
		City:     "Las Vegas",
	}

	tests := []struct {
		name           string
		course         *CourseDetails
		status         string
		note           string
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:   "with nil course",
			course: nil,
			status: "interested",
			note:   "",
			wantContains: []string{
				"Test Event",
				"⭐",
				"Interested",
			},
			wantNotContain: []string{
				"Tees:",
			},
		},
		{
			name: "with course but no tees",
			course: &CourseDetails{
				Name: "Test Course",
				Tees: []TeeDetails{},
			},
			status: "registered",
			note:   "",
			wantContains: []string{
				"Test Event",
				"✅",
				"Registered",
			},
			wantNotContain: []string{
				"Tees:",
				"Test Course", // Course name not shown without tees
			},
		},
		{
			name: "with multiple tees",
			course: &CourseDetails{
				Name: "Test Course",
				Tees: []TeeDetails{
					{Name: "Championship", Par: 72, Yardage: 7200},
					{Name: "Blue", Par: 72, Yardage: 6800},
					{Name: "White", Par: 72, Yardage: 6400},
				},
			},
			status: "maybe",
			note:   "Thinking about it",
			wantContains: []string{
				"Test Event",
				"🤔",
				"Maybe",
				"Test Course",
				"Championship",
				"Blue",
				"White",
				"7,200",
				"6,800",
				"6,400",
				"📝",
				"Thinking about it",
			},
			wantNotContain: []string{},
		},
		{
			name: "with skip status and note",
			course: &CourseDetails{
				Name: "Test Course",
				Tees: []TeeDetails{
					{Name: "Championship", Par: 72, Yardage: 7200},
				},
			},
			status: "skip",
			note:   "Too expensive",
			wantContains: []string{
				"Test Event",
				"❌",
				"Skipped",
				"Test Course",
				"📝",
				"Too expensive",
			},
			wantNotContain: []string{},
		},
		{
			name: "with no status",
			course: &CourseDetails{
				Name: "Test Course",
				Tees: []TeeDetails{
					{Name: "Championship", Par: 72, Yardage: 7200},
				},
			},
			status: "",
			note:   "",
			wantContains: []string{
				"Test Event",
				"Test Course",
				"Championship",
			},
			wantNotContain: []string{
				"⭐", "✅", "🤔", "❌",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create minimal preferences object
			prefs := preferences.NewPreferences()
			msg, keyboard := FormatEventWithStatusAndCourse(evt, tt.course, tt.status, tt.note, "12345", prefs)

			// Check that message contains expected text
			for _, want := range tt.wantContains {
				if !strings.Contains(msg, want) {
					t.Errorf("Expected %q in message but not found:\n%s", want, msg)
				}
			}

			// Check that message does NOT contain unwanted text
			for _, notWant := range tt.wantNotContain {
				if strings.Contains(msg, notWant) {
					t.Errorf("Did not expect %q in message but found it:\n%s", notWant, msg)
				}
			}

			// Keyboard should always be present
			if keyboard == nil {
				t.Error("Keyboard should not be nil")
			}
		})
	}
}
