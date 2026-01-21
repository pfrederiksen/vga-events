package preferences

import (
	"testing"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

func TestPreferences(t *testing.T) {
	prefs := NewPreferences()

	// Test adding states
	if !prefs.AddState("123", "NV") {
		t.Error("Failed to add NV to user 123")
	}

	if !prefs.AddState("123", "CA") {
		t.Error("Failed to add CA to user 123")
	}

	// Test duplicate add
	if prefs.AddState("123", "NV") {
		t.Error("Should not add duplicate state NV")
	}

	// Test HasState
	if !prefs.HasState("123", "NV") {
		t.Error("HasState should return true for NV")
	}

	if prefs.HasState("123", "TX") {
		t.Error("HasState should return false for TX")
	}

	// Test GetStates
	states := prefs.GetStates("123")
	if len(states) != 2 {
		t.Errorf("Expected 2 states, got %d", len(states))
	}

	// Test RemoveState
	if !prefs.RemoveState("123", "NV") {
		t.Error("Failed to remove NV")
	}

	if prefs.HasState("123", "NV") {
		t.Error("NV should be removed")
	}

	// Test remove non-existent
	if prefs.RemoveState("123", "TX") {
		t.Error("Should not remove non-existent state")
	}

	// Test GetUser creates user
	user := prefs.GetUser("456")
	if user == nil {
		t.Fatal("GetUser should create new user")
	}

	if !user.Active {
		t.Error("New user should be active")
	}

	if len(user.States) != 0 {
		t.Error("New user should have no states")
	}
}

func TestIsValidState(t *testing.T) {
	tests := []struct {
		state string
		valid bool
	}{
		{"NV", true},
		{"CA", true},
		{"TX", true},
		{"ALL", true},
		{"nv", true}, // case insensitive
		{"ca", true},
		{"ZZ", false},    // invalid
		{"", false},      // empty
		{"  NV  ", true}, // whitespace trimmed
	}

	for _, tt := range tests {
		t.Run(tt.state, func(t *testing.T) {
			got := IsValidState(tt.state)
			if got != tt.valid {
				t.Errorf("IsValidState(%q) = %v, want %v", tt.state, got, tt.valid)
			}
		})
	}
}

func TestGetStateName(t *testing.T) {
	tests := []struct {
		code string
		name string
	}{
		{"NV", "Nevada"},
		{"CA", "California"},
		{"TX", "Texas"},
		{"ALL", "All States"},
		{"nv", "Nevada"}, // case insensitive
		{"ZZ", "ZZ"},     // unknown returns code
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			got := GetStateName(tt.code)
			if got != tt.name {
				t.Errorf("GetStateName(%q) = %q, want %q", tt.code, got, tt.name)
			}
		})
	}
}

func TestGetAllUsers(t *testing.T) {
	prefs := NewPreferences()

	// Add users with states
	prefs.AddState("123", "NV")
	prefs.AddState("456", "CA")

	// Add user with no states
	prefs.GetUser("789")

	// Add inactive user
	prefs.AddState("999", "TX")
	prefs.GetUser("999").Active = false

	users := prefs.GetAllUsers()

	// Should only return active users with states
	if len(users) != 2 {
		t.Errorf("Expected 2 active users with states, got %d", len(users))
	}

	// Check that 123 and 456 are in the list
	hasUser123 := false
	hasUser456 := false
	for _, user := range users {
		if user == "123" {
			hasUser123 = true
		}
		if user == "456" {
			hasUser456 = true
		}
	}

	if !hasUser123 || !hasUser456 {
		t.Error("Missing expected users in GetAllUsers")
	}
}

func TestJSONMarshaling(t *testing.T) {
	prefs := NewPreferences()
	prefs.AddState("123", "NV")
	prefs.AddState("123", "CA")
	prefs.AddState("456", "TX")

	// Marshal to JSON
	data, err := prefs.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON failed: %v", err)
	}

	// Unmarshal from JSON
	loaded, err := FromJSON(data)
	if err != nil {
		t.Fatalf("FromJSON failed: %v", err)
	}

	// Verify data
	if len(loaded) != 2 {
		t.Errorf("Expected 2 users, got %d", len(loaded))
	}

	states123 := loaded.GetStates("123")
	if len(states123) != 2 {
		t.Errorf("Expected 2 states for user 123, got %d", len(states123))
	}

	states456 := loaded.GetStates("456")
	if len(states456) != 1 {
		t.Errorf("Expected 1 state for user 456, got %d", len(states456))
	}
}

func TestCaseInsensitivity(t *testing.T) {
	prefs := NewPreferences()

	// Add state with lowercase
	prefs.AddState("123", "nv")

	// Check with uppercase
	if !prefs.HasState("123", "NV") {
		t.Error("State check should be case-insensitive")
	}

	// Remove with mixed case
	if !prefs.RemoveState("123", "Nv") {
		t.Error("State removal should be case-insensitive")
	}
}

