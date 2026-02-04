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

func TestDiffWithRemovedEvents(t *testing.T) {
	// Create some test events
	evt1 := NewEvent("NV", "Event 1", "4.4.26", "Las Vegas", "NV - Event 1 4.4.26 - Las Vegas", "https://example.com")
	evt2 := NewEvent("NV", "Event 2", "5.5.26", "Las Vegas", "NV - Event 2 5.5.26 - Las Vegas", "https://example.com")
	evt3 := NewEvent("CA", "Event 3", "6.6.26", "Los Angeles", "CA - Event 3 6.6.26 - Los Angeles", "https://example.com")

	t.Run("detects removed events", func(t *testing.T) {
		// Previous snapshot had evt1, evt2, evt3
		previous := NewSnapshot()
		previous.Events[evt1.ID] = evt1
		previous.Events[evt2.ID] = evt2
		previous.Events[evt3.ID] = evt3

		// Current events only have evt1 (evt2 and evt3 were removed)
		current := []*Event{evt1}

		result := Diff(previous, current, "")

		if len(result.RemovedEvents) != 2 {
			t.Errorf("expected 2 removed events, got %d", len(result.RemovedEvents))
		}

		// Check that evt2 and evt3 are in removed events
		foundEvt2 := false
		foundEvt3 := false
		for _, evt := range result.RemovedEvents {
			if evt.ID == evt2.ID {
				foundEvt2 = true
			}
			if evt.ID == evt3.ID {
				foundEvt3 = true
			}
		}

		if !foundEvt2 {
			t.Error("expected evt2 to be in removed events")
		}
		if !foundEvt3 {
			t.Error("expected evt3 to be in removed events")
		}
	})

	t.Run("filters removed events by state", func(t *testing.T) {
		// Previous snapshot had all three events
		previous := NewSnapshot()
		previous.Events[evt1.ID] = evt1
		previous.Events[evt2.ID] = evt2
		previous.Events[evt3.ID] = evt3

		// Current events are empty (all removed)
		current := []*Event{}

		// Filter by NV only
		result := Diff(previous, current, "NV")

		// Should only detect NV events as removed
		if len(result.RemovedEvents) != 2 {
			t.Errorf("expected 2 removed NV events, got %d", len(result.RemovedEvents))
		}

		for _, evt := range result.RemovedEvents {
			if evt.State != "NV" {
				t.Errorf("expected only NV events in removed, got %s", evt.State)
			}
		}
	})

	t.Run("handles no removed events", func(t *testing.T) {
		// Previous snapshot had evt1
		previous := NewSnapshot()
		previous.Events[evt1.ID] = evt1

		// Current still has evt1 (no removals)
		current := []*Event{evt1}

		result := Diff(previous, current, "")

		if len(result.RemovedEvents) != 0 {
			t.Errorf("expected 0 removed events, got %d", len(result.RemovedEvents))
		}
	})
}

func TestCompareSnapshotsWithRemovals(t *testing.T) {
	// Create previous snapshot
	evt1 := NewEvent("NV", "Event 1", "Apr 4 2026", "Las Vegas", "NV - Event 1 Apr 4 2026 - Las Vegas", "https://example.com")
	evt2 := NewEvent("CA", "Event 2", "May 15 2026", "San Francisco", "CA - Event 2 May 15 2026 - San Francisco", "https://example.com")
	previousEvents := map[string]*Event{evt1.ID: evt1, evt2.ID: evt2}
	previousIndex := map[string]string{evt1.StableKey: evt1.ID, evt2.StableKey: evt2.ID}

	t.Run("detects removed events", func(t *testing.T) {
		// Current snapshot only has evt1 (evt2 was removed)
		currentEvents := map[string]*Event{evt1.ID: evt1}
		currentIndex := map[string]string{evt1.StableKey: evt1.ID}

		changes := CompareSnapshots(previousEvents, currentEvents, previousIndex, currentIndex)

		// Should have one removal change
		foundRemoval := false
		for _, change := range changes {
			if change.ChangeType == "removed" {
				foundRemoval = true
				if change.StableKey != evt2.StableKey {
					t.Errorf("expected removed event to be evt2, got StableKey %s", change.StableKey)
				}
				if change.OldValue != evt2.Title {
					t.Errorf("expected OldValue to be %s, got %s", evt2.Title, change.OldValue)
				}
				if change.NewValue != "" {
					t.Errorf("expected NewValue to be empty for removal, got %s", change.NewValue)
				}
			}
		}

		if !foundRemoval {
			t.Error("expected to find removal change")
		}
	})
}

