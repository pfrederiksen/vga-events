package course

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// USGAClient provides access to the USGA Course Rating Database
// No authentication required - this is a public database
type USGAClient struct {
	BaseURL    string
	HTTPClient *http.Client
	UserAgent  string
}

// NewUSGAClient creates a new USGA API client
func NewUSGAClient() *USGAClient {
	return &USGAClient{
		BaseURL: "https://ncrdb.usga.org",
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		UserAgent: "Mozilla/5.0 (compatible; VGA-Events-Bot/1.0)",
	}
}

// USGACourseResult represents a course from USGA search results
type USGACourseResult struct {
	FacilityName string `json:"facilityName"`
	CourseName   string `json:"courseName"`
	City         string `json:"city"`
	State        string `json:"state"`
	CourseID     string `json:"courseId"`
}

// USGASearchResponse represents the response from LoadCourses endpoint
type USGASearchResponse struct {
	Data []USGACourseResult `json:"data"`
}

// USGACourseDetails represents detailed course information
type USGACourseDetails struct {
	CourseName string    `json:"courseName"`
	City       string    `json:"city"`
	State      string    `json:"state"`
	Tees       []USGATee `json:"tees"`
}

// USGATee represents tee box information
type USGATee struct {
	TeeColor     string  `json:"teeColor"`
	TeeName      string  `json:"teeName"`
	CourseRating float64 `json:"courseRating"`
	SlopeRating  int     `json:"slopeRating"`
	Par          int     `json:"par"`
	Yardage      int     `json:"yardage"`
}

// SearchCourse searches the USGA database for a course by name and state
// Returns nil if no match found or API unavailable
func (c *USGAClient) SearchCourse(name, state string) (*CourseInfo, error) {
	// Step 1: Search for courses matching the name
	courses, err := c.searchCourses(name, "", state, "USA")
	if err != nil {
		return nil, fmt.Errorf("searching courses: %w", err)
	}

	if len(courses) == 0 {
		return nil, nil // No results
	}

	// Step 2: Find best match
	bestMatch := findBestUSGAMatch(courses, name, state)
	if bestMatch == nil {
		return nil, nil
	}

	// Step 3: Get detailed course information
	details, err := c.getCourseDetails(bestMatch.CourseID)
	if err != nil {
		// If we can't get details, return basic info from search result
		return &CourseInfo{
			Name:   bestMatch.FacilityName + " - " + bestMatch.CourseName,
			City:   bestMatch.City,
			State:  bestMatch.State,
			Source: "usga",
		}, nil
	}

	// Step 4: Select championship tees (longest/hardest)
	tee := selectChampionshipTee(details.Tees)
	if tee == nil {
		return &CourseInfo{
			Name:   details.CourseName,
			City:   details.City,
			State:  details.State,
			Source: "usga",
		}, nil
	}

	// Step 5: Build CourseInfo from details
	info := &CourseInfo{
		Name:    details.CourseName,
		City:    details.City,
		State:   details.State,
		Yardage: tee.Yardage,
		Par:     tee.Par,
		Slope:   tee.SlopeRating,
		Rating:  tee.CourseRating,
		Website: "", // USGA doesn't include website URLs
		Source:  "usga",
	}

	return info, nil
}

// searchCourses queries the USGA LoadCourses endpoint
func (c *USGAClient) searchCourses(clubName, clubCity, clubState, clubCountry string) ([]USGACourseResult, error) {
	searchURL := c.BaseURL + "/NCRListing?handler=LoadCourses"

	// Build form data
	formData := url.Values{}
	formData.Set("clubName", clubName)
	formData.Set("clubCity", clubCity)
	formData.Set("clubState", clubState)
	formData.Set("clubCountry", clubCountry)

	// Create POST request
	req, err := http.NewRequest("POST", searchURL, strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", c.UserAgent)
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("api request failed: %w", err)
	}
	defer resp.Body.Close()

	// Check status
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("api returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var result USGASearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("parsing response: %w", err)
	}

	return result.Data, nil
}

// getCourseDetails fetches detailed tee information for a course by parsing HTML
// Note: This uses HTML parsing since USGA doesn't provide a JSON API for course details.
// The parsing is best-effort and may fail if the HTML structure changes.
// Graceful degradation: If parsing fails, SearchCourse falls back to basic info.
func (c *USGAClient) getCourseDetails(courseID string) (*USGACourseDetails, error) {
	detailsURL := fmt.Sprintf("%s/courseTeeInfo?CourseID=%s", c.BaseURL, courseID)

	req, err := http.NewRequest("GET", detailsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("User-Agent", c.UserAgent)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching course details: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	htmlContent := string(body)

	// Parse course name and location
	details := &USGACourseDetails{
		Tees: []USGATee{},
	}

	// Extract course name - look for heading or title
	courseNameRe := regexp.MustCompile(`<h2[^>]*>([^<]+)</h2>`)
	if matches := courseNameRe.FindStringSubmatch(htmlContent); len(matches) > 1 {
		details.CourseName = strings.TrimSpace(matches[1])
	}

	// Parse tee information from table rows
	// Look for patterns like: White | 72 | 6500 | 125 | 71.5
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

	if len(details.Tees) == 0 {
		return nil, fmt.Errorf("no tee information found in HTML")
	}

	return details, nil
}

// findBestUSGAMatch finds the best matching course from search results
func findBestUSGAMatch(courses []USGACourseResult, searchName, searchState string) *USGACourseResult {
	if len(courses) == 0 {
		return nil
	}

	searchNorm := strings.ToLower(strings.TrimSpace(searchName))
	stateNorm := strings.ToLower(strings.TrimSpace(searchState))

	// First pass: exact facility name match with state match
	for i := range courses {
		facilityNorm := strings.ToLower(courses[i].FacilityName)
		courseStateNorm := strings.ToLower(courses[i].State)

		if facilityNorm == searchNorm && courseStateNorm == stateNorm {
			return &courses[i]
		}
	}

	// Second pass: partial facility name match with state match
	for i := range courses {
		facilityNorm := strings.ToLower(courses[i].FacilityName)
		courseStateNorm := strings.ToLower(courses[i].State)

		if strings.Contains(facilityNorm, searchNorm) && courseStateNorm == stateNorm {
			return &courses[i]
		}
	}

	// Third pass: any match with correct state
	for i := range courses {
		courseStateNorm := strings.ToLower(courses[i].State)
		if courseStateNorm == stateNorm {
			return &courses[i]
		}
	}

	// Last resort: return first result
	return &courses[0]
}

// selectChampionshipTee selects the championship tees (longest/hardest)
func selectChampionshipTee(tees []USGATee) *USGATee {
	if len(tees) == 0 {
		return nil
	}

	// Prefer "Championship", "Black", "Blue", "Gold" tees
	championshipColors := map[string]int{
		"championship": 5,
		"black":        4,
		"blue":         3,
		"gold":         2,
		"white":        1,
	}

	var best *USGATee
	bestScore := -1

	for i := range tees {
		colorNorm := strings.ToLower(tees[i].TeeColor)
		score, exists := championshipColors[colorNorm]

		if exists && score > bestScore {
			best = &tees[i]
			bestScore = score
		}
	}

	// If no known championship tee, take the one with highest slope
	if best == nil {
		maxSlope := 0
		for i := range tees {
			if tees[i].SlopeRating > maxSlope {
				maxSlope = tees[i].SlopeRating
				best = &tees[i]
			}
		}
	}

	// If still nothing, take first tee
	if best == nil && len(tees) > 0 {
		best = &tees[0]
	}

	return best
}
