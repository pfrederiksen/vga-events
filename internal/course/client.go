package course

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Client is a client for the Golf Course API
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	cache      *Cache
}

// NewClient creates a new Golf Course API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.golfcourseapi.com",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache: NewCache(),
	}
}

// NewClientWithCache creates a client with an existing cache
func NewClientWithCache(apiKey string, cache *Cache) *Client {
	client := NewClient(apiKey)
	client.cache = cache
	return client
}

// GetCache returns the client's cache
func (c *Client) GetCache() *Cache {
	return c.cache
}

// Location represents course location information
type Location struct {
	Address   string  `json:"address"`
	City      string  `json:"city"`
	State     string  `json:"state"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// TeeInfo represents tee-specific course information
type TeeInfo struct {
	TeeName       string  `json:"tee_name"`
	CourseRating  float64 `json:"course_rating"`
	SlopeRating   int     `json:"slope_rating"`
	BogeyRating   float64 `json:"bogey_rating"`
	TotalYards    int     `json:"total_yards"`
	TotalMeters   int     `json:"total_meters"`
	NumberOfHoles int     `json:"number_of_holes"`
	ParTotal      int     `json:"par_total"`
}

// Tees represents all tee options at a course
type Tees struct {
	Female []TeeInfo `json:"female"`
	Male   []TeeInfo `json:"male"`
}

// CourseInfo represents golf course information from the API
type CourseInfo struct {
	ID         int      `json:"id"`
	ClubName   string   `json:"club_name"`
	CourseName string   `json:"course_name"`
	Location   Location `json:"location"`
	Tees       Tees     `json:"tees"`
}

// SearchResult represents the API search response
type SearchResult struct {
	Courses []CourseInfo `json:"courses"`
}

// Search searches for golf courses by name
func (c *Client) Search(searchQuery string) ([]CourseInfo, error) {
	// Build query parameters
	params := url.Values{}
	params.Add("search_query", searchQuery)

	// Construct URL
	reqURL := fmt.Sprintf("%s/v1/search?%s", c.baseURL, params.Encode())

	// Create request
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	// Add API key header in format: Authorization: Key <key>
	req.Header.Set("Authorization", fmt.Sprintf("Key %s", c.apiKey))
	req.Header.Set("Content-Type", "application/json")

	// Make request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}

	// Parse response
	var result SearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result.Courses, nil
}

// FindBestMatch searches for a course and returns the best match
func (c *Client) FindBestMatch(courseName, city, state string) (*CourseInfo, error) {
	// Clean up course name (remove dates, special chars)
	cleanName := CleanCourseName(courseName)

	// Check cache first
	if c.cache != nil {
		if cached := c.cache.Get(cleanName, city, state); cached != nil {
			return cached, nil
		}
	}

	// Build search query with course name and location
	searchQuery := cleanName
	if city != "" {
		searchQuery = fmt.Sprintf("%s %s", cleanName, city)
	}
	if state != "" {
		searchQuery = fmt.Sprintf("%s %s", searchQuery, state)
	}

	// Search via API
	courses, err := c.Search(searchQuery)
	if err != nil {
		return nil, fmt.Errorf("searching courses: %w", err)
	}

	if len(courses) == 0 {
		// Cache negative result (store nil) to avoid repeated failed lookups
		if c.cache != nil {
			c.cache.Set(cleanName, city, state, nil)
		}
		return nil, nil
	}

	// Find best match
	var bestMatch *CourseInfo

	// If we have a city, try to match by city as well
	if city != "" {
		for i := range courses {
			if strings.EqualFold(courses[i].Location.City, city) {
				bestMatch = &courses[i]
				break
			}
		}
	}

	// If we have a state, try to match by state
	if bestMatch == nil && state != "" {
		for i := range courses {
			if strings.EqualFold(courses[i].Location.State, state) {
				bestMatch = &courses[i]
				break
			}
		}
	}

	// Return first match if no location match
	if bestMatch == nil {
		bestMatch = &courses[0]
	}

	// Cache the result
	if c.cache != nil {
		c.cache.Set(cleanName, city, state, bestMatch)
	}

	return bestMatch, nil
}

// GetBestTee returns the best tee information (prefers men's championship tees)
func (info *CourseInfo) GetBestTee() *TeeInfo {
	// Prefer men's tees
	if len(info.Tees.Male) > 0 {
		// Find championship/black tees or return first
		for i := range info.Tees.Male {
			teeName := strings.ToLower(info.Tees.Male[i].TeeName)
			if strings.Contains(teeName, "champ") || strings.Contains(teeName, "black") || strings.Contains(teeName, "tips") {
				return &info.Tees.Male[i]
			}
		}
		return &info.Tees.Male[0]
	}

	// Fallback to women's tees
	if len(info.Tees.Female) > 0 {
		return &info.Tees.Female[0]
	}

	return nil
}

// GetDisplayName returns the best display name for the course
func (info *CourseInfo) GetDisplayName() string {
	if info.CourseName != "" && info.CourseName != info.ClubName {
		return fmt.Sprintf("%s - %s", info.ClubName, info.CourseName)
	}
	if info.ClubName != "" {
		return info.ClubName
	}
	return info.CourseName
}

// CleanCourseName removes dates and other noise from course names
func CleanCourseName(name string) string {
	// Remove common date patterns like "1.25.26" or "(12/25/26)"
	name = strings.TrimSpace(name)

	// Remove trailing dates in format: M.D.YY or MM.DD.YY
	parts := strings.Fields(name)
	if len(parts) > 1 {
		lastPart := parts[len(parts)-1]
		// Check if last part looks like a date
		if strings.Count(lastPart, ".") == 2 || strings.Count(lastPart, "/") == 2 {
			name = strings.Join(parts[:len(parts)-1], " ")
		}
	}

	return strings.TrimSpace(name)
}
