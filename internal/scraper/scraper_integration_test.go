package scraper

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFetchEvents(t *testing.T) {
	tests := []struct {
		name          string
		htmlContent   string
		statusCode    int
		wantError     bool
		wantMinEvents int
	}{
		{
			name: "successful fetch with events",
			htmlContent: `
				<html>
					<body>
						NV - Chimera Golf Club 4.4.26 - Las Vegas
						CA - Pebble Beach Golf Links - Monterey
					</body>
				</html>
			`,
			statusCode:    http.StatusOK,
			wantError:     false,
			wantMinEvents: 2,
		},
		{
			name:        "HTTP error",
			htmlContent: "",
			statusCode:  http.StatusNotFound,
			wantError:   true,
		},
		{
			name: "empty page",
			htmlContent: `
				<html>
					<body>
						<p>No events</p>
					</body>
				</html>
			`,
			statusCode:    http.StatusOK,
			wantError:     false,
			wantMinEvents: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify User-Agent is set
				if userAgent := r.Header.Get("User-Agent"); !strings.Contains(userAgent, "vga-events") {
					t.Errorf("User-Agent = %q, should contain 'vga-events'", userAgent)
				}

				w.WriteHeader(tt.statusCode)
				w.Write([]byte(tt.htmlContent))
			}))
			defer server.Close()

			// Create scraper with test server URL
			scraper := New()
			scraper.url = server.URL

			events, err := scraper.FetchEvents()

			if tt.wantError {
				if err == nil {
					t.Error("FetchEvents() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("FetchEvents() unexpected error: %v", err)
				}
				if len(events) < tt.wantMinEvents {
					t.Errorf("FetchEvents() returned %d events, want at least %d", len(events), tt.wantMinEvents)
				}
			}
		})
	}
}

func TestParseEvents_EdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		html          string
		wantEventCount int
		checkEvent    func(*testing.T, string, string, string, string) // Check state, title, date, city
	}{
		{
			name: "event with date in title",
			html: `NV - Chimera Golf Club 4.4.26 - Las Vegas`,
			wantEventCount: 1,
			checkEvent: func(t *testing.T, state, title, dateText, city string) {
				if state != "NV" {
					t.Errorf("state = %q, want NV", state)
				}
				if !strings.Contains(title, "Chimera") {
					t.Errorf("title = %q, should contain Chimera", title)
				}
				if city != "Las Vegas" {
					t.Errorf("city = %q, want Las Vegas", city)
				}
			},
		},
		{
			name: "event without city",
			html: `CA - Torrey Pines Golf Course`,
			wantEventCount: 1,
			checkEvent: func(t *testing.T, state, title, dateText, city string) {
				if state != "CA" {
					t.Errorf("state = %q, want CA", state)
				}
				if city != "" {
					t.Errorf("city = %q, want empty", city)
				}
			},
		},
		{
			name: "bracketed date format",
			html: `[Feb 13 2026] NV - Championship Event - Reno`,
			wantEventCount: 1,
			checkEvent: func(t *testing.T, state, title, dateText, city string) {
				if dateText != "Feb 13 2026" {
					t.Errorf("dateText = %q, want 'Feb 13 2026'", dateText)
				}
			},
		},
		{
			name: "multiple date formats",
			html: `
				NV - Event 1 4.4.26 - Las Vegas
				CA - Event 2 - Monterey
				[Mar 13 2026] TX - Event 3 - Dallas
			`,
			wantEventCount: 3,
		},
		{
			name: "duplicate events",
			html: `
				NV - Chimera Golf Club - Las Vegas
				NV - Chimera Golf Club - Las Vegas
			`,
			wantEventCount: 1, // Should deduplicate
		},
		{
			name: "malformed lines ignored",
			html: `
				NV - Valid Event - Las Vegas
				This is not an event
				- Also not an event
				NV - Another Valid Event - Reno
			`,
			wantEventCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := New()
			events, err := s.parseEvents(strings.NewReader(tt.html), "https://test.example.com")

			if err != nil {
				t.Fatalf("parseEvents() error: %v", err)
			}

			if len(events) != tt.wantEventCount {
				t.Errorf("parseEvents() returned %d events, want %d", len(events), tt.wantEventCount)
			}

			if tt.checkEvent != nil && len(events) > 0 {
				evt := events[0]
				tt.checkEvent(t, evt.State, evt.Title, evt.DateText, evt.City)
			}
		})
	}
}

