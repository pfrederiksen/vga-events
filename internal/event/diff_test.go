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

	// Test StableIndex
	if len(snapshot.StableIndex) != 2 {
		t.Errorf("expected 2 entries in StableIndex, got %d", len(snapshot.StableIndex))
	}

	if snapshot.StableIndex[evt1.StableKey] != evt1.ID {
		t.Error("expected evt1's StableKey to map to evt1's ID")
	}

	if snapshot.StableIndex[evt2.StableKey] != evt2.ID {
		t.Error("expected evt2's StableKey to map to evt2's ID")
	}
}

func TestDetectChanges(t *testing.T) {
	t.Run("detects new event", func(t *testing.T) {
		current := NewEvent("NV", "Pebble Beach", "Apr 4 2026", "Las Vegas", "NV - Pebble Beach Apr 4 2026 - Las Vegas", "https://example.com")

		changes := DetectChanges(nil, current)

		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}

		if changes[0].ChangeType != "new" {
			t.Errorf("expected change type 'new', got '%s'", changes[0].ChangeType)
		}

		if changes[0].NewValue != "Pebble Beach" {
			t.Errorf("expected new value 'Pebble Beach', got '%s'", changes[0].NewValue)
		}
	})

	t.Run("detects date change", func(t *testing.T) {
		previous := NewEvent("NV", "Pebble Beach", "Apr 4 2026", "Las Vegas", "NV - Pebble Beach Apr 4 2026 - Las Vegas", "https://example.com")
		current := NewEvent("NV", "Pebble Beach", "Apr 11 2026", "Las Vegas", "NV - Pebble Beach Apr 11 2026 - Las Vegas", "https://example.com")

		changes := DetectChanges(previous, current)

		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}

		if changes[0].ChangeType != "date" {
			t.Errorf("expected change type 'date', got '%s'", changes[0].ChangeType)
		}

		if changes[0].OldValue != "Apr 4 2026" {
			t.Errorf("expected old value 'Apr 4 2026', got '%s'", changes[0].OldValue)
		}

		if changes[0].NewValue != "Apr 11 2026" {
			t.Errorf("expected new value 'Apr 11 2026', got '%s'", changes[0].NewValue)
		}
	})

	t.Run("detects title change", func(t *testing.T) {
		previous := NewEvent("CA", "Pebble Beach", "May 15 2026", "Monterey", "CA - Pebble Beach May 15 2026 - Monterey", "https://example.com")
		current := NewEvent("CA", "Pebble Beach Golf Links", "May 15 2026", "Monterey", "CA - Pebble Beach Golf Links May 15 2026 - Monterey", "https://example.com")

		changes := DetectChanges(previous, current)

		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}

		if changes[0].ChangeType != "title" {
			t.Errorf("expected change type 'title', got '%s'", changes[0].ChangeType)
		}
	})

	t.Run("detects city change", func(t *testing.T) {
		previous := NewEvent("TX", "Dallas Country Club", "Jun 1 2026", "Dallas", "TX - Dallas Country Club Jun 1 2026 - Dallas", "https://example.com")
		current := NewEvent("TX", "Dallas Country Club", "Jun 1 2026", "Irving", "TX - Dallas Country Club Jun 1 2026 - Irving", "https://example.com")

		changes := DetectChanges(previous, current)

		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}

		if changes[0].ChangeType != "city" {
			t.Errorf("expected change type 'city', got '%s'", changes[0].ChangeType)
		}
	})

	t.Run("detects multiple changes", func(t *testing.T) {
		previous := NewEvent("AZ", "Phoenix Golf Resort", "Jul 10 2026", "Phoenix", "AZ - Phoenix Golf Resort Jul 10 2026 - Phoenix", "https://example.com")
		current := NewEvent("AZ", "Phoenix Golf Resort", "Jul 17 2026", "Scottsdale", "AZ - Phoenix Golf Resort Jul 17 2026 - Scottsdale", "https://example.com")

		changes := DetectChanges(previous, current)

		if len(changes) != 2 {
			t.Fatalf("expected 2 changes, got %d", len(changes))
		}

		// Check both date and city changes are detected
		foundDate := false
		foundCity := false
		for _, change := range changes {
			if change.ChangeType == "date" {
				foundDate = true
			}
			if change.ChangeType == "city" {
				foundCity = true
			}
		}

		if !foundDate {
			t.Error("expected to find date change")
		}
		if !foundCity {
			t.Error("expected to find city change")
		}
	})

	t.Run("no changes detected", func(t *testing.T) {
		previous := NewEvent("NV", "Pebble Beach", "Apr 4 2026", "Las Vegas", "NV - Pebble Beach Apr 4 2026 - Las Vegas", "https://example.com")
		current := NewEvent("NV", "Pebble Beach", "Apr 4 2026", "Las Vegas", "NV - Pebble Beach Apr 4 2026 - Las Vegas", "https://example.com")

		changes := DetectChanges(previous, current)

		if len(changes) != 0 {
			t.Errorf("expected 0 changes, got %d", len(changes))
		}
	})
}

func TestCompareSnapshots(t *testing.T) {
	// Create previous snapshot
	evt1 := NewEvent("NV", "Event 1", "Apr 4 2026", "Las Vegas", "NV - Event 1 Apr 4 2026 - Las Vegas", "https://example.com")
	previousEvents := map[string]*Event{evt1.ID: evt1}
	previousIndex := map[string]string{evt1.StableKey: evt1.ID}

	t.Run("detects changes to existing events", func(t *testing.T) {
		// Create current snapshot with same stable key but changed date
		evt1Updated := NewEvent("NV", "Event 1", "Apr 11 2026", "Las Vegas", "NV - Event 1 Apr 11 2026 - Las Vegas", "https://example.com")
		currentEvents := map[string]*Event{evt1Updated.ID: evt1Updated}
		currentIndex := map[string]string{evt1Updated.StableKey: evt1Updated.ID}

		changes := CompareSnapshots(previousEvents, currentEvents, previousIndex, currentIndex)

		if len(changes) != 1 {
			t.Fatalf("expected 1 change, got %d", len(changes))
		}

		if changes[0].ChangeType != "date" {
			t.Errorf("expected date change, got %s", changes[0].ChangeType)
		}
	})

	t.Run("detects new events", func(t *testing.T) {
		evt2 := NewEvent("CA", "New Event", "May 15 2026", "San Francisco", "CA - New Event May 15 2026 - San Francisco", "https://example.com")
		currentEvents := map[string]*Event{evt1.ID: evt1, evt2.ID: evt2}
		currentIndex := map[string]string{evt1.StableKey: evt1.ID, evt2.StableKey: evt2.ID}

		changes := CompareSnapshots(previousEvents, currentEvents, previousIndex, currentIndex)

		if len(changes) != 1 {
			t.Fatalf("expected 1 change (new event), got %d", len(changes))
		}

		if changes[0].ChangeType != "new" {
			t.Errorf("expected new event, got %s", changes[0].ChangeType)
		}
	})
}