func TestSnapshotRemovedEventsStorage(t *testing.T) {
	t.Run("stores removed events", func(t *testing.T) {
		snapshot := NewSnapshot()
		evt1 := NewEvent("NV", "Event 1", "Apr 4 2026", "Las Vegas", "NV - Event 1 - Las Vegas", "https://example.com")
		evt2 := NewEvent("CA", "Event 2", "May 15 2026", "San Francisco", "CA - Event 2 - San Francisco", "https://example.com")

		removedEvents := []*Event{evt1, evt2}
		snapshot.StoreRemovedEvents(removedEvents)

		if len(snapshot.RemovedEvents) != 2 {
			t.Errorf("expected 2 removed events stored, got %d", len(snapshot.RemovedEvents))
		}

		if _, ok := snapshot.RemovedEvents[evt1.ID]; !ok {
			t.Error("expected evt1 to be in removed events")
		}
		if _, ok := snapshot.RemovedEvents[evt2.ID]; !ok {
			t.Error("expected evt2 to be in removed events")
		}
	})

	t.Run("cleans up old removed events", func(t *testing.T) {
		snapshot := NewSnapshot()

		// Create an old event (removed 40 days ago)
		oldEvt := NewEvent("NV", "Old Event", "Apr 4 2026", "Las Vegas", "NV - Old Event - Las Vegas", "https://example.com")
		oldEvt.RemovedAt = time.Now().AddDate(0, 0, -40)

		// Create a recent event (removed 10 days ago)
		recentEvt := NewEvent("CA", "Recent Event", "May 15 2026", "San Francisco", "CA - Recent Event - San Francisco", "https://example.com")
		recentEvt.RemovedAt = time.Now().AddDate(0, 0, -10)

		snapshot.RemovedEvents[oldEvt.ID] = oldEvt
		snapshot.RemovedEvents[recentEvt.ID] = recentEvt

		// Cleanup should remove events older than 30 days
		removed := snapshot.CleanupRemovedEvents()

		if removed != 1 {
			t.Errorf("expected to clean up 1 event, got %d", removed)
		}

		if len(snapshot.RemovedEvents) != 1 {
			t.Errorf("expected 1 event remaining, got %d", len(snapshot.RemovedEvents))
		}

		if _, ok := snapshot.RemovedEvents[recentEvt.ID]; !ok {
			t.Error("expected recent event to still be in removed events")
		}

		if _, ok := snapshot.RemovedEvents[oldEvt.ID]; ok {
			t.Error("expected old event to be cleaned up")
		}
	})
}