func TestNewUserDefaults(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("new-user")

	// Check default values for new users
	if user.DigestFrequency != "immediate" {
		t.Errorf("New user should have immediate digest, got %q", user.DigestFrequency)
	}

	if user.DigestHour != 9 {
		t.Errorf("New user should have digest hour 9, got %d", user.DigestHour)
	}

	if user.DigestDayOfWeek != 1 {
		t.Errorf("New user should have digest day 1 (Monday), got %d", user.DigestDayOfWeek)
	}

	if !user.HidePastEvents {
		t.Error("New user should hide past events by default")
	}

	if user.DaysAhead != 0 {
		t.Errorf("New user should have DaysAhead disabled (0), got %d", user.DaysAhead)
	}

	if user.SeenEventIDs == nil {
		t.Error("New user should have initialized SeenEventIDs map")
	}

	if user.PendingEvents == nil {
		t.Error("New user should have initialized PendingEvents slice")
	}
}

func TestMigration(t *testing.T) {
	// Simulate an existing user without new fields (loaded from old JSON)
	prefs := make(Preferences)
	prefs["legacy-user"] = &UserPreferences{
		States:          []string{"NV", "CA"},
		Active:          true,
		SeenEventIDs:    nil, // Old user doesn't have this
		DigestFrequency: "",  // Old user doesn't have this
	}

	// GetUser should initialize missing fields
	user := prefs.GetUser("legacy-user")

	if user.DigestFrequency != "immediate" {
		t.Errorf("Migrated user should default to immediate, got %q", user.DigestFrequency)
	}

	if user.SeenEventIDs == nil {
		t.Error("Migrated user should have initialized SeenEventIDs")
	}

	// Existing fields should be preserved
	if len(user.States) != 2 {
		t.Errorf("Migration should preserve existing states, got %d", len(user.States))
	}

	if !user.Active {
		t.Error("Migration should preserve Active status")
	}
}

func TestEventHistory(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("test-user")

	// Mark events as seen
	user.MarkEventSeen("event1")
	user.MarkEventSeen("event2")

	// Check HasSeenEvent
	if !user.HasSeenEvent("event1") {
		t.Error("Should have seen event1")
	}

	if !user.HasSeenEvent("event2") {
		t.Error("Should have seen event2")
	}

	if user.HasSeenEvent("event3") {
		t.Error("Should not have seen event3")
	}

	// Check that timestamps are set
	if user.SeenEventIDs["event1"] == 0 {
		t.Error("event1 should have a non-zero timestamp")
	}
}

func TestCleanupOldHistory(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("test-user")

	now := time.Now().Unix()
	oldTime := time.Now().AddDate(0, 0, -100).Unix() // 100 days ago

	// Add mix of old and recent events
	user.SeenEventIDs["old-event-1"] = oldTime
	user.SeenEventIDs["old-event-2"] = oldTime
	user.SeenEventIDs["recent-event"] = now

	// Clean up events older than 90 days
	removed := user.CleanupOldHistory(90)

	if removed != 2 {
		t.Errorf("Should have removed 2 old events, removed %d", removed)
	}

	if user.HasSeenEvent("old-event-1") {
		t.Error("old-event-1 should have been cleaned up")
	}

	if user.HasSeenEvent("old-event-2") {
		t.Error("old-event-2 should have been cleaned up")
	}

	if !user.HasSeenEvent("recent-event") {
		t.Error("recent-event should still be present")
	}
}

func TestPendingEvents(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("test-user")

	// Create test events
	evt1 := &event.Event{ID: "evt1", Title: "Event 1", State: "NV"}
	evt2 := &event.Event{ID: "evt2", Title: "Event 2", State: "CA"}

	// Add pending events
	user.AddPendingEvent(evt1)
	user.AddPendingEvent(evt2)

	if len(user.PendingEvents) != 2 {
		t.Errorf("Expected 2 pending events, got %d", len(user.PendingEvents))
	}

	// Clear pending events
	user.ClearPendingEvents()

	if len(user.PendingEvents) != 0 {
		t.Errorf("Expected 0 pending events after clear, got %d", len(user.PendingEvents))
	}
}

func TestSetDigestFrequency(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("test-user")

	// Test valid frequencies
	if !user.SetDigestFrequency("daily") {
		t.Error("Should accept 'daily'")
	}
	if user.DigestFrequency != "daily" {
		t.Errorf("Expected 'daily', got %q", user.DigestFrequency)
	}

	if !user.SetDigestFrequency("weekly") {
		t.Error("Should accept 'weekly'")
	}
	if user.DigestFrequency != "weekly" {
		t.Errorf("Expected 'weekly', got %q", user.DigestFrequency)
	}

	if !user.SetDigestFrequency("immediate") {
		t.Error("Should accept 'immediate'")
	}
	if user.DigestFrequency != "immediate" {
		t.Errorf("Expected 'immediate', got %q", user.DigestFrequency)
	}

	// Test invalid frequency
	if user.SetDigestFrequency("monthly") {
		t.Error("Should reject invalid frequency 'monthly'")
	}

	// Test case insensitivity
	if !user.SetDigestFrequency("DAILY") {
		t.Error("Should accept uppercase 'DAILY'")
	}
	if user.DigestFrequency != "daily" {
		t.Errorf("Should normalize to lowercase, got %q", user.DigestFrequency)
	}
}
