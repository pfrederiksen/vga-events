package telegram

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// FormatDigest formats a batch of events as a digest message
func FormatDigest(events []*event.Event, frequency string) string {
	if len(events) == 0 {
		return "No new events in this digest period."
	}

	// Header
	msg := "📬 <b>Your VGA Events Digest</b>\n\n"
	msg += fmt.Sprintf("🗓 %s digest • %d new event(s)\n\n", strings.Title(frequency), len(events))

	// Group events by state
	byState := make(map[string][]*event.Event)
	for _, evt := range events {
		byState[evt.State] = append(byState[evt.State], evt)
	}

	// Sort states alphabetically
	states := make([]string, 0, len(byState))
	for state := range byState {
		states = append(states, state)
	}
	sort.Strings(states)

	// Format each state section
	for _, state := range states {
		stateEvents := byState[state]
		msg += fmt.Sprintf("📍 <b>%s</b> (%d event%s)\n", state, len(stateEvents), pluralize(len(stateEvents)))

		for _, evt := range stateEvents {
			msg += fmt.Sprintf("  • %s", evt.Title)
			if evt.DateText != "" {
				msg += fmt.Sprintf(" (%s)", evt.DateText)
			}
			if evt.City != "" {
				msg += fmt.Sprintf(" - %s", evt.City)
			}
			msg += "\n"
		}
		msg += "\n"
	}

	msg += "🔗 <b>Register:</b> https://vgagolf.org/state-events\n\n"
	msg += "💬 <i>/settings to change digest frequency</i>"

	return msg
}

// FormatDigestSummary creates a short summary for a digest
func FormatDigestSummary(events []*event.Event, frequency string) string {
	if len(events) == 0 {
		return fmt.Sprintf("📬 Your %s digest: No new events", frequency)
	}

	// Count events by state
	byState := make(map[string]int)
	for _, evt := range events {
		byState[evt.State]++
	}

	stateList := make([]string, 0, len(byState))
	for state, count := range byState {
		stateList = append(stateList, fmt.Sprintf("%s (%d)", state, count))
	}
	sort.Strings(stateList)

	return fmt.Sprintf("📬 Your %s digest: %d new event%s in %s",
		frequency,
		len(events),
		pluralize(len(events)),
		strings.Join(stateList, ", "))
}

func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
