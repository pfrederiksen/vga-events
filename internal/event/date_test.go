package event

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		name     string
		dateText string
		wantYear int
		wantMonth time.Month
		wantDay  int
		wantZero bool
	}{
		{
			name:      "Full date Mar 13 2026",
			dateText:  "Mar 13 2026",
			wantYear:  2026,
			wantMonth: time.March,
			wantDay:   13,
		},
		{
			name:      "Full date single digit day Jan 2 2026",
			dateText:  "Jan 2 2026",
			wantYear:  2026,
			wantMonth: time.January,
			wantDay:   2,
		},
		{
			name:      "Dot format 4.4.26",
			dateText:  "4.4.26",
			wantYear:  2026,
			wantMonth: time.April,
			wantDay:   4,
		},
		{
			name:      "Dot format with leading zeros 04.04.26",
			dateText:  "04.04.26",
			wantYear:  2026,
			wantMonth: time.April,
			wantDay:   4,
		},
		{
			name:      "Slash format 02/15/26",
			dateText:  "02/15/26",
			wantYear:  2026,
			wantMonth: time.February,
			wantDay:   15,
		},
		{
			name:      "Month and day only Jan 24",
			dateText:  "Jan 24",
			wantYear:  time.Now().Year(),
			wantMonth: time.January,
			wantDay:   24,
		},
		{
			name:      "Month and single digit day Jan 2",
			dateText:  "Jan 2",
			wantYear:  time.Now().Year(),
			wantMonth: time.January,
			wantDay:   2,
		},
		{
			name:      "Empty string",
			dateText:  "",
			wantZero:  true,
		},
		{
			name:      "Invalid format",
			dateText:  "Not a date",
			wantZero:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseDate(tt.dateText)

			if tt.wantZero {
				if !got.IsZero() {
					t.Errorf("ParseDate(%q) = %v, want zero time", tt.dateText, got)
				}
				return
			}

			if got.Year() != tt.wantYear {
				t.Errorf("ParseDate(%q).Year() = %d, want %d", tt.dateText, got.Year(), tt.wantYear)
			}
			if got.Month() != tt.wantMonth {
				t.Errorf("ParseDate(%q).Month() = %v, want %v", tt.dateText, got.Month(), tt.wantMonth)
			}
			if got.Day() != tt.wantDay {
				t.Errorf("ParseDate(%q).Day() = %d, want %d", tt.dateText, got.Day(), tt.wantDay)
			}
		})
	}
}

func TestEvent_IsPastEvent(t *testing.T) {
	tests := []struct {
		name     string
		dateText string
		want     bool
	}{
		{
			name:     "Past date",
			dateText: "Jan 1 2020",
			want:     true,
		},
		{
			name:     "Future date",
			dateText: "Dec 31 2099",
			want:     false,
		},
		{
			name:     "Unparseable date",
			dateText: "invalid",
			want:     false, // Safe default: don't filter
		},
		{
			name:     "Empty date",
			dateText: "",
			want:     false, // Safe default: don't filter
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &Event{DateText: tt.dateText}
			if got := evt.IsPastEvent(); got != tt.want {
				t.Errorf("Event.IsPastEvent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestEvent_IsWithinDays(t *testing.T) {
	tomorrow := time.Now().AddDate(0, 0, 1).Format("Jan 2 2006")
	nextWeek := time.Now().AddDate(0, 0, 7).Format("Jan 2 2006")
	nextMonth := time.Now().AddDate(0, 0, 35).Format("Jan 2 2006")

	tests := []struct {
		name     string
		dateText string
		days     int
		want     bool
	}{
		{
			name:     "Within 30 days - tomorrow",
			dateText: tomorrow,
			days:     30,
			want:     true,
		},
		{
			name:     "Within 30 days - next week",
			dateText: nextWeek,
			days:     30,
			want:     true,
		},
		{
			name:     "Beyond 30 days - next month",
			dateText: nextMonth,
			days:     30,
			want:     false,
		},
		{
			name:     "Feature disabled (days=0)",
			dateText: nextMonth,
			days:     0,
			want:     true, // Always include when disabled
		},
		{
			name:     "Feature disabled (days negative)",
			dateText: nextMonth,
			days:     -1,
			want:     true, // Always include when disabled
		},
		{
			name:     "Unparseable date",
			dateText: "invalid",
			days:     30,
			want:     true, // Safe default: include
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &Event{DateText: tt.dateText}
			if got := evt.IsWithinDays(tt.days); got != tt.want {
				t.Errorf("Event.IsWithinDays(%d) = %v, want %v", tt.days, got, tt.want)
			}
		})
	}
}

func TestEvent_IsUpcoming(t *testing.T) {
	tests := []struct {
		name     string
		dateText string
		want     bool
	}{
		{
			name:     "Future date",
			dateText: "Dec 31 2099",
			want:     true,
		},
		{
			name:     "Past date",
			dateText: "Jan 1 2020",
			want:     false,
		},
		{
			name:     "Unparseable date",
			dateText: "invalid",
			want:     true, // Safe default: include
		},
		{
			name:     "Empty date",
			dateText: "",
			want:     true, // Safe default: include
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			evt := &Event{DateText: tt.dateText}
			if got := evt.IsUpcoming(); got != tt.want {
				t.Errorf("Event.IsUpcoming() = %v, want %v", got, tt.want)
			}
		})
	}
}
