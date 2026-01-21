package course

import (
	"testing"
)

func TestLookupCourse(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		state    string
		wantName string
		wantNil  bool
	}{
		{
			name:     "Exact match - Pebble Beach",
			title:    "Pebble Beach",
			state:    "CA",
			wantName: "Pebble Beach Golf Links",
			wantNil:  false,
		},
		{
			name:     "Exact match with suffix - Pebble Beach Golf Links",
			title:    "Pebble Beach Golf Links",
			state:    "CA",
			wantName: "Pebble Beach Golf Links",
			wantNil:  false,
		},
		{
			name:     "Case insensitive - SHADOW CREEK",
			title:    "SHADOW CREEK",
			state:    "NV",
			wantName: "Shadow Creek Golf Course",
			wantNil:  false,
		},
		{
			name:     "Partial match - Pinehurst (either No. 2 or No. 4)",
			title:    "Pinehurst",
			state:    "NC",
			wantName: "", // Don't check exact name since either is valid
			wantNil:  false,
		},
		{
			name:     "With country club suffix",
			title:    "Merion Country Club",
			state:    "PA",
			wantName: "Merion Golf Club (East Course)",
			wantNil:  false,
		},
		{
			name:     "Unknown course",
			title:    "Unknown Golf Club",
			state:    "CA",
			wantName: "",
			wantNil:  true,
		},
		{
			name:     "Wrong state",
			title:    "Pebble Beach",
			state:    "NV",
			wantName: "",
			wantNil:  true,
		},
		{
			name:     "No state provided (should still match)",
			title:    "Shadow Creek",
			state:    "",
			wantName: "Shadow Creek Golf Course",
			wantNil:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LookupCourse(tt.title, tt.state)

			if tt.wantNil {
				if got != nil {
					t.Errorf("LookupCourse(%q, %q) = %v, want nil", tt.title, tt.state, got)
				}
				return
			}

			if got == nil {
				t.Fatalf("LookupCourse(%q, %q) = nil, want course info", tt.title, tt.state)
			}

			if tt.wantName != "" && got.Name != tt.wantName {
				t.Errorf("LookupCourse(%q, %q).Name = %q, want %q", tt.title, tt.state, got.Name, tt.wantName)
			}

			if got.Source != "manual" {
				t.Errorf("LookupCourse(%q, %q).Source = %q, want 'manual'", tt.title, tt.state, got.Source)
			}

			// Verify basic info is populated
			if got.Yardage == 0 {
				t.Errorf("LookupCourse(%q, %q).Yardage = 0, want > 0", tt.title, tt.state)
			}
			if got.Par == 0 {
				t.Errorf("LookupCourse(%q, %q).Par = 0, want > 0", tt.title, tt.state)
			}
		})
	}
}

func TestNormalizeTitle(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: "Pebble Beach Golf Links",
			want:  "pebble beach",
		},
		{
			input: "Shadow Creek Golf Course",
			want:  "shadow creek",
		},
		{
			input: "Merion Country Club",
			want:  "merion",
		},
		{
			input: "TPC Sawgrass C.C.",
			want:  "tpc sawgrass",
		},
		{
			input: "Pine Valley G.C.",
			want:  "pine valley",
		},
		{
			input: "  Spyglass Hill  ",
			want:  "spyglass hill",
		},
		{
			input: "AUGUSTA NATIONAL",
			want:  "augusta national",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeTitle(tt.input)
			if got != tt.want {
				t.Errorf("normalizeTitle(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestManualDatabaseCompleteness(t *testing.T) {
	// Verify that all entries in the manual database have required fields
	for key, info := range manualDatabase {
		if info.Name == "" {
			t.Errorf("Entry %q has empty Name", key)
		}
		if info.City == "" {
			t.Errorf("Entry %q has empty City", key)
		}
		if info.State == "" {
			t.Errorf("Entry %q has empty State", key)
		}
		if info.Yardage == 0 {
			t.Errorf("Entry %q has zero Yardage", key)
		}
		if info.Par == 0 {
			t.Errorf("Entry %q has zero Par", key)
		}
		if info.Source != "manual" {
			t.Errorf("Entry %q has Source = %q, want 'manual'", key, info.Source)
		}
	}

	// Verify we have at least 40 courses
	if len(manualDatabase) < 40 {
		t.Errorf("Manual database has %d courses, want at least 40", len(manualDatabase))
	}
}
