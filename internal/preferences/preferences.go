package preferences

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

const (
	// DigestFrequency constants
	DigestFrequencyImmediate = "immediate"
	DigestFrequencyDaily     = "daily"
	DigestFrequencyWeekly    = "weekly"

	// EventStatus constants
	EventStatusInterested = "interested"
	EventStatusRegistered = "registered"
	EventStatusMaybe      = "maybe"
	EventStatusSkip       = "skip"
)

// UserPreferences represents a user's subscription preferences
type UserPreferences struct {
	// Core subscription settings
	States []string `json:"states"`
	Active bool     `json:"active"`

	// Event history tracking (Feature 1)
	// Key: event.ID, Value: Unix timestamp when first seen
	SeenEventIDs map[string]int64 `json:"seen_event_ids,omitempty"`

	// Digest mode configuration (Feature 4)
	DigestFrequency string         `json:"digest_frequency,omitempty"`   // "immediate", "daily", "weekly"
	DigestDayOfWeek int            `json:"digest_day_of_week,omitempty"` // 0-6 for weekly digest
	DigestHour      int            `json:"digest_hour,omitempty"`        // 0-23 UTC
	PendingEvents   []*event.Event `json:"pending_events,omitempty"`     // Events queued for digest

	// Time-based filtering (Feature 3)
	DaysAhead      int  `json:"days_ahead,omitempty"`       // 0 = disabled, >0 = only show events within N days
	HidePastEvents bool `json:"hide_past_events,omitempty"` // Default: true

	// Event status tracking (Feature 9)
	// Key: event.ID, Value: status ("interested", "registered", "maybe", "skip")
	EventStatuses map[string]string `json:"event_statuses,omitempty"`

	// Event reminders (Week 3)
	// Days before event to send reminders (e.g., [1, 3, 7] means 1 day, 3 days, and 1 week before)
	ReminderDays []int `json:"reminder_days,omitempty"`

	// Personal event notes
	// Key: event.ID, Value: user's personal note
	EventNotes map[string]string `json:"event_notes,omitempty"`
}

// Preferences maps chat IDs to user preferences
type Preferences map[string]*UserPreferences

// Storage defines the interface for preferences storage
type Storage interface {
	Load() (Preferences, error)
	Save(prefs Preferences) error
}

// NewPreferences creates a new empty preferences map
func NewPreferences() Preferences {
	return make(Preferences)
}

// GetUser retrieves preferences for a specific user, creating them if they don't exist.
// For existing users, initializes new fields with default values (migration).
func (p Preferences) GetUser(chatID string) *UserPreferences {
	user, exists := p[chatID]

	if exists {
		// Migration: initialize new fields if they don't exist
		if user.SeenEventIDs == nil {
			user.SeenEventIDs = make(map[string]int64)
		}
		if user.DigestFrequency == "" {
			user.DigestFrequency = DigestFrequencyImmediate // Keep current behavior
		}
		if user.DigestHour == 0 && user.DigestFrequency != "immediate" {
			user.DigestHour = 9 // 9 AM UTC default
		}
		if user.DigestDayOfWeek == 0 && user.DigestFrequency == "weekly" {
			user.DigestDayOfWeek = 1 // Monday default
		}
		if user.EventStatuses == nil {
			user.EventStatuses = make(map[string]string)
		}
		if user.EventNotes == nil {
			user.EventNotes = make(map[string]string)
		}
		// Note: HidePastEvents defaults to false (zero value) for backward compatibility
		// Users can enable it via settings
		return user
	}

	// Create new user with default preferences
	p[chatID] = &UserPreferences{
		States:          []string{},
		Active:          true,
		SeenEventIDs:    make(map[string]int64),
		DigestFrequency: DigestFrequencyImmediate,
		DigestHour:      9,
		DigestDayOfWeek: 1,
		DaysAhead:       0,    // Disabled by default
		HidePastEvents:  true, // New users hide past events by default
		PendingEvents:   []*event.Event{},
		EventStatuses:   make(map[string]string),
		ReminderDays:    []int{}, // No reminders by default, user can configure
		EventNotes:      make(map[string]string),
	}
	return p[chatID]
}

// AddState adds a state to a user's subscriptions
func (p Preferences) AddState(chatID, state string) bool {
	state = strings.ToUpper(strings.TrimSpace(state))
	if !IsValidState(state) {
		return false
	}

	user := p.GetUser(chatID)

	// Check if already subscribed
	for _, s := range user.States {
		if s == state {
			return false // Already subscribed
		}
	}

	user.States = append(user.States, state)
	return true
}

