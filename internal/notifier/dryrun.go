package notifier

import (
	"fmt"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// DryRunNotifier prints what would be tweeted without actually posting
type DryRunNotifier struct{}

// NewDryRunNotifier creates a new dry-run notifier
func NewDryRunNotifier() *DryRunNotifier {
	return &DryRunNotifier{}
}

// Notify prints the tweets that would be posted
func (n *DryRunNotifier) Notify(events []*event.Event) error {
	for i, evt := range events {
		tweet := formatTweet(evt)
		fmt.Printf("--- Tweet %d/%d ---\n", i+1, len(events))
		fmt.Println(tweet)
		fmt.Printf("\n(Length: %d characters)\n\n", len(tweet))
	}
	return nil
}
