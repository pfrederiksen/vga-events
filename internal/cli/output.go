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
	CheckedAt  time.Time              `json:"checked_at"`
	States     []string               `json:"states"`
	NewEvents  []*event.Event         `json:"new_events"`
	EventCount int                    `json:"event_count"`
	ByState    map[string][]*event.Event `json:"by_state,omitempty"`
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
	if result.EventCount == 0 {
		fmt.Fprintln(w, "No new events found.")
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

			fmt.Fprintf(w, "\n%s (%d new):\n", state, len(events))
			for _, evt := range events {
				fmt.Fprintf(w, "  NEW: %s\n", evt.Raw)
				if verbose {
					fmt.Fprintf(w, "       ID: %s\n", evt.ID)
					if evt.DateText != "" {
						fmt.Fprintf(w, "       Date: %s\n", evt.DateText)
					}
					if evt.City != "" {
						fmt.Fprintf(w, "       City: %s\n", evt.City)
					}
				}
			}
		}
		fmt.Fprintf(w, "\nTotal: %d new events across %d states\n", result.EventCount, len(result.ByState))
	} else {
		// Simple list for single-state queries
		for _, evt := range result.NewEvents {
			fmt.Fprintf(w, "NEW (%s): %s\n", evt.State, evt.Raw)
			if verbose {
				fmt.Fprintf(w, "     ID: %s\n", evt.ID)
				if evt.DateText != "" {
					fmt.Fprintf(w, "     Date: %s\n", evt.DateText)
				}
				if evt.City != "" {
					fmt.Fprintf(w, "     City: %s\n", evt.City)
				}
			}
		}
		fmt.Fprintf(w, "\nTotal: %d new events\n", result.EventCount)
	}

	return nil
}
