// Package course provides course information lookup for VGA events
package course

import (
	"strings"
)

// CourseInfo contains detailed information about a golf course
type CourseInfo struct {
	Name    string  // Full course name
	City    string  // City location
	State   string  // State code (e.g., "CA", "NV")
	Yardage int     // Course yardage (0 if unknown)
	Par     int     // Course par (0 if unknown)
	Slope   int     // Course slope rating (0 if unknown)
	Rating  float64 // Course rating (0 if unknown)
	Website string  // Course website URL
	Source  string  // Data source: "manual" or "ghin"
}

// LookupCourse attempts to find course information by title and state
// Returns nil if no information is available
func LookupCourse(title, state string) *CourseInfo {
	// Try manual database first (fast lookup)
	if info := lookupManual(title, state); info != nil {
		return info
	}

	// Future: Try GHIN API if manual lookup fails
	// if info := lookupGHIN(title, state); info != nil {
	//     return info
	// }

	// No information available
	return nil
}

// normalizeTitle converts a course title to a normalized form for matching
func normalizeTitle(title string) string {
	// Convert to lowercase
	normalized := strings.ToLower(title)
	// Trim whitespace
	normalized = strings.TrimSpace(normalized)
	// Remove common suffixes
	normalized = strings.TrimSuffix(normalized, " golf links")
	normalized = strings.TrimSuffix(normalized, " golf club")
	normalized = strings.TrimSuffix(normalized, " golf course")
	normalized = strings.TrimSuffix(normalized, " country club")
	normalized = strings.TrimSuffix(normalized, " c.c.")
	normalized = strings.TrimSuffix(normalized, " g.c.")
	return normalized
}
