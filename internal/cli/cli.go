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
	flagVersion    bool
	flagSort       string
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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
	cmd.Flags().StringVar(&flagSort, "sort", "date", "Sort order: date, state, or title")
	cmd.Flags().BoolVarP(&flagVersion, "version", "v", false, "Print version information")

	// Make check-state optional if version is requested
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if flagVersion {
			fmt.Printf("vga-events version %s\n", version)
			fmt.Printf("commit: %s\n", commit)
			fmt.Printf("built: %s\n", date)
			os.Exit(ExitSuccess)
		}
		// Validate check-state is provided if not version
		if flagCheckState == "" {
			return fmt.Errorf("--check-state is required")
		}
		return nil
	}

	return cmd
}

// filterEventsByState filters events by state code
func filterEventsByState(events []*event.Event, state string) []*event.Event {
	// If checking all states, return all events
	if state == StateAll {
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
func handleShowAll(currentEvents []*event.Event, state string, format OutputFormat, verbose bool, sortOrder SortOrder, store *storage.Storage) error {
	// Filter events by state
	filteredEvents := make([]*event.Event, 0)
	stateMap := make(map[string][]*event.Event)

	for _, evt := range currentEvents {
		// Apply state filter
		if state != StateAll && !strings.EqualFold(evt.State, state) {
			continue
		}

		filteredEvents = append(filteredEvents, evt)

		// Group by state for "all" mode
		if state == StateAll {
			if stateMap[evt.State] == nil {
				stateMap[evt.State] = make([]*event.Event, 0)
			}
			stateMap[evt.State] = append(stateMap[evt.State], evt)
		}
	}

	// Sort events
	sortEvents(filteredEvents, sortOrder)

	// Sort events within each state group
	if state == StateAll {
		for _, events := range stateMap {
			sortEvents(events, sortOrder)
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
	if state == StateAll {
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

func validateRunConfig() (string, OutputFormat, SortOrder, error) {
	state := strings.ToUpper(strings.TrimSpace(flagCheckState))
	if state == "" {
		return "", "", "", fmt.Errorf("--check-state is required")
	}

	format := OutputFormat(strings.ToLower(flagFormat))
	if format != FormatText && format != FormatJSON {
		return "", "", "", fmt.Errorf("invalid format: %s (must be 'text' or 'json')", flagFormat)
	}

	sortOrder := SortOrder(strings.ToLower(flagSort))
	if sortOrder != SortByDate && sortOrder != SortByState && sortOrder != SortByTitle {
		return "", "", "", fmt.Errorf("invalid sort order: %s (must be 'date', 'state', or 'title')", flagSort)
	}

	return state, format, sortOrder, nil
}

func logRunConfig(state string, sortOrder SortOrder) {
	if !flagVerbose {
		return
	}

	fmt.Fprintf(os.Stderr, "Checking state: %s\n", state)
	fmt.Fprintf(os.Stderr, "Data directory: %s\n", flagDataDir)
	fmt.Fprintf(os.Stderr, "Sort order: %s\n", sortOrder)
}

func fetchCurrentEvents() ([]*event.Event, error) {
	sc := scraper.New()

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "Fetching events from %s\n", scraper.StateEventsURL)
	}

	currentEvents, err := sc.FetchEvents()
	if err != nil {
		return nil, fmt.Errorf("fetching events: %w", err)
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "Fetched %d total events\n", len(currentEvents))
	}

	return currentEvents, nil
}

func detectChangedEvents(previous *event.Snapshot, eventsToSave []*event.Event, newSnapshot *event.Snapshot) []*event.EventChange {
	if previous == nil {
		return nil
	}

	currentEventsMap := make(map[string]*event.Event)
	for _, evt := range eventsToSave {
		currentEventsMap[evt.ID] = evt
	}

	changedEvents := event.CompareSnapshots(previous.Events, currentEventsMap, previous.StableIndex, newSnapshot.StableIndex)
	filteredChanges := make([]*event.EventChange, 0, len(changedEvents))
	for _, change := range changedEvents {
		if change.ChangeType != "new" && change.ChangeType != "removed" {
			filteredChanges = append(filteredChanges, change)
		}
	}

	newSnapshot.ChangeLog = append(newSnapshot.ChangeLog, filteredChanges...)
	if len(newSnapshot.ChangeLog) > 100 {
		newSnapshot.ChangeLog = newSnapshot.ChangeLog[len(newSnapshot.ChangeLog)-100:]
	}

	return filteredChanges
}

func buildDiffOutputResult(state string, diff *event.DiffResult, changedEvents []*event.EventChange) *OutputResult {
	result := &OutputResult{
		CheckedAt:     time.Now().UTC(),
		NewEvents:     diff.NewEvents,
		RemovedEvents: diff.RemovedEvents,
		ChangedEvents: changedEvents,
		EventCount:    len(diff.NewEvents),
	}

	if state == StateAll {
		states := make([]string, 0, len(diff.States))
		for s := range diff.States {
			states = append(states, s)
		}
		result.States = states
		result.ByState = diff.States
		return result
	}

	result.States = []string{state}
	if len(diff.NewEvents) > 0 {
		result.ByState = map[string][]*event.Event{
			state: diff.NewEvents,
		}
	}

	return result
}

func handleRefreshOutput(result *OutputResult, format OutputFormat) {
	if format == FormatText {
		fmt.Println("Snapshot refreshed successfully.")
	} else {
		result.NewEvents = nil
		result.EventCount = 0
		result.ByState = nil
		_ = WriteOutput(os.Stdout, result, format, flagVerbose)
	}
	os.Exit(ExitSuccess)
}

// runCheck is the main command logic
func runCheck(cmd *cobra.Command, args []string) error {
	state, format, sortOrder, err := validateRunConfig()
	if err != nil {
		return err
	}
	logRunConfig(state, sortOrder)

	store, err := storage.New(flagDataDir)
	if err != nil {
		return fmt.Errorf("initializing storage: %w", err)
	}

	currentEvents, err := fetchCurrentEvents()
	if err != nil {
		return err
	}

	if flagShowAll {
		return handleShowAll(currentEvents, state, format, flagVerbose, sortOrder, store)
	}

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

	diff := event.Diff(previous, currentEvents, state)
	sortEvents(diff.NewEvents, sortOrder)
	if state == StateAll {
		for _, events := range diff.States {
			sortEvents(events, sortOrder)
		}
	}

	eventsToSave := filterEventsByState(currentEvents, state)
	newSnapshot := event.CreateSnapshot(eventsToSave, time.Now().UTC().Format(time.RFC3339))
	changedEvents := detectChangedEvents(previous, eventsToSave, newSnapshot)

	if len(diff.RemovedEvents) > 0 {
		newSnapshot.StoreRemovedEvents(diff.RemovedEvents)
	}
	newSnapshot.CleanupRemovedEvents()
	if err := store.SaveSnapshot(newSnapshot, state); err != nil {
		return fmt.Errorf("saving snapshot: %w", err)
	}

	if flagVerbose {
		fmt.Fprintf(os.Stderr, "Saved snapshot\n")
	}

	result := buildDiffOutputResult(state, diff, changedEvents)

	if flagRefresh {
		handleRefreshOutput(result, format)
		return nil
	}

	if err := WriteOutput(os.Stdout, result, format, flagVerbose); err != nil {
		return fmt.Errorf("writing output: %w", err)
	}

	if len(diff.NewEvents) > 0 {
		os.Exit(ExitNewEvents)
	} else {
		os.Exit(ExitSuccess)
	}

	return nil
}

// Execute runs the CLI
func Execute(v, c, d string) {
	// Set version information
	version = v
	commit = c
	date = d

	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(ExitError)
	}
}
