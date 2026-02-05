package main

import (
	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
)

// processUICommand handles UI/keyboard-based commands
// Returns (responseText, events, handled) where handled indicates if the command was processed
func processUICommand(command string, prefs preferences.Preferences, chatID string, botToken string, dryRun bool) (string, []*event.Event, bool) {
	switch command {
	case "/manage":
		responseText, _ := handleManageWithKeyboard(prefs, chatID, botToken, dryRun)
		return responseText, nil, true

	case "/settings":
		responseText, _ := handleSettingsWithKeyboard(prefs, chatID, botToken, dryRun)
		return responseText, nil, true

	case "/menu":
		responseText, _ := handleMenuWithKeyboard(chatID, botToken, dryRun)
		return responseText, nil, true

	case "/list":
		return handleList(prefs, chatID), nil, true

	default:
		return "", nil, false // Command not handled
	}
}

// processEventCommand handles event viewing commands
// Returns (responseText, events, handled) where handled indicates if the command was processed
func processEventCommand(command string, prefs preferences.Preferences, chatID string, botToken string, dryRun bool, modified *bool) (string, []*event.Event, bool) {
	switch command {
	case "/my-events":
		responseText, events := handleMyEvents(prefs, chatID, botToken, dryRun, modified)
		return responseText, events, true

	case "/events":
		responseText, events := handleAllEvents(prefs, chatID, botToken, dryRun, modified)
		return responseText, events, true

	default:
		return "", nil, false // Command not handled
	}
}