// RemoveState removes a state from a user's subscriptions
func (p Preferences) RemoveState(chatID, state string) bool {
	state = strings.ToUpper(strings.TrimSpace(state))

	user, exists := p[chatID]
	if !exists {
		return false
	}

	// Find and remove the state
	for i, s := range user.States {
		if s == state {
			user.States = append(user.States[:i], user.States[i+1:]...)
			return true
		}
	}

	return false
}

// GetStates returns all states a user is subscribed to
func (p Preferences) GetStates(chatID string) []string {
	if user, exists := p[chatID]; exists {
		return user.States
	}
	return []string{}
}

// HasState checks if a user is subscribed to a specific state
func (p Preferences) HasState(chatID, state string) bool {
	state = strings.ToUpper(strings.TrimSpace(state))
	states := p.GetStates(chatID)
	for _, s := range states {
		if s == state {
			return true
		}
	}
	return false
}

// GetAllUsers returns all chat IDs with active subscriptions
func (p Preferences) GetAllUsers() []string {
	users := make([]string, 0, len(p))
	for chatID, user := range p {
		if user.Active && len(user.States) > 0 {
			users = append(users, chatID)
		}
	}
	return users
}

// ToJSON marshals preferences to JSON
func (p Preferences) ToJSON() ([]byte, error) {
	return json.MarshalIndent(p, "", "  ")
}

// FromJSON unmarshals preferences from JSON
func FromJSON(data []byte) (Preferences, error) {
	var prefs Preferences
	if err := json.Unmarshal(data, &prefs); err != nil {
		return nil, fmt.Errorf("unmarshaling preferences: %w", err)
	}
	return prefs, nil
}

// IsValidState checks if a state code is valid
func IsValidState(state string) bool {
	state = strings.ToUpper(strings.TrimSpace(state))

	// Special case for ALL
	if state == "ALL" {
		return true
	}

	// Valid US state codes
	validStates := map[string]bool{
		"AL": true, "AK": true, "AZ": true, "AR": true, "CA": true,
		"CO": true, "CT": true, "DE": true, "FL": true, "GA": true,
		"HI": true, "ID": true, "IL": true, "IN": true, "IA": true,
		"KS": true, "KY": true, "LA": true, "ME": true, "MD": true,
		"MA": true, "MI": true, "MN": true, "MS": true, "MO": true,
		"MT": true, "NE": true, "NV": true, "NH": true, "NJ": true,
		"NM": true, "NY": true, "NC": true, "ND": true, "OH": true,
		"OK": true, "OR": true, "PA": true, "RI": true, "SC": true,
		"SD": true, "TN": true, "TX": true, "UT": true, "VT": true,
		"VA": true, "WA": true, "WV": true, "WI": true, "WY": true,
		"DC": true, // Washington, D.C.
	}

	return validStates[state]
}

// GetStateName returns the full name of a state given its code
func GetStateName(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))

	stateNames := map[string]string{
		"AL": "Alabama", "AK": "Alaska", "AZ": "Arizona", "AR": "Arkansas",
		"CA": "California", "CO": "Colorado", "CT": "Connecticut", "DE": "Delaware",
		"FL": "Florida", "GA": "Georgia", "HI": "Hawaii", "ID": "Idaho",
		"IL": "Illinois", "IN": "Indiana", "IA": "Iowa", "KS": "Kansas",
		"KY": "Kentucky", "LA": "Louisiana", "ME": "Maine", "MD": "Maryland",
		"MA": "Massachusetts", "MI": "Michigan", "MN": "Minnesota", "MS": "Mississippi",
		"MO": "Missouri", "MT": "Montana", "NE": "Nebraska", "NV": "Nevada",
		"NH": "New Hampshire", "NJ": "New Jersey", "NM": "New Mexico", "NY": "New York",
		"NC": "North Carolina", "ND": "North Dakota", "OH": "Ohio", "OK": "Oklahoma",
		"OR": "Oregon", "PA": "Pennsylvania", "RI": "Rhode Island", "SC": "South Carolina",
		"SD": "South Dakota", "TN": "Tennessee", "TX": "Texas", "UT": "Utah",
		"VT": "Vermont", "VA": "Virginia", "WA": "Washington", "WV": "West Virginia",
		"WI": "Wisconsin", "WY": "Wyoming", "DC": "Washington, D.C.",
		"ALL": "All States",
	}

	if name, exists := stateNames[code]; exists {
		return name
	}
	return code
}

// CleanupOldHistory removes event history entries older than the specified number of days.
// This prevents SeenEventIDs from growing unbounded.
func (u *UserPreferences) CleanupOldHistory(daysToKeep int) int {
	if u.SeenEventIDs == nil {
		return 0
	}

	cutoff := time.Now().AddDate(0, 0, -daysToKeep).Unix()
	removed := 0

	for eventID, timestamp := range u.SeenEventIDs {
		if timestamp < cutoff {
			delete(u.SeenEventIDs, eventID)
			removed++
		}
	}

	return removed
}

