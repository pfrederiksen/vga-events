package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pfrederiksen/vga-events/internal/event"
)

// Storage handles persistence of event snapshots
type Storage struct {
	dataDir string
}

// New creates a new Storage instance
func New(dataDir string) (*Storage, error) {
	// Expand ~ to home directory
	if strings.HasPrefix(dataDir, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home directory: %w", err)
		}
		dataDir = filepath.Join(home, dataDir[2:])
	}

	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("creating data directory: %w", err)
	}

	return &Storage{
		dataDir: dataDir,
	}, nil
}

// getSnapshotPath returns the path to the snapshot file
func (s *Storage) getSnapshotPath(state string) string {
	if state == "" || strings.ToUpper(state) == "ALL" {
		return filepath.Join(s.dataDir, "snapshot.json")
	}
	return filepath.Join(s.dataDir, fmt.Sprintf("snapshot_%s.json", strings.ToUpper(state)))
}

// LoadSnapshot loads a snapshot from disk
func (s *Storage) LoadSnapshot(state string) (*event.Snapshot, error) {
	path := s.getSnapshotPath(state)

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// No previous snapshot, return empty one
			return event.NewSnapshot(), nil
		}
		return nil, fmt.Errorf("reading snapshot: %w", err)
	}

	var snapshot event.Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("parsing snapshot: %w", err)
	}

	// Ensure Events map is initialized
	if snapshot.Events == nil {
		snapshot.Events = make(map[string]*event.Event)
	}

	// Restore CourseCache TTL (excluded from JSON with json:"-")
	if snapshot.CourseCache != nil {
		snapshot.CourseCache.TTL = 7 * 24 * time.Hour
	}

	return &snapshot, nil
}

// SaveSnapshot saves a snapshot to disk
func (s *Storage) SaveSnapshot(snapshot *event.Snapshot, state string) error {
	path := s.getSnapshotPath(state)

	// Set updated timestamp
	snapshot.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding snapshot: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing snapshot: %w", err)
	}

	return nil
}

// CreateSnapshotFromEvents creates and saves a snapshot from a list of events
func (s *Storage) CreateSnapshotFromEvents(events []*event.Event, state string) error {
	snapshot := event.CreateSnapshot(events, time.Now().UTC().Format(time.RFC3339))
	return s.SaveSnapshot(snapshot, state)
}

// GetEventByID retrieves an event by ID from the snapshot
// It searches the "all" snapshot first, then falls back to state-specific snapshots
func (s *Storage) GetEventByID(eventID string) (*event.Event, error) {
	// Try loading the "all" snapshot first
	snapshot, err := s.LoadSnapshot("all")
	if err != nil {
		return nil, fmt.Errorf("loading snapshot: %w", err)
	}

	if evt, exists := snapshot.Events[eventID]; exists {
		return evt, nil
	}

	return nil, fmt.Errorf("event not found: %s", eventID)
}
