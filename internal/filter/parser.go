package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseDateRange parses a date range string into start and end times.
//
// Supported formats:
//   - "Mar 1-15" or "March 1-15" - Same month, different days
//   - "March 1 - April 15" - Different months
//   - "March" - Entire month
//
// The parser automatically infers the year:
//   - If the month is in the past, assumes next year
//   - Otherwise, uses current year
//   - For cross-month ranges, if end month < start month, end is in next year
//
// Returns (dateFrom, dateTo, error). Times are in UTC.
// Start time is at 00:00:00, end time is at 23:59:59.
func ParseDateRange(input string) (*time.Time, *time.Time, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil, fmt.Errorf("date range cannot be empty")
	}

	// Try various formats

	// Format 1: "Mar 1-15" or "March 1-15"
	re1 := regexp.MustCompile(`(?i)^(jan|january|feb|february|mar|march|apr|april|may|jun|june|jul|july|aug|august|sep|september|oct|october|nov|november|dec|december)\s+(\d{1,2})\s*-\s*(\d{1,2})$`)
	if matches := re1.FindStringSubmatch(input); matches != nil {
		month := parseMonth(matches[1])
		if month == 0 {
			return nil, nil, fmt.Errorf("invalid month: %s", matches[1])
		}

		day1, err := strconv.Atoi(matches[2])
		if err != nil || day1 < 1 || day1 > 31 {
			return nil, nil, fmt.Errorf("invalid day: %s", matches[2])
		}

		day2, err := strconv.Atoi(matches[3])
		if err != nil || day2 < 1 || day2 > 31 {
			return nil, nil, fmt.Errorf("invalid day: %s", matches[3])
		}

		year := getYearForMonth(month)
		from := time.Date(year, month, day1, 0, 0, 0, 0, time.UTC)
		to := time.Date(year, month, day2, 23, 59, 59, 0, time.UTC)

		if from.After(to) {
			return nil, nil, fmt.Errorf("start date must be before end date")
		}

		return &from, &to, nil
	}

	// Format 2: "Mar 1 - Mar 15" or "March 1 - March 15"
	re2 := regexp.MustCompile(`(?i)^(jan|january|feb|february|mar|march|apr|april|may|jun|june|jul|july|aug|august|sep|september|oct|october|nov|november|dec|december)\s+(\d{1,2})\s*-\s*(jan|january|feb|february|mar|march|apr|april|may|jun|june|jul|july|aug|august|sep|september|oct|october|nov|november|dec|december)\s+(\d{1,2})$`)
	if matches := re2.FindStringSubmatch(input); matches != nil {
		month1 := parseMonth(matches[1])
		if month1 == 0 {
			return nil, nil, fmt.Errorf("invalid month: %s", matches[1])
		}

		day1, err := strconv.Atoi(matches[2])
		if err != nil || day1 < 1 || day1 > 31 {
			return nil, nil, fmt.Errorf("invalid day: %s", matches[2])
		}

		month2 := parseMonth(matches[3])
		if month2 == 0 {
			return nil, nil, fmt.Errorf("invalid month: %s", matches[3])
		}

		day2, err := strconv.Atoi(matches[4])
		if err != nil || day2 < 1 || day2 > 31 {
			return nil, nil, fmt.Errorf("invalid day: %s", matches[4])
		}

		year1 := getYearForMonth(month1)
		year2 := getYearForMonth(month2)

		// If month2 < month1, assume month2 is in the next year
		if month2 < month1 {
			year2++
		}

		from := time.Date(year1, month1, day1, 0, 0, 0, 0, time.UTC)
		to := time.Date(year2, month2, day2, 23, 59, 59, 0, time.UTC)

		if from.After(to) {
			return nil, nil, fmt.Errorf("start date must be before end date")
		}

		return &from, &to, nil
	}

	// Format 3: Single month "March" or "Mar" (entire month)
	re3 := regexp.MustCompile(`(?i)^(jan|january|feb|february|mar|march|apr|april|may|jun|june|jul|july|aug|august|sep|september|oct|october|nov|november|dec|december)$`)
	if matches := re3.FindStringSubmatch(input); matches != nil {
		month := parseMonth(matches[1])
		if month == 0 {
			return nil, nil, fmt.Errorf("invalid month: %s", matches[1])
		}

		year := getYearForMonth(month)
		from := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		// Last day of month
		to := time.Date(year, month+1, 0, 23, 59, 59, 0, time.UTC)

		return &from, &to, nil
	}

	return nil, nil, fmt.Errorf("invalid date range format. Use 'Mar 1-15', 'March 1 - March 15', or 'March'")
}

// parseMonth converts a month name to time.Month
func parseMonth(name string) time.Month {
	name = strings.ToLower(strings.TrimSpace(name))

	months := map[string]time.Month{
		"jan": time.January, "january": time.January,
		"feb": time.February, "february": time.February,
		"mar": time.March, "march": time.March,
		"apr": time.April, "april": time.April,
		"may": time.May,
		"jun": time.June, "june": time.June,
		"jul": time.July, "july": time.July,
		"aug": time.August, "august": time.August,
		"sep": time.September, "september": time.September,
		"oct": time.October, "october": time.October,
		"nov": time.November, "november": time.November,
		"dec": time.December, "december": time.December,
	}

	return months[name]
}

// getYearForMonth returns the appropriate year for a given month
// If the month has already passed this year, returns next year
func getYearForMonth(month time.Month) int {
	now := time.Now()
	year := now.Year()

	// If month is in the past, use next year
	if month < now.Month() {
		year++
	}

	return year
}
