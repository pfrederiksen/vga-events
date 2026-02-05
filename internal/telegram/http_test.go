package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestSendMessage_Success tests successful message sending
func TestSendMessage_Success(t *testing.T) {
	// Create a test server that mimics Telegram API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and content type
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type application/json, got %s", r.Header.Get("Content-Type"))
		}

		// Return successful response
		response := map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": 123,
				"chat": map[string]interface{}{
					"id": 789,
				},
				"text": "Test message",
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	// Override the API base URL for testing
	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.SendMessage("Test message")
	if err != nil {
		t.Errorf("SendMessage() unexpected error: %v", err)
	}
}

// TestSendMessage_APIError tests API error handling
func TestSendMessage_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return error response
		response := map[string]interface{}{
			"ok":          false,
			"description": "Bad Request: chat not found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.SendMessage("Test message")
	if err == nil {
		t.Error("SendMessage() expected error for API failure, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "Bad Request") {
		t.Errorf("SendMessage() error = %v, want error containing 'Bad Request'", err)
	}
}

// TestSendMessage_HTTPError tests HTTP error handling
func TestSendMessage_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return HTTP error
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.SendMessage("Test message")
	if err == nil {
		t.Error("SendMessage() expected error for HTTP error, got nil")
	}
	if err != nil && !strings.Contains(err.Error(), "status 500") {
		t.Errorf("SendMessage() error = %v, want error containing 'status 500'", err)
	}
}

// TestSendMessageWithKeyboard_Success tests successful message with keyboard
func TestSendMessageWithKeyboard_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": 123,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "Button 1", CallbackData: "cb1"},
				{Text: "Button 2", CallbackData: "cb2"},
			},
		},
	}

	err := client.SendMessageWithKeyboard("Test message", keyboard)
	if err != nil {
		t.Errorf("SendMessageWithKeyboard() unexpected error: %v", err)
	}
}

// TestSendMessageWithKeyboard_APIError tests keyboard message API error
func TestSendMessageWithKeyboard_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok":          false,
			"description": "Bad Request: invalid keyboard",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "Button", CallbackData: "cb"}},
		},
	}

	err := client.SendMessageWithKeyboard("Test", keyboard)
	if err == nil {
		t.Error("SendMessageWithKeyboard() expected error, got nil")
	}
}

// TestAnswerCallbackQuery_Success tests successful callback query answer
func TestAnswerCallbackQuery_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok":     true,
			"result": true,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.AnswerCallbackQuery("callback123", "Success!", false)
	if err != nil {
		t.Errorf("AnswerCallbackQuery() unexpected error: %v", err)
	}
}

// TestAnswerCallbackQuery_WithEmptyID tests behavior with empty callback ID
// Note: The function doesn't validate parameters, so this will make an HTTP call
func TestAnswerCallbackQuery_WithEmptyID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return error for empty callback_query_id
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"ok":          false,
			"description": "Bad Request: query is empty",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.AnswerCallbackQuery("", "text", false)
	if err == nil {
		t.Error("AnswerCallbackQuery() expected error for empty callback_query_id, got nil")
	}
}

// TestAnswerCallbackQuery_APIError tests API error handling
func TestAnswerCallbackQuery_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return HTTP error status
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"ok":          false,
			"description": "Query is too old",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.AnswerCallbackQuery("old-callback", "text", false)
	if err == nil {
		t.Error("AnswerCallbackQuery() expected error, got nil")
	}
}

// TestEditMessageText_Success tests successful message edit
func TestEditMessageText_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": 123,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.EditMessageText("12345", 123, "Updated text", nil)
	if err != nil {
		t.Errorf("EditMessageText() unexpected error: %v", err)
	}
}

// TestEditMessageText_EmptyChatID tests validation
func TestEditMessageText_EmptyChatID(t *testing.T) {
	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.EditMessageText("", 123, "text", nil)
	if err == nil {
		t.Error("EditMessageText() expected error for empty chat_id, got nil")
	}
}

// TestEditMessageText_ZeroMessageID tests validation
func TestEditMessageText_ZeroMessageID(t *testing.T) {
	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.EditMessageText("12345", 0, "text", nil)
	if err == nil {
		t.Error("EditMessageText() expected error for message_id = 0, got nil")
	}
}

// TestEditMessageText_EmptyText tests validation
func TestEditMessageText_EmptyText(t *testing.T) {
	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.EditMessageText("12345", 123, "", nil)
	if err == nil {
		t.Error("EditMessageText() expected error for empty text, got nil")
	}
}

// TestEditMessageText_WithKeyboard tests editing with keyboard
func TestEditMessageText_WithKeyboard(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": 123,
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{{Text: "Updated", CallbackData: "updated"}},
		},
	}

	err := client.EditMessageText("12345", 123, "Updated", keyboard)
	if err != nil {
		t.Errorf("EditMessageText() with keyboard unexpected error: %v", err)
	}
}

// TestEditMessageText_APIError tests API error handling
func TestEditMessageText_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return HTTP error status
		w.WriteHeader(http.StatusBadRequest)
		response := map[string]interface{}{
			"ok":          false,
			"description": "Message not found",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.EditMessageText("12345", 999, "text", nil)
	if err == nil {
		t.Error("EditMessageText() expected error, got nil")
	}
}

// TestSendDocument_Success tests successful document sending
func TestSendDocument_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify multipart form data
		if !strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data") {
			t.Errorf("Expected multipart/form-data, got %s", r.Header.Get("Content-Type"))
		}

		response := map[string]interface{}{
			"ok": true,
			"result": map[string]interface{}{
				"message_id": 123,
				"document": map[string]interface{}{
					"file_id": "doc123",
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	data := []byte("test file content")
	err := client.SendDocument("test.txt", data, "Test document")
	if err != nil {
		t.Errorf("SendDocument() unexpected error: %v", err)
	}
}

// TestSendDocument_EmptyData tests validation
func TestSendDocument_EmptyData(t *testing.T) {
	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	err := client.SendDocument("test.txt", nil, "caption")
	if err == nil {
		t.Error("SendDocument() expected error for empty data, got nil")
	}
}

// TestSendDocument_EmptyFilename tests validation
func TestSendDocument_EmptyFilename(t *testing.T) {
	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	data := []byte("content")
	err := client.SendDocument("", data, "caption")
	if err == nil {
		t.Error("SendDocument() expected error for empty filename, got nil")
	}
}

// TestSendDocument_APIError tests API error handling
func TestSendDocument_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok":          false,
			"description": "File too large",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	originalURL := apiBaseURL
	apiBaseURL = server.URL + "/"
	defer func() { apiBaseURL = originalURL }()

	client := &Client{
		botToken:   "test-token",
		chatID:     "12345",
		httpClient: &http.Client{},
	}

	data := []byte("content")
	err := client.SendDocument("test.txt", data, "caption")
	if err == nil {
		t.Error("SendDocument() expected error, got nil")
	}
}
