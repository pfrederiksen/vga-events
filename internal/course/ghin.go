package course

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// GHINClient provides access to the GHIN course database API
// Note: This implementation assumes public GHIN API access.
// In production, you may need authentication or alternative data sources.
type GHINClient struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

// NewGHINClient creates a new GHIN API client
func NewGHINClient() *GHINClient {
	return &GHINClient{
		BaseURL: "https://api.ghin.com", // Placeholder - actual endpoint may differ
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		UserAgent: "VGA-Events-Bot/1.0",
	}
}

// GHINSearchResult represents a course search result from GHIN
type GHINSearchResult struct {
	Courses []GHINCourse `json:"courses"`
}

// GHINCourse represents course data from GHIN API
type GHINCourse struct {
	Name      string  `json:"name"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	Yardage   int     `json:"yardage"`
	Par       int     `json:"par"`
	Slope     int     `json:"slope"`
	Rating    float64 `json:"rating"`
	CourseID  string  `json:"course_id"`
	TeeColor  string  `json:"tee_color"`
}

// SearchCourse searches GHIN database for a course by name and state
// Returns nil if no match found or API unavailable
func (c *GHINClient) SearchCourse(name, state string) (*CourseInfo, error) {
	// Note: This is a placeholder implementation
	// Real GHIN API may require different endpoints, authentication, or parameters

	// Build search URL
	searchURL := fmt.Sprintf("%s/api/v1/courses/search", c.BaseURL)
	params := url.Values{}
	params.Add("name", name)
	if state != "" {
		params.Add("state", state)
	}

	fullURL := fmt.Sprintf("%s?%s", searchURL, params.Encode())

	// Create request
	req, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		// API not available - gracefully degrade
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	// Handle non-200 responses gracefully
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result GHINSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	// Find best match
	if len(result.Courses) == 0 {
		return nil, nil // No results
	}

	// Take first result (could be enhanced with fuzzy matching)
	ghinCourse := result.Courses[0]

	// Convert to CourseInfo
	info := &CourseInfo{
		Name:    ghinCourse.Name,
		City:    ghinCourse.City,
		State:   ghinCourse.State,
		Yardage: ghinCourse.Yardage,
		Par:     ghinCourse.Par,
		Slope:   ghinCourse.Slope,
		Rating:  ghinCourse.Rating,
		Website: "", // GHIN doesn't typically include website
		Source:  "ghin",
	}

	return info, nil
}

// findBestMatch finds the best matching course from search results
// Uses simple fuzzy matching on course name
func findBestMatch(courses []GHINCourse, searchName string) *GHINCourse {
	if len(courses) == 0 {
		return nil
	}

	searchNorm := strings.ToLower(strings.TrimSpace(searchName))

	// First pass: exact match
	for i := range courses {
		courseNorm := strings.ToLower(courses[i].Name)
		if courseNorm == searchNorm {
			return &courses[i]
		}
	}

	// Second pass: contains match
	for i := range courses {
		courseNorm := strings.ToLower(courses[i].Name)
		if strings.Contains(courseNorm, searchNorm) || strings.Contains(searchNorm, courseNorm) {
			return &courses[i]
		}
	}

	// No good match, return first result
	return &courses[0]
}
