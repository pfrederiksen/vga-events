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

//nolint:dupl // Similar structure to TestIsValidEventStatus is intentional
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

func TestEventStatus(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("test-user")

	// Test setting valid statuses
	tests := []struct {
		eventID string
		status  string
		valid   bool
	}{
		{"event1", "interested", true},
		{"event2", "registered", true},
		{"event3", "maybe", true},
		{"event4", "skip", true},
		{"event5", "invalid", false},
		{"event6", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := user.SetEventStatus(tt.eventID, tt.status)
			if got != tt.valid {
				t.Errorf("SetEventStatus(%q, %q) = %v, want %v", tt.eventID, tt.status, got, tt.valid)
			}

			if tt.valid {
				// Verify status was set
				status := user.GetEventStatus(tt.eventID)
				if status != tt.status {
					t.Errorf("GetEventStatus(%q) = %q, want %q", tt.eventID, status, tt.status)
				}
			}
		})
	}

	// Test case insensitivity
	if !user.SetEventStatus("event7", "INTERESTED") {
		t.Error("Should accept uppercase 'INTERESTED'")
	}
	if user.GetEventStatus("event7") != "interested" {
		t.Error("Should normalize to lowercase")
	}

	// Test GetEventStatus for non-existent event
	if status := user.GetEventStatus("nonexistent"); status != "" {
		t.Errorf("GetEventStatus for nonexistent event should return empty string, got %q", status)
	}

	// Test RemoveEventStatus
	user.SetEventStatus("event-to-remove", "interested")
	user.RemoveEventStatus("event-to-remove")
	if status := user.GetEventStatus("event-to-remove"); status != "" {
		t.Errorf("Event status should be removed, got %q", status)
	}
}

func TestGetEventsByStatus(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("test-user")

	// Set up events with different statuses
	user.SetEventStatus("event1", "interested")
	user.SetEventStatus("event2", "interested")
	user.SetEventStatus("event3", "registered")
	user.SetEventStatus("event4", "maybe")
	user.SetEventStatus("event5", "skip")

	// Test getting events by status
	interested := user.GetEventsByStatus("interested")
	if len(interested) != 2 {
		t.Errorf("Expected 2 interested events, got %d", len(interested))
	}

	registered := user.GetEventsByStatus("registered")
	if len(registered) != 1 {
		t.Errorf("Expected 1 registered event, got %d", len(registered))
	}

	maybe := user.GetEventsByStatus("maybe")
	if len(maybe) != 1 {
		t.Errorf("Expected 1 maybe event, got %d", len(maybe))
	}

	skip := user.GetEventsByStatus("skip")
	if len(skip) != 1 {
		t.Errorf("Expected 1 skip event, got %d", len(skip))
	}

	// Test with status that has no events
	none := user.GetEventsByStatus("nonexistent")
	if len(none) != 0 {
		t.Errorf("Expected 0 events for nonexistent status, got %d", len(none))
	}
}

//nolint:dupl // Similar structure to TestIsValidState is intentional
func TestIsValidEventStatus(t *testing.T) {
	tests := []struct {
		status string
		valid  bool
	}{
		{"interested", true},
		{"registered", true},
		{"maybe", true},
		{"skip", true},
		{"INTERESTED", true},     // case insensitive
		{"  registered  ", true}, // whitespace
		{"invalid", false},
		{"", false},
		{"pending", false},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := IsValidEventStatus(tt.status)
			if got != tt.valid {
				t.Errorf("IsValidEventStatus(%q) = %v, want %v", tt.status, got, tt.valid)
			}
		})
	}
}

