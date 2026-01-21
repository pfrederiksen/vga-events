package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
)

// User represents a Telegram user
type User struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

// Message represents a Telegram message
type Message struct {
	MessageID int    `json:"message_id"`
	From      User   `json:"from"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text,omitempty"`
}

// Chat represents a Telegram chat
type Chat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

// CallbackQuery represents an incoming callback query from a button press
type CallbackQuery struct {
	ID      string   `json:"id"`
	From    User     `json:"from"`
	Message *Message `json:"message,omitempty"`
	Data    string   `json:"data"`
}

// AnswerCallbackQuery sends an answer to a callback query
func (c *Client) AnswerCallbackQuery(callbackID string, text string, showAlert bool) error {
	url := fmt.Sprintf("%s%s/answerCallbackQuery", apiBaseURL, c.botToken)

	payload := map[string]interface{}{
		"callback_query_id": callbackID,
	}

	if text != "" {
		payload["text"] = text
		payload["show_alert"] = showAlert
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error (status %d)", resp.StatusCode)
	}

	return nil
}

// EditMessageText edits the text of a message
func (c *Client) EditMessageText(chatID string, messageID int, text string, keyboard *InlineKeyboardMarkup) error {
	url := fmt.Sprintf("%s%s/editMessageText", apiBaseURL, c.botToken)

	payload := map[string]interface{}{
		"chat_id":                  chatID,
		"message_id":               messageID,
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

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API error (status %d)", resp.StatusCode)
	}

	return nil
}
