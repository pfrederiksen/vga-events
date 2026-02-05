package main

import (
	"fmt"
	"strings"

	"github.com/pfrederiksen/vga-events/internal/preferences"
)

// formatStatusCounts appends formatted status counts to a string builder
func formatStatusCounts(msg *strings.Builder, eventCounts map[string]int) {
	statusList := []struct {
		key   string
		emoji string
		text  string
	}{
		{preferences.EventStatusInterested, "⭐", "Interested"},
		{preferences.EventStatusRegistered, "✅", "Registered"},
		{preferences.EventStatusMaybe, "🤔", "Maybe"},
		{preferences.EventStatusSkip, "❌", "Skipped"},
	}

	for _, s := range statusList {
		if count, ok := eventCounts[s.key]; ok && count > 0 {
			msg.WriteString(fmt.Sprintf("  %s %s: %d\n", s.emoji, s.text, count))
		}
	}
}
