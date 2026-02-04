package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// OutputFormat specifies the output format
type OutputFormat string

const (
	FormatText OutputFormat = "text"
	FormatJSON OutputFormat = "json"
)

// OutputResult contains data to be output
type OutputResult struct {
	CheckedAt      time.Time                 `json:"checked_at"`
	States         []string                  `json:"states"`
	NewEvents      []*event.Event            `json:"new_events"`
	RemovedEvents  []*event.Event            `json:"removed_events,omitempty"`
	ChangedEvents  []*event.EventChange      `json:"changed_events,omitempty"`
	EventCount     int                       `json:"event_count"`
	ByState        map[string][]*event.Event `json:"by_state,omitempty"`
	ShowAll        bool                      `json:"show_all,omitempty"`
}

// WriteOutput writes the result in the specified format
func WriteOutput(w io.Writer, result *OutputResult, format OutputFormat, verbose bool) error {
	switch format {
	case FormatJSON:
		return writeJSON(w, result)
	case FormatText:
		return writeText(w, result, verbose)
	default:
		return fmt.Errorf("unknown format: %s", format)
	}
}

// writeJSON outputs results as JSON
func writeJSON(w io.Writer, result *OutputResult) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

// writeText outputs results as human-readable text
func writeText(w io.Writer, result *OutputResult, verbose bool) error {
	// Determine labels based on ShowAll mode
	eventLabel := "new"
	eventPrefix := "NEW"
	if result.ShowAll {
		eventLabel = "events"
		eventPrefix = ""
	}

	if result.EventCount == 0 {
		if result.ShowAll {
			fmt.Fprintln(w, "No events found.")
		} else {
			fmt.Fprintln(w, "No new events found.")
		}
		return nil
	}

	// If we have state grouping, show grouped output
	if len(result.ByState) > 0 {
		// Get sorted state codes
		states := make([]string, 0, len(result.ByState))
		for state := range result.ByState {
			states = append(states, state)
		}
		sort.Strings(states)

		for _, state := range states {
			events := result.ByState[state]
			if len(events) == 0 {
				continue
			}

			fmt.Fprintf(w, "\n%s (%d %s):\n", state, len(events), eventLabel)
			for _, evt := range events {
				if eventPrefix != "" {
					fmt.Fprintf(w, "  %s: %s\n", eventPrefix, evt.Raw)
				} else {
					fmt.Fprintf(w, "  %s\n", evt.Raw)
				}
				if evt.DateText != "" {
					fmt.Fprintf(w, "       Date: %s\n", evt.DateText)
				}
				if len(evt.AlsoIn) > 0 {
					fmt.Fprintf(w, "       Also in: %s\n", formatStateList(evt.AlsoIn))
				}
				if verbose {
					fmt.Fprintf(w, "       ID: %s\n", evt.ID)
					if evt.City != "" {
						fmt.Fprintf(w, "       City: %s\n", evt.City)
					}
				}
			}
		}
		fmt.Fprintf(w, "\nTotal: %d %s across %d states\n", result.EventCount, eventLabel, len(result.ByState))
	} else {
		// Simple list for single-state queries
		for _, evt := range result.NewEvents {
			if eventPrefix != "" {
				fmt.Fprintf(w, "%s (%s): %s\n", eventPrefix, evt.State, evt.Raw)
			} else {
				fmt.Fprintf(w, "%s: %s\n", evt.State, evt.Raw)
			}
			if evt.DateText != "" {
				fmt.Fprintf(w, "     Date: %s\n", evt.DateText)
			}
			if len(evt.AlsoIn) > 0 {
				fmt.Fprintf(w, "     Also in: %s\n", formatStateList(evt.AlsoIn))
			}
			if verbose {
				fmt.Fprintf(w, "     ID: %s\n", evt.ID)
				if evt.City != "" {
					fmt.Fprintf(w, "     City: %s\n", evt.City)
				}
			}
		}
		fmt.Fprintf(w, "\nTotal: %d %s\n", result.EventCount, eventLabel)
	}

	return nil
}

// formatStateList formats a list of state codes
func formatStateList(states []string) string {
	if len(states) == 0 {
		return ""
	}
	if len(states) == 1 {
		return states[0]
	}
	// Sort for consistent output
	sorted := make([]string, len(states))
	copy(sorted, states)
	sort.Strings(sorted)

	// Join with commas
	result := ""
	for i, state := range sorted {
		if i > 0 {
			result += ", "
		}
		result += state
	}
	return result
}
