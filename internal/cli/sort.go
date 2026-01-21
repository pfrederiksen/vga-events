package cli

import (
	"sort"
	"strings"

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
	dateI := event.ParseDate(i.DateText)
	dateJ := event.ParseDate(j.DateText)

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
