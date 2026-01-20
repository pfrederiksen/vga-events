package event

import (
	"crypto/sha1"
	"fmt"
	"time"
)

// Event represents a VGA Golf state event
type Event struct {
	ID        string    `json:"id"`
	State     string    `json:"state"`
	Title     string    `json:"title"`
	DateText  string    `json:"date_text"`
	City      string    `json:"city,omitempty"`
	Raw       string    `json:"raw"`
	SourceURL string    `json:"source_url"`
	FirstSeen time.Time `json:"first_seen"`
}

// GenerateID creates a deterministic ID for an event based on stable fields
func GenerateID(state, raw string) string {
	h := sha1.New()
	h.Write([]byte(state + "|" + raw))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// NewEvent creates a new Event with ID and FirstSeen populated
func NewEvent(state, title, dateText, city, raw, sourceURL string) *Event {
	return &Event{
		ID:        GenerateID(state, raw),
		State:     state,
		Title:     title,
		DateText:  dateText,
		City:      city,
		Raw:       raw,
		SourceURL: sourceURL,
		FirstSeen: time.Now().UTC(),
	}
}
