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

	// Change notifications (v0.5.0 Enhancement #3)
	// Whether to be notified when tracked events change (date, title, city)
	NotifyOnChanges bool `json:"notify_on_changes,omitempty"` // Default: true

	// Weekly statistics (v0.5.0 Enhancement #4)
	WeeklyStats  *WeeklyStats              `json:"weekly_stats,omitempty"`
	StatsHistory map[string]*WeeklyStats   `json:"stats_history,omitempty"` // week key → stats
	EnableStats  bool                      `json:"enable_stats,omitempty"`  // Default: true

	// Friends and sharing (v0.5.0 Enhancement #7)
	FriendChatIDs      []string          `json:"friend_chat_ids,omitempty"`       // List of friend chat IDs
	PendingInvites     map[string]string `json:"pending_invites,omitempty"`       // invite code → sender chat ID
	ShareEvents        bool              `json:"share_events,omitempty"`          // Default: false (privacy)
	InviteCode         string            `json:"invite_code,omitempty"`           // This user's invite code
	GroupSubscriptions map[string][]string `json:"group_subscriptions,omitempty"` // group ID → member chat IDs
}

// WeeklyStats tracks user engagement metrics for a week
type WeeklyStats struct {
	WeekStart        time.Time      `json:"week_start"`
	EventsViewed     int            `json:"events_viewed"`
	EventsMarked     map[string]int `json:"events_marked"`     // status → count
	EventsRegistered int            `json:"events_registered"` // Count of events marked as registered
	TopStates        []string       `json:"top_states"`        // States with most activity
}

