package preferences

import (
	"testing"
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
		t.Error("GetUser should create new user")
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
		{"nv", true},  // case insensitive
		{"ca", true},
		{"ZZ", false}, // invalid
		{"", false},   // empty
		{"  NV  ", true},  // whitespace trimmed
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
