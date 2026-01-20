package scraper

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pfrederiksen/vga-events/internal/event"
)

const (
	StateEventsURL = "https://vgagolf.org/state-events/"
	UserAgent      = "vga-events-cli/1.0 (github.com/pfrederiksen/vga-events)"
	Timeout        = 30 * time.Second
)

// Scraper handles fetching and parsing VGA Golf state events
type Scraper struct {
	client *http.Client
	url    string
}

// New creates a new Scraper instance
func New() *Scraper {
	return &Scraper{
		client: &http.Client{
			Timeout: Timeout,
		},
		url: StateEventsURL,
	}
}

// FetchEvents fetches and parses all state events from the VGA Golf website
func (s *Scraper) FetchEvents() ([]*event.Event, error) {
	req, err := http.NewRequest("GET", s.url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return s.parseEvents(resp.Body, s.url)
}

// parseEvents extracts events from HTML
func (s *Scraper) parseEvents(r io.Reader, sourceURL string) ([]*event.Event, error) {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	events := make([]*event.Event, 0)

	// Pattern to match state event lines: "STATE - Course/Event - City"
	// Example: "NV - Chimera Golf Club 4.4.26 - Las Vegas"
	stateEventPattern := regexp.MustCompile(`^([A-Z]{2})\s*-\s*(.+?)\s*-\s*(.+)$`)

	// Alternative pattern for events without city
	stateEventPatternNoCity := regexp.MustCompile(`^([A-Z]{2})\s*-\s*(.+)$`)

	// Pattern to match bracketed dates: "[Feb 13 2026]" or "[Feb 13 2026]"
	bracketedDatePattern := regexp.MustCompile(`^\[(.*?)\]$`)

	// Pattern to match date + event on same line: "[Mar 13 2026] UT - Sunbrook Golf Club - St. George"
	dateEventPattern := regexp.MustCompile(`^\[(.*?)\]\s+([A-Z]{2})\s*-\s*(.+?)\s*-\s*(.+)$`)
	dateEventPatternNoCity := regexp.MustCompile(`^\[(.*?)\]\s+([A-Z]{2})\s*-\s*(.+)$`)

	// Get all text content and process line by line to preserve order of dates and events
	allText := doc.Text()
	lines := strings.Split(allText, "\n")

	var currentDate string // Track the most recent date
	monthPattern := regexp.MustCompile(`^(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)$`)
	dayPattern := regexp.MustCompile(`^\d{1,2}$`)
	yearPattern := regexp.MustCompile(`^20\d{2}$`)

	var recentMonth, recentDay, recentYear string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Check if this line is a month name
		if monthPattern.MatchString(line) {
			recentMonth = line
			recentDay = ""
			recentYear = ""
			continue
		}

		// Check if this line is a day number (after seeing a month)
		if recentMonth != "" && dayPattern.MatchString(line) {
			recentDay = line
			continue
		}

		// Check if this line is a year (after seeing month and day)
		if recentMonth != "" && recentDay != "" && yearPattern.MatchString(line) {
			recentYear = line
			// Assemble the current date
			currentDate = fmt.Sprintf("%s %s %s", recentMonth, recentDay, recentYear)
			continue
		}

		// Check for date + event on same line (with city)
		if matches := dateEventPattern.FindStringSubmatch(line); matches != nil {
			dateText := strings.TrimSpace(matches[1])
			state := matches[2]
			title := strings.TrimSpace(matches[3])
			city := strings.TrimSpace(matches[4])

			// Extract raw event line without the date prefix
			rawLine := strings.TrimSpace(strings.TrimPrefix(line, "["+matches[1]+"]"))

			evt := event.NewEvent(state, title, dateText, city, rawLine, sourceURL)
			events = append(events, evt)
			continue
		}

		// Check for date + event on same line (without city)
		if matches := dateEventPatternNoCity.FindStringSubmatch(line); matches != nil {
			dateText := strings.TrimSpace(matches[1])
			state := matches[2]
			title := strings.TrimSpace(matches[3])

			// Skip if this looks like it might be part of a different pattern
			if strings.Contains(title, "http") || len(title) < 5 {
				continue
			}

			// Extract raw event line without the date prefix
			rawLine := strings.TrimSpace(strings.TrimPrefix(line, "["+matches[1]+"]"))

			evt := event.NewEvent(state, title, dateText, "", rawLine, sourceURL)
			events = append(events, evt)
			continue
		}

		// Check if this line is a standalone bracketed date
		if matches := bracketedDatePattern.FindStringSubmatch(line); matches != nil {
			currentDate = strings.TrimSpace(matches[1])
			continue
		}

		// Try pattern with city
		if matches := stateEventPattern.FindStringSubmatch(line); matches != nil {
			state := matches[1]
			title := strings.TrimSpace(matches[2])
			city := strings.TrimSpace(matches[3])

			// Use bracketed date if available, otherwise extract from title
			dateText := currentDate
			if dateText == "" {
				dateText = extractDate(title)
			}

			evt := event.NewEvent(state, title, dateText, city, line, sourceURL)
			events = append(events, evt)
			currentDate = "" // Reset after use
			continue
		}

		// Try pattern without city
		if matches := stateEventPatternNoCity.FindStringSubmatch(line); matches != nil {
			state := matches[1]
			title := strings.TrimSpace(matches[2])

			// Skip if this looks like it might be part of a different pattern
			if strings.Contains(title, "http") || len(title) < 5 {
				continue
			}

			// Use bracketed date if available, otherwise extract from title
			dateText := currentDate
			if dateText == "" {
				dateText = extractDate(title)
			}

			evt := event.NewEvent(state, title, dateText, "", line, sourceURL)
			events = append(events, evt)
			currentDate = "" // Reset after use
		}
	}

	// Deduplicate events by ID
	seen := make(map[string]bool)
	unique := make([]*event.Event, 0, len(events))
	for _, evt := range events {
		if !seen[evt.ID] {
			seen[evt.ID] = true
			unique = append(unique, evt)
		}
	}

	return unique, nil
}

// extractDate attempts to extract date text from a title
// Looks for patterns like "4.4.26", "Jan 24", "02/15/26", etc.
func extractDate(title string) string {
	// Pattern for dates like "4.4.26" or "04.04.26"
	datePattern1 := regexp.MustCompile(`\d{1,2}\.\d{1,2}\.\d{2,4}`)
	if match := datePattern1.FindString(title); match != "" {
		return match
	}

	// Pattern for dates like "Jan 24" or "Feb 08"
	datePattern2 := regexp.MustCompile(`(?i)(Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)\s+\d{1,2}`)
	if match := datePattern2.FindString(title); match != "" {
		return match
	}

	// Pattern for dates like "02/15/26"
	datePattern3 := regexp.MustCompile(`\d{1,2}/\d{1,2}/\d{2,4}`)
	if match := datePattern3.FindString(title); match != "" {
		return match
	}

	return ""
}
