package filter

import (
	"testing"
	"time"
)

// nolint:gocyclo // Test function with many test cases
func TestParseDateRange(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		checkResult func(from, to *time.Time) bool
	}{
		{
			name:    "Mar 1-15",
			input:   "Mar 1-15",
			wantErr: false,
			checkResult: func(from, to *time.Time) bool {
				return from.Month() == time.March && from.Day() == 1 &&
					to.Month() == time.March && to.Day() == 15
			},
		},
		{
			name:    "March 1-15",
			input:   "March 1-15",
			wantErr: false,
			checkResult: func(from, to *time.Time) bool {
				return from.Month() == time.March && from.Day() == 1 &&
					to.Month() == time.March && to.Day() == 15
			},
		},
		{
			name:    "Mar 1 - Mar 15",
			input:   "Mar 1 - Mar 15",
			wantErr: false,
			checkResult: func(from, to *time.Time) bool {
				return from.Month() == time.March && from.Day() == 1 &&
					to.Month() == time.March && to.Day() == 15
			},
		},
		{
			name:    "March 1 - March 15",
			input:   "March 1 - March 15",
			wantErr: false,
			checkResult: func(from, to *time.Time) bool {
				return from.Month() == time.March && from.Day() == 1 &&
					to.Month() == time.March && to.Day() == 15
			},
		},
		{
			name:    "Dec 25 - Jan 5 (cross year)",
			input:   "Dec 25 - Jan 5",
			wantErr: false,
			checkResult: func(from, to *time.Time) bool {
				return from.Month() == time.December && from.Day() == 25 &&
					to.Month() == time.January && to.Day() == 5 &&
					to.Year() > from.Year()
			},
		},
		{
			name:    "March (entire month)",
			input:   "March",
			wantErr: false,
			checkResult: func(from, to *time.Time) bool {
				return from.Month() == time.March && from.Day() == 1 &&
					to.Month() == time.March && to.Day() == 31
			},
		},
		{
			name:    "Feb (entire month)",
			input:   "Feb",
			wantErr: false,
			checkResult: func(from, to *time.Time) bool {
				// February can be 28 or 29 days
				return from.Month() == time.February && from.Day() == 1 &&
					to.Month() == time.February && (to.Day() == 28 || to.Day() == 29)
			},
		},
		{
			name:    "empty string",
			input:   "",
			wantErr: true,
		},
		{
			name:    "invalid format",
			input:   "not a date",
			wantErr: true,
		},
		{
			name:    "invalid day",
			input:   "Mar 50-60",
			wantErr: true,
		},
		{
			name:    "invalid month",
			input:   "Xxx 1-15",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, err := ParseDateRange(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDateRange() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("ParseDateRange() unexpected error: %v", err)
				return
			}

			if from == nil || to == nil {
				t.Errorf("ParseDateRange() returned nil date(s)")
				return
			}

			if tt.checkResult != nil && !tt.checkResult(from, to) {
				t.Errorf("ParseDateRange() result check failed. From: %v, To: %v", from, to)
			}

			// Verify from is before to
			if from.After(*to) {
				t.Errorf("ParseDateRange() from (%v) is after to (%v)", from, to)
			}
		})
	}
}

func TestParseMonth(t *testing.T) {
	tests := []struct {
		input string
		want  time.Month
	}{
		{"jan", time.January},
		{"January", time.January},
		{"JANUARY", time.January},
		{"feb", time.February},
		{"mar", time.March},
		{"apr", time.April},
		{"may", time.May},
		{"jun", time.June},
		{"jul", time.July},
		{"aug", time.August},
		{"sep", time.September},
		{"oct", time.October},
		{"nov", time.November},
		{"dec", time.December},
		{"invalid", time.Month(0)},
		{"", time.Month(0)},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			if got := parseMonth(tt.input); got != tt.want {
				t.Errorf("parseMonth(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGetYearForMonth(t *testing.T) {
	now := time.Now()
	currentYear := now.Year()
	currentMonth := now.Month()

	tests := []struct {
		name  string
		month time.Month
		want  int
	}{
		{
			name:  "current month",
			month: currentMonth,
			want:  currentYear,
		},
		{
			name:  "future month",
			month: currentMonth + 1,
			want:  currentYear,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getYearForMonth(tt.month)
			// Allow for edge cases around year boundary
			if got != tt.want && got != tt.want+1 {
				t.Errorf("getYearForMonth(%v) = %v, want %v (or %v for year boundary)", tt.month, got, tt.want, tt.want+1)
			}
		})
	}
}
