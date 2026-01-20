package scraper

import (
	"os"
	"strings"
	"testing"
)

func TestParseEvents(t *testing.T) {
	// Load test fixture
	data, err := os.ReadFile("../../testdata/fixtures/sample_events.html")
	if err != nil {
		t.Fatalf("failed to load test fixture: %v", err)
	}

	s := New()
	events, err := s.parseEvents(strings.NewReader(string(data)), "https://test.example.com")
	if err != nil {
		t.Fatalf("parseEvents failed: %v", err)
	}

	if len(events) == 0 {
		t.Fatal("expected events to be parsed, got 0")
	}

	// Check for expected states
	stateCount := make(map[string]int)
	for _, evt := range events {
		stateCount[evt.State]++
	}

	expectedStates := map[string]int{
		"NV": 2,
		"CA": 3,
		"TX": 2,
	}

	for state, expectedCount := range expectedStates {
		if count, ok := stateCount[state]; !ok {
			t.Errorf("expected state %s to be present", state)
		} else if count != expectedCount {
			t.Errorf("expected %d events for state %s, got %d", expectedCount, state, count)
		}
	}

	// Verify event fields are populated
	for _, evt := range events {
		if evt.ID == "" {
			t.Error("event ID should not be empty")
		}
		if evt.State == "" {
			t.Error("event state should not be empty")
		}
		if evt.Raw == "" {
			t.Error("event raw should not be empty")
		}
		if evt.SourceURL != "https://test.example.com" {
			t.Errorf("expected source URL to be 'https://test.example.com', got '%s'", evt.SourceURL)
		}
	}
}

func TestExtractDate(t *testing.T) {
	tests := []struct {
		title    string
		expected string
	}{
		{"Chimera Golf Club 4.4.26", "4.4.26"},
		{"Valley Oaks GC 1.24.26", "1.24.26"},
		{"Event with Jan 24 date", "Jan 24"},
		{"Event with Feb 08 date", "Feb 08"},
		{"Event with 02/15/26 date", "02/15/26"},
		{"No date here", ""},
	}

	for _, tt := range tests {
		t.Run(tt.title, func(t *testing.T) {
			result := extractDate(tt.title)
			if result != tt.expected {
				t.Errorf("extractDate(%q) = %q, expected %q", tt.title, result, tt.expected)
			}
		})
	}
}
