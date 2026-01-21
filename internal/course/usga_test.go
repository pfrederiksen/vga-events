package course

import (
	"os"
	"regexp"
	"strconv"
	"strings"
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

//nolint:dupl // Test intentionally duplicates parsing logic to verify it
func TestParseUSGACourseDetailsHTML(t *testing.T) {
	tests := []struct {
		name          string
		htmlFile      string
		wantTeeCount  int
		wantFirstTee  string
		wantFirstPar  int
		wantFirstYard int
		wantError     bool
	}{
		{
			name:          "valid course details",
			htmlFile:      "../../testdata/fixtures/usga_course_details.html",
			wantTeeCount:  4,
			wantFirstTee:  "Championship",
			wantFirstPar:  72,
			wantFirstYard: 6828,
			wantError:     false,
		},
		{
			name:         "malformed HTML without tee table",
			htmlFile:     "../../testdata/fixtures/usga_course_malformed.html",
			wantTeeCount: 0,
			wantError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Read HTML fixture
			htmlBytes, err := os.ReadFile(tt.htmlFile)
			if err != nil {
				t.Fatalf("Failed to read fixture: %v", err)
			}

			// Parse HTML using the same regex logic as getCourseDetails
			htmlContent := string(htmlBytes)

			details := &USGACourseDetails{
				Tees: []USGATee{},
			}

			// Extract course name
			courseNameRe := regexp.MustCompile(`<h2[^>]*>([^<]+)</h2>`)
			if matches := courseNameRe.FindStringSubmatch(htmlContent); len(matches) > 1 {
				details.CourseName = strings.TrimSpace(matches[1])
			}

			// Parse tee information
			teeRowRe := regexp.MustCompile(`(?i)<tr[^>]*>.*?<td[^>]*>([^<]+)</td>.*?<td[^>]*>(\d+)</td>.*?<td[^>]*>(\d+)</td>.*?<td[^>]*>(\d+)</td>.*?<td[^>]*>([\d.]+)</td>`)

			matches := teeRowRe.FindAllStringSubmatch(htmlContent, -1)
			for _, match := range matches {
				if len(match) >= 6 {
					teeColor := strings.TrimSpace(match[1])
					par, _ := strconv.Atoi(match[2])
					yardage, _ := strconv.Atoi(match[3])
					slope, _ := strconv.Atoi(match[4])
					rating, _ := strconv.ParseFloat(match[5], 64)

					if par > 0 && yardage > 0 {
						details.Tees = append(details.Tees, USGATee{
							TeeColor:     teeColor,
							TeeName:      teeColor,
							Par:          par,
							Yardage:      yardage,
							SlopeRating:  slope,
							CourseRating: rating,
						})
					}
				}
			}

			// Check results
			if tt.wantError && len(details.Tees) > 0 {
				t.Errorf("Expected error (no tees), but got %d tees", len(details.Tees))
			}

			if !tt.wantError {
				if len(details.Tees) != tt.wantTeeCount {
					t.Errorf("Tee count = %d, want %d", len(details.Tees), tt.wantTeeCount)
				}

				if tt.wantTeeCount > 0 {
					firstTee := details.Tees[0]
					if firstTee.TeeColor != tt.wantFirstTee {
						t.Errorf("First tee color = %q, want %q", firstTee.TeeColor, tt.wantFirstTee)
					}
					if firstTee.Par != tt.wantFirstPar {
						t.Errorf("First tee par = %d, want %d", firstTee.Par, tt.wantFirstPar)
					}
					if firstTee.Yardage != tt.wantFirstYard {
						t.Errorf("First tee yardage = %d, want %d", firstTee.Yardage, tt.wantFirstYard)
					}
				}
			}
		})
	}
}

//nolint:dupl // Test intentionally duplicates parsing logic to verify edge cases
func TestParseUSGACourseDetailsEdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		html         string
		wantTeeCount int
	}{
		{
			name:         "incomplete row (missing rating)",
			html:         `<tr><td>Blue</td><td>72</td><td>6500</td><td>135</td></tr>`,
			wantTeeCount: 0, // Should skip incomplete rows
		},
		{
			name:         "zero values",
			html:         `<tr><td>Red</td><td>0</td><td>0</td><td>120</td><td>68.5</td></tr>`,
			wantTeeCount: 0, // Should skip rows with par=0 or yardage=0
		},
		{
			name: "multiple valid tees",
			html: `
				<tr><td>Championship</td><td>72</td><td>7000</td><td>145</td><td>74.5</td></tr>
				<tr><td>Blue</td><td>72</td><td>6500</td><td>140</td><td>72.3</td></tr>
			`,
			wantTeeCount: 2,
		},
		{
			name:         "non-numeric values",
			html:         `<tr><td>White</td><td>N/A</td><td>6000</td><td>130</td><td>70.0</td></tr>`,
			wantTeeCount: 0, // Should skip rows with non-numeric values
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			details := &USGACourseDetails{
				Tees: []USGATee{},
			}

			teeRowRe := regexp.MustCompile(`(?i)<tr[^>]*>.*?<td[^>]*>([^<]+)</td>.*?<td[^>]*>(\d+)</td>.*?<td[^>]*>(\d+)</td>.*?<td[^>]*>(\d+)</td>.*?<td[^>]*>([\d.]+)</td>`)

			matches := teeRowRe.FindAllStringSubmatch(tt.html, -1)
			for _, match := range matches {
				if len(match) >= 6 {
					teeColor := strings.TrimSpace(match[1])
					par, _ := strconv.Atoi(match[2])
					yardage, _ := strconv.Atoi(match[3])
					slope, _ := strconv.Atoi(match[4])
					rating, _ := strconv.ParseFloat(match[5], 64)

					if par > 0 && yardage > 0 {
						details.Tees = append(details.Tees, USGATee{
							TeeColor:     teeColor,
							TeeName:      teeColor,
							Par:          par,
							Yardage:      yardage,
							SlopeRating:  slope,
							CourseRating: rating,
						})
					}
				}
			}

			if len(details.Tees) != tt.wantTeeCount {
				t.Errorf("Tee count = %d, want %d", len(details.Tees), tt.wantTeeCount)
			}
		})
	}
}

func TestParseUSGACourseName(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		wantName string
	}{
		{
			name:     "standard h2 heading",
			html:     `<h2>Pebble Beach Golf Links</h2>`,
			wantName: "Pebble Beach Golf Links",
		},
		{
			name:     "h2 with class",
			html:     `<h2 class="course-title">Augusta National Golf Club</h2>`,
			wantName: "Augusta National Golf Club",
		},
		{
			name:     "h2 with whitespace",
			html:     `<h2>  Shadow Creek  </h2>`,
			wantName: "Shadow Creek",
		},
		{
			name:     "no h2 heading",
			html:     `<div>Course Name</div>`,
			wantName: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			courseNameRe := regexp.MustCompile(`<h2[^>]*>([^<]+)</h2>`)
			var courseName string
			if matches := courseNameRe.FindStringSubmatch(tt.html); len(matches) > 1 {
				courseName = strings.TrimSpace(matches[1])
			}

			if courseName != tt.wantName {
				t.Errorf("Course name = %q, want %q", courseName, tt.wantName)
			}
		})
	}
}
