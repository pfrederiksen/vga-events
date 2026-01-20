package notifier

import (
	"fmt"
	"os"
	"time"

	"github.com/dghubble/go-twitter/twitter" //nolint:staticcheck // Using stable v1.1 API
	"github.com/dghubble/oauth1"
	"github.com/pfrederiksen/vga-events/internal/event"
)

// TwitterNotifier posts events to Twitter
type TwitterNotifier struct {
	client *twitter.Client
}

// NewTwitterNotifier creates a new Twitter notifier using environment variables
// Required environment variables:
// - TWITTER_API_KEY
// - TWITTER_API_SECRET
// - TWITTER_ACCESS_TOKEN
// - TWITTER_ACCESS_SECRET
func NewTwitterNotifier() (*TwitterNotifier, error) {
	apiKey := os.Getenv("TWITTER_API_KEY")
	apiSecret := os.Getenv("TWITTER_API_SECRET")
	accessToken := os.Getenv("TWITTER_ACCESS_TOKEN")
	accessSecret := os.Getenv("TWITTER_ACCESS_SECRET")

	if apiKey == "" || apiSecret == "" || accessToken == "" || accessSecret == "" {
		return nil, fmt.Errorf("missing required Twitter credentials in environment variables")
	}

	config := oauth1.NewConfig(apiKey, apiSecret)
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	return &TwitterNotifier{client: client}, nil
}

// Notify posts tweets for each event
func (n *TwitterNotifier) Notify(events []*event.Event) error {
	for i, evt := range events {
		tweet := formatTweet(evt)

		_, _, err := n.client.Statuses.Update(tweet, nil)
		if err != nil {
			return fmt.Errorf("failed to post tweet for event %s: %w", evt.ID, err)
		}

		// Rate limiting: wait between tweets
		if i < len(events)-1 {
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

// formatTweet formats an event as a tweet
func formatTweet(evt *event.Event) string {
	tweet := "🏌️ New VGA Golf Event!\n\n"
	tweet += fmt.Sprintf("📍 %s - %s\n", evt.State, evt.Title)

	if evt.DateText != "" {
		tweet += fmt.Sprintf("📅 %s\n", evt.DateText)
	}

	if evt.City != "" {
		tweet += fmt.Sprintf("🏢 %s\n", evt.City)
	}

	// Add registration link (login required for specific event)
	tweet += "\n🔗 Register at vgagolf.org/state-events (login required)\n"
	tweet += "\n#VGAGolf #Golf"

	// Twitter limit is 280 characters
	if len(tweet) > 280 {
		// Truncate and add ellipsis
		tweet = tweet[:277] + "..."
	}

	return tweet
}
