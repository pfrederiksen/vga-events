package telegram

import (
	"strings"
	"testing"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

func TestFormatDigest(t *testing.T) {
	tests := []struct {
		name         string
		events       []*event.Event
		frequency    string
		wantContains []string
		wantEmpty    bool
	}{
		{
			name:      "empty events",
			events:    []*event.Event{},
			frequency: "daily",
			wantContains: []string{
				"No new events",
			},
			wantEmpty: false,
		},
		{
			name: "single event",
			events: []*event.Event{
				{
					ID:        "evt1",
					State:     "NV",
					Title:     "Chimera Golf Club",
					DateText:  "Apr 4 2026",
					City:      "Las Vegas",
					FirstSeen: time.Now(),
				},
			},
			frequency: "daily",
			wantContains: []string{
				"📬",
				"Your VGA Events Digest",
				"Daily digest",
				"1 new event(s)",
				"📍",
				"NV",
				"Chimera Golf Club",
				"Apr 4 2026",
				"Las Vegas",
				"vgagolf.org/state-events",
				"/settings",
			},
			wantEmpty: false,
		},
		{
			name: "multiple events in multiple states",
			events: []*event.Event{
				{
					ID:        "evt1",
					State:     "NV",
					Title:     "Chimera Golf Club",
					DateText:  "Apr 4 2026",
					City:      "Las Vegas",
					FirstSeen: time.Now(),
				},
				{
					ID:        "evt2",
					State:     "NV",
					Title:     "Paiute Golf Resort",
					DateText:  "Apr 10 2026",
					City:      "Las Vegas",
					FirstSeen: time.Now(),
				},
				{
					ID:        "evt3",
					State:     "CA",
					Title:     "Pebble Beach",
					DateText:  "May 15 2026",
					City:      "Monterey",
					FirstSeen: time.Now(),
				},
			},
			frequency: "weekly",
			wantContains: []string{
				"Weekly digest",
				"3 new event(s)",
				"NV",
				"(2 events)",
				"CA",
				"(1 event)",
				"Chimera Golf Club",
				"Paiute Golf Resort",
				"Pebble Beach",
			},
			wantEmpty: false,
		},
		{
			name: "event without date",
			events: []*event.Event{
				{
					ID:        "evt1",
					State:     "TX",
					Title:     "Dallas Country Club",
					DateText:  "",
					City:      "Dallas",
					FirstSeen: time.Now(),
				},
			},
			frequency: "daily",
			wantContains: []string{
				"Dallas Country Club",
				"Dallas",
				"TX",
			},
			wantEmpty: false,
		},
		{
			name: "event without city",
			events: []*event.Event{
				{
					ID:        "evt1",
					State:     "AZ",
					Title:     "Phoenix Golf Resort",
					DateText:  "Jun 1 2026",
					City:      "",
					FirstSeen: time.Now(),
				},
			},
			frequency: "immediate",
			wantContains: []string{
				"Phoenix Golf Resort",
				"Jun 1 2026",
				"AZ",
				"Immediate digest",
			},
			wantEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDigest(tt.events, tt.frequency)

			if tt.wantEmpty && got != "" {
				t.Errorf("FormatDigest() = %q, want empty string", got)
			}

			if !tt.wantEmpty && got == "" {
				t.Error("FormatDigest() returned empty string")
			}

			// Check all required substrings
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatDigest() missing %q in message:\n%s", want, got)
				}
			}

			// Message should be within Telegram's limit
			if len(got) > 4096 {
				t.Errorf("FormatDigest() length = %d, exceeds Telegram limit of 4096", len(got))
			}
		})
	}
}

