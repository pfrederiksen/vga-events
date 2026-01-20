package event

import (
	"testing"
)

func TestGenerateID(t *testing.T) {
	tests := []struct {
		name     string
		state    string
		raw      string
		expected string
	}{
		{
			name:     "same input produces same ID",
			state:    "NV",
			raw:      "NV - Chimera Golf Club 4.4.26 - Las Vegas",
			expected: GenerateID("NV", "NV - Chimera Golf Club 4.4.26 - Las Vegas"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id1 := GenerateID(tt.state, tt.raw)
			id2 := GenerateID(tt.state, tt.raw)

			if id1 != id2 {
				t.Errorf("GenerateID should be deterministic, got different IDs: %s vs %s", id1, id2)
			}

			if id1 == "" {
				t.Error("GenerateID should not return empty string")
			}

			if len(id1) != 40 { // SHA1 produces 40 hex characters
				t.Errorf("expected ID length of 40, got %d", len(id1))
			}
		})
	}
}

func TestNewEvent(t *testing.T) {
	evt := NewEvent("NV", "Chimera Golf Club", "4.4.26", "Las Vegas", "NV - Chimera Golf Club 4.4.26 - Las Vegas", "https://example.com")

	if evt.ID == "" {
		t.Error("expected ID to be generated")
	}

	if evt.State != "NV" {
		t.Errorf("expected state to be 'NV', got '%s'", evt.State)
	}

	if evt.Title != "Chimera Golf Club" {
		t.Errorf("expected title to be 'Chimera Golf Club', got '%s'", evt.Title)
	}

	if evt.FirstSeen.IsZero() {
		t.Error("expected FirstSeen to be set")
	}
}
