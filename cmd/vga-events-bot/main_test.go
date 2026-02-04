package main

import (
	"strings"
	"testing"
)

func TestGetCommandHelp(t *testing.T) {
	tests := []struct {
		name     string
		cmdName  string
		wantText string
	}{
		{
			name:     "subscribe command",
			cmdName:  "subscribe",
			wantText: "/subscribe - Subscribe to State Events",
		},
		{
			name:     "search command",
			cmdName:  "search",
			wantText: "/search - Search for Events",
		},
		{
			name:     "note command",
			cmdName:  "note",
			wantText: "/note - Add Notes to Events",
		},
		{
			name:     "stats command",
			cmdName:  "stats",
			wantText: "/stats - View Engagement Statistics",
		},
		{
			name:     "unknown command",
			cmdName:  "nonexistent",
			wantText: "Unknown Command: /nonexistent",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getCommandHelp(tt.cmdName)
			if !strings.Contains(result, tt.wantText) {
				t.Errorf("getCommandHelp(%q) = %v, want to contain %v", tt.cmdName, result, tt.wantText)
			}
		})
	}
}

func TestGetCommandHelpFormat(t *testing.T) {
	commands := []string{
		"subscribe", "unsubscribe", "search", "near", "note", "notes",
		"events", "my-events", "reminders", "notify-removals", "stats",
		"bulk", "export-calendar", "invite", "friends", "join",
		"list", "manage", "settings", "menu", "check", "start",
	}

	for _, cmd := range commands {
		t.Run(cmd, func(t *testing.T) {
			result := getCommandHelp(cmd)

			// Check that help contains expected sections
			if !strings.Contains(result, "<b>Description:</b>") {
				t.Errorf("Command %q help missing Description section", cmd)
			}
			if !strings.Contains(result, "<b>Usage:</b>") {
				t.Errorf("Command %q help missing Usage section", cmd)
			}

			// Check that result is not empty
			if len(result) < 100 {
				t.Errorf("Command %q help seems too short: %d characters", cmd, len(result))
			}
		})
	}
}

func TestGetHelpMessage(t *testing.T) {
	result := getHelpMessage()

	// Check for key sections
	requiredSections := []string{
		"VGA Events Bot",
		"Commands:",
		"Event Tracking:",
		"Reminders:",
		"State Codes:",
		"Get detailed help for any command",
	}

	for _, section := range requiredSections {
		if !strings.Contains(result, section) {
			t.Errorf("getHelpMessage() missing section: %q", section)
		}
	}

	// Check that it contains some command examples
	requiredCommands := []string{"/subscribe", "/events", "/search", "/help"}
	for _, cmd := range requiredCommands {
		if !strings.Contains(result, cmd) {
			t.Errorf("getHelpMessage() missing command: %q", cmd)
		}
	}
}
