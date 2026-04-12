package filter

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const monthPattern = `(?i)(jan|january|feb|february|mar|march|apr|april|may|jun|june|jul|july|aug|august|sep|september|oct|october|nov|november|dec|december)`

var (
	sameMonthRangeRE  = regexp.MustCompile(`^` + monthPattern + `\s+(\d{1,2})\s*-\s*(\d{1,2})$`)
	crossMonthRangeRE = regexp.MustCompile(`^` + monthPattern + `\s+(\d{1,2})\s*-\s*` + monthPattern + `\s+(\d{1,2})$`)
	singleMonthRE     = regexp.MustCompile(`^` + monthPattern + `$`)
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

	if matches := sameMonthRangeRE.FindStringSubmatch(input); matches != nil {
		month, day1, day2, err := parseMonthAndDays(matches[1], matches[2], matches[3])
		if err != nil {
			return nil, nil, err
		}

		year := getYearForMonth(month)
		return createDateRange(year, month, day1, year, month, day2)
	}

	if matches := crossMonthRangeRE.FindStringSubmatch(input); matches != nil {
		month1, day1, err := parseMonthAndDay(matches[1], matches[2])
		if err != nil {
			return nil, nil, err
		}
		month2, day2, err := parseMonthAndDay(matches[3], matches[4])
		if err != nil {
			return nil, nil, err
		}
		year1 := getYearForMonth(month1)
		year2 := getYearForMonth(month2)
		if month2 < month1 {
			year2++
		}
		return createDateRange(year1, month1, day1, year2, month2, day2)
	}

	if matches := singleMonthRE.FindStringSubmatch(input); matches != nil {
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

func parseMonthAndDay(monthText, dayText string) (time.Month, int, error) {
	month := parseMonth(monthText)
	if month == 0 {
		return 0, 0, fmt.Errorf("invalid month: %s", monthText)
	}

	day, err := strconv.Atoi(dayText)
	if err != nil || day < 1 || day > 31 {
		return 0, 0, fmt.Errorf("invalid day: %s", dayText)
	}

	return month, day, nil
}

func parseMonthAndDays(monthText, day1Text, day2Text string) (time.Month, int, int, error) {
	month, day1, err := parseMonthAndDay(monthText, day1Text)
	if err != nil {
		return 0, 0, 0, err
	}

	_, day2, err := parseMonthAndDay(monthText, day2Text)
	if err != nil {
		return 0, 0, 0, err
	}

	return month, day1, day2, nil
}

func createDateRange(year1 int, month1 time.Month, day1 int, year2 int, month2 time.Month, day2 int) (*time.Time, *time.Time, error) {
	from := time.Date(year1, month1, day1, 0, 0, 0, 0, time.UTC)
	to := time.Date(year2, month2, day2, 23, 59, 59, 0, time.UTC)

	if from.After(to) {
		return nil, nil, fmt.Errorf("start date must be before end date")
	}

	return &from, &to, nil
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
