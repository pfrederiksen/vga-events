package calendar

import (
	"fmt"
	"strings"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// GenerateICS generates an iCalendar (.ics) file for an event
func GenerateICS(evt *event.Event) string {
	var ics strings.Builder

	ics.WriteString("BEGIN:VCALENDAR\r\n")
	ics.WriteString("VERSION:2.0\r\n")
	ics.WriteString("PRODID:-//VGA Events//vga-events//EN\r\n")
	ics.WriteString("CALSCALE:GREGORIAN\r\n")
	ics.WriteString("METHOD:PUBLISH\r\n")
	ics.WriteString("BEGIN:VEVENT\r\n")

	// UID - unique identifier for the event
	ics.WriteString(fmt.Sprintf("UID:%s@vgagolf.org\r\n", evt.ID))

	// DTSTAMP - timestamp when this calendar entry was created
	now := time.Now().UTC()
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

	// SUMMARY - event title
	summary := fmt.Sprintf("VGA Golf - %s", evt.Title)
	ics.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(summary)))

	// DESCRIPTION - event details
	description := fmt.Sprintf("%s - %s\n\nRegister at: https://vgagolf.org/state-events", evt.State, evt.Title)
	if evt.DateText != "" {
		description = fmt.Sprintf("Date: %s\n%s", evt.DateText, description)
	}
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

	// SEQUENCE - version number for updates
	ics.WriteString("SEQUENCE:0\r\n")

	// TRANSP - show as busy
	ics.WriteString("TRANSP:OPAQUE\r\n")

	ics.WriteString("END:VEVENT\r\n")
	ics.WriteString("END:VCALENDAR\r\n")

	return ics.String()
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
