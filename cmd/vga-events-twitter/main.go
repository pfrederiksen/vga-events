package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/notifier"
)

var (
	eventsFile = flag.String("events-file", "", "Path to events JSON file (or read from stdin)")
	dryRun     = flag.Bool("dry-run", false, "Print tweets without posting")
	maxTweets  = flag.Int("max-tweets", 10, "Maximum number of tweets to post")
	stateFilter = flag.String("state", "", "Only tweet events for this state")
	version    = "dev"
)

func main() {
	flag.Parse()

	// Read events from file or stdin
	var reader io.Reader
	if *eventsFile != "" {
		f, err := os.Open(*eventsFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening events file: %v\n", err)
			os.Exit(1)
		}
		defer f.Close()
		reader = f
	} else {
		reader = os.Stdin
	}

	// Parse JSON
	var result struct {
		NewEvents []*event.Event `json:"new_events"`
	}

	decoder := json.NewDecoder(reader)
	if err := decoder.Decode(&result); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	if len(result.NewEvents) == 0 {
		fmt.Println("No new events to tweet")
		os.Exit(0)
	}

	// Filter events by state if specified
	events := result.NewEvents
	if *stateFilter != "" {
		filtered := make([]*event.Event, 0)
		for _, evt := range events {
			if evt.State == *stateFilter {
				filtered = append(filtered, evt)
			}
		}
		events = filtered
	}

	// Limit number of tweets
	if len(events) > *maxTweets {
		events = events[:*maxTweets]
	}

	if len(events) == 0 {
		fmt.Println("No events match criteria")
		os.Exit(0)
	}

	// Initialize Twitter client
	var tw notifier.Notifier
	if *dryRun {
		tw = notifier.NewDryRunNotifier()
		fmt.Printf("DRY RUN MODE - Would tweet %d events:\n\n", len(events))
	} else {
		client, err := notifier.NewTwitterNotifier()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error initializing Twitter client: %v\n", err)
			os.Exit(1)
		}
		tw = client
	}

	// Post tweets
	if err := tw.Notify(events); err != nil {
		fmt.Fprintf(os.Stderr, "Error posting tweets: %v\n", err)
		os.Exit(1)
	}

	if !*dryRun {
		fmt.Printf("Successfully posted %d tweets\n", len(events))
	}
}
