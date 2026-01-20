package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	apiBaseURL = "https://api.telegram.org/bot"
	timeout    = 10 * time.Second
)

// Client represents a Telegram Bot API client
type Client struct {
	botToken string
	chatID   string
	httpClient *http.Client
}

// NewClient creates a new Telegram client
func NewClient(botToken, chatID string) (*Client, error) {
	if botToken == "" {
		return nil, fmt.Errorf("bot token is required")
	}
	if chatID == "" {
		return nil, fmt.Errorf("chat ID is required")
	}

	return &Client{
		botToken: botToken,
		chatID:   chatID,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}, nil
}

// SendMessage sends a text message to the configured chat
func (c *Client) SendMessage(text string) error {
	if text == "" {
		return fmt.Errorf("message text is required")
	}

	url := fmt.Sprintf("%s%s/sendMessage", apiBaseURL, c.botToken)

	payload := map[string]interface{}{
		"chat_id": c.chatID,
		"text":    text,
		"parse_mode": "HTML",
		"disable_web_page_preview": true,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshaling payload: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, string(body))
	}

	// Parse response to check for errors
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram API error: %s", result.Description)
	}

	return nil
}
