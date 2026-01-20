package cli

import (
	"sort"
	"strings"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// SortOrder represents the available sorting options
type SortOrder string

const (
	SortByDate  SortOrder = "date"
	SortByState SortOrder = "state"
	SortByTitle SortOrder = "title"
)

// sortEvents sorts a slice of events based on the specified sort order
func sortEvents(events []*event.Event, sortOrder SortOrder) {
	switch sortOrder {
	case SortByDate:
		sort.Slice(events, func(i, j int) bool {
			return compareByDate(events[i], events[j])
		})
	case SortByState:
		sort.Slice(events, func(i, j int) bool {
			if events[i].State != events[j].State {
				return events[i].State < events[j].State
			}
			// If states are equal, sort by date
			return compareByDate(events[i], events[j])
		})
	case SortByTitle:
		sort.Slice(events, func(i, j int) bool {
			if events[i].Title != events[j].Title {
				return strings.ToLower(events[i].Title) < strings.ToLower(events[j].Title)
			}
			// If titles are equal, sort by date
			return compareByDate(events[i], events[j])
		})
	}
}

// compareByDate compares two events by their date
// Returns true if event i should come before event j
func compareByDate(i, j *event.Event) bool {
	dateI := parseEventDate(i.DateText)
	dateJ := parseEventDate(j.DateText)

	// If both dates are valid, compare them
	if !dateI.IsZero() && !dateJ.IsZero() {
		return dateI.Before(dateJ)
	}

	// If only one date is valid, put the valid one first
	if !dateI.IsZero() {
		return true
	}
	if !dateJ.IsZero() {
		return false
	}

	// If neither has a valid date, sort by state then title
	if i.State != j.State {
		return i.State < j.State
	}
	return strings.ToLower(i.Title) < strings.ToLower(j.Title)
}

// parseEventDate attempts to parse the DateText field into a time.Time
// Supports formats like "Mar 13 2026", "4.4.26", "Jan 24", "02/15/26"
func parseEventDate(dateText string) time.Time {
	if dateText == "" {
		return time.Time{}
	}

	// Try "Mar 13 2026" format
	t, err := time.Parse("Jan 02 2006", dateText)
	if err == nil {
		return t
	}

	// Try "Jan 2 2026" format (single digit day)
	t, err = time.Parse("Jan 2 2006", dateText)
	if err == nil {
		return t
	}

	// Try "4.4.26" format (month.day.year)
	t, err = time.Parse("1.2.06", dateText)
	if err == nil {
		return t
	}

	// Try "04.04.26" format
	t, err = time.Parse("01.02.06", dateText)
	if err == nil {
		return t
	}

	// Try "02/15/26" format
	t, err = time.Parse("01/02/06", dateText)
	if err == nil {
		return t
	}

	// Try "Jan 24" format (no year, assume current year)
	t, err = time.Parse("Jan 02", dateText)
	if err == nil {
		// Add the current year
		now := time.Now()
		return time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}

	// Try "Jan 2" format (single digit day, no year)
	t, err = time.Parse("Jan 2", dateText)
	if err == nil {
		now := time.Now()
		return time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}

	// Could not parse, return zero time
	return time.Time{}
}
