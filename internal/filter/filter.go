// Package filter provides event filtering capabilities for the VGA Events bot.
//
// Users can create filters to narrow down event notifications based on various criteria:
//   - Date ranges (from/to dates)
//   - Course names (substring matching, case-insensitive)
//   - Cities (substring matching, case-insensitive)
//   - States (subset of user's subscribed states)
//   - Weekends only (Saturday/Sunday)
//   - Maximum price (placeholder for future use)
//
// Filters can be saved as named presets and loaded later for reuse.
//
// Example usage:
//
//	// Create a filter for weekend events at Pebble Beach
//	f := filter.NewFilter()
//	f.WeekendsOnly = true
//	f.Courses = []string{"Pebble Beach"}
//
//	// Apply filter to events
//	filtered := f.Apply(events)
//
//	// Save as preset
//	preset := filter.NewFilterPreset("weekend-pebble", f)
package filter

import (
	"fmt"
	"strings"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// Filter represents event filtering criteria
type Filter struct {
	// Date range filtering
	DateFrom *time.Time `json:"date_from,omitempty"`
	DateTo   *time.Time `json:"date_to,omitempty"`

	// Course name filtering (case-insensitive substring match)
	Courses []string `json:"courses,omitempty"`

	// Weekend-only filtering (Saturday/Sunday)
	WeekendsOnly bool `json:"weekends_only,omitempty"`

	// State filtering (in addition to user's subscribed states)
	// This allows filtering within subscribed states
	States []string `json:"states,omitempty"`

	// City filtering (case-insensitive substring match)
	Cities []string `json:"cities,omitempty"`

	// Price filtering (if available in future)
	MaxPrice float64 `json:"max_price,omitempty"`
}

// FilterPreset represents a saved filter configuration
type FilterPreset struct {
	Name      string    `json:"name"`
	Filter    *Filter   `json:"filter"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewFilter creates a new empty filter with no active criteria.
// The filter will match all events until criteria are added.
func NewFilter() *Filter {
	return &Filter{
		Courses: []string{},
		States:  []string{},
		Cities:  []string{},
	}
}

// NewFilterPreset creates a new filter preset with the given name and filter.
// Created and Updated timestamps are set to the current time.
func NewFilterPreset(name string, f *Filter) *FilterPreset {
	now := time.Now().UTC()
	return &FilterPreset{
		Name:      name,
		Filter:    f,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsEmpty checks if the filter has any active criteria.
// Returns true if the filter would match all events.
func (f *Filter) IsEmpty() bool {
	return f.DateFrom == nil &&
		f.DateTo == nil &&
		len(f.Courses) == 0 &&
		!f.WeekendsOnly &&
		len(f.States) == 0 &&
		len(f.Cities) == 0 &&
		f.MaxPrice == 0
}

// Matches checks if an event matches all active filter criteria.
// Returns true if the event passes all filters, false otherwise.
// An empty filter matches all events.
//
// Matching logic:
//   - Date range: Event date must be within DateFrom and DateTo (inclusive)
//   - Courses: Event title must contain at least one course name (case-insensitive)
//   - Cities: Event city must contain at least one city name (case-insensitive)
//   - States: Event state must match at least one state code (case-insensitive)
//   - WeekendsOnly: Event must be on Saturday or Sunday
//   - MaxPrice: Currently a no-op, reserved for future use
func (f *Filter) Matches(evt *event.Event) bool {
	// Empty filter matches all events
	if f.IsEmpty() {
		return true
	}

	// Parse event date for date-based filtering
	eventDate := parseEventDate(evt.DateText)

	// Check date range
	if f.DateFrom != nil && eventDate != nil {
		if eventDate.Before(*f.DateFrom) {
			return false
		}
	}

	if f.DateTo != nil && eventDate != nil {
		if eventDate.After(*f.DateTo) {
			return false
		}
	}

	// Check weekends only
	if f.WeekendsOnly && eventDate != nil {
		weekday := eventDate.Weekday()
		if weekday != time.Saturday && weekday != time.Sunday {
			return false
		}
	}

	// Check course name (case-insensitive substring match)
	if len(f.Courses) > 0 {
		matched := false
		titleLower := strings.ToLower(evt.Title)
		for _, course := range f.Courses {
			if strings.Contains(titleLower, strings.ToLower(course)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check state filtering
	if len(f.States) > 0 {
		matched := false
		for _, state := range f.States {
			if strings.EqualFold(evt.State, state) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check city filtering (case-insensitive substring match)
	if len(f.Cities) > 0 {
		matched := false
		cityLower := strings.ToLower(evt.City)
		for _, city := range f.Cities {
			if strings.Contains(cityLower, strings.ToLower(city)) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check max price (placeholder for future implementation)
	// Price data would need to be added to Event struct
	// TODO: Implement when price data is available
	_ = f.MaxPrice // Placeholder for future price filtering

	return true
}

// Apply applies the filter to a list of events and returns only matching events.
// If the filter is empty, returns the original list unchanged.
// Otherwise, returns a new slice containing only events that match all criteria.
func (f *Filter) Apply(events []*event.Event) []*event.Event {
	if f.IsEmpty() {
		return events
	}

	var filtered []*event.Event
	for _, evt := range events {
		if f.Matches(evt) {
			filtered = append(filtered, evt)
		}
	}

	return filtered
}

// String returns a human-readable description of the active filter criteria.
// Returns "No active filters" if the filter is empty.
// Format: "From: Jan 2, 2026 | To: Jan 15, 2026 | Courses: Pebble Beach | Weekends only"
func (f *Filter) String() string {
	if f.IsEmpty() {
		return "No active filters"
	}

	var parts []string

	if f.DateFrom != nil {
		parts = append(parts, fmt.Sprintf("From: %s", f.DateFrom.Format("Jan 2, 2006")))
	}

	if f.DateTo != nil {
		parts = append(parts, fmt.Sprintf("To: %s", f.DateTo.Format("Jan 2, 2006")))
	}

	if len(f.Courses) > 0 {
		parts = append(parts, fmt.Sprintf("Courses: %s", strings.Join(f.Courses, ", ")))
	}

	if f.WeekendsOnly {
		parts = append(parts, "Weekends only")
	}

	if len(f.States) > 0 {
		parts = append(parts, fmt.Sprintf("States: %s", strings.Join(f.States, ", ")))
	}

	if len(f.Cities) > 0 {
		parts = append(parts, fmt.Sprintf("Cities: %s", strings.Join(f.Cities, ", ")))
	}

	if f.MaxPrice > 0 {
		parts = append(parts, fmt.Sprintf("Max price: $%.2f", f.MaxPrice))
	}

	return strings.Join(parts, " | ")
}

// parseEventDate attempts to parse an event's date text
// Returns nil if parsing fails
func parseEventDate(dateText string) *time.Time {
	// Try common date formats
	formats := []string{
		"January 2, 2006",
		"Jan 2, 2006",
		"1/2/2006",
		"01/02/2006",
		"2006-01-02",
		"Jan 2",
		"January 2",
	}

	// Normalize the date text
	normalized := strings.TrimSpace(dateText)

	// For formats without year, add current year
	currentYear := time.Now().Year()
	yearlessFormats := []string{"Jan 2", "January 2"}

	for _, format := range formats {
		t, err := time.Parse(format, normalized)
		if err == nil {
			return &t
		}
	}

	// Try yearless formats with current year appended
	for _, format := range yearlessFormats {
		t, err := time.Parse(format, normalized)
		if err == nil {
			// Add current year
			t = time.Date(currentYear, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			// If date is in the past, assume next year
			if t.Before(time.Now()) {
				t = time.Date(currentYear+1, t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
			}
			return &t
		}
	}

	return nil
}

// Clone creates a deep copy of the filter.
// All slices and pointers are copied to new memory locations,
// ensuring modifications to the clone don't affect the original.
func (f *Filter) Clone() *Filter {
	clone := &Filter{
		WeekendsOnly: f.WeekendsOnly,
		MaxPrice:     f.MaxPrice,
	}

	if f.DateFrom != nil {
		df := *f.DateFrom
		clone.DateFrom = &df
	}

	if f.DateTo != nil {
		dt := *f.DateTo
		clone.DateTo = &dt
	}

	if len(f.Courses) > 0 {
		clone.Courses = make([]string, len(f.Courses))
		copy(clone.Courses, f.Courses)
	} else {
		clone.Courses = []string{}
	}

	if len(f.States) > 0 {
		clone.States = make([]string, len(f.States))
		copy(clone.States, f.States)
	} else {
		clone.States = []string{}
	}

	if len(f.Cities) > 0 {
		clone.Cities = make([]string, len(f.Cities))
		copy(clone.Cities, f.Cities)
	} else {
		clone.Cities = []string{}
	}

	return clone
}
