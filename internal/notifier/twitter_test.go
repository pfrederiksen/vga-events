package notifier

import (
	"testing"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

func TestFormatTweet(t *testing.T) {
	tests := []struct {
		name     string
		event    *event.Event
		wantLen  int
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
			wantLen: 280,
			contains: []string{
				"NV",
				"Chimera Golf Club",
				"Apr 04 2026",
				"Las Vegas",
				"#VGAGolf",
				"#Golf",
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
			wantLen: 280,
			contains: []string{
				"CA",
				"Pebble Beach",
				"Monterey",
				"#VGAGolf",
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
			wantLen: 280,
			contains: []string{
				"TX",
				"Dallas Country Club",
				"May 15 2026",
				"#Golf",
			},
		},
		{
			name: "very long title gets truncated",
			event: &event.Event{
				ID:        "test000",
				State:     "CA",
				Title:     "This is an extremely long golf course name that goes on and on and will definitely exceed the Twitter character limit of 280 characters when combined with all the other information we want to include in the tweet including emojis and hashtags",
				DateText:  "Jun 20 2026",
				City:      "Very Long City Name That Also Contributes To Length",
				Raw:       "CA - Long Course - Long City",
				SourceURL: "https://vgagolf.org/state-events/",
				FirstSeen: time.Now(),
			},
			wantLen: 280,
			contains: []string{
				"...",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTweet(tt.event)

			// Check length
			if len(got) > tt.wantLen {
				t.Errorf("formatTweet() length = %d, want <= %d", len(got), tt.wantLen)
			}

			// Check contains
			for _, want := range tt.contains {
				if !contains(got, want) {
					t.Errorf("formatTweet() missing %q in tweet:\n%s", want, got)
				}
			}
		})
	}
}

func TestDryRunNotifier(t *testing.T) {
	notifier := NewDryRunNotifier()

	events := []*event.Event{
		{
			ID:        "test1",
			State:     "NV",
			Title:     "Test Event 1",
			DateText:  "Apr 01 2026",
			City:      "Las Vegas",
			Raw:       "NV - Test Event 1 - Las Vegas",
			SourceURL: "https://vgagolf.org/state-events/",
			FirstSeen: time.Now(),
		},
		{
			ID:        "test2",
			State:     "CA",
			Title:     "Test Event 2",
			DateText:  "Apr 02 2026",
			City:      "San Francisco",
			Raw:       "CA - Test Event 2 - San Francisco",
			SourceURL: "https://vgagolf.org/state-events/",
			FirstSeen: time.Now(),
		},
	}

	// Should not error
	if err := notifier.Notify(events); err != nil {
		t.Errorf("DryRunNotifier.Notify() error = %v, want nil", err)
	}
}

// contains checks if s contains substr
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || findSubstring(s, substr))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