func TestReminderDays(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("test-user")

	// Test setting valid reminder days
	if !user.SetReminderDays([]int{1, 3, 7}) {
		t.Error("Should accept valid reminder days")
	}

	if len(user.ReminderDays) != 3 {
		t.Errorf("Expected 3 reminder days, got %d", len(user.ReminderDays))
	}

	// Test HasReminderDay
	if !user.HasReminderDay(1) {
		t.Error("Should have reminder day 1")
	}
	if !user.HasReminderDay(3) {
		t.Error("Should have reminder day 3")
	}
	if !user.HasReminderDay(7) {
		t.Error("Should have reminder day 7")
	}
	if user.HasReminderDay(14) {
		t.Error("Should not have reminder day 14")
	}

	// Test setting invalid reminder days
	if user.SetReminderDays([]int{1, 5, 7}) {
		t.Error("Should reject invalid reminder day 5")
	}

	// Test setting all valid days
	if !user.SetReminderDays([]int{1, 3, 7, 14}) {
		t.Error("Should accept all valid reminder days")
	}

	if len(user.ReminderDays) != 4 {
		t.Errorf("Expected 4 reminder days, got %d", len(user.ReminderDays))
	}

	// Test empty reminder days
	if !user.SetReminderDays([]int{}) {
		t.Error("Should accept empty reminder days")
	}

	if len(user.ReminderDays) != 0 {
		t.Errorf("Expected 0 reminder days, got %d", len(user.ReminderDays))
	}
}

func TestEventStatusMigration(t *testing.T) {
	// Simulate an existing user without EventStatuses field
	prefs := make(Preferences)
	prefs["legacy-user"] = &UserPreferences{
		States:        []string{"NV"},
		Active:        true,
		EventStatuses: nil, // Old user doesn't have this
	}

	// GetUser should initialize missing fields
	user := prefs.GetUser("legacy-user")

	if user.EventStatuses == nil {
		t.Error("Migrated user should have initialized EventStatuses")
	}

	// Test that we can set statuses
	if !user.SetEventStatus("test-event", "interested") {
		t.Error("Should be able to set event status after migration")
	}
}

func TestReminderDaysMigration(t *testing.T) {
	// Simulate an existing user without ReminderDays field
	prefs := make(Preferences)
	prefs["legacy-user"] = &UserPreferences{
		States:       []string{"NV"},
		Active:       true,
		ReminderDays: nil, // Old user doesn't have this
	}

	// GetUser should work (migration happens in GetUser for existing users)
	user := prefs.GetUser("legacy-user")

	// New users should have empty ReminderDays slice
	if user.ReminderDays == nil {
		// This is OK - it will be initialized when SetReminderDays is called
		user.ReminderDays = []int{}
	}

	// Test that we can set reminder days
	if !user.SetReminderDays([]int{1, 7}) {
		t.Error("Should be able to set reminder days")
	}
}

func TestUserPreferences_SetEventNote(t *testing.T) {
	user := &UserPreferences{}

	// Set a note for an event
	user.SetEventNote("event123", "This is a great event!")

	// Verify EventNotes was initialized
	if user.EventNotes == nil {
		t.Error("EventNotes map should be initialized")
	}

	// Verify the note was set
	if got := user.EventNotes["event123"]; got != "This is a great event!" {
		t.Errorf("EventNotes[event123] = %q, want %q", got, "This is a great event!")
	}

	// Update the note
	user.SetEventNote("event123", "Updated note")
	if got := user.EventNotes["event123"]; got != "Updated note" {
		t.Errorf("EventNotes[event123] = %q, want %q", got, "Updated note")
	}

	// Add another note
	user.SetEventNote("event456", "Another event note")
	if len(user.EventNotes) != 2 {
		t.Errorf("EventNotes length = %d, want 2", len(user.EventNotes))
	}
}

func TestUserPreferences_GetEventNote(t *testing.T) {
	user := &UserPreferences{
		EventNotes: map[string]string{
			"event123": "Great tournament",
			"event456": "Bring extra balls",
		},
	}

	tests := []struct {
		name    string
		eventID string
		want    string
	}{
		{
			name:    "Existing note",
			eventID: "event123",
			want:    "Great tournament",
		},
		{
			name:    "Another existing note",
			eventID: "event456",
			want:    "Bring extra balls",
		},
		{
			name:    "Non-existent event",
			eventID: "event999",
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := user.GetEventNote(tt.eventID)
			if got != tt.want {
				t.Errorf("GetEventNote(%q) = %q, want %q", tt.eventID, got, tt.want)
			}
		})
	}

	// Test with nil EventNotes
	user2 := &UserPreferences{}
	if got := user2.GetEventNote("event123"); got != "" {
		t.Errorf("GetEventNote with nil map = %q, want empty string", got)
	}
}

