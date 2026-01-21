package event

import (
	"sort"
	"time"
)

// ParseDate attempts to parse event DateText into a time.Time.
// Returns time.Time{} (zero value) if parsing fails.
// Supports formats: "Mar 13 2026", "4.4.26", "Jan 24", "02/15/26"
func ParseDate(dateText string) time.Time {
	if dateText == "" {
		return time.Time{}
	}

	// Try "Mar 13 2026" format
	t, err := time.Parse("Jan 02 2006", dateText)
	if err == nil {
		return t
	}

	// Try "Jan 2 2026" format (single digit day)
	t, err = time.Parse("Jan 2 2006", dateText)
	if err == nil {
		return t
	}

	// Try "4.4.26" format (month.day.year)
	t, err = time.Parse("1.2.06", dateText)
	if err == nil {
		return t
	}

	// Try "04.04.26" format
	t, err = time.Parse("01.02.06", dateText)
	if err == nil {
		return t
	}

	// Try "02/15/26" format
	t, err = time.Parse("01/02/06", dateText)
	if err == nil {
		return t
	}

	// Try "Jan 24" format (no year, assume current year)
	t, err = time.Parse("Jan 02", dateText)
	if err == nil {
		// Add the current year
		now := time.Now()
		return time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}

	// Try "Jan 2" format (single digit day, no year)
	t, err = time.Parse("Jan 2", dateText)
	if err == nil {
		now := time.Now()
		return time.Date(now.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
	}

	// Could not parse, return zero time
	return time.Time{}
}

// IsPastEvent checks if an event's date has passed.
// Returns false if the date cannot be parsed (safer default).
func (e *Event) IsPastEvent() bool {
	parsed := ParseDate(e.DateText)
	if parsed.IsZero() {
		return false // Can't determine, don't filter
	}
	return parsed.Before(time.Now())
}

// IsWithinDays checks if an event is within N days from now.
// Returns true if days <= 0 (feature disabled) or date is unparseable.
func (e *Event) IsWithinDays(days int) bool {
	if days <= 0 {
		return true // Feature disabled
	}
	parsed := ParseDate(e.DateText)
	if parsed.IsZero() {
		return true // Can't determine, include it
	}
	now := time.Now()
	cutoff := now.AddDate(0, 0, days)
	// Compare dates only (truncate to start of day) to include today's events
	nowDate := now.Truncate(24 * time.Hour)
	return !parsed.Before(nowDate) && parsed.Before(cutoff)
}

// IsUpcoming checks if an event is in the future (not past).
// Returns true if the date cannot be parsed (safer default).
func (e *Event) IsUpcoming() bool {
	parsed := ParseDate(e.DateText)
	if parsed.IsZero() {
		return true // Can't determine, include it
	}
	return parsed.After(time.Now())
}

// SortByDate sorts events by date (soonest first).
// Events with unparseable dates are placed at the end.
func SortByDate(events []*Event) {
	sort.Slice(events, func(i, j int) bool {
		dateI := ParseDate(events[i].DateText)
		dateJ := ParseDate(events[j].DateText)

		// If both dates are zero (unparseable), maintain current order
		if dateI.IsZero() && dateJ.IsZero() {
			return false
		}

		// If only i is zero, put it after j
		if dateI.IsZero() {
			return false
		}

		// If only j is zero, put i before j
		if dateJ.IsZero() {
			return true
		}

		// Both dates are valid, sort by date
		return dateI.Before(dateJ)
	})
}

// FormatDateNice formats a date with day of week and countdown
// Returns enhanced date string like "Fri, Mar 15, 2026 (in 45 days)"
// Falls back to original dateText if parsing fails
func FormatDateNice(dateText string) string {
	if dateText == "" {
		return ""
	}

	parsed := ParseDate(dateText)
	if parsed.IsZero() {
		// Can't parse, return original
		return dateText
	}

	// Format: "Mon, Jan 2, 2006"
	formatted := parsed.Format("Mon, Jan 2, 2006")

	// Add countdown
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	eventDate := time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, now.Location())
	daysUntil := int(eventDate.Sub(today).Hours() / 24)

	if daysUntil == 0 {
		formatted += " (today!)"
	} else if daysUntil == 1 {
		formatted += " (tomorrow)"
	} else if daysUntil > 1 && daysUntil <= 30 {
		formatted += " (in " + formatDays(daysUntil) + ")"
	} else if daysUntil < 0 {
		daysAgo := -daysUntil
		if daysAgo == 1 {
			formatted += " (yesterday)"
		} else {
			formatted += " (" + formatDays(daysAgo) + " ago)"
		}
	}

	return formatted
}

// formatDays returns a nice string for day counts
func formatDays(days int) string {
	if days == 7 {
		return "1 week"
	} else if days == 14 {
		return "2 weeks"
	} else if days == 21 {
		return "3 weeks"
	} else if days > 7 && days < 14 {
		weeks := days / 7
		remainingDays := days % 7
		if remainingDays == 0 {
			return formatDays(days / 7 * 7)
		}
		return formatDays(weeks*7) + " " + formatDays(remainingDays)
	}
	if days == 1 {
		return "1 day"
	}
	return formatInt(days) + " days"
}

// formatInt converts int to string without importing strconv
func formatInt(n int) string {
	if n < 0 {
		return "-" + formatInt(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return formatInt(n/10) + string(rune('0'+n%10))
}
