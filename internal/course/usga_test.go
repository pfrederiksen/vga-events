package course

import (
	"testing"
)

func TestNewUSGAClient(t *testing.T) {
	client := NewUSGAClient()

	if client == nil {
		t.Fatal("NewUSGAClient() returned nil")
	}

	if client.BaseURL != "https://ncrdb.usga.org" {
		t.Errorf("BaseURL = %q, want %q", client.BaseURL, "https://ncrdb.usga.org")
	}

	if client.UserAgent == "" {
		t.Error("UserAgent is empty")
	}

	if client.HTTPClient == nil {
		t.Error("HTTPClient is nil")
	}
}

func TestFindBestUSGAMatch(t *testing.T) {
	courses := []USGACourseResult{
		{FacilityName: "Pebble Beach Golf Links", CourseName: "Pebble Beach", City: "Pebble Beach", State: "CA", CourseID: "12345"},
		{FacilityName: "Pebble Beach", CourseName: "Pebble Beach", City: "Monterey", State: "CA", CourseID: "23456"},
		{FacilityName: "Pebble Creek Golf Club", CourseName: "Pebble Creek", City: "Las Vegas", State: "NV", CourseID: "34567"},
	}

	tests := []struct {
		name        string
		searchName  string
		searchState string
		wantID      string
	}{
		{
			name:        "exact facility name and state match",
			searchName:  "Pebble Beach",
			searchState: "CA",
			wantID:      "23456",
		},
		{
			name:        "partial facility name match",
			searchName:  "Pebble",
			searchState: "CA",
			wantID:      "12345", // First partial match
		},
		{
			name:        "state match only",
			searchName:  "Unknown Course",
			searchState: "NV",
			wantID:      "34567",
		},
		{
			name:        "case insensitive matching",
			searchName:  "PEBBLE BEACH",
			searchState: "ca",
			wantID:      "23456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findBestUSGAMatch(courses, tt.searchName, tt.searchState)
			if result == nil {
				t.Fatal("findBestUSGAMatch returned nil")
			}
			if result.CourseID != tt.wantID {
				t.Errorf("CourseID = %q, want %q", result.CourseID, tt.wantID)
			}
		})
	}
}

func TestFindBestUSGAMatch_EmptySlice(t *testing.T) {
	result := findBestUSGAMatch(nil, "Test", "CA")
	if result != nil {
		t.Errorf("findBestUSGAMatch(nil) = %v, want nil", result)
	}

	result = findBestUSGAMatch([]USGACourseResult{}, "Test", "CA")
	if result != nil {
		t.Errorf("findBestUSGAMatch(empty slice) = %v, want nil", result)
	}
}

func TestSelectChampionshipTee(t *testing.T) {
	tees := []USGATee{
		{TeeColor: "White", TeeName: "White", CourseRating: 70.0, SlopeRating: 125, Par: 72, Yardage: 6000},
		{TeeColor: "Blue", TeeName: "Blue", CourseRating: 72.5, SlopeRating: 135, Par: 72, Yardage: 6500},
		{TeeColor: "Championship", TeeName: "Championship", CourseRating: 74.5, SlopeRating: 145, Par: 72, Yardage: 7000},
		{TeeColor: "Black", TeeName: "Black", CourseRating: 75.0, SlopeRating: 150, Par: 72, Yardage: 7200},
	}

	tests := []struct {
		name      string
		tees      []USGATee
		wantColor string
	}{
		{
			name:      "select championship tee",
			tees:      tees,
			wantColor: "Championship",
		},
		{
			name: "select black when no championship",
			tees: []USGATee{
				{TeeColor: "Blue", TeeName: "Blue", CourseRating: 72.5, SlopeRating: 135, Par: 72, Yardage: 6500},
				{TeeColor: "Black", TeeName: "Black", CourseRating: 75.0, SlopeRating: 150, Par: 72, Yardage: 7200},
			},
			wantColor: "Black",
		},
		{
			name: "select blue when no championship or black",
			tees: []USGATee{
				{TeeColor: "White", TeeName: "White", CourseRating: 70.0, SlopeRating: 125, Par: 72, Yardage: 6000},
				{TeeColor: "Blue", TeeName: "Blue", CourseRating: 72.5, SlopeRating: 135, Par: 72, Yardage: 6500},
			},
			wantColor: "Blue",
		},
		{
			name: "select highest slope when no known colors",
			tees: []USGATee{
				{TeeColor: "Red", TeeName: "Red", CourseRating: 68.0, SlopeRating: 120, Par: 72, Yardage: 5500},
				{TeeColor: "Green", TeeName: "Green", CourseRating: 73.0, SlopeRating: 140, Par: 72, Yardage: 6800},
			},
			wantColor: "Green", // Higher slope
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := selectChampionshipTee(tt.tees)
			if result == nil {
				t.Fatal("selectChampionshipTee returned nil")
			}
			if result.TeeColor != tt.wantColor {
				t.Errorf("TeeColor = %q, want %q", result.TeeColor, tt.wantColor)
			}
		})
	}
}

func TestSelectChampionshipTee_EmptySlice(t *testing.T) {
	result := selectChampionshipTee(nil)
	if result != nil {
		t.Errorf("selectChampionshipTee(nil) = %v, want nil", result)
	}

	result = selectChampionshipTee([]USGATee{})
	if result != nil {
		t.Errorf("selectChampionshipTee(empty slice) = %v, want nil", result)
	}
}

func TestSelectChampionshipTee_FirstTeeWhenNoMatch(t *testing.T) {
	tees := []USGATee{
		{TeeColor: "Unknown", TeeName: "Unknown", CourseRating: 70.0, SlopeRating: 0, Par: 72, Yardage: 6000},
	}

	result := selectChampionshipTee(tees)
	if result == nil {
		t.Fatal("selectChampionshipTee returned nil")
	}
	if result.TeeColor != "Unknown" {
		t.Errorf("TeeColor = %q, want %q", result.TeeColor, "Unknown")
	}
}
