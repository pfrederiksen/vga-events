package preferences

import (
	"encoding/json"
	"testing"
)

// TestGistStorage_LoadWithPreferences tests loading preferences from Gist
func TestGistStorage_LoadWithPreferences(t *testing.T) {
	// Note: Load() makes actual HTTP calls to GitHub API, so these are integration tests
	// For unit testing, we would need to refactor GistStorage to accept a custom HTTP client
	// and use httptest.Server. For now, we test the JSON parsing logic through FromJSON.

	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
		wantLen  int
	}{
		{
			name:     "valid single user",
			jsonData: `{"123":{"states":["NV"],"active":true}}`,
			wantErr:  false,
			wantLen:  1,
		},
		{
			name:     "valid multiple users",
			jsonData: `{"123":{"states":["NV"],"active":true},"456":{"states":["CA"],"active":false}}`,
			wantErr:  false,
			wantLen:  2,
		},
		{
			name:     "empty preferences",
			jsonData: `{}`,
			wantErr:  false,
			wantLen:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefs, err := FromJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && len(prefs) != tt.wantLen {
				t.Errorf("FromJSON() loaded %d users, want %d", len(prefs), tt.wantLen)
			}
		})
	}
}

// TestGistStorage_SavePreferences tests saving preferences to JSON
func TestGistStorage_SavePreferences(t *testing.T) {
	tests := []struct {
		name    string
		setup   func() Preferences
		wantErr bool
	}{
		{
			name: "save single user",
			setup: func() Preferences {
				prefs := NewPreferences()
				user := prefs.GetUser("123")
				user.States = []string{"NV"}
				user.Active = true
				return prefs
			},
			wantErr: false,
		},
		{
			name: "save multiple users",
			setup: func() Preferences {
				prefs := NewPreferences()
				prefs.GetUser("123").States = []string{"NV"}
				prefs.GetUser("456").States = []string{"CA"}
				return prefs
			},
			wantErr: false,
		},
		{
			name: "save with event tracking",
			setup: func() Preferences {
				prefs := NewPreferences()
				user := prefs.GetUser("123")
				user.States = []string{"NV"}
				user.SetEventNote("evt1", "Test note")
				user.SetEventStatus("evt1", "interested")
				return prefs
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prefs := tt.setup()

			// Convert to JSON
			jsonBytes, err := json.Marshal(prefs)
			if (err != nil) != tt.wantErr {
				t.Errorf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Verify it's valid JSON
			if !tt.wantErr {
				var decoded map[string]interface{}
				if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
					t.Errorf("Saved JSON is not valid: %v", err)
				}

				// Verify we can load it back
				loadedPrefs, err := FromJSON(jsonBytes)
				if err != nil {
					t.Errorf("FromJSON() failed to load saved data: %v", err)
				}

				if len(loadedPrefs) != len(prefs) {
					t.Errorf("Roundtrip changed user count: got %d, want %d", len(loadedPrefs), len(prefs))
				}
			}
		})
	}
}

// TestGistStorage_LoadSaveRoundtrip tests that data survives save/load cycle
func TestGistStorage_LoadSaveRoundtrip(t *testing.T) {
	original := NewPreferences()

	// Setup complex user data
	user1 := original.GetUser("user1")
	user1.States = []string{"NV", "CA"}
	user1.Active = true
	user1.SetEventNote("evt1", "Test note 1")
	user1.SetEventNote("evt2", "Test note 2")
	user1.SetEventStatus("evt1", "interested")
	user1.SetEventStatus("evt2", "registered")
	user1.DaysAhead = 30
	user1.HidePastEvents = true

	user2 := original.GetUser("user2")
	user2.States = []string{"TX", "AZ"}
	user2.Active = false
	user2.DigestFrequency = DigestFrequencyDaily

	// Convert to JSON
	jsonBytes, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal() error: %v", err)
	}

	// Load back from JSON
	loaded, err := FromJSON(jsonBytes)
	if err != nil {
		t.Fatalf("FromJSON() error: %v", err)
	}

	// Verify user count
	if len(loaded) != len(original) {
		t.Errorf("User count mismatch: got %d, want %d", len(loaded), len(original))
	}

	// Verify user1 data
	loadedUser1 := loaded.GetUser("user1")
	if len(loadedUser1.States) != 2 {
		t.Errorf("User1 states count = %d, want 2", len(loadedUser1.States))
	}
	if loadedUser1.States[0] != "NV" || loadedUser1.States[1] != "CA" {
		t.Errorf("User1 states = %v, want [NV CA]", loadedUser1.States)
	}
	if !loadedUser1.Active {
		t.Error("User1 Active should be true")
	}
	if loadedUser1.EventNotes["evt1"] != "Test note 1" {
		t.Errorf("User1 note evt1 = %q, want 'Test note 1'", loadedUser1.EventNotes["evt1"])
	}
	if loadedUser1.EventStatuses["evt1"] != "interested" {
		t.Errorf("User1 status evt1 = %q, want 'interested'", loadedUser1.EventStatuses["evt1"])
	}
	if loadedUser1.DaysAhead != 30 {
		t.Errorf("User1 DaysAhead = %d, want 30", loadedUser1.DaysAhead)
	}
	if !loadedUser1.HidePastEvents {
		t.Error("User1 HidePastEvents should be true")
	}

	// Verify user2 data
	loadedUser2 := loaded.GetUser("user2")
	if loadedUser2.Active {
		t.Error("User2 Active should be false")
	}
	if loadedUser2.DigestFrequency != "daily" {
		t.Errorf("User2 DigestFrequency = %q, want 'daily'", loadedUser2.DigestFrequency)
	}
}

// TestGistStorage_ErrorHandling tests error cases
func TestGistStorage_ErrorHandling(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		wantErr  bool
	}{
		{
			name:     "invalid JSON",
			jsonData: `{invalid json`,
			wantErr:  true,
		},
		{
			name:     "wrong type",
			jsonData: `[]`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := FromJSON([]byte(tt.jsonData))

			if (err != nil) != tt.wantErr {
				t.Errorf("FromJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestNewGistStorageValidation tests GistStorage creation validation
func TestNewGistStorageValidation(t *testing.T) {
	t.Run("valid storage creation", func(t *testing.T) {
		storage, err := NewGistStorage("gist123", "token123")
		if err != nil {
			t.Errorf("NewGistStorage() unexpected error: %v", err)
		}
		if storage == nil {
			t.Fatal("NewGistStorage() returned nil storage")
		}
		if storage.httpClient == nil {
			t.Error("NewGistStorage() did not initialize httpClient")
		}
	})

	t.Run("validates required fields", func(t *testing.T) {
		_, err := NewGistStorage("", "token")
		if err == nil {
			t.Error("NewGistStorage() should error with empty gistID")
		}

		_, err = NewGistStorage("gist", "")
		if err == nil {
			t.Error("NewGistStorage() should error with empty token")
		}
	})
}
