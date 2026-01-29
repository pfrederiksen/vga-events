package course

import (
	"testing"
)

func TestCleanCourseName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "course with date suffix",
			input:    "Stallion Mountain 1.25.26",
			expected: "Stallion Mountain",
		},
		{
			name:     "course with slash date",
			input:    "Pebble Beach 12/15/25",
			expected: "Pebble Beach",
		},
		{
			name:     "course without date",
			input:    "Augusta National",
			expected: "Augusta National",
		},
		{
			name:     "course with extra spaces",
			input:    "  TPC Sawgrass  ",
			expected: "TPC Sawgrass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CleanCourseName(tt.input)
			if result != tt.expected {
				t.Errorf("CleanCourseName(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
