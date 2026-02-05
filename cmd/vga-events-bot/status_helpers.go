package main

import "github.com/pfrederiksen/vga-events/internal/preferences"

// getStatusDisplay returns emoji and display text for a status
func getStatusDisplay(status string) (emoji, text string) {
	switch status {
	case preferences.EventStatusInterested:
		return "⭐", "Interested"
	case preferences.EventStatusRegistered:
		return "✅", "Registered"
	case preferences.EventStatusMaybe:
		return "🤔", "Maybe"
	case preferences.EventStatusSkip:
		return "❌", "Skipped"
	}
	return "", ""
}