// NewWeeklyStats creates a new WeeklyStats for the current week
func NewWeeklyStats() *WeeklyStats {
	return &WeeklyStats{
		WeekStart:    time.Now().UTC().Truncate(24 * time.Hour), // Start of today
		EventsViewed: 0,
		EventsMarked: make(map[string]int),
	}
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
		// Migration: enable change notifications for existing users (if they have tracked events)
		if !user.NotifyOnChanges && len(user.EventStatuses) > 0 {
			user.NotifyOnChanges = true
		}
		// Migration: initialize weekly stats for existing users
		if user.WeeklyStats == nil {
			user.WeeklyStats = NewWeeklyStats()
		}
		if user.StatsHistory == nil {
			user.StatsHistory = make(map[string]*WeeklyStats)
		}
		if !user.EnableStats && len(user.States) > 0 {
			user.EnableStats = true // Enable for existing users
		}
		// Migration: initialize friend fields for existing users
		if user.FriendChatIDs == nil {
			user.FriendChatIDs = []string{}
		}
		if user.PendingInvites == nil {
			user.PendingInvites = make(map[string]string)
		}
		if user.InviteCode == "" {
			user.InviteCode = generateInviteCode(chatID)
		}
		if user.GroupSubscriptions == nil {
			user.GroupSubscriptions = make(map[string][]string)
		}
		// Note: ShareEvents defaults to false (zero value) - user must opt in
		// Note: HidePastEvents defaults to false (zero value) for backward compatibility
		// Users can enable it via settings
		return user
	}

	// Create new user with default preferences
	p[chatID] = &UserPreferences{
		States:             []string{},
		Active:             true,
		SeenEventIDs:       make(map[string]int64),
		DigestFrequency:    DigestFrequencyImmediate,
		DigestHour:         9,
		DigestDayOfWeek:    1,
		DaysAhead:          0,    // Disabled by default
		HidePastEvents:     true, // New users hide past events by default
		PendingEvents:      []*event.Event{},
		EventStatuses:      make(map[string]string),
		ReminderDays:       []int{}, // No reminders by default, user can configure
		EventNotes:         make(map[string]string),
		NotifyOnChanges:    true,              // New feature: notify about event changes
		WeeklyStats:        NewWeeklyStats(),  // New feature: track weekly stats
		StatsHistory:       make(map[string]*WeeklyStats),
		EnableStats:        true,              // Enable stats tracking by default
		FriendChatIDs:      []string{},        // New feature: friends list
		PendingInvites:     make(map[string]string),
		ShareEvents:        false,             // Privacy: opt-in only
		InviteCode:         generateInviteCode(chatID),
		GroupSubscriptions: make(map[string][]string),
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

// IncrementEventsViewed increments the events viewed counter
func (u *UserPreferences) IncrementEventsViewed(count int) {
	if !u.EnableStats {
		return
	}
	if u.WeeklyStats == nil {
		u.WeeklyStats = NewWeeklyStats()
	}
	u.WeeklyStats.EventsViewed += count
}

// IncrementEventStatus increments the counter for a specific status
func (u *UserPreferences) IncrementEventStatus(status string) {
	if !u.EnableStats {
		return
	}
	if u.WeeklyStats == nil {
		u.WeeklyStats = NewWeeklyStats()
	}
	if u.WeeklyStats.EventsMarked == nil {
		u.WeeklyStats.EventsMarked = make(map[string]int)
	}
	u.WeeklyStats.EventsMarked[status]++

	// Track registered events separately
	if status == EventStatusRegistered {
		u.WeeklyStats.EventsRegistered++
	}
}

// GetWeekKey generates a week key for stats history (format: "2026-W01")
func GetWeekKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// ArchiveCurrentWeek moves current week stats to history and starts a new week
func (u *UserPreferences) ArchiveCurrentWeek() {
	if u.WeeklyStats == nil {
		return
	}

	// Generate key for current week
	weekKey := GetWeekKey(u.WeeklyStats.WeekStart)

	// Store in history
	if u.StatsHistory == nil {
		u.StatsHistory = make(map[string]*WeeklyStats)
	}
	u.StatsHistory[weekKey] = u.WeeklyStats

	// Start new week
	u.WeeklyStats = NewWeeklyStats()
}

// GetAllTimeStats aggregates stats from all history plus current week
func (u *UserPreferences) GetAllTimeStats() *WeeklyStats {
	total := &WeeklyStats{
		WeekStart:    u.WeeklyStats.WeekStart,
		EventsViewed: u.WeeklyStats.EventsViewed,
		EventsMarked: make(map[string]int),
	}

	// Add current week
	for status, count := range u.WeeklyStats.EventsMarked {
		total.EventsMarked[status] += count
	}
	total.EventsRegistered = u.WeeklyStats.EventsRegistered

	// Add history
	for _, stats := range u.StatsHistory {
		total.EventsViewed += stats.EventsViewed
		total.EventsRegistered += stats.EventsRegistered
		for status, count := range stats.EventsMarked {
			total.EventsMarked[status] += count
		}
	}

	return total
}

// Friend Management Methods

// generateInviteCode generates a unique invite code for a user
func generateInviteCode(chatID string) string {
	// Use last 6 characters of chat ID for simplicity
	// In production, could use a proper random code generator
	if len(chatID) >= 6 {
		return chatID[len(chatID)-6:]
	}
	return chatID
}

// AddFriend adds a friend to the user's friend list
func (u *UserPreferences) AddFriend(friendChatID string) bool {
	// Check if already friends
	for _, id := range u.FriendChatIDs {
		if id == friendChatID {
			return false // Already friends
		}
	}

	u.FriendChatIDs = append(u.FriendChatIDs, friendChatID)
	return true
}

// RemoveFriend removes a friend from the user's friend list
func (u *UserPreferences) RemoveFriend(friendChatID string) bool {
	for i, id := range u.FriendChatIDs {
		if id == friendChatID {
			// Remove by swapping with last element
			u.FriendChatIDs[i] = u.FriendChatIDs[len(u.FriendChatIDs)-1]
			u.FriendChatIDs = u.FriendChatIDs[:len(u.FriendChatIDs)-1]
			return true
		}
	}
	return false
}

// IsFriend checks if a user is a friend
func (u *UserPreferences) IsFriend(chatID string) bool {
	for _, id := range u.FriendChatIDs {
		if id == chatID {
			return true
		}
	}
	return false
}

// GetFriendCount returns the number of friends
func (u *UserPreferences) GetFriendCount() int {
	return len(u.FriendChatIDs)
}

// CreatePendingInvite creates a pending invite with a code
func (u *UserPreferences) CreatePendingInvite() string {
	return u.InviteCode
}

// GetFriendsForEvent returns list of friend chat IDs who are registered/interested in an event
func (p Preferences) GetFriendsForEvent(chatID, eventID string) []string {
	user := p.GetUser(chatID)
	if !user.ShareEvents {
		return []string{} // Privacy: sharing disabled
	}

	var friendsForEvent []string
	for _, friendChatID := range user.FriendChatIDs {
		friend := p.GetUser(friendChatID)
		if !friend.ShareEvents {
			continue // Friend has sharing disabled
		}

		// Check if friend is registered or interested
		status := friend.GetEventStatus(eventID)
		if status == EventStatusInterested || status == EventStatusRegistered {
			friendsForEvent = append(friendsForEvent, friendChatID)
		}
	}

	return friendsForEvent
}
