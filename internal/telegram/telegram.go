package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"time"
)

const timeout = 10 * time.Second

// apiBaseURL is a package variable (not const) to allow test overriding
var apiBaseURL = "https://api.telegram.org/bot"

// InlineKeyboardButton represents a button in an inline keyboard
type InlineKeyboardButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data,omitempty"`
	URL          string `json:"url,omitempty"`
}

// InlineKeyboardMarkup represents an inline keyboard
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton `json:"inline_keyboard"`
}

// Client represents a Telegram Bot API client
type Client struct {
	botToken   string
	chatID     string
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
		"chat_id":                  c.chatID,
		"text":                     text,
		"parse_mode":               "HTML",
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

// SendMessageWithKeyboard sends a text message with an inline keyboard to the configured chat
func (c *Client) SendMessageWithKeyboard(text string, keyboard *InlineKeyboardMarkup) error {
	if text == "" {
		return fmt.Errorf("message text is required")
	}

	url := fmt.Sprintf("%s%s/sendMessage", apiBaseURL, c.botToken)

	payload := map[string]interface{}{
		"chat_id":                  c.chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}

	if keyboard != nil {
		payload["reply_markup"] = keyboard
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

// SendDocument sends a file document to the configured chat
func (c *Client) SendDocument(filename string, content []byte, caption string) error {
	url := fmt.Sprintf("%s%s/sendDocument", apiBaseURL, c.botToken)

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add chat_id field
	if err := writer.WriteField("chat_id", c.chatID); err != nil {
		return fmt.Errorf("writing chat_id field: %w", err)
	}

	// Add caption if provided
	if caption != "" {
		if err := writer.WriteField("caption", caption); err != nil {
			return fmt.Errorf("writing caption field: %w", err)
		}
		if err := writer.WriteField("parse_mode", "HTML"); err != nil {
			return fmt.Errorf("writing parse_mode field: %w", err)
		}
	}

	// Add file
	part, err := writer.CreateFormFile("document", filename)
	if err != nil {
		return fmt.Errorf("creating form file: %w", err)
	}

	if _, err := part.Write(content); err != nil {
		return fmt.Errorf("writing file content: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("closing multipart writer: %w", err)
	}

	// Send POST request
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	// Parse response to check for errors
	var result struct {
		OK          bool   `json:"ok"`
		Description string `json:"description"`
	}

	if err := json.Unmarshal(respBody, &result); err != nil {
		return fmt.Errorf("parsing response: %w", err)
	}

	if !result.OK {
		return fmt.Errorf("telegram API error: %s", result.Description)
	}

	return nil
}
