package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// EventOptions contains optional configuration for ICS generation
type EventOptions struct {
	Status         string        // Event status (interested, registered, maybe, skip)
	Note           string        // User note for the event
	CourseDetails  *CourseInfo   // Course information (optional)
	ReminderBefore time.Duration // How far before event to set reminder (default: 24h)
}

// CourseInfo contains golf course details for the ICS description
type CourseInfo struct {
	Name    string
	Par     int
	Yardage int
	Tees    []TeeInfo
}

// TeeInfo represents a single tee's information
type TeeInfo struct {
	Name    string
	Par     int
	Yardage int
}

// GenerateICS generates an iCalendar (.ics) file for an event
func GenerateICS(evt *event.Event) string {
	return GenerateICSWithOptions(evt, nil)
}

// GenerateICSWithOptions generates an iCalendar file for an event with optional configuration
func GenerateICSWithOptions(evt *event.Event, opts *EventOptions) string {
	var ics strings.Builder

	ics.WriteString("BEGIN:VCALENDAR\r\n")
	ics.WriteString("VERSION:2.0\r\n")
	ics.WriteString("PRODID:-//VGA Events//vga-events//EN\r\n")
	ics.WriteString("CALSCALE:GREGORIAN\r\n")
	ics.WriteString("METHOD:PUBLISH\r\n")
	ics.WriteString("BEGIN:VEVENT\r\n")

	writeEvent(&ics, evt, opts)

	ics.WriteString("END:VEVENT\r\n")
	ics.WriteString("END:VCALENDAR\r\n")

	return ics.String()
}

// GenerateMultiEventICS generates an iCalendar file with multiple events
// This is a convenience wrapper for GenerateBulkICS with a default name
func GenerateMultiEventICS(events []*event.Event) string {
	return GenerateBulkICS(events, "VGA Registered Events")
}

// GenerateBulkICS generates an iCalendar (.ics) file with multiple events
func GenerateBulkICS(events []*event.Event, calendarName string) string {
	return GenerateBulkICSWithOptions(events, calendarName, nil)
}

// GenerateBulkICSWithOptions generates an iCalendar file with multiple events and optional per-event configuration
func GenerateBulkICSWithOptions(events []*event.Event, calendarName string, optsMap map[string]*EventOptions) string {
	if len(events) == 0 {
		return ""
	}

	var ics strings.Builder

	// Calendar header
	ics.WriteString("BEGIN:VCALENDAR\r\n")
	ics.WriteString("VERSION:2.0\r\n")
	ics.WriteString("PRODID:-//VGA Events//vga-events//EN\r\n")
	ics.WriteString("CALSCALE:GREGORIAN\r\n")
	ics.WriteString("METHOD:PUBLISH\r\n")

	if calendarName != "" {
		ics.WriteString(fmt.Sprintf("X-WR-CALNAME:%s\r\n", escapeICS(calendarName)))
	}

	// Add each event
	for _, evt := range events {
		ics.WriteString("BEGIN:VEVENT\r\n")

		// Get options for this event (if available)
		var opts *EventOptions
		if optsMap != nil {
			opts = optsMap[evt.ID]
		}

		writeEvent(&ics, evt, opts)

		ics.WriteString("END:VEVENT\r\n")
	}

	// Calendar footer
	ics.WriteString("END:VCALENDAR\r\n")

	return ics.String()
}

// writeEvent writes a single VEVENT to the ICS builder
func writeEvent(ics *strings.Builder, evt *event.Event, opts *EventOptions) {
	now := time.Now().UTC()

	// UID - unique identifier for the event
	ics.WriteString(fmt.Sprintf("UID:%s@vgagolf.org\r\n", evt.ID))

	// DTSTAMP - timestamp when this calendar entry was created
	ics.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", formatICSTime(now)))

	// DTSTART and DTEND - event date and time
	eventDate := event.ParseDate(evt.DateText)
	if eventDate.IsZero() {
		// If we can't parse the date, use one week from now
		eventDate = time.Now().AddDate(0, 0, 7)
	}

	// Set event to 9 AM - 1 PM (4 hours) local time
	startTime := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 9, 0, 0, 0, time.UTC)
	endTime := startTime.Add(4 * time.Hour)

	ics.WriteString(fmt.Sprintf("DTSTART:%s\r\n", formatICSTime(startTime)))
	ics.WriteString(fmt.Sprintf("DTEND:%s\r\n", formatICSTime(endTime)))

	// SUMMARY - event title with status emoji if available
	summary := fmt.Sprintf("VGA Golf - %s", evt.Title)
	if opts != nil && opts.Status != "" {
		emoji := getStatusEmoji(opts.Status)
		if emoji != "" {
			summary = fmt.Sprintf("%s %s", emoji, summary)
		}
	}
	ics.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(summary)))

	// DESCRIPTION - event details with course info if available
	description := buildDescription(evt, opts)
	ics.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(description)))

	// LOCATION - event location
	location := evt.Title
	if evt.City != "" {
		location = fmt.Sprintf("%s, %s", evt.Title, evt.City)
	}
	ics.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICS(location)))

	// URL - link to registration
	ics.WriteString("URL:https://vgagolf.org/state-events\r\n")

	// STATUS - confirmed
	ics.WriteString("STATUS:CONFIRMED\r\n")

	// CATEGORIES and COLOR based on event status
	if opts != nil && opts.Status != "" {
		category := getStatusCategory(opts.Status)
		color := getStatusColor(opts.Status)
		if category != "" {
			ics.WriteString(fmt.Sprintf("CATEGORIES:%s\r\n", category))
		}
		if color != "" {
			ics.WriteString(fmt.Sprintf("COLOR:%s\r\n", color))
		}
	}

	// SEQUENCE - version number for updates
	ics.WriteString("SEQUENCE:0\r\n")

	// TRANSP - show as busy
	ics.WriteString("TRANSP:OPAQUE\r\n")

	// VALARM - reminder/alarm
	reminderBefore := 24 * time.Hour // default: 1 day before
	if opts != nil && opts.ReminderBefore > 0 {
		reminderBefore = opts.ReminderBefore
	}
	writeAlarm(ics, reminderBefore)
}

