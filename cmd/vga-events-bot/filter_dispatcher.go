package main

import (
	"strings"

	"github.com/pfrederiksen/vga-events/internal/event"
	"github.com/pfrederiksen/vga-events/internal/preferences"
)

// processFilterCommand handles all /filter subcommands
func processFilterCommand(parts []string, prefs preferences.Preferences, chatID string, modified *bool) (string, []*event.Event) {
	// No subcommand - show current status
	if len(parts) < 2 {
		return handleFilterStatus(prefs, chatID)
	}

	subcommand := strings.ToLower(parts[1])

	switch subcommand {
	case "date":
		if len(parts) < 3 {
			return `❌ Please specify a date range.

Usage: /filter date "Mar 1-15"

Examples:
/filter date "Mar 1-15" - March 1 to 15
/filter date "Mar 1 - Apr 15" - March 1 to April 15
/filter date "March" - Entire month of March`, nil
		}
		dateRange := strings.Join(parts[2:], " ")
		return handleFilterDate(prefs, chatID, dateRange, modified)

	case "course":
		if len(parts) < 3 {
			return `❌ Please specify a course name.

Usage: /filter course "Pebble Beach"

Examples:
/filter course "Pebble Beach"
/filter course "Shadow Creek"`, nil
		}
		courseName := strings.Join(parts[2:], " ")
		return handleFilterCourse(prefs, chatID, courseName, modified)

	case "city":
		if len(parts) < 3 {
			return `❌ Please specify a city name.

Usage: /filter city "Las Vegas"

Examples:
/filter city "Las Vegas"
/filter city "San Diego"`, nil
		}
		cityName := strings.Join(parts[2:], " ")
		return handleFilterCity(prefs, chatID, cityName, modified)

	case "weekends":
		return handleFilterWeekends(prefs, chatID, modified)

	case "clear":
		return handleFilterClear(prefs, chatID, modified)

	case "save":
		if len(parts) < 3 {
			return `❌ Please specify a name for this filter.

Usage: /filter save "My Weekend Events"

Examples:
/filter save "March Weekends"
/filter save "Pebble Beach Events"`, nil
		}
		filterName := strings.Join(parts[2:], " ")
		// Remove quotes if present
		filterName = strings.Trim(filterName, "\"")
		return handleFilterSave(prefs, chatID, filterName, modified)

	case "load":
		if len(parts) < 3 {
			return `❌ Please specify the filter name to load.

Usage: /filter load "My Weekend Events"

Use /filters to see all saved filters.`, nil
		}
		filterName := strings.Join(parts[2:], " ")
		// Remove quotes if present
		filterName = strings.Trim(filterName, "\"")
		return handleFilterLoad(prefs, chatID, filterName, modified)

	case "delete":
		if len(parts) < 3 {
			return `❌ Please specify the filter name to delete.

Usage: /filter delete "My Weekend Events"

Use /filters to see all saved filters.`, nil
		}
		filterName := strings.Join(parts[2:], " ")
		// Remove quotes if present
		filterName = strings.Trim(filterName, "\"")
		return handleFilterDelete(prefs, chatID, filterName, modified)

	default:
		return `❌ Unknown filter subcommand.

Available subcommands:
• /filter date "Mar 1-15" - Set date range
• /filter course "Pebble Beach" - Filter by course
• /filter city "Las Vegas" - Filter by city
• /filter weekends - Toggle weekends-only
• /filter save "name" - Save current filter
• /filter load "name" - Load saved filter
• /filter delete "name" - Delete saved filter
• /filter clear - Remove active filter
• /filter - Show current filter status

Use /filters to list all saved filters.`, nil
	}
}