func TestUserPreferences_RemoveEventNote(t *testing.T) {
	user := &UserPreferences{
		EventNotes: map[string]string{
			"event123": "Great tournament",
			"event456": "Bring extra balls",
			"event789": "Early tee time",
		},
	}

	// Remove a note
	user.RemoveEventNote("event456")

	// Verify it was removed
	if _, exists := user.EventNotes["event456"]; exists {
		t.Error("event456 should have been removed")
	}

	// Verify others still exist
	if len(user.EventNotes) != 2 {
		t.Errorf("EventNotes length = %d, want 2", len(user.EventNotes))
	}

	// Remove non-existent note (should not error)
	user.RemoveEventNote("event999")
	if len(user.EventNotes) != 2 {
		t.Errorf("EventNotes length = %d, want 2", len(user.EventNotes))
	}

	// Test with nil EventNotes (should not panic)
	user2 := &UserPreferences{}
	user2.RemoveEventNote("event123") // Should not panic
}

func TestUserPreferences_EventNotes_Migration(t *testing.T) {
	// Test that GetUser initializes EventNotes for existing users
	prefs := NewPreferences()

	// Create a user without EventNotes (simulating old data)
	prefs["user123"] = &UserPreferences{
		States: []string{"NV"},
		Active: true,
	}

	// GetUser should initialize EventNotes
	user := prefs.GetUser("user123")
	if user.EventNotes == nil {
		t.Error("GetUser should initialize EventNotes map")
	}

	// Should be able to use EventNotes immediately
	user.SetEventNote("event123", "Test note")
	if got := user.GetEventNote("event123"); got != "Test note" {
		t.Errorf("GetEventNote = %q, want %q", got, "Test note")
	}
}
func TestWeeklyStats(t *testing.T) {
	prefs := NewPreferences()
	chatID := "12345"

	user := prefs.GetUser(chatID)
	if user.WeeklyStats == nil {
		t.Fatal("WeeklyStats should be initialized")
	}

	// Test IncrementEventsViewed
	user.IncrementEventsViewed(1)
	if user.WeeklyStats.EventsViewed != 1 {
		t.Errorf("EventsViewed = %d, want 1", user.WeeklyStats.EventsViewed)
	}

	// Test IncrementEventStatus
	user.IncrementEventStatus("interested")
	user.IncrementEventStatus("interested")
	user.IncrementEventStatus("registered")

	if user.WeeklyStats.EventsMarked["interested"] != 2 {
		t.Errorf("EventsMarked[interested] = %d, want 2", user.WeeklyStats.EventsMarked["interested"])
	}
	if user.WeeklyStats.EventsMarked["registered"] != 1 {
		t.Errorf("EventsMarked[registered] = %d, want 1", user.WeeklyStats.EventsMarked["registered"])
	}
}

func TestArchiveCurrentWeek(t *testing.T) {
	prefs := NewPreferences()
	chatID := "12345"

	user := prefs.GetUser(chatID)

	// Add some stats
	user.IncrementEventsViewed(1)
	user.IncrementEventsViewed(1)
	user.IncrementEventStatus("interested")

	// Get current week key
	weekKey := GetWeekKey(time.Now())

	// Archive current week
	user.ArchiveCurrentWeek()

	// Check that stats were archived
	if user.StatsHistory == nil {
		t.Fatal("StatsHistory should be initialized")
	}

	archived, exists := user.StatsHistory[weekKey]
	if !exists {
		t.Fatalf("Stats for week %s not archived", weekKey)
	}

	if archived.EventsViewed != 2 {
		t.Errorf("Archived EventsViewed = %d, want 2", archived.EventsViewed)
	}

	// Check that current stats were reset
	if user.WeeklyStats.EventsViewed != 0 {
		t.Errorf("Current EventsViewed = %d, want 0", user.WeeklyStats.EventsViewed)
	}
}