func TestFormatDigestSummary(t *testing.T) {
	tests := []struct {
		name         string
		events       []*event.Event
		frequency    string
		wantContains []string
	}{
		{
			name:      "empty events",
			events:    []*event.Event{},
			frequency: "daily",
			wantContains: []string{
				"📬",
				"daily digest",
				"No new events",
			},
		},
		{
			name: "single event single state",
			events: []*event.Event{
				{
					ID:    "evt1",
					State: "NV",
					Title: "Chimera Golf Club",
				},
			},
			frequency: "daily",
			wantContains: []string{
				"📬",
				"daily digest",
				"1 new event",
				"NV (1)",
			},
		},
		{
			name: "multiple events multiple states",
			events: []*event.Event{
				{ID: "evt1", State: "NV", Title: "Event 1"},
				{ID: "evt2", State: "NV", Title: "Event 2"},
				{ID: "evt3", State: "CA", Title: "Event 3"},
				{ID: "evt4", State: "TX", Title: "Event 4"},
			},
			frequency: "weekly",
			wantContains: []string{
				"📬",
				"weekly digest",
				"4 new events",
				"CA (1)",
				"NV (2)",
				"TX (1)",
			},
		},
		{
			name: "many events same state",
			events: []*event.Event{
				{ID: "evt1", State: "CA", Title: "Event 1"},
				{ID: "evt2", State: "CA", Title: "Event 2"},
				{ID: "evt3", State: "CA", Title: "Event 3"},
				{ID: "evt4", State: "CA", Title: "Event 4"},
				{ID: "evt5", State: "CA", Title: "Event 5"},
			},
			frequency: "immediate",
			wantContains: []string{
				"5 new events",
				"CA (5)",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDigestSummary(tt.events, tt.frequency)

			if got == "" {
				t.Error("FormatDigestSummary() returned empty string")
			}

			// Check all required substrings
			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("FormatDigestSummary() missing %q in message:\n%s", want, got)
				}
			}

			// Summary should be much shorter than full digest
			if len(got) > 500 {
				t.Errorf("FormatDigestSummary() length = %d, should be concise", len(got))
			}
		})
	}
}

func TestPluralize(t *testing.T) {
	tests := []struct {
		count int
		want  string
	}{
		{count: 0, want: "s"},
		{count: 1, want: ""},
		{count: 2, want: "s"},
		{count: 10, want: "s"},
		{count: 100, want: "s"},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.count)), func(t *testing.T) {
			got := pluralize(tt.count)
			if got != tt.want {
				t.Errorf("pluralize(%d) = %q, want %q", tt.count, got, tt.want)
			}
		})
	}
}

func TestFormatDigest_StateOrdering(t *testing.T) {
	// Test that states are alphabetically ordered
	events := []*event.Event{
		{ID: "evt1", State: "TX", Title: "Event 1"},
		{ID: "evt2", State: "CA", Title: "Event 2"},
		{ID: "evt3", State: "NV", Title: "Event 3"},
		{ID: "evt4", State: "AZ", Title: "Event 4"},
	}

	result := FormatDigest(events, "daily")

	// Find positions of state names
	posAZ := strings.Index(result, "📍 <b>AZ</b>")
	posCA := strings.Index(result, "📍 <b>CA</b>")
	posNV := strings.Index(result, "📍 <b>NV</b>")
	posTX := strings.Index(result, "📍 <b>TX</b>")

	// Verify alphabetical order
	if posAZ == -1 || posCA == -1 || posNV == -1 || posTX == -1 {
		t.Error("Not all states found in digest")
	}

	if !(posAZ < posCA && posCA < posNV && posNV < posTX) {
		t.Errorf("States not in alphabetical order. Positions: AZ=%d, CA=%d, NV=%d, TX=%d",
			posAZ, posCA, posNV, posTX)
	}
}

func TestFormatDigestSummary_StateOrdering(t *testing.T) {
	// Test that states are alphabetically ordered in summary
	events := []*event.Event{
		{ID: "evt1", State: "TX", Title: "Event 1"},
		{ID: "evt2", State: "CA", Title: "Event 2"},
		{ID: "evt3", State: "NV", Title: "Event 3"},
	}

	result := FormatDigestSummary(events, "daily")

	// Find positions of state codes
	posCA := strings.Index(result, "CA")
	posNV := strings.Index(result, "NV")
	posTX := strings.Index(result, "TX")

	if posCA == -1 || posNV == -1 || posTX == -1 {
		t.Error("Not all states found in summary")
	}

	// CA should come before NV, and NV before TX
	if !(posCA < posNV && posNV < posTX) {
		t.Errorf("States not in alphabetical order. Positions: CA=%d, NV=%d, TX=%d",
			posCA, posNV, posTX)
	}
}
