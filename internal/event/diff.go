package event

import (
	"sort"
	"strings"
)

// Snapshot represents a collection of events at a point in time
type Snapshot struct {
	Events    map[string]*Event `json:"events"`     // keyed by Event.ID
	UpdatedAt string            `json:"updated_at"` // RFC3339 timestamp
}

// NewSnapshot creates an empty snapshot
func NewSnapshot() *Snapshot {
	return &Snapshot{
		Events: make(map[string]*Event),
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
	}

	return snap
}
