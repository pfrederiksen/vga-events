package telegram

import (
	"testing"
)

func TestNewClient_Validation(t *testing.T) {
	tests := []struct {
		name      string
		botToken  string
		chatID    string
		wantError bool
	}{
		{
			name:      "valid parameters",
			botToken:  "test-token",
			chatID:    "12345",
			wantError: false,
		},
		{
			name:      "empty bot token",
			botToken:  "",
			chatID:    "12345",
			wantError: true,
		},
		{
			name:      "empty chat ID",
			botToken:  "test-token",
			chatID:    "",
			wantError: true,
		},
		{
			name:      "both empty",
			botToken:  "",
			chatID:    "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.botToken, tt.chatID)
			if tt.wantError {
				if err == nil {
					t.Error("NewClient() expected error, got nil")
				}
				if client != nil {
					t.Error("NewClient() should return nil client on error")
				}
			} else {
				if err != nil {
					t.Errorf("NewClient() unexpected error: %v", err)
				}
				if client == nil {
					t.Error("NewClient() returned nil client")
				}
				return
				if client.botToken != tt.botToken {
					t.Errorf("botToken = %q, want %q", client.botToken, tt.botToken)
				}
				if client.chatID != tt.chatID {
					t.Errorf("chatID = %q, want %q", client.chatID, tt.chatID)
				}
				if client.httpClient == nil {
					t.Error("httpClient should not be nil")
				}
			}
		})
	}
}

func TestSendMessage_Validation(t *testing.T) {
	client := &Client{
		botToken: "test-token",
		chatID:   "12345",
	}

	// Test empty message
	err := client.SendMessage("")
	if err == nil {
		t.Error("SendMessage() expected error for empty message, got nil")
	}
	if err != nil && err.Error() != "message text is required" {
		t.Errorf("SendMessage() error = %v, want 'message text is required'", err)
	}
}

func TestSendMessageWithKeyboard_Validation(t *testing.T) {
	client := &Client{
		botToken: "test-token",
		chatID:   "12345",
	}

	// Test empty message
	err := client.SendMessageWithKeyboard("", nil)
	if err == nil {
		t.Error("SendMessageWithKeyboard() expected error for empty message, got nil")
	}
	if err != nil && err.Error() != "message text is required" {
		t.Errorf("SendMessageWithKeyboard() error = %v, want 'message text is required'", err)
	}
}

func TestInlineKeyboardButton(t *testing.T) {
	button := InlineKeyboardButton{
		Text:         "Test Button",
		CallbackData: "callback_data",
		URL:          "https://example.com",
	}

	if button.Text != "Test Button" {
		t.Errorf("Text = %q, want 'Test Button'", button.Text)
	}
	if button.CallbackData != "callback_data" {
		t.Errorf("CallbackData = %q, want 'callback_data'", button.CallbackData)
	}
	if button.URL != "https://example.com" {
		t.Errorf("URL = %q, want 'https://example.com'", button.URL)
	}
}

func TestInlineKeyboardMarkup(t *testing.T) {
	keyboard := &InlineKeyboardMarkup{
		InlineKeyboard: [][]InlineKeyboardButton{
			{
				{Text: "Button 1", CallbackData: "cb1"},
				{Text: "Button 2", CallbackData: "cb2"},
			},
			{
				{Text: "Button 3", URL: "https://example.com"},
			},
		},
	}

	if len(keyboard.InlineKeyboard) != 2 {
		t.Errorf("InlineKeyboard rows = %d, want 2", len(keyboard.InlineKeyboard))
	}

	if len(keyboard.InlineKeyboard[0]) != 2 {
		t.Errorf("First row buttons = %d, want 2", len(keyboard.InlineKeyboard[0]))
	}

	if len(keyboard.InlineKeyboard[1]) != 1 {
		t.Errorf("Second row buttons = %d, want 1", len(keyboard.InlineKeyboard[1]))
	}
}

func TestUser(t *testing.T) {
	user := User{
		ID:        12345,
		FirstName: "John",
		LastName:  "Doe",
		Username:  "johndoe",
	}

	if user.ID != 12345 {
		t.Errorf("ID = %d, want 12345", user.ID)
	}
	if user.FirstName != "John" {
		t.Errorf("FirstName = %q, want 'John'", user.FirstName)
	}
	if user.LastName != "Doe" {
		t.Errorf("LastName = %q, want 'Doe'", user.LastName)
	}
	if user.Username != "johndoe" {
		t.Errorf("Username = %q, want 'johndoe'", user.Username)
	}
}

func TestMessage(t *testing.T) {
	msg := Message{
		MessageID: 123,
		From: User{
			ID:        456,
			FirstName: "Test",
		},
		Chat: Chat{
			ID:   789,
			Type: "private",
		},
		Text: "Hello World",
	}

	if msg.MessageID != 123 {
		t.Errorf("MessageID = %d, want 123", msg.MessageID)
	}
	if msg.From.ID != 456 {
		t.Errorf("From.ID = %d, want 456", msg.From.ID)
	}
	if msg.Chat.ID != 789 {
		t.Errorf("Chat.ID = %d, want 789", msg.Chat.ID)
	}
	if msg.Text != "Hello World" {
		t.Errorf("Text = %q, want 'Hello World'", msg.Text)
	}
}

func TestCallbackQuery(t *testing.T) {
	cbq := CallbackQuery{
		ID: "callback123",
		From: User{
			ID:        456,
			FirstName: "Test",
		},
		Data: "button_clicked",
	}

	if cbq.ID != "callback123" {
		t.Errorf("ID = %q, want 'callback123'", cbq.ID)
	}
	if cbq.Data != "button_clicked" {
		t.Errorf("Data = %q, want 'button_clicked'", cbq.Data)
	}
}
