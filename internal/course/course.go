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
	Source  string  // Data source: "manual", "usga", or "cache"
}

// globalCache is the shared course cache instance
var globalCache *CourseCache

// globalUSGAClient is the shared USGA API client
var globalUSGAClient *USGAClient

// init initializes global cache and USGA client
func init() {
	globalCache = NewCourseCache()
	globalUSGAClient = NewUSGAClient()
}

// LookupCourse attempts to find course information by title and state
// Uses three-tier lookup: manual DB → cache → USGA API
// Returns nil if no information is available
func LookupCourse(title, state string) *CourseInfo {
	return LookupCourseWithCache(title, state, globalCache, globalUSGAClient)
}

// LookupCourseWithCache performs course lookup with explicit cache and client
// Useful for testing and custom configurations
func LookupCourseWithCache(title, state string, cache *CourseCache, usgaClient *USGAClient) *CourseInfo {
	// 1. Try manual database first (instant, no API call)
	if info := lookupManual(title, state); info != nil {
		return info
	}

	// 2. Try cache (fast, no API call)
	if cache != nil {
		if info := cache.Get(title, state); info != nil {
			return info
		}
	}

	// 3. Try USGA API (slow, requires network)
	if usgaClient != nil {
		info, err := usgaClient.SearchCourse(title, state)
		if err != nil {
			// Log error but don't fail - graceful degradation
			// In production, could log: fmt.Fprintf(os.Stderr, "USGA API error: %v\n", err)
		} else if info != nil {
			// Cache successful lookup
			if cache != nil {
				cache.Set(title, state, info)
			}
			return info
		}
	}

	// No information available from any source
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
