package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/scraper"
	"github.com/pfrederiksen/vga-events/internal/storage"
	"github.com/spf13/cobra"
)

const (
	ExitSuccess   = 0
	ExitError     = 1
	ExitNewEvents = 2

	StateAll = "ALL"
)

var (
	flagCheckState string
	flagDataDir    string
	flagFormat     string
	flagRefresh    bool
	flagVerbose    bool
	flagShowAll    bool
)

// NewRootCmd creates the root command
func NewRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vga-events",
		Short: "Check for newly-added VGA Golf state events",
		Long: `A CLI tool to check for newly-added VGA Golf state events.
Tracks events across runs and reports only new events since last check.`,
		RunE: runCheck,
	}

	// Define flags
	cmd.Flags().StringVar(&flagCheckState, "check-state", "", "State code (e.g., NV) or 'all' (required)")
	cmd.Flags().StringVar(&flagDataDir, "data-dir", "~/.local/share/vga-events", "Data directory for snapshots")
	cmd.Flags().StringVar(&flagFormat, "format", "text", "Output format: text or json")
	cmd.Flags().BoolVar(&flagRefresh, "refresh", false, "Refresh snapshot without showing new events")
	cmd.Flags().BoolVar(&flagVerbose, "verbose", false, "Enable verbose logging")
	cmd.Flags().BoolVar(&flagShowAll, "show-all", false, "Show all events, not just new ones")

	cmd.MarkFlagRequired("check-state")

	return cmd
}

// filterEventsByState filters events by state code
func filterEventsByState(events []*event.Event, state string) []*event.Event {
	// If checking all states, return all events
	if state == "StateAll" {
		return events
	}

	// Filter for specific state
	filtered := make([]*event.Event, 0)
	for _, evt := range events {
		if strings.EqualFold(evt.State, state) {
			filtered = append(filtered, evt)
		}
	}
	return filtered
}

// handleShowAll handles the --show-all flag to display all events
func handleShowAll(currentEvents []*event.Event, state string, format OutputFormat, verbose bool, store *storage.Storage) error {
	// Filter events by state
	filteredEvents := make([]*event.Event, 0)
	stateMap := make(map[string][]*event.Event)

	for _, evt := range currentEvents {
		// Apply state filter
		if state != "StateAll" && !strings.EqualFold(evt.State, state) {
			continue
		}

		filteredEvents = append(filteredEvents, evt)

		// Group by state for "all" mode
		if state == "StateAll" {
			if stateMap[evt.State] == nil {
				stateMap[evt.State] = make([]*event.Event, 0)
			}
			stateMap[evt.State] = append(stateMap[evt.State], evt)
		}
	}

	// Filter events by state before saving snapshot
	eventsToSave := filterEventsByState(currentEvents, state)

	// Save snapshot with only the filtered events
	if err := store.CreateSnapshotFromEvents(eventsToSave, state); err != nil {
		return fmt.Errorf("saving snapshot: %w", err)
	}

	if verbose {
		fmt.Fprintf(os.Stderr, "Saved snapshot\n")
	}

	// Prepare output
	result := &OutputResult{
		CheckedAt:  time.Now().UTC(),
		NewEvents:  filteredEvents,
		EventCount: len(filteredEvents),
		ShowAll:    true,
	}

	// Determine states
	if state == "StateAll" {
		states := make([]string, 0, len(stateMap))
		for s := range stateMap {
			states = append(states, s)
		}
		result.States = states
		result.ByState = stateMap
	} else {
		result.States = []string{state}
		if len(filteredEvents) > 0 {
			result.ByState = map[string][]*event.Event{
				state: filteredEvents,
			}
		}
	}

	// Write output
	if err := WriteOutput(os.Stdout, result, format, verbose); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	os.Exit(ExitSuccess)
	return nil
}

// runCheck is the main command logic
func runCheck(cmd *cobra.Command, args []string) error {
	// Normalize state code
	state := strings.ToUpper(strings.TrimSpace(flagCheckState))
	if state == "" {
		return fmt.Errorf("--check-state is required")
	}

	// Validate format
	format := OutputFormat(strings.ToLower(flagFormat))
	if format != FormatText && format != FormatJSON {
		return fmt.Errorf("invalid format: %s (must be 'text' or 'json')", flagFormat)
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "Checking state: %s\n", state)
		fmt.Fprintf(os.Stderr, "Data directory: %s\n", flagDataDir)
	}

	// Initialize storage
	store, err := storage.New(flagDataDir)
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}

	// Initialize scraper
	sc := scraper.New()

	// Fetch current events
	if flagVerbose {
		fmt.Fprintf(os.Stderr, "Fetching events from %s\n", scraper.StateEventsURL)
	}

	currentEvents, err := sc.FetchEvents()
	if err != nil {
		return fmt.Errorf("fetching events: %w", err)
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "Fetched %d total events\n", len(currentEvents))
	}

	// Handle --show-all mode
	if flagShowAll {
		return handleShowAll(currentEvents, state, format, flagVerbose, store)
	}

	// Load previous snapshot
	var previous *event.Snapshot
	if !flagRefresh {
		previous, err = store.LoadSnapshot(state)
		if err != nil {
			return fmt.Errorf("loading snapshot: %w", err)
		}

		if flagVerbose {
			fmt.Fprintf(os.Stderr, "Loaded previous snapshot with %d events\n", len(previous.Events))
		}
	}

	// Compute diff
	diff := event.Diff(previous, currentEvents, state)

	// Filter events by state before saving snapshot
	eventsToSave := filterEventsByState(currentEvents, state)

	// Save updated snapshot with only the filtered events
	if err := store.CreateSnapshotFromEvents(eventsToSave, state); err != nil {
		return fmt.Errorf("saving snapshot: %w", err)
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "Saved snapshot\n")
	}

	// Prepare output
	result := &OutputResult{
		CheckedAt:  time.Now().UTC(),
		NewEvents:  diff.NewEvents,
		EventCount: len(diff.NewEvents),
	}

	// Determine states checked
	if state == "StateAll" {
		states := make([]string, 0, len(diff.States))
		for s := range diff.States {
			states = append(states, s)
		}
		result.States = states
		result.ByState = diff.States
	} else {
		result.States = []string{state}
		if len(diff.NewEvents) > 0 {
			result.ByState = map[string][]*event.Event{
				state: diff.NewEvents,
			}
		}
	}

	// In refresh mode, don't output new events
	if flagRefresh {
		if format == FormatText {
			fmt.Println("Snapshot refreshed successfully.")
		} else {
			// Still output JSON but with zero new events
			result.NewEvents = nil
			result.EventCount = 0
			result.ByState = nil
			_ = WriteOutput(os.Stdout, result, format, flagVerbose)
		}
		os.Exit(ExitSuccess)
		return nil
	}

	// Write output
	if err := WriteOutput(os.Stdout, result, format, flagVerbose); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	// Set exit code based on whether new events were found
	if len(diff.NewEvents) > 0 {
		os.Exit(ExitNewEvents)
	} else {
		os.Exit(ExitSuccess)
	}

	return nil
}

// Execute runs the CLI
func Execute() {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}
}
