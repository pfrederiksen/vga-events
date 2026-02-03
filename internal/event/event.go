package event

import (
	"crypto/sha1"
	"fmt"
	"strings"
	"time"
)

// Event represents a VGA Golf state event
type Event struct {
	ID        string    `json:"id"`
	StableKey string    `json:"stable_key"` // Stable identifier based on normalized title
	State     string    `json:"state"`
	Title     string    `json:"title"`
	DateText  string    `json:"date_text"`
	City      string    `json:"city,omitempty"`
	Raw       string    `json:"raw"`
	SourceURL string    `json:"source_url"`
	FirstSeen time.Time `json:"first_seen"`
	RemovedAt time.Time `json:"removed_at,omitempty"` // When event was removed from VGA website
}

// GenerateID creates a deterministic ID for an event based on stable fields
func GenerateID(state, raw string) string {
	h := sha1.New()
	h.Write([]byte(state + "|" + raw))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GenerateStableKey creates a stable identifier based on normalized title
// This key stays the same even if dates or cities change
func GenerateStableKey(state, title string) string {
	// Normalize title: lowercase, trim spaces
	normalized := strings.ToLower(strings.TrimSpace(title))
	// Remove common date-related words that might appear in titles
	// (In future, could add more sophisticated normalization)

	h := sha1.New()
	h.Write([]byte(state + "|" + normalized))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// NewEvent creates a new Event with ID, StableKey, and FirstSeen populated
func NewEvent(state, title, dateText, city, raw, sourceURL string) *Event {
	return &Event{
		ID:        GenerateID(state, raw),
		StableKey: GenerateStableKey(state, title),
		State:     state,
		Title:     title,
		DateText:  dateText,
		City:      city,
		Raw:       raw,
		SourceURL: sourceURL,
		FirstSeen: time.Now().UTC(),
	}
}
