package event

import (
	"sort"
	"strings"
	"time"
)

// Snapshot represents a collection of events at a point in time
type Snapshot struct {
	Events      map[string]*Event `json:"events"`       // keyed by Event.ID
	StableIndex map[string]string `json:"stable_index"` // StableKey → ID mapping
	ChangeLog   []*EventChange    `json:"change_log"`   // Recent changes
	UpdatedAt   string            `json:"updated_at"`   // RFC3339 timestamp
}

// NewSnapshot creates an empty snapshot
func NewSnapshot() *Snapshot {
	return &Snapshot{
		Events:      make(map[string]*Event),
		StableIndex: make(map[string]string),
		ChangeLog:   make([]*EventChange, 0),
	}
}

// DiffResult contains the results of comparing two snapshots
type DiffResult struct {
	NewEvents []*Event
	States    map[string][]*Event // new events grouped by state
}

// Diff compares current events against a previous snapshot and returns new events
func Diff(previous *Snapshot, current []*Event, stateFilter string) *DiffResult {
	result := &DiffResult{
		NewEvents: make([]*Event, 0),
		States:    make(map[string][]*Event),
	}

	if previous == nil {
		previous = NewSnapshot()
	}

	for _, evt := range current {
		// Apply state filter
		if stateFilter != "" && stateFilter != "ALL" {
			if !strings.EqualFold(evt.State, stateFilter) {
				continue
			}
		}

		// Check if this event exists in previous snapshot
		if _, exists := previous.Events[evt.ID]; !exists {
			result.NewEvents = append(result.NewEvents, evt)

			// Group by state
			if result.States[evt.State] == nil {
				result.States[evt.State] = make([]*Event, 0)
			}
			result.States[evt.State] = append(result.States[evt.State], evt)
		}
	}

	// Sort new events for consistent output
	sort.Slice(result.NewEvents, func(i, j int) bool {
		if result.NewEvents[i].State != result.NewEvents[j].State {
			return result.NewEvents[i].State < result.NewEvents[j].State
		}
		return result.NewEvents[i].Raw < result.NewEvents[j].Raw
	})

	// Sort within each state group
	for state := range result.States {
		sort.Slice(result.States[state], func(i, j int) bool {
			return result.States[state][i].Raw < result.States[state][j].Raw
		})
	}

	return result
}

// CreateSnapshot creates a snapshot from a list of events
func CreateSnapshot(events []*Event, updatedAt string) *Snapshot {
	snap := NewSnapshot()
	snap.UpdatedAt = updatedAt

	for _, evt := range events {
		snap.Events[evt.ID] = evt
		// Build stable index
		if evt.StableKey != "" {
			snap.StableIndex[evt.StableKey] = evt.ID
		}
	}

	return snap
}

// EventChange represents a change detected in an event
type EventChange struct {
	EventID    string    `json:"event_id"`
	StableKey  string    `json:"stable_key"`
	ChangeType string    `json:"change_type"` // "date", "title", "city", "new"
	OldValue   string    `json:"old_value"`
	NewValue   string    `json:"new_value"`
	DetectedAt time.Time `json:"detected_at"`
}

// DetectChanges compares two events and returns detected changes
func DetectChanges(previous, current *Event) []*EventChange {
	var changes []*EventChange

	// If no previous event, this is a new event
	if previous == nil {
		return []*EventChange{
			{
				EventID:    current.ID,
				StableKey:  current.StableKey,
				ChangeType: "new",
				OldValue:   "",
				NewValue:   current.Title,
				DetectedAt: time.Now().UTC(),
			},
		}
	}

	// Detect date change
	if previous.DateText != current.DateText {
		changes = append(changes, &EventChange{
			EventID:    current.ID,
			StableKey:  current.StableKey,
			ChangeType: "date",
			OldValue:   previous.DateText,
			NewValue:   current.DateText,
			DetectedAt: time.Now().UTC(),
		})
	}

	// Detect title change
	if previous.Title != current.Title {
		changes = append(changes, &EventChange{
			EventID:    current.ID,
			StableKey:  current.StableKey,
			ChangeType: "title",
			OldValue:   previous.Title,
			NewValue:   current.Title,
			DetectedAt: time.Now().UTC(),
		})
	}

	// Detect city change
	if previous.City != current.City {
		changes = append(changes, &EventChange{
			EventID:    current.ID,
			StableKey:  current.StableKey,
			ChangeType: "city",
			OldValue:   previous.City,
			NewValue:   current.City,
			DetectedAt: time.Now().UTC(),
		})
	}

	return changes
}

// CompareSnapshots compares two sets of events and returns all detected changes
func CompareSnapshots(previousEvents, currentEvents map[string]*Event, previousIndex, currentIndex map[string]string) []*EventChange {
	var allChanges []*EventChange

	// Check each stable key in current snapshot
	for stableKey, currentID := range currentIndex {
		currentEvent := currentEvents[currentID]

		// Look for previous event with same stable key
		if previousID, exists := previousIndex[stableKey]; exists {
			previousEvent := previousEvents[previousID]
			changes := DetectChanges(previousEvent, currentEvent)
			allChanges = append(allChanges, changes...)
		} else {
			// New event (stable key doesn't exist in previous)
			changes := DetectChanges(nil, currentEvent)
			allChanges = append(allChanges, changes...)
		}
	}

	return allChanges
}
