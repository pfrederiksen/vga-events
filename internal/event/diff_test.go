package event

import (
	"testing"
	"time"
)

func TestDiff(t *testing.T) {
	// Create some test events
	evt1 := NewEvent("NV", "Event 1", "4.4.26", "Las Vegas", "NV - Event 1 4.4.26 - Las Vegas", "https://example.com")
	evt2 := NewEvent("NV", "Event 2", "5.5.26", "Las Vegas", "NV - Event 2 5.5.26 - Las Vegas", "https://example.com")
	evt3 := NewEvent("CA", "Event 3", "6.6.26", "Los Angeles", "CA - Event 3 6.6.26 - Los Angeles", "https://example.com")

	// Create previous snapshot with evt1
	previous := NewSnapshot()
	previous.Events[evt1.ID] = evt1
	previous.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Current events include evt1, evt2, and evt3
	current := []*Event{evt1, evt2, evt3}

	t.Run("finds new events", func(t *testing.T) {
		result := Diff(previous, current, "")

		if len(result.NewEvents) != 2 {
			t.Errorf("expected 2 new events, got %d", len(result.NewEvents))
		}

		// Check that evt2 and evt3 are in new events
		foundEvt2 := false
		foundEvt3 := false
		for _, evt := range result.NewEvents {
			if evt.ID == evt2.ID {
				foundEvt2 = true
			}
			if evt.ID == evt3.ID {
				foundEvt3 = true
			}
		}

		if !foundEvt2 {
			t.Error("expected evt2 to be in new events")
		}
		if !foundEvt3 {
			t.Error("expected evt3 to be in new events")
		}
	})

	t.Run("filters by state", func(t *testing.T) {
		result := Diff(previous, current, "NV")

		if len(result.NewEvents) != 1 {
			t.Errorf("expected 1 new event for NV, got %d", len(result.NewEvents))
		}

		if result.NewEvents[0].ID != evt2.ID {
			t.Error("expected evt2 to be the only new NV event")
		}
	})

	t.Run("groups by state", func(t *testing.T) {
		result := Diff(previous, current, "")

		if len(result.States) != 2 {
			t.Errorf("expected 2 states, got %d", len(result.States))
		}

		if len(result.States["NV"]) != 1 {
			t.Errorf("expected 1 new event for NV, got %d", len(result.States["NV"]))
		}

		if len(result.States["CA"]) != 1 {
			t.Errorf("expected 1 new event for CA, got %d", len(result.States["CA"]))
		}
	})

	t.Run("handles nil previous snapshot", func(t *testing.T) {
		result := Diff(nil, current, "")

		if len(result.NewEvents) != 3 {
			t.Errorf("expected all 3 events to be new, got %d", len(result.NewEvents))
		}
	})
}

func TestCreateSnapshot(t *testing.T) {
	evt1 := NewEvent("NV", "Event 1", "4.4.26", "Las Vegas", "NV - Event 1 4.4.26 - Las Vegas", "https://example.com")
	evt2 := NewEvent("CA", "Event 2", "5.5.26", "Los Angeles", "CA - Event 2 5.5.26 - Los Angeles", "https://example.com")

	events := []*Event{evt1, evt2}
	updatedAt := time.Now().UTC().Format(time.RFC3339)

	snapshot := CreateSnapshot(events, updatedAt)

	if len(snapshot.Events) != 2 {
		t.Errorf("expected 2 events in snapshot, got %d", len(snapshot.Events))
	}

	if snapshot.UpdatedAt != updatedAt {
		t.Errorf("expected UpdatedAt to be '%s', got '%s'", updatedAt, snapshot.UpdatedAt)
	}

	if _, ok := snapshot.Events[evt1.ID]; !ok {
		t.Error("expected evt1 to be in snapshot")
	}

	if _, ok := snapshot.Events[evt2.ID]; !ok {
		t.Error("expected evt2 to be in snapshot")
	}
}
