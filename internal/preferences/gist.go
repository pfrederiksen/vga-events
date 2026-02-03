package preferences

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/pfrederiksen/vga-events/internal/crypto"
)

const (
	gistAPIURL   = "https://api.github.com/gists"
	gistFilename = "preferences.json"
	timeout      = 15 * time.Second
)

// GistStorage implements Storage using GitHub Gists
type GistStorage struct {
	gistID      string
	githubToken string
	httpClient  *http.Client
	encryptor   *crypto.Encryptor
}

// NewGistStorage creates a new Gist-based storage
func NewGistStorage(gistID, githubToken string) (*GistStorage, error) {
	return NewGistStorageWithEncryption(gistID, githubToken, "")
}

// NewGistStorageWithEncryption creates a new Gist-based storage with optional encryption
func NewGistStorageWithEncryption(gistID, githubToken, encryptionKey string) (*GistStorage, error) {
	if gistID == "" {
		return nil, fmt.Errorf("gist ID is required")
	}
	if githubToken == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	var encryptor *crypto.Encryptor
	if encryptionKey != "" {
		encryptor = crypto.NewEncryptor(encryptionKey)
	}

	return &GistStorage{
		gistID:      gistID,
		githubToken: githubToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		encryptor: encryptor,
	}, nil
}

// Load retrieves preferences from the Gist
func (g *GistStorage) Load() (Preferences, error) {
	url := fmt.Sprintf("%s/%s", gistAPIURL, g.gistID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.githubToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching gist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Don't include response body in error to prevent information leakage
		return nil, fmt.Errorf("GitHub API error (status %d)", resp.StatusCode)
	}

	var gistResp struct {
		Files map[string]struct {
			Content string `json:"content"`
		} `json:"files"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return nil, fmt.Errorf("decoding gist response: %w", err)
	}

	file, exists := gistResp.Files[gistFilename]
	if !exists {
		// File doesn't exist yet, return empty preferences
		return NewPreferences(), nil
	}

	prefs, err := FromJSON([]byte(file.Content))
	if err != nil {
		return nil, fmt.Errorf("parsing preferences: %w", err)
	}

	// Decrypt sensitive fields if encryptor is configured
	if g.encryptor != nil {
		if err := g.decryptPreferences(prefs); err != nil {
			return nil, fmt.Errorf("decrypting preferences: %w", err)
		}
	}

	return prefs, nil
}

// Save updates the Gist with new preferences
func (g *GistStorage) Save(prefs Preferences) error {
	// Encrypt sensitive fields if encryptor is configured
	if g.encryptor != nil {
		// Create a copy to avoid modifying the original
		prefsCopy := make(Preferences)
		for chatID, userPrefs := range prefs {
			// Deep copy user preferences
			userPrefsCopy := *userPrefs
			prefsCopy[chatID] = &userPrefsCopy
		}
		if err := g.encryptPreferences(prefsCopy); err != nil {
			return fmt.Errorf("encrypting preferences: %w", err)
		}
		prefs = prefsCopy
	}

	prefsJSON, err := prefs.ToJSON()
	if err != nil {
		return fmt.Errorf("marshaling preferences: %w", err)
	}

	url := fmt.Sprintf("%s/%s", gistAPIURL, g.gistID)

	payload := map[string]interface{}{
		"files": map[string]interface{}{
			gistFilename: map[string]string{
				"content": string(prefsJSON),
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.githubToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("updating gist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Don't include response body in error to prevent information leakage
		return fmt.Errorf("GitHub API error (status %d)", resp.StatusCode)
	}

	return nil
}

// CreateGist creates a new private Gist and returns its ID
func CreateGist(githubToken, description string) (string, error) {
	if githubToken == "" {
		return "", fmt.Errorf("GitHub token is required")
	}

	initialPrefs := NewPreferences()
	prefsJSON, err := initialPrefs.ToJSON()
	if err != nil {
		return "", fmt.Errorf("marshaling initial preferences: %w", err)
	}

	payload := map[string]interface{}{
		"description": description,
		"public":      false,
		"files": map[string]interface{}{
			gistFilename: map[string]string{
				"content": string(prefsJSON),
			},
		},
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequest("POST", gistAPIURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("token %s", githubToken))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("creating gist: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		// Don't include response body in error to prevent information leakage
		return "", fmt.Errorf("GitHub API error (status %d)", resp.StatusCode)
	}

	var gistResp struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return gistResp.ID, nil
}

// encryptPreferences encrypts sensitive fields in all user preferences
func (g *GistStorage) encryptPreferences(prefs Preferences) error {
	if g.encryptor == nil {
		return nil
	}

	for _, userPrefs := range prefs {
		// Encrypt event notes
		if len(userPrefs.EventNotes) > 0 {
			encrypted, err := g.encryptor.EncryptMap(userPrefs.EventNotes)
			if err != nil {
				return fmt.Errorf("encrypting event notes: %w", err)
			}
			userPrefs.EventNotes = encrypted
		}

		// Encrypt event statuses
		if len(userPrefs.EventStatuses) > 0 {
			encrypted, err := g.encryptor.EncryptMap(userPrefs.EventStatuses)
			if err != nil {
				return fmt.Errorf("encrypting event statuses: %w", err)
			}
			userPrefs.EventStatuses = encrypted
		}

		// Encrypt invite code
		if userPrefs.InviteCode != "" {
			encrypted, err := g.encryptor.Encrypt(userPrefs.InviteCode)
			if err != nil {
				return fmt.Errorf("encrypting invite code: %w", err)
			}
			userPrefs.InviteCode = encrypted
		}
	}

	return nil
}

// decryptPreferences decrypts sensitive fields in all user preferences
func (g *GistStorage) decryptPreferences(prefs Preferences) error {
	if g.encryptor == nil {
		return nil
	}

	for _, userPrefs := range prefs {
		// Decrypt event notes
		if len(userPrefs.EventNotes) > 0 {
			decrypted, err := g.encryptor.DecryptMap(userPrefs.EventNotes)
			if err != nil {
				return fmt.Errorf("decrypting event notes: %w", err)
			}
			userPrefs.EventNotes = decrypted
		}

		// Decrypt event statuses
		if len(userPrefs.EventStatuses) > 0 {
			decrypted, err := g.encryptor.DecryptMap(userPrefs.EventStatuses)
			if err != nil {
				return fmt.Errorf("decrypting event statuses: %w", err)
			}
			userPrefs.EventStatuses = decrypted
		}

		// Decrypt invite code
		if userPrefs.InviteCode != "" {
			decrypted, err := g.encryptor.Decrypt(userPrefs.InviteCode)
			if err != nil {
				return fmt.Errorf("decrypting invite code: %w", err)
			}
			userPrefs.InviteCode = decrypted
		}
	}

	return nil
}
