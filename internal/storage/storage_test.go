package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

func TestGetEventByID(t *testing.T) {
	// Create a temporary directory for test snapshots
	tmpDir, err := os.MkdirTemp("", "storage-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage instance
	storage, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Test events
	event1 := &event.Event{
		ID:        "event-123",
		State:     "NV",
		Title:     "Spring Championship",
		DateText:  "Mar 15 2026",
		City:      "Las Vegas",
		SourceURL: "https://example.com/event1",
		FirstSeen: time.Now(),
	}

	event2 := &event.Event{
		ID:        "event-456",
		State:     "CA",
		Title:     "Summer Classic",
		DateText:  "Jun 20 2026",
		City:      "San Francisco",
		SourceURL: "https://example.com/event2",
		FirstSeen: time.Now(),
	}

	tests := []struct {
		name          string
		setup         func() // Setup function to create snapshots
		eventID       string
		wantEvent     *event.Event
		wantErr       bool
		wantErrString string
	}{
		{
			name: "Successfully retrieve event from 'all' snapshot",
			setup: func() {
				snapshot := event.CreateSnapshot([]*event.Event{event1, event2}, time.Now().Format(time.RFC3339))
				if err := storage.SaveSnapshot(snapshot, "all"); err != nil {
					t.Fatalf("Failed to save snapshot: %v", err)
				}
			},
			eventID:   "event-123",
			wantEvent: event1,
			wantErr:   false,
		},
		{
			name: "Retrieve different event from same snapshot",
			setup: func() {
				// Snapshot already exists from previous test
			},
			eventID:   "event-456",
			wantEvent: event2,
			wantErr:   false,
		},
		{
			name: "Event not found in snapshot",
			setup: func() {
				// Snapshot exists but doesn't contain event-789
			},
			eventID:       "event-789",
			wantEvent:     nil,
			wantErr:       true,
			wantErrString: "event not found: event-789",
		},
		{
			name: "Event not found - nonexistent ID",
			setup: func() {
				// Snapshot exists
			},
			eventID:       "nonexistent-id",
			wantEvent:     nil,
			wantErr:       true,
			wantErrString: "event not found: nonexistent-id",
		},
		{
			name: "Empty snapshot",
			setup: func() {
				// Create a new empty snapshot, overwriting previous
				snapshot := event.CreateSnapshot([]*event.Event{}, time.Now().Format(time.RFC3339))
				if err := storage.SaveSnapshot(snapshot, "all"); err != nil {
					t.Fatalf("Failed to save empty snapshot: %v", err)
				}
			},
			eventID:       "event-123",
			wantEvent:     nil,
			wantErr:       true,
			wantErrString: "event not found: event-123",
		},
		{
			name: "No snapshot file exists",
			setup: func() {
				// Remove all snapshot files
				os.RemoveAll(filepath.Join(tmpDir, "snapshot.json"))
			},
			eventID:       "event-123",
			wantEvent:     nil,
			wantErr:       true,
			wantErrString: "event not found: event-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup()
			}

			got, err := storage.GetEventByID(tt.eventID)

			// Check error
			if (err != nil) != tt.wantErr {
				t.Errorf("GetEventByID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if tt.wantErr && err != nil {
				if tt.wantErrString != "" && err.Error() != tt.wantErrString {
					t.Errorf("GetEventByID() error = %q, want %q", err.Error(), tt.wantErrString)
				}
				return
			}

			// Check event
			if !eventsEqual(got, tt.wantEvent) {
				t.Errorf("GetEventByID() = %+v, want %+v", got, tt.wantEvent)
			}
		})
	}
}

// eventsEqual compares two events for equality (ignoring time precision)
func eventsEqual(a, b *event.Event) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a.ID == b.ID &&
		a.State == b.State &&
		a.Title == b.Title &&
		a.DateText == b.DateText &&
		a.City == b.City &&
		a.SourceURL == b.SourceURL
}

func TestGetEventByID_StateSpecificFallback(t *testing.T) {
	// Create a temporary directory for test snapshots
	tmpDir, err := os.MkdirTemp("", "storage-test-fallback-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create storage instance
	storage, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Test event
	nvEvent := &event.Event{
		ID:        "nv-event-001",
		State:     "NV",
		Title:     "Nevada Championship",
		DateText:  "Apr 10 2026",
		City:      "Reno",
		SourceURL: "https://example.com/nv",
		FirstSeen: time.Now(),
	}

	// Note: Current implementation only checks "all" snapshot
	// This test documents the current behavior
	t.Run("Event not in 'all' snapshot - state-specific not implemented", func(t *testing.T) {
		// Create a state-specific snapshot
		snapshot := event.CreateSnapshot([]*event.Event{nvEvent}, time.Now().Format(time.RFC3339))
		if err := storage.SaveSnapshot(snapshot, "NV"); err != nil {
			t.Fatalf("Failed to save NV snapshot: %v", err)
		}

		// Create an empty 'all' snapshot
		emptySnapshot := event.CreateSnapshot([]*event.Event{}, time.Now().Format(time.RFC3339))
		if err := storage.SaveSnapshot(emptySnapshot, "all"); err != nil {
			t.Fatalf("Failed to save all snapshot: %v", err)
		}

		// Try to retrieve the event - should fail with current implementation
		_, err := storage.GetEventByID("nv-event-001")
		if err == nil {
			t.Error("GetEventByID() expected error when event only in state snapshot, got nil")
		}

		// This documents that the current implementation only checks "all" snapshot
		// Future enhancement could search state-specific snapshots
	})
}