// MarkEventSeen records that a user has seen a specific event.
func (u *UserPreferences) MarkEventSeen(eventID string) {
	if u.SeenEventIDs == nil {
		u.SeenEventIDs = make(map[string]int64)
	}
	u.SeenEventIDs[eventID] = time.Now().Unix()
}

// HasSeenEvent checks if a user has already seen a specific event.
func (u *UserPreferences) HasSeenEvent(eventID string) bool {
	if u.SeenEventIDs == nil {
		return false
	}
	_, seen := u.SeenEventIDs[eventID]
	return seen
}

// AddPendingEvent adds an event to the user's pending digest queue.
func (u *UserPreferences) AddPendingEvent(evt *event.Event) {
	if u.PendingEvents == nil {
		u.PendingEvents = []*event.Event{}
	}
	u.PendingEvents = append(u.PendingEvents, evt)
}

// ClearPendingEvents removes all pending events (called after digest is sent).
func (u *UserPreferences) ClearPendingEvents() {
	u.PendingEvents = []*event.Event{}
}

// SetDigestFrequency updates the digest frequency setting.
// Valid values: "immediate", "daily", "weekly"
func (u *UserPreferences) SetDigestFrequency(frequency string) bool {
	frequency = strings.ToLower(strings.TrimSpace(frequency))
	if frequency != DigestFrequencyImmediate && frequency != DigestFrequencyDaily && frequency != DigestFrequencyWeekly {
		return false
	}
	u.DigestFrequency = frequency
	return true
}

// SetEventStatus sets the status for an event.
// Valid statuses: "interested", "registered", "maybe", "skip"
func (u *UserPreferences) SetEventStatus(eventID, status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != EventStatusInterested && status != EventStatusRegistered &&
		status != EventStatusMaybe && status != EventStatusSkip {
		return false
	}

	if u.EventStatuses == nil {
		u.EventStatuses = make(map[string]string)
	}

	u.EventStatuses[eventID] = status
	return true
}

// GetEventStatus returns the status of an event, or empty string if not set.
func (u *UserPreferences) GetEventStatus(eventID string) string {
	if u.EventStatuses == nil {
		return ""
	}
	return u.EventStatuses[eventID]
}

// RemoveEventStatus removes the status for an event.
func (u *UserPreferences) RemoveEventStatus(eventID string) {
	if u.EventStatuses != nil {
		delete(u.EventStatuses, eventID)
	}
}

// GetEventsByStatus returns all event IDs with a specific status.
func (u *UserPreferences) GetEventsByStatus(status string) []string {
	if u.EventStatuses == nil {
		return []string{}
	}

	var eventIDs []string
	for eventID, eventStatus := range u.EventStatuses {
		if eventStatus == status {
			eventIDs = append(eventIDs, eventID)
		}
	}
	return eventIDs
}

// IsValidEventStatus checks if a status string is valid.
func IsValidEventStatus(status string) bool {
	status = strings.ToLower(strings.TrimSpace(status))
	return status == EventStatusInterested || status == EventStatusRegistered ||
		status == EventStatusMaybe || status == EventStatusSkip
}

// SetReminderDays sets the reminder days for a user.
// Valid values: 1, 3, 7, 14 (1 day, 3 days, 1 week, 2 weeks)
func (u *UserPreferences) SetReminderDays(days []int) bool {
	// Validate all days
	for _, day := range days {
		if day != 1 && day != 3 && day != 7 && day != 14 {
			return false
		}
	}
	u.ReminderDays = days
	return true
}

// HasReminderDay checks if reminders are enabled for a specific number of days before.
func (u *UserPreferences) HasReminderDay(day int) bool {
	for _, d := range u.ReminderDays {
		if d == day {
			return true
		}
	}
	return false
}

// SetEventNote sets a personal note for an event.
func (u *UserPreferences) SetEventNote(eventID, note string) {
	if u.EventNotes == nil {
		u.EventNotes = make(map[string]string)
	}
	u.EventNotes[eventID] = note
}

// GetEventNote retrieves the note for an event.
// Returns empty string if no note exists.
func (u *UserPreferences) GetEventNote(eventID string) string {
	if u.EventNotes == nil {
		return ""
	}
	return u.EventNotes[eventID]
}

// RemoveEventNote removes a note for an event.
func (u *UserPreferences) RemoveEventNote(eventID string) {
	if u.EventNotes != nil {
		delete(u.EventNotes, eventID)
	}
}
