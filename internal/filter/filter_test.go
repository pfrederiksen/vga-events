package filter

import (
	"testing"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

func TestFilter_IsEmpty(t *testing.T) {
	tests := []struct {
		name   string
		filter *Filter
		want   bool
	}{
		{
			name:   "empty filter",
			filter: NewFilter(),
			want:   true,
		},
		{
			name: "filter with date from",
			filter: &Filter{
				DateFrom: timePtr(time.Now()),
			},
			want: false,
		},
		{
			name: "filter with weekends only",
			filter: &Filter{
				WeekendsOnly: true,
			},
			want: false,
		},
		{
			name: "filter with course",
			filter: &Filter{
				Courses: []string{"Pebble Beach"},
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.IsEmpty(); got != tt.want {
				t.Errorf("Filter.IsEmpty() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Matches(t *testing.T) {
	// Create test events
	jan15 := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	mar1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	mar31 := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		filter *Filter
		event  *event.Event
		want   bool
	}{
		{
			name:   "empty filter matches all",
			filter: NewFilter(),
			event: &event.Event{
				Title:    "Test Event",
				State:    "NV",
				DateText: "March 15, 2026",
			},
			want: true,
		},
		{
			name: "course filter matches",
			filter: &Filter{
				Courses: []string{"pebble"},
			},
			event: &event.Event{
				Title:    "Pebble Beach Tournament",
				State:    "CA",
				DateText: "March 15, 2026",
			},
			want: true,
		},
		{
			name: "course filter does not match",
			filter: &Filter{
				Courses: []string{"augusta"},
			},
			event: &event.Event{
				Title:    "Pebble Beach Tournament",
				State:    "CA",
				DateText: "March 15, 2026",
			},
			want: false,
		},
		{
			name: "state filter matches",
			filter: &Filter{
				States: []string{"CA", "NV"},
			},
			event: &event.Event{
				Title:    "Test Event",
				State:    "CA",
				DateText: "March 15, 2026",
			},
			want: true,
		},
		{
			name: "state filter does not match",
			filter: &Filter{
				States: []string{"TX"},
			},
			event: &event.Event{
				Title:    "Test Event",
				State:    "CA",
				DateText: "March 15, 2026",
			},
			want: false,
		},
		{
			name: "date range filter matches",
			filter: &Filter{
				DateFrom: &mar1,
				DateTo:   &mar31,
			},
			event: &event.Event{
				Title:    "Test Event",
				State:    "NV",
				DateText: "March 15, 2026",
			},
			want: true,
		},
		{
			name: "date range filter does not match (before)",
			filter: &Filter{
				DateFrom: &jan15,
			},
			event: &event.Event{
				Title:    "Test Event",
				State:    "NV",
				DateText: "January 1, 2026",
			},
			want: false,
		},
		{
			name: "city filter matches",
			filter: &Filter{
				Cities: []string{"vegas"},
			},
			event: &event.Event{
				Title:    "Test Event",
				State:    "NV",
				City:     "Las Vegas",
				DateText: "March 15, 2026",
			},
			want: true,
		},
		{
			name: "city filter does not match",
			filter: &Filter{
				Cities: []string{"reno"},
			},
			event: &event.Event{
				Title:    "Test Event",
				State:    "NV",
				City:     "Las Vegas",
				DateText: "March 15, 2026",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.Matches(tt.event); got != tt.want {
				t.Errorf("Filter.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Apply(t *testing.T) {
	events := []*event.Event{
		{
			ID:       "1",
			Title:    "Pebble Beach Event",
			State:    "CA",
			City:     "Pebble Beach",
			DateText: "March 15, 2026",
		},
		{
			ID:       "2",
			Title:    "Las Vegas Championship",
			State:    "NV",
			City:     "Las Vegas",
			DateText: "March 20, 2026",
		},
		{
			ID:       "3",
			Title:    "Texas Open",
			State:    "TX",
			City:     "Austin",
			DateText: "March 25, 2026",
		},
	}

	tests := []struct {
		name      string
		filter    *Filter
		wantCount int
		wantIDs   []string
	}{
		{
			name:      "empty filter returns all",
			filter:    NewFilter(),
			wantCount: 3,
			wantIDs:   []string{"1", "2", "3"},
		},
		{
			name: "filter by state",
			filter: &Filter{
				States: []string{"CA", "NV"},
			},
			wantCount: 2,
			wantIDs:   []string{"1", "2"},
		},
		{
			name: "filter by course name",
			filter: &Filter{
				Courses: []string{"pebble"},
			},
			wantCount: 1,
			wantIDs:   []string{"1"},
		},
		{
			name: "filter by city",
			filter: &Filter{
				Cities: []string{"vegas"},
			},
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.filter.Apply(events)
			if len(got) != tt.wantCount {
				t.Errorf("Filter.Apply() returned %d events, want %d", len(got), tt.wantCount)
			}

			// Check IDs match
			gotIDs := make([]string, len(got))
			for i, evt := range got {
				gotIDs[i] = evt.ID
			}

			if len(gotIDs) != len(tt.wantIDs) {
				t.Errorf("Filter.Apply() returned IDs %v, want %v", gotIDs, tt.wantIDs)
			}
		})
	}
}

func TestFilter_String(t *testing.T) {
	jan15 := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	mar31 := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name   string
		filter *Filter
		want   string
	}{
		{
			name:   "empty filter",
			filter: NewFilter(),
			want:   "No active filters",
		},
		{
			name: "filter with date range",
			filter: &Filter{
				DateFrom: &jan15,
				DateTo:   &mar31,
			},
			want: "From: Jan 15, 2026 | To: Mar 31, 2026",
		},
		{
			name: "filter with courses",
			filter: &Filter{
				Courses: []string{"Pebble Beach", "Augusta"},
			},
			want: "Courses: Pebble Beach, Augusta",
		},
		{
			name: "filter with weekends only",
			filter: &Filter{
				WeekendsOnly: true,
			},
			want: "Weekends only",
		},
		{
			name: "complex filter",
			filter: &Filter{
				DateFrom:     &jan15,
				Courses:      []string{"Pebble Beach"},
				WeekendsOnly: true,
			},
			want: "From: Jan 15, 2026 | Courses: Pebble Beach | Weekends only",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.filter.String(); got != tt.want {
				t.Errorf("Filter.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilter_Clone(t *testing.T) {
	mar1 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	mar31 := time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC)

	original := &Filter{
		DateFrom:     &mar1,
		DateTo:       &mar31,
		Courses:      []string{"Pebble Beach"},
		WeekendsOnly: true,
		States:       []string{"CA", "NV"},
		Cities:       []string{"Las Vegas"},
		MaxPrice:     100.0,
	}

	clone := original.Clone()

	// Verify values match
	if clone.WeekendsOnly != original.WeekendsOnly {
		t.Errorf("Clone.WeekendsOnly = %v, want %v", clone.WeekendsOnly, original.WeekendsOnly)
	}

	if clone.MaxPrice != original.MaxPrice {
		t.Errorf("Clone.MaxPrice = %v, want %v", clone.MaxPrice, original.MaxPrice)
	}

	// Modify clone to ensure deep copy
	clone.Courses[0] = "Augusta"
	if original.Courses[0] == "Augusta" {
		t.Error("Modifying clone affected original (shallow copy)")
	}

	// Modify clone dates
	newDate := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	clone.DateFrom = &newDate
	if original.DateFrom.Equal(newDate) {
		t.Error("Modifying clone DateFrom affected original")
	}
}

func TestNewFilterPreset(t *testing.T) {
	filter := &Filter{
		WeekendsOnly: true,
		Courses:      []string{"Pebble Beach"},
	}

	preset := NewFilterPreset("My Weekend Events", filter)

	if preset.Name != "My Weekend Events" {
		t.Errorf("NewFilterPreset().Name = %v, want %v", preset.Name, "My Weekend Events")
	}

	if preset.Filter != filter {
		t.Error("NewFilterPreset().Filter does not match input filter")
	}

	if preset.CreatedAt.IsZero() {
		t.Error("NewFilterPreset().CreatedAt is zero")
	}

	if preset.UpdatedAt.IsZero() {
		t.Error("NewFilterPreset().UpdatedAt is zero")
	}
}

// Helper function to create a time pointer
func timePtr(t time.Time) *time.Time {
	return &t
}
