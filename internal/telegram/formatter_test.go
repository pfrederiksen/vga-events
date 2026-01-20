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
			name:  "single event, single state",
			count: 1,
			states: []string{"NV"},
			contains: []string{
				"<b>1</b> new event",
				"1 state",
				"NV",
				"#VGAGolf",
			},
		},
		{
			name:  "multiple events, multiple states",
			count: 5,
			states: []string{"NV", "CA", "TX"},
			contains: []string{
				"<b>5</b> new events",
				"3 states",
				"NV, CA, TX",
				"#VGAGolf",
			},
		},
		{
			name:  "multiple events, no states specified",
			count: 10,
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