func TestMarkDuplicates(t *testing.T) {
	t.Run("detects duplicates across states", func(t *testing.T) {
		// Create same event in three states
		evt1 := NewEvent("CA", "Pebble Beach Golf Links", "Apr 15 2026", "Monterey", "CA - Pebble Beach Golf Links Apr 15 2026 - Monterey", "https://example.com")
		evt2 := NewEvent("NV", "Pebble Beach Golf Links", "Apr 15 2026", "Las Vegas", "NV - Pebble Beach Golf Links Apr 15 2026 - Las Vegas", "https://example.com")
		evt3 := NewEvent("TX", "Pebble Beach Golf Links", "Apr 15 2026", "Austin", "TX - Pebble Beach Golf Links Apr 15 2026 - Austin", "https://example.com")

		events := []*Event{evt1, evt2, evt3}
		deduped := MarkDuplicates(events)

		// Should only have one event (CA comes first alphabetically)
		if len(deduped) != 1 {
			t.Errorf("expected 1 deduplicated event, got %d", len(deduped))
		}

		if deduped[0].State != "CA" {
			t.Errorf("expected CA event to be primary, got %s", deduped[0].State)
		}

		// Should have NV and TX in AlsoIn
		if len(deduped[0].AlsoIn) != 2 {
			t.Errorf("expected 2 states in AlsoIn, got %d", len(deduped[0].AlsoIn))
		}

		// Check that AlsoIn contains NV and TX
		foundNV := false
		foundTX := false
		for _, state := range deduped[0].AlsoIn {
			if state == "NV" {
				foundNV = true
			}
			if state == "TX" {
				foundTX = true
			}
		}

		if !foundNV {
			t.Error("expected NV in AlsoIn")
		}
		if !foundTX {
			t.Error("expected TX in AlsoIn")
		}
	})

	t.Run("handles course name variations", func(t *testing.T) {
		// Same course with different name variations
		evt1 := NewEvent("AZ", "Shadow Creek Golf Club", "May 1 2026", "Phoenix", "AZ - Shadow Creek Golf Club May 1 2026 - Phoenix", "https://example.com")
		evt2 := NewEvent("CA", "Shadow Creek Golf Course", "May 1 2026", "Las Vegas", "CA - Shadow Creek Golf Course May 1 2026 - Las Vegas", "https://example.com")
		evt3 := NewEvent("NV", "Shadow Creek", "May 1 2026", "Las Vegas", "NV - Shadow Creek May 1 2026 - Las Vegas", "https://example.com")

		events := []*Event{evt1, evt2, evt3}
		deduped := MarkDuplicates(events)

		// Should deduplicate since "golf club", "golf course" suffixes are normalized
		if len(deduped) != 1 {
			t.Errorf("expected 1 deduplicated event, got %d", len(deduped))
		}

		if len(deduped[0].AlsoIn) != 2 {
			t.Errorf("expected 2 states in AlsoIn, got %d", len(deduped[0].AlsoIn))
		}
	})

	t.Run("does not deduplicate different dates", func(t *testing.T) {
		// Same course but different dates
		evt1 := NewEvent("CA", "Pebble Beach", "Apr 15 2026", "Monterey", "CA - Pebble Beach Apr 15 2026 - Monterey", "https://example.com")
		evt2 := NewEvent("NV", "Pebble Beach", "Apr 22 2026", "Las Vegas", "NV - Pebble Beach Apr 22 2026 - Las Vegas", "https://example.com")

		events := []*Event{evt1, evt2}
		deduped := MarkDuplicates(events)

		// Should NOT deduplicate (different dates)
		if len(deduped) != 2 {
			t.Errorf("expected 2 events (different dates), got %d", len(deduped))
		}

		// Neither should have AlsoIn
		for _, evt := range deduped {
			if len(evt.AlsoIn) > 0 {
				t.Errorf("expected no AlsoIn for different dates, got %v", evt.AlsoIn)
			}
		}
	})

	t.Run("does not deduplicate different courses", func(t *testing.T) {
		// Different courses
		evt1 := NewEvent("CA", "Pebble Beach", "Apr 15 2026", "Monterey", "CA - Pebble Beach Apr 15 2026 - Monterey", "https://example.com")
		evt2 := NewEvent("NV", "TPC Las Vegas", "Apr 15 2026", "Las Vegas", "NV - TPC Las Vegas Apr 15 2026 - Las Vegas", "https://example.com")

		events := []*Event{evt1, evt2}
		deduped := MarkDuplicates(events)

		// Should NOT deduplicate (different courses)
		if len(deduped) != 2 {
			t.Errorf("expected 2 events (different courses), got %d", len(deduped))
		}
	})

	t.Run("handles no duplicates", func(t *testing.T) {
		// All unique events
		evt1 := NewEvent("CA", "Event 1", "Apr 15 2026", "Monterey", "CA - Event 1 Apr 15 2026 - Monterey", "https://example.com")
		evt2 := NewEvent("NV", "Event 2", "Apr 16 2026", "Las Vegas", "NV - Event 2 Apr 16 2026 - Las Vegas", "https://example.com")
		evt3 := NewEvent("TX", "Event 3", "Apr 17 2026", "Austin", "TX - Event 3 Apr 17 2026 - Austin", "https://example.com")

		events := []*Event{evt1, evt2, evt3}
		deduped := MarkDuplicates(events)

		// Should keep all events
		if len(deduped) != 3 {
			t.Errorf("expected 3 events (no duplicates), got %d", len(deduped))
		}

		// None should have AlsoIn
		for _, evt := range deduped {
			if len(evt.AlsoIn) > 0 {
				t.Errorf("expected no AlsoIn for unique events, got %v", evt.AlsoIn)
			}
		}
	})
}