func TestExtractDate_EdgeCases(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"Event 1.24.26", "1.24.26"},
		{"Event 12.31.25", "12.31.25"},
		{"Event with Jan 15 date", "Jan 15"},
		{"Event with Feb 08 date", "Feb 08"},
		{"Event with 02/15/26 date", "02/15/26"},
		{"Event with 2/5/26 date", "2/5/26"},
		{"No date in this title", ""},
		{"Multiple 1.1.26 dates 2.2.26", "1.1.26"}, // Should get first
		{"", ""},
		{"Date at end 3.15.26", "3.15.26"},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := extractDate(tt.title)
			if result != tt.expected {
				t.Errorf("extractDate(%q) = %q, want %q", tt.title, result, tt.expected)
			}
		})
	}
}

func TestNew(t *testing.T) {
	s := New()

	if s == nil {
		t.Fatal("New() returned nil")
	}

	if s.client == nil {
		t.Error("scraper client is nil")
	}

	if s.url != StateEventsURL {
		t.Errorf("scraper url = %q, want %q", s.url, StateEventsURL)
	}
}

func TestParseEvents_SourceURL(t *testing.T) {
	html := `NV - Test Event - Las Vegas`
	sourceURL := "https://custom.url/test"

	s := New()
	events, err := s.parseEvents(strings.NewReader(html), sourceURL)

	if err != nil {
		t.Fatalf("parseEvents() error: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("parseEvents() returned no events")
	}

	if events[0].SourceURL != sourceURL {
		t.Errorf("event SourceURL = %q, want %q", events[0].SourceURL, sourceURL)
	}
}

func TestParseEvents_EventIDs(t *testing.T) {
	html := `
		NV - Event One - Las Vegas
		CA - Event Two - Monterey
		TX - Event Three - Dallas
	`

	s := New()
	events, err := s.parseEvents(strings.NewReader(html), "https://test.url")

	if err != nil {
		t.Fatalf("parseEvents() error: %v", err)
	}

	// All events should have unique IDs
	seenIDs := make(map[string]bool)
	for _, evt := range events {
		if evt.ID == "" {
			t.Error("Event has empty ID")
		}
		if seenIDs[evt.ID] {
			t.Errorf("Duplicate event ID: %s", evt.ID)
		}
		seenIDs[evt.ID] = true
	}
}

func TestParseEvents_StableKeys(t *testing.T) {
	html := `NV - Test Event - Las Vegas`

	s := New()
	events, err := s.parseEvents(strings.NewReader(html), "https://test.url")

	if err != nil {
		t.Fatalf("parseEvents() error: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("parseEvents() returned no events")
	}

	// Event should have a stable key
	if events[0].StableKey == "" {
		t.Error("Event has empty StableKey")
	}
}

func TestParseEvents_MonthDayYearFormat(t *testing.T) {
	// Test the pattern where month, day, and year are on separate lines
	html := `
		Jan
		15
		2026
		NV - Test Event - Las Vegas
	`

	s := New()
	events, err := s.parseEvents(strings.NewReader(html), "https://test.url")

	if err != nil {
		t.Fatalf("parseEvents() error: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("parseEvents() returned no events")
	}

	// Event should have picked up the date
	if !strings.Contains(events[0].DateText, "Jan") {
		t.Errorf("Event DateText = %q, should contain 'Jan'", events[0].DateText)
	}
}

func TestParseEvents_HTMLEntities(t *testing.T) {
	html := `
		<html>
			<body>
				<p>NV - Test &amp; Event - Las Vegas</p>
			</body>
		</html>
	`

	s := New()
	events, err := s.parseEvents(strings.NewReader(html), "https://test.url")

	if err != nil {
		t.Fatalf("parseEvents() error: %v", err)
	}

	// Should handle HTML entities properly
	// goquery automatically decodes HTML entities
	if len(events) > 0 && strings.Contains(events[0].Title, "&amp;") {
		t.Error("Event title contains unescaped HTML entity")
	}
}

func TestParseEvents_WhitespaceHandling(t *testing.T) {
	html := `
		NV   -   Event With Spaces   -   Las Vegas
	`

	s := New()
	events, err := s.parseEvents(strings.NewReader(html), "https://test.url")

	if err != nil {
		t.Fatalf("parseEvents() error: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("parseEvents() returned no events")
	}

	evt := events[0]

	// Title and city should have trimmed whitespace
	if strings.HasPrefix(evt.Title, " ") || strings.HasSuffix(evt.Title, " ") {
		t.Errorf("Event title has untrimmed whitespace: %q", evt.Title)
	}

	if strings.HasPrefix(evt.City, " ") || strings.HasSuffix(evt.City, " ") {
		t.Errorf("Event city has untrimmed whitespace: %q", evt.City)
	}
}