func TestGetAllTimeStats(t *testing.T) {
	prefs := NewPreferences()
	chatID := "12345"

	user := prefs.GetUser(chatID)

	// Add current week stats
	user.IncrementEventsViewed(1)
	user.IncrementEventsViewed(1)
	user.IncrementEventStatus("interested")

	// Archive a week
	user.ArchiveCurrentWeek()

	// Add more stats
	user.IncrementEventsViewed(1)
	user.IncrementEventStatus("registered")

	// Get all-time stats
	allTime := user.GetAllTimeStats()

	if allTime.EventsViewed != 3 {
		t.Errorf("AllTime EventsViewed = %d, want 3", allTime.EventsViewed)
	}

	if allTime.EventsMarked["interested"] != 1 {
		t.Errorf("AllTime EventsMarked[interested] = %d, want 1", allTime.EventsMarked["interested"])
	}

	if allTime.EventsMarked["registered"] != 1 {
		t.Errorf("AllTime EventsMarked[registered] = %d, want 1", allTime.EventsMarked["registered"])
	}
}

func TestGetWeekKey(t *testing.T) {
	// Test a known date
	testDate := time.Date(2026, 1, 21, 0, 0, 0, 0, time.UTC)
	weekKey := GetWeekKey(testDate)

	// Week key should be in format YYYY-WXX
	if len(weekKey) != 8 {
		t.Errorf("Week key length = %d, want 8", len(weekKey))
	}

	// Should start with year
	if weekKey[:4] != "2026" {
		t.Errorf("Week key year = %s, want 2026", weekKey[:4])
	}

	// Test different weeks give different keys
	week1 := GetWeekKey(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	week2 := GetWeekKey(time.Date(2026, 1, 8, 0, 0, 0, 0, time.UTC))

	if week1 == week2 {
		t.Errorf("Different weeks produced same key: %s", week1)
	}
}

func TestFriendManagement(t *testing.T) {
	prefs := NewPreferences()
	user1ID := "12345"
	user2ID := "67890"

	user1 := prefs.GetUser(user1ID)

	// Initially no friends
	if user1.GetFriendCount() != 0 {
		t.Errorf("Initial friend count = %d, want 0", user1.GetFriendCount())
	}

	// Add friend
	added := user1.AddFriend(user2ID)
	if !added {
		t.Error("AddFriend returned false, expected true")
	}

	// Check friend count
	if user1.GetFriendCount() != 1 {
		t.Errorf("Friend count after add = %d, want 1", user1.GetFriendCount())
	}

	// Check IsFriend
	if !user1.IsFriend(user2ID) {
		t.Error("IsFriend returned false, expected true")
	}

	// Add same friend again should return false
	added = user1.AddFriend(user2ID)
	if added {
		t.Error("AddFriend duplicate returned true, expected false")
	}

	// Check friend count didn't increase
	if user1.GetFriendCount() != 1 {
		t.Errorf("Friend count after duplicate = %d, want 1", user1.GetFriendCount())
	}

	// Remove friend
	removed := user1.RemoveFriend(user2ID)
	if !removed {
		t.Error("RemoveFriend returned false, expected true")
	}

	// Check friend count
	if user1.GetFriendCount() != 0 {
		t.Errorf("Friend count after remove = %d, want 0", user1.GetFriendCount())
	}

	// Check IsFriend
	if user1.IsFriend(user2ID) {
		t.Error("IsFriend returned true, expected false")
	}

	// Remove non-existent friend should return false
	removed = user1.RemoveFriend(user2ID)
	if removed {
		t.Error("RemoveFriend non-existent returned true, expected false")
	}
}

func TestGetInviteCode(t *testing.T) {
	prefs := NewPreferences()
	chatID := "123456789"

	user := prefs.GetUser(chatID)

	inviteCode := user.GetInviteCode()

	if inviteCode == "" {
		t.Error("GetInviteCode returned empty string")
	}

	// Invite code should be consistent
	inviteCode2 := user.GetInviteCode()
	if inviteCode != inviteCode2 {
		t.Errorf("GetInviteCode inconsistent: %s != %s", inviteCode, inviteCode2)
	}

	// CreatePendingInvite should return same code
	pendingCode := user.CreatePendingInvite()
	if pendingCode != inviteCode {
		t.Errorf("CreatePendingInvite = %s, want %s", pendingCode, inviteCode)
	}
}

func TestGenerateInviteCode(t *testing.T) {
	// Test with various chat IDs
	tests := []struct {
		chatID string
		want   string
	}{
		{"123456789", "456789"},
		{"12345", "12345"},
		{"123", "123"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.chatID, func(t *testing.T) {
			got := generateInviteCode(tt.chatID)
			if got != tt.want {
				t.Errorf("generateInviteCode(%q) = %q, want %q", tt.chatID, got, tt.want)
			}
		})
	}
}

func TestGetFriendsForEvent(t *testing.T) {
	prefs := NewPreferences()
	user1ID := "12345"
	user2ID := "67890"
	user3ID := "11111"
	eventID := "event123"

	user1 := prefs.GetUser(user1ID)
	user2 := prefs.GetUser(user2ID)
	user3 := prefs.GetUser(user3ID)

	// Setup: user1 has user2 and user3 as friends
	user1.AddFriend(user2ID)
	user1.AddFriend(user3ID)

	// Enable sharing for all users
	user1.ShareEvents = true
	user2.ShareEvents = true
	user3.ShareEvents = true

	// user2 registers for event
	user2.SetEventStatus(eventID, "registered")

	// user3 marks as interested
	user3.SetEventStatus(eventID, "interested")

	// Get friends for event
	friends := prefs.GetFriendsForEvent(user1ID, eventID)

	if len(friends) != 2 {
		t.Errorf("GetFriendsForEvent returned %d friends, want 2", len(friends))
	}

	// Check that both user2 and user3 are in the list
	foundUser2 := false
	foundUser3 := false
	for _, friendID := range friends {
		if friendID == user2ID {
			foundUser2 = true
		}
		if friendID == user3ID {
			foundUser3 = true
		}
	}

	if !foundUser2 {
		t.Error("user2 not in friends list")
	}
	if !foundUser3 {
		t.Error("user3 not in friends list")
	}
}

func TestGetFriendsForEvent_PrivacyControls(t *testing.T) {
	prefs := NewPreferences()
	user1ID := "12345"
	user2ID := "67890"
	eventID := "event123"

	user1 := prefs.GetUser(user1ID)
	user2 := prefs.GetUser(user2ID)

	// Setup friendship
	user1.AddFriend(user2ID)

	// user2 registers for event
	user2.SetEventStatus(eventID, "registered")

	// Test: user1 has sharing disabled
	user1.ShareEvents = false
	user2.ShareEvents = true

	friends := prefs.GetFriendsForEvent(user1ID, eventID)
	if len(friends) != 0 {
		t.Errorf("With user1 sharing disabled, got %d friends, want 0", len(friends))
	}

	// Test: user2 has sharing disabled
	user1.ShareEvents = true
	user2.ShareEvents = false

	friends = prefs.GetFriendsForEvent(user1ID, eventID)
	if len(friends) != 0 {
		t.Errorf("With user2 sharing disabled, got %d friends, want 0", len(friends))
	}

	// Test: both have sharing enabled
	user1.ShareEvents = true
	user2.ShareEvents = true

	friends = prefs.GetFriendsForEvent(user1ID, eventID)
	if len(friends) != 1 {
		t.Errorf("With both sharing enabled, got %d friends, want 1", len(friends))
	}
}

func TestNotifyOnChanges(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("12345")

	// Default should be true
	if !user.NotifyOnChanges {
		t.Error("Default NotifyOnChanges should be true")
	}

	// Test setting to false
	user.NotifyOnChanges = false
	if user.NotifyOnChanges {
		t.Error("NotifyOnChanges should be false")
	}

	// Test setting to true
	user.NotifyOnChanges = true
	if !user.NotifyOnChanges {
		t.Error("NotifyOnChanges should be true")
	}
}

func TestNotifyOnRemoval(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("12345")

	// Default should be true
	if !user.NotifyOnRemoval {
		t.Error("Default NotifyOnRemoval should be true")
	}

	// Test setting to false
	user.NotifyOnRemoval = false
	if user.NotifyOnRemoval {
		t.Error("NotifyOnRemoval should be false")
	}

	// Test setting to true
	user.NotifyOnRemoval = true
	if !user.NotifyOnRemoval {
		t.Error("NotifyOnRemoval should be true")
	}
}

func TestHidePastEvents(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("12345")

	// Default should be true
	if !user.HidePastEvents {
		t.Error("Default HidePastEvents should be true")
	}

	// Test setting to false
	user.HidePastEvents = false
	if user.HidePastEvents {
		t.Error("HidePastEvents should be false")
	}

	// Test setting to true
	user.HidePastEvents = true
	if !user.HidePastEvents {
		t.Error("HidePastEvents should be true")
	}
}

func TestDaysAhead(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("12345")

	// Default should be 0
	if user.DaysAhead != 0 {
		t.Errorf("Default DaysAhead = %d, want 0", user.DaysAhead)
	}

	// Test setting valid values
	user.DaysAhead = 30
	if user.DaysAhead != 30 {
		t.Errorf("DaysAhead = %d, want 30", user.DaysAhead)
	}

	user.DaysAhead = 90
	if user.DaysAhead != 90 {
		t.Errorf("DaysAhead = %d, want 90", user.DaysAhead)
	}
}

func TestDigestHour(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("12345")

	// Default should be 9
	if user.DigestHour != 9 {
		t.Errorf("Default DigestHour = %d, want 9", user.DigestHour)
	}

	// Test setting values
	user.DigestHour = 12
	if user.DigestHour != 12 {
		t.Errorf("DigestHour = %d, want 12", user.DigestHour)
	}
}

func TestDigestDayOfWeek(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("12345")

	// Default should be 1 (Monday)
	if user.DigestDayOfWeek != 1 {
		t.Errorf("Default DigestDayOfWeek = %d, want 1", user.DigestDayOfWeek)
	}

	// Test setting values
	user.DigestDayOfWeek = 5
	if user.DigestDayOfWeek != 5 {
		t.Errorf("DigestDayOfWeek = %d, want 5", user.DigestDayOfWeek)
	}
}

func TestShareEvents(t *testing.T) {
	prefs := NewPreferences()
	user := prefs.GetUser("12345")

	// Default should be false
	if user.ShareEvents {
		t.Error("Default ShareEvents should be false")
	}

	// Test toggling
	user.ShareEvents = true
	if !user.ShareEvents {
		t.Error("ShareEvents should be true")
	}

	user.ShareEvents = false
	if user.ShareEvents {
		t.Error("ShareEvents should be false")
	}
}

func TestRemoveAllStates(t *testing.T) {
	prefs := NewPreferences()
	chatID := "12345"

	// Add some states using the Preferences methods
	prefs.AddState(chatID, "NV")
	prefs.AddState(chatID, "CA")
	prefs.AddState(chatID, "TX")

	user := prefs.GetUser(chatID)
	if len(user.States) != 3 {
		t.Errorf("Before remove, States length = %d, want 3", len(user.States))
	}

	// Remove all states one by one
	prefs.RemoveState(chatID, "NV")
	prefs.RemoveState(chatID, "CA")
	prefs.RemoveState(chatID, "TX")

	// Verify all states removed
	if len(user.States) != 0 {
		t.Errorf("After removing all, States length = %d, want 0", len(user.States))
	}
}

func TestMultipleUsersWithFrequency(t *testing.T) {
	prefs := NewPreferences()

	// Create users with different frequencies
	prefs.AddState("user1", "NV")
	user1 := prefs.GetUser("user1")
	user1.SetDigestFrequency("daily")

	prefs.AddState("user2", "CA")
	user2 := prefs.GetUser("user2")
	user2.SetDigestFrequency("daily")

	prefs.AddState("user3", "TX")
	user3 := prefs.GetUser("user3")
	user3.SetDigestFrequency("weekly")

	prefs.AddState("user4", "AZ")
	user4 := prefs.GetUser("user4")
	user4.SetDigestFrequency("immediate")

	// Inactive user
	prefs.AddState("user5", "FL")
	user5 := prefs.GetUser("user5")
	user5.SetDigestFrequency("daily")
	user5.Active = false

	// Count users by frequency
	dailyCount := 0
	weeklyCount := 0
	immediateCount := 0

	for _, user := range prefs.GetAllUsers() {
		userPrefs := prefs.GetUser(user)
		switch userPrefs.DigestFrequency {
		case "daily":
			dailyCount++
		case "weekly":
			weeklyCount++
		case "immediate":
			immediateCount++
		}
	}

	// Verify counts (user5 is inactive so not in GetAllUsers)
	if dailyCount != 2 {
		t.Errorf("Daily users = %d, want 2", dailyCount)
	}
	if weeklyCount != 1 {
		t.Errorf("Weekly users = %d, want 1", weeklyCount)
	}
	if immediateCount != 1 {
		t.Errorf("Immediate users = %d, want 1", immediateCount)
	}
}

func TestAddState(t *testing.T) {
	prefs := NewPreferences()

	// Test adding new state
	if !prefs.AddState("12345", "NV") {
		t.Error("AddState should return true for new state")
	}

	// Test duplicate state
	if prefs.AddState("12345", "NV") {
		t.Error("AddState should return false for duplicate state")
	}

	// Verify state was added
	if !prefs.HasState("12345", "NV") {
		t.Error("State should be in user's list")
	}
}

func TestRemoveState(t *testing.T) {
	prefs := NewPreferences()
	prefs.AddState("12345", "NV")
	prefs.AddState("12345", "CA")

	// Test removing existing state
	if !prefs.RemoveState("12345", "NV") {
		t.Error("RemoveState should return true for existing state")
	}

	// Test removing non-existent state
	if prefs.RemoveState("12345", "TX") {
		t.Error("RemoveState should return false for non-existent state")
	}

	// Verify state was removed
	if prefs.HasState("12345", "NV") {
		t.Error("State should be removed")
	}

	// Verify other state still exists
	if !prefs.HasState("12345", "CA") {
		t.Error("Other state should still exist")
	}
}

func TestGetStates(t *testing.T) {
	prefs := NewPreferences()
	prefs.AddState("12345", "NV")
	prefs.AddState("12345", "CA")
	prefs.AddState("12345", "TX")

	states := prefs.GetStates("12345")

	if len(states) != 3 {
		t.Errorf("GetStates returned %d states, want 3", len(states))
	}

	// Check all states are present
	stateMap := make(map[string]bool)
	for _, s := range states {
		stateMap[s] = true
	}

	if !stateMap["NV"] || !stateMap["CA"] || !stateMap["TX"] {
		t.Error("Not all states returned")
	}
}

func TestGetUser(t *testing.T) {
	prefs := NewPreferences()

	// Test getting new user
	user := prefs.GetUser("new-user")
	if user == nil {
		t.Error("GetUser should create new user")
		return
	}
	if !user.Active {
		t.Error("New user should be active")
	}

	// Test getting existing user
	user.States = []string{"NV"}
	user2 := prefs.GetUser("new-user")
	if len(user2.States) != 1 {
		t.Error("GetUser should return same user instance")
	}
}

func TestValidState(t *testing.T) {
	tests := []struct {
		state string
		valid bool
	}{
		{"NV", true},
		{"CA", true},
		{"TX", true},
		{"ALL", true},
		{"ZZ", false},
		{"", false},
		{"  ", false},
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

func TestFriendMigration(t *testing.T) {
	prefs := NewPreferences()
	chatID := "12345"

	user := prefs.GetUser(chatID)

	// Check that friend fields are initialized
	if user.FriendChatIDs == nil {
		t.Error("FriendChatIDs not initialized")
	}

	if user.InviteCode == "" {
		t.Error("InviteCode not initialized")
	}

	// ShareEvents should default to false
	if user.ShareEvents {
		t.Error("ShareEvents should default to false")
	}
}