// buildDescription constructs the event description with course details and notes
func buildDescription(evt *event.Event, opts *EventOptions) string {
	var desc strings.Builder

	// Basic event info
	if evt.DateText != "" {
		desc.WriteString(fmt.Sprintf("Date: %s\n", evt.DateText))
	}
	desc.WriteString(fmt.Sprintf("%s - %s\n", evt.State, evt.Title))

	// Course details if available
	if opts != nil && opts.CourseDetails != nil {
		course := opts.CourseDetails
		desc.WriteString(fmt.Sprintf("\nCourse: %s\n", course.Name))

		if len(course.Tees) > 0 {
			desc.WriteString("\nTees:\n")
			for _, tee := range course.Tees {
				desc.WriteString(fmt.Sprintf("  %s: ", tee.Name))
				if tee.Par > 0 && tee.Yardage > 0 {
					desc.WriteString(fmt.Sprintf("Par %d, %d yds\n", tee.Par, tee.Yardage))
				} else {
					desc.WriteString("\n")
				}
			}
		} else if course.Par > 0 && course.Yardage > 0 {
			desc.WriteString(fmt.Sprintf("Par %d, %d yards\n", course.Par, course.Yardage))
		}
	}

	// User note if available
	if opts != nil && opts.Note != "" {
		desc.WriteString(fmt.Sprintf("\nNote: %s\n", opts.Note))
	}

	// Registration link
	desc.WriteString("\nRegister at: https://vgagolf.org/state-events")

	return desc.String()
}

// writeAlarm writes a VALARM component for event reminders
func writeAlarm(ics *strings.Builder, reminderBefore time.Duration) {
	ics.WriteString("BEGIN:VALARM\r\n")
	ics.WriteString("ACTION:DISPLAY\r\n")
	ics.WriteString("DESCRIPTION:VGA Golf Event Reminder\r\n")

	// Convert duration to ISO 8601 format (e.g., -PT24H for 24 hours before)
	hours := int(reminderBefore.Hours())
	trigger := fmt.Sprintf("-PT%dH", hours)
	ics.WriteString(fmt.Sprintf("TRIGGER:%s\r\n", trigger))

	ics.WriteString("END:VALARM\r\n")
}

// getStatusEmoji returns an emoji for the given status
func getStatusEmoji(status string) string {
	switch status {
	case "registered":
		return "✅"
	case "interested":
		return "⭐"
	case "maybe":
		return "🤔"
	case "skip":
		return "❌"
	default:
		return ""
	}
}

// getStatusCategory returns a category name for the given status
func getStatusCategory(status string) string {
	switch status {
	case "registered":
		return "Registered"
	case "interested":
		return "Interested"
	case "maybe":
		return "Maybe"
	case "skip":
		return "Skipped"
	default:
		return "VGA Golf"
	}
}

// getStatusColor returns a color for the given status
// Colors follow common calendar color schemes
func getStatusColor(status string) string {
	switch status {
	case "registered":
		return "green"
	case "interested":
		return "yellow"
	case "maybe":
		return "gray"
	case "skip":
		return "black"
	default:
		return ""
	}
}

// formatICSTime formats a time.Time as an iCalendar datetime string
func formatICSTime(t time.Time) string {
	return t.UTC().Format("20060102T150405Z")
}

// escapeICS escapes special characters for iCalendar format
func escapeICS(s string) string {
	// Replace special characters according to RFC 5545
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}