func TestNormalizeCourseTitle(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "removes golf club suffix",
			input:    "Shadow Creek Golf Club",
			expected: "shadow creek",
		},
		{
			name:     "removes golf course suffix",
			input:    "Pebble Beach Golf Course",
			expected: "pebble beach",
		},
		{
			name:     "removes country club suffix",
			input:    "Augusta Country Club",
			expected: "augusta",
		},
		{
			name:     "removes cc abbreviation",
			input:    "Dallas CC",
			expected: "dallas",
		},
		{
			name:     "removes gc abbreviation",
			input:    "Phoenix GC",
			expected: "phoenix",
		},
		{
			name:     "removes 'the' prefix",
			input:    "The Sanctuary",
			expected: "sanctuary",
		},
		{
			name:     "handles multiple variations",
			input:    "The Pebble Beach Golf Links",
			expected: "pebble beach golf links",
		},
		{
			name:     "handles extra whitespace",
			input:    "Shadow   Creek    Golf   Club",
			expected: "shadow creek",
		},
		{
			name:     "handles lowercase input",
			input:    "shadow creek golf club",
			expected: "shadow creek",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizeCourseTitle(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeCourseTitle(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGenerateDuplicationKey(t *testing.T) {
	t.Run("same title and date produces same key", func(t *testing.T) {
		key1 := GenerateDuplicationKey("Pebble Beach Golf Club", "Apr 15 2026")
		key2 := GenerateDuplicationKey("Pebble Beach Golf Course", "Apr 15 2026")

		if key1 != key2 {
			t.Error("expected same duplication key for normalized titles")
		}
	})

	t.Run("different dates produce different keys", func(t *testing.T) {
		key1 := GenerateDuplicationKey("Pebble Beach", "Apr 15 2026")
		key2 := GenerateDuplicationKey("Pebble Beach", "Apr 22 2026")

		if key1 == key2 {
			t.Error("expected different keys for different dates")
		}
	})

	t.Run("different titles produce different keys", func(t *testing.T) {
		key1 := GenerateDuplicationKey("Pebble Beach", "Apr 15 2026")
		key2 := GenerateDuplicationKey("Shadow Creek", "Apr 15 2026")

		if key1 == key2 {
			t.Error("expected different keys for different courses")
		}
	})
}

func TestDiffWithDeduplication(t *testing.T) {
	t.Run("deduplicates new events across states", func(t *testing.T) {
		// Create previous snapshot (empty)
		previous := NewSnapshot()

		// Create current events - same event in three states
		evt1 := NewEvent("AZ", "Shadow Creek Golf Club", "May 1 2026", "Phoenix", "AZ - Shadow Creek Golf Club May 1 2026 - Phoenix", "https://example.com")
		evt2 := NewEvent("CA", "Shadow Creek Golf Course", "May 1 2026", "Los Angeles", "CA - Shadow Creek Golf Course May 1 2026 - Los Angeles", "https://example.com")
		evt3 := NewEvent("NV", "Shadow Creek", "May 1 2026", "Las Vegas", "NV - Shadow Creek May 1 2026 - Las Vegas", "https://example.com")
		evt4 := NewEvent("TX", "Dallas Country Club", "May 1 2026", "Dallas", "TX - Dallas Country Club May 1 2026 - Dallas", "https://example.com")

		current := []*Event{evt1, evt2, evt3, evt4}

		// Run diff without state filter to check all states
		result := Diff(previous, current, "")

		// Should have 2 unique events (Shadow Creek deduplicated, Dallas unique)
		if len(result.NewEvents) != 2 {
			t.Errorf("expected 2 deduplicated new events, got %d", len(result.NewEvents))
		}

		// Find the Shadow Creek event (should be AZ, alphabetically first)
		var shadowCreek *Event
		var dallas *Event
		for _, evt := range result.NewEvents {
			normalized := NormalizeCourseTitle(evt.Title)
			if normalized == "shadow creek" {
				shadowCreek = evt
			} else if normalized == "dallas" {
				dallas = evt
			}
		}

		if shadowCreek == nil {
			t.Fatal("expected to find Shadow Creek event")
		}
		if dallas == nil {
			t.Fatal("expected to find Dallas event")
		}

		// Shadow Creek should be the AZ event (alphabetically first)
		if shadowCreek.State != "AZ" {
			t.Errorf("expected AZ to be primary state, got %s", shadowCreek.State)
		}

		// Shadow Creek should have CA and NV in AlsoIn
		if len(shadowCreek.AlsoIn) != 2 {
			t.Errorf("expected 2 states in AlsoIn, got %d: %v", len(shadowCreek.AlsoIn), shadowCreek.AlsoIn)
		}

		// Check that AlsoIn contains CA and NV
		foundCA := false
		foundNV := false
		for _, state := range shadowCreek.AlsoIn {
			if state == "CA" {
				foundCA = true
			}
			if state == "NV" {
				foundNV = true
			}
		}
		if !foundCA {
			t.Error("expected CA in AlsoIn")
		}
		if !foundNV {
			t.Error("expected NV in AlsoIn")
		}

		// Dallas should not have AlsoIn
		if len(dallas.AlsoIn) > 0 {
			t.Errorf("expected Dallas to have no AlsoIn, got %v", dallas.AlsoIn)
		}

		// State grouping should only have AZ and TX (deduplicated)
		if len(result.States) != 2 {
			t.Errorf("expected 2 states in result, got %d", len(result.States))
		}

		if _, ok := result.States["AZ"]; !ok {
			t.Error("expected AZ in state groups")
		}
		if _, ok := result.States["TX"]; !ok {
			t.Error("expected TX in state groups")
		}
		if _, ok := result.States["CA"]; ok {
			t.Error("CA should not be in state groups (deduplicated)")
		}
		if _, ok := result.States["NV"]; ok {
			t.Error("NV should not be in state groups (deduplicated)")
		}
	})

	t.Run("does not deduplicate events with different dates", func(t *testing.T) {
		previous := NewSnapshot()

		// Same course, different dates
		evt1 := NewEvent("AZ", "Pebble Beach", "May 1 2026", "Phoenix", "AZ - Pebble Beach May 1 2026 - Phoenix", "https://example.com")
		evt2 := NewEvent("CA", "Pebble Beach", "May 8 2026", "Los Angeles", "CA - Pebble Beach May 8 2026 - Los Angeles", "https://example.com")

		current := []*Event{evt1, evt2}
		result := Diff(previous, current, "")

		// Should keep both events (different dates)
		if len(result.NewEvents) != 2 {
			t.Errorf("expected 2 events (different dates), got %d", len(result.NewEvents))
		}

		// Neither should have AlsoIn
		for _, evt := range result.NewEvents {
			if len(evt.AlsoIn) > 0 {
				t.Errorf("expected no AlsoIn for different dates, got %v for %s", evt.AlsoIn, evt.State)
			}
		}
	})
}
