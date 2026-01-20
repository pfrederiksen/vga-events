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
)

var (
	flagCheckState string
	flagDataDir    string
	flagFormat     string
	flagRefresh    bool
	flagVerbose    bool
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

	cmd.MarkFlagRequired("check-state")

	return cmd
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

	// Save updated snapshot
	if err := store.CreateSnapshotFromEvents(currentEvents, state); err != nil {
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
	if state == "ALL" {
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
			WriteOutput(os.Stdout, result, format, flagVerbose)
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
