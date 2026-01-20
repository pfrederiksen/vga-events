package preferences

import (
	"encoding/json"
	"fmt"
	"strings"
)

// UserPreferences represents a user's subscription preferences
type UserPreferences struct {
	States []string `json:"states"`
	Active bool     `json:"active"`
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

// GetUser retrieves preferences for a specific user, creating them if they don't exist
func (p Preferences) GetUser(chatID string) *UserPreferences {
	if user, exists := p[chatID]; exists {
		return user
	}
	// Create new user with default preferences
	p[chatID] = &UserPreferences{
		States: []string{},
		Active: true,
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
