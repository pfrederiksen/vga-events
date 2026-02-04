package main

import (
	"reflect"
	"strings"
	"testing"

	"github.com/pfrederiksen/vga-events/internal/preferences"
)

// Helper function for string containment check
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

func TestParseBulkEventIDs(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "space separated",
			input:    []string{"abc123", "def456", "ghi789"},
			expected: []string{"abc123", "def456", "ghi789"},
		},
		{
			name:     "comma separated",
			input:    []string{"abc123,def456,ghi789"},
			expected: []string{"abc123", "def456", "ghi789"},
		},
		{
			name:     "mixed with spaces",
			input:    []string{"abc123,def456", "ghi789"},
			expected: []string{"abc123", "def456", "ghi789"},
		},
		{
			name:     "with extra spaces",
			input:    []string{"abc123, def456 ,ghi789"},
			expected: []string{"abc123", "def456", "ghi789"},
		},
		{
			name:     "empty input",
			input:    []string{},
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseBulkEventIDs(tt.input)
			if len(result) == 0 && len(tt.expected) == 0 {
				return // Both empty, test passes
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseBulkEventIDs(%v) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseEventIDList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "comma separated",
			input:    "abc123,def456,ghi789",
			expected: []string{"abc123", "def456", "ghi789"},
		},
		{
			name:     "space separated",
			input:    "abc123 def456 ghi789",
			expected: []string{"abc123", "def456", "ghi789"},
		},
		{
			name:     "single ID",
			input:    "abc123",
			expected: []string{"abc123"},
		},
		{
			name:     "with extra spaces",
			input:    "abc123, def456 , ghi789",
			expected: []string{"abc123", "def456", "ghi789"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEventIDList(tt.input)
			if len(result) == 0 && len(tt.expected) == 0 {
				return // Both empty, test passes
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseEventIDList(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestHandleBulkRegister(t *testing.T) {
	tests := []struct {
		name         string
		eventIDs     []string
		expectModify bool
		expectError  bool
	}{
		{
			name:         "register multiple events",
			eventIDs:     []string{"event1", "event2", "event3"},
			expectModify: true,
			expectError:  false,
		},
		{
			name:         "register single event",
			eventIDs:     []string{"event1"},
			expectModify: true,
			expectError:  false,
		},
		{
			name:         "empty event list",
			eventIDs:     []string{},
			expectModify: false,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefs := preferences.NewPreferences()
			chatID := "test123"
			modified := false

			response, _ := handleBulkRegister(prefs, chatID, tt.eventIDs, &modified)

			if tt.expectModify && !modified {
				t.Errorf("Expected preferences to be modified")
			}
			if !tt.expectModify && modified {
				t.Errorf("Expected preferences not to be modified")
			}

			if len(tt.eventIDs) == 0 {
				if response != "❌ No event IDs provided." {
					t.Errorf("Expected error message for empty input, got: %s", response)
				}
			} else {
				// Check that events were registered
				user := prefs.GetUser(chatID)
				for _, eventID := range tt.eventIDs {
					status := user.GetEventStatus(eventID)
					if status != preferences.EventStatusRegistered {
						t.Errorf("Event %s not registered, got status: %s", eventID, status)
					}
				}
			}
		})
	}
}

func TestHandleBulkNote(t *testing.T) {
	tests := []struct {
		name         string
		eventIDs     []string
		noteText     string
		expectModify bool
	}{
		{
			name:         "add note to multiple events",
			eventIDs:     []string{"event1", "event2"},
			noteText:     "Test note",
			expectModify: true,
		},
		{
			name:         "add note to single event",
			eventIDs:     []string{"event1"},
			noteText:     "Single note",
			expectModify: true,
		},
		{
			name:         "empty event list",
			eventIDs:     []string{},
			noteText:     "Note",
			expectModify: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefs := preferences.NewPreferences()
			chatID := "test123"
			modified := false

			response, _ := handleBulkNote(prefs, chatID, tt.eventIDs, tt.noteText, &modified)

			if tt.expectModify && !modified {
				t.Errorf("Expected preferences to be modified")
			}
			if !tt.expectModify && modified {
				t.Errorf("Expected preferences not to be modified")
			}

			if len(tt.eventIDs) == 0 {
				if response != "❌ No event IDs provided." {
					t.Errorf("Expected error message for empty input, got: %s", response)
				}
			} else {
				// Check that notes were added
				user := prefs.GetUser(chatID)
				for _, eventID := range tt.eventIDs {
					if user.EventNotes[eventID] != tt.noteText {
						t.Errorf("Event %s note not set correctly, got: %s", eventID, user.EventNotes[eventID])
					}
				}
			}
		})
	}
}

func TestHandleBulkStatus(t *testing.T) {
	tests := []struct {
		name         string
		status       string
		eventIDs     []string
		expectModify bool
		expectError  bool
	}{
		{
			name:         "set interested status",
			status:       "interested",
			eventIDs:     []string{"event1", "event2"},
			expectModify: true,
			expectError:  false,
		},
		{
			name:         "set registered status",
			status:       "registered",
			eventIDs:     []string{"event1"},
			expectModify: true,
			expectError:  false,
		},
		{
			name:         "invalid status",
			status:       "invalid",
			eventIDs:     []string{"event1"},
			expectModify: false,
			expectError:  true,
		},
		{
			name:         "empty event list",
			status:       "interested",
			eventIDs:     []string{},
			expectModify: false,
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefs := preferences.NewPreferences()
			chatID := "test123"
			modified := false

			response, _ := handleBulkStatus(prefs, chatID, tt.status, tt.eventIDs, &modified)

			if tt.expectModify && !modified {
				t.Errorf("Expected preferences to be modified")
			}
			if !tt.expectModify && modified {
				t.Errorf("Expected preferences not to be modified")
			}

			if tt.expectError {
				if !contains(response, "Invalid status") {
					t.Errorf("Expected error message about invalid status, got: %s", response)
				}
			} else if len(tt.eventIDs) == 0 {
				if response != "❌ No event IDs provided." {
					t.Errorf("Expected error message for empty input, got: %s", response)
				}
			} else if !tt.expectError {
				// Check that status was set
				user := prefs.GetUser(chatID)
				for _, eventID := range tt.eventIDs {
					status := user.GetEventStatus(eventID)
					if status != tt.status {
						t.Errorf("Event %s status not set correctly, expected %s, got: %s", eventID, tt.status, status)
					}
				}
			}
		})
	}
}
