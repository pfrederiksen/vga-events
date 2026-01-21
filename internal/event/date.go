package event

import "time"

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
	return parsed.After(now) && parsed.Before(cutoff)
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
