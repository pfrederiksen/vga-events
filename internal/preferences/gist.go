package preferences

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
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
}

// NewGistStorage creates a new Gist-based storage
func NewGistStorage(gistID, githubToken string) (*GistStorage, error) {
	if gistID == "" {
		return nil, fmt.Errorf("gist ID is required")
	}
	if githubToken == "" {
		return nil, fmt.Errorf("GitHub token is required")
	}

	return &GistStorage{
		gistID:      gistID,
		githubToken: githubToken,
		httpClient: &http.Client{
			Timeout: timeout,
		},
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
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
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

	return prefs, nil
}

// Save updates the Gist with new preferences
func (g *GistStorage) Save(prefs Preferences) error {
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
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
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
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("GitHub API error (status %d): %s", resp.StatusCode, string(body))
	}

	var gistResp struct {
		ID string `json:"id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gistResp); err != nil {
		return "", fmt.Errorf("decoding response: %w", err)
	}

	return gistResp.ID, nil
}
