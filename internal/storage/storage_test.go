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
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to clean up temp dir: %v", err)
		}
	}()

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
				if err := os.RemoveAll(filepath.Join(tmpDir, "snapshot.json")); err != nil {
					t.Logf("Warning: failed to remove snapshot file: %v", err)
				}
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

// validateSingleEventSnapshot validates that a snapshot contains exactly one event with the given ID
func validateSingleEventSnapshot(t *testing.T, storage *Storage, state, eventID string) {
	snapshot, err := storage.LoadSnapshot(state)
	if err != nil {
		t.Fatalf("Failed to load snapshot: %v", err)
	}
	if len(snapshot.Events) != 1 {
		t.Errorf("Snapshot has %d events, want 1", len(snapshot.Events))
	}
	if snapshot.Events[eventID] == nil {
		t.Errorf("Event %s not found in snapshot", eventID)
	}
}

func TestGetEventByID_StateSpecificFallback(t *testing.T) {
	// Create a temporary directory for test snapshots
	tmpDir, err := os.MkdirTemp("", "storage-test-fallback-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to clean up temp dir: %v", err)
		}
	}()

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

func TestCreateSnapshotFromEvents(t *testing.T) {
	// Create a temporary directory for test snapshots
	tmpDir, err := os.MkdirTemp("", "storage-test-create-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("Failed to clean up temp dir: %v", err)
		}
	}()

	// Create storage instance
	storage, err := New(tmpDir)
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	tests := []struct {
		name     string
		events   []*event.Event
		state    string
		wantErr  bool
		validate func(t *testing.T, storage *Storage, state string)
	}{
		{
			name: "Create snapshot from event list",
			events: []*event.Event{
				{
					ID:        "evt-001",
					State:     "NV",
					Title:     "Test Event 1",
					DateText:  "Mar 15 2026",
					City:      "Las Vegas",
					SourceURL: "https://example.com/1",
					FirstSeen: time.Now(),
				},
				{
					ID:        "evt-002",
					State:     "NV",
					Title:     "Test Event 2",
					DateText:  "Apr 20 2026",
					City:      "Reno",
					SourceURL: "https://example.com/2",
					FirstSeen: time.Now(),
				},
			},
			state:   "all",
			wantErr: false,
			validate: func(t *testing.T, storage *Storage, state string) {
				snapshot, err := storage.LoadSnapshot(state)
				if err != nil {
					t.Fatalf("Failed to load snapshot: %v", err)
				}
				if len(snapshot.Events) != 2 {
					t.Errorf("Snapshot has %d events, want 2", len(snapshot.Events))
				}
				if snapshot.Events["evt-001"] == nil {
					t.Error("Event evt-001 not found in snapshot")
				}
				if snapshot.Events["evt-002"] == nil {
					t.Error("Event evt-002 not found in snapshot")
				}
			},
		},
		{
			name: "Create snapshot for specific state",
			events: []*event.Event{
				{
					ID:        "ca-evt-001",
					State:     "CA",
					Title:     "California Event",
					DateText:  "May 10 2026",
					City:      "San Francisco",
					SourceURL: "https://example.com/ca1",
					FirstSeen: time.Now(),
				},
			},
			state:   "CA",
			wantErr: false,
			validate: func(t *testing.T, storage *Storage, state string) {
				validateSingleEventSnapshot(t, storage, state, "ca-evt-001")
			},
		},
		{
			name:    "Create empty snapshot",
			events:  []*event.Event{},
			state:   "all",
			wantErr: false,
			validate: func(t *testing.T, storage *Storage, state string) {
				snapshot, err := storage.LoadSnapshot(state)
				if err != nil {
					t.Fatalf("Failed to load snapshot: %v", err)
				}
				if len(snapshot.Events) != 0 {
					t.Errorf("Snapshot has %d events, want 0", len(snapshot.Events))
				}
			},
		},
		{
			name: "Overwrite existing snapshot",
			events: []*event.Event{
				{
					ID:        "new-evt-001",
					State:     "TX",
					Title:     "New Event",
					DateText:  "Jun 1 2026",
					City:      "Austin",
					SourceURL: "https://example.com/new",
					FirstSeen: time.Now(),
				},
			},
			state:   "all",
			wantErr: false,
			validate: func(t *testing.T, storage *Storage, state string) {
				// Should only have 1 event (overwrites previous snapshot)
				validateSingleEventSnapshot(t, storage, state, "new-evt-001")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.CreateSnapshotFromEvents(tt.events, tt.state)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateSnapshotFromEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, storage, tt.state)
			}
		})
	}
}

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		dataDir  string
		setup    func(t *testing.T) string // Returns actual path to use
		wantErr  bool
		validate func(t *testing.T, storage *Storage, dataDir string)
	}{
		{
			name: "Create storage with absolute path",
			setup: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "storage-new-test-*")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				t.Cleanup(func() {
					_ = os.RemoveAll(tmpDir)
				})
				return tmpDir
			},
			wantErr: false,
			validate: func(t *testing.T, storage *Storage, dataDir string) {
				if storage == nil {
					t.Fatal("Storage is nil")
				}
				if storage.dataDir != dataDir {
					t.Errorf("Storage.dataDir = %q, want %q", storage.dataDir, dataDir)
				}
				// Verify directory exists
				if _, err := os.Stat(dataDir); os.IsNotExist(err) {
					t.Error("Data directory was not created")
				}
			},
		},
		{
			name: "Create storage creates nested directories",
			setup: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "storage-new-nested-*")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				t.Cleanup(func() {
					_ = os.RemoveAll(tmpDir)
				})
				return filepath.Join(tmpDir, "nested", "path", "data")
			},
			wantErr: false,
			validate: func(t *testing.T, storage *Storage, dataDir string) {
				if storage == nil {
					t.Fatal("Storage is nil")
				}
				// Verify nested directory was created
				if _, err := os.Stat(dataDir); os.IsNotExist(err) {
					t.Error("Nested data directory was not created")
				}
			},
		},
		{
			name: "Create storage with existing directory",
			setup: func(t *testing.T) string {
				tmpDir, err := os.MkdirTemp("", "storage-new-existing-*")
				if err != nil {
					t.Fatalf("Failed to create temp dir: %v", err)
				}
				t.Cleanup(func() {
					_ = os.RemoveAll(tmpDir)
				})
				// Directory already exists
				return tmpDir
			},
			wantErr: false,
			validate: func(t *testing.T, storage *Storage, dataDir string) {
				if storage == nil {
					t.Fatal("Storage is nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dataDir := tt.setup(t)

			storage, err := New(dataDir)

			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.validate != nil {
				tt.validate(t, storage, dataDir)
			}
		})
	}
}
