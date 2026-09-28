package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestProcessTelegramUpdate(t *testing.T) {
	allowedChatID := int64(12345678)
	unauthorizedChatID := int64(99999999)

	t.Run("Authorized chat ID with valid skill intent", func(t *testing.T) {
		handlerCalled := false
		handler := func(intent string) (string, error) {
			handlerCalled = true
			if intent != "open notepad" {
				return "", fmt.Errorf("unexpected intent: %s", intent)
			}
			return "Opened notepad", nil
		}

		mockJSON := fmt.Sprintf(`{
			"update_id": 1001,
			"message": {
				"message_id": 50,
				"chat": {
					"id": %d,
					"type": "private"
				},
				"text": "open notepad",
				"date": 1600000000
			}
		}`, allowedChatID)

		resp, processed, err := ProcessTelegramUpdateJSON([]byte(mockJSON), allowedChatID, handler)
		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}
		if !processed {
			t.Fatalf("Expected message to be processed")
		}
		if !handlerCalled {
			t.Fatalf("Expected intent handler to be called")
		}
		if resp != "Opened notepad" {
			t.Fatalf("Expected response 'Opened notepad', got: '%s'", resp)
		}
	})

	t.Run("Unauthorized chat ID is rejected", func(t *testing.T) {
		handlerCalled := false
		handler := func(intent string) (string, error) {
			handlerCalled = true
			return "OK", nil
		}

		mockJSON := fmt.Sprintf(`{
			"update_id": 1002,
			"message": {
				"message_id": 51,
				"chat": {
					"id": %d,
					"type": "private"
				},
				"text": "open notepad",
				"date": 1600000000
			}
		}`, unauthorizedChatID)

		resp, processed, err := ProcessTelegramUpdateJSON([]byte(mockJSON), allowedChatID, handler)
		if err == nil {
			t.Fatalf("Expected authorization error for unauthorized chat ID")
		}
		if processed {
			t.Fatalf("Expected message to NOT be processed for unauthorized chat ID")
		}
		if handlerCalled {
			t.Fatalf("Expected intent handler NOT to be called for unauthorized chat ID")
		}
		if resp != "Unauthorized chat ID" {
			t.Fatalf("Expected 'Unauthorized chat ID' response, got: '%s'", resp)
		}
	})

	t.Run("Nil or empty message payload handled safely", func(t *testing.T) {
		handlerCalled := false
		handler := func(intent string) (string, error) {
			handlerCalled = true
			return "OK", nil
		}

		mockJSON := `{
			"update_id": 1003,
			"message": null
		}`

		_, processed, err := ProcessTelegramUpdateJSON([]byte(mockJSON), allowedChatID, handler)
		if err != nil {
			t.Fatalf("Unexpected error for nil message: %v", err)
		}
		if processed {
			t.Fatalf("Expected processed=false for nil message")
		}
		if handlerCalled {
			t.Fatalf("Expected handler NOT called for nil message")
		}
	})

	t.Run("Handler error formatting", func(t *testing.T) {
		handler := func(intent string) (string, error) {
			return "", fmt.Errorf("unknown intent")
		}

		update := TelegramUpdate{
			UpdateID: 1004,
			Message: &TelegramMessage{
				MessageID: 52,
				Chat:      TelegramChat{ID: allowedChatID},
				Text:      "invalid intent",
			},
		}

		resp, processed, err := ProcessTelegramUpdate(update, allowedChatID, handler)
		if err == nil {
			t.Fatalf("Expected error from handler, got nil")
		}
		if !processed {
			t.Fatalf("Expected processed=true even when handler errors")
		}
		if resp != "Error executing intent: unknown intent" {
			t.Fatalf("Unexpected response text: %s", resp)
		}
	})
}

func TestStartListenerWithMockServer(t *testing.T) {
	allowedChatID := int64(777)
	botToken := "TEST_BOT_TOKEN"

	var updateCount int32
	var sendMessageReceived int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bot"+botToken+"/getUpdates" {
			count := atomic.AddInt32(&updateCount, 1)
			w.Header().Set("Content-Type", "application/json")
			if count == 1 {
				// Send one update on first call
				resp := GetUpdatesResponse{
					Ok: true,
					Result: []TelegramUpdate{
						{
							UpdateID: 1,
							Message: &TelegramMessage{
								MessageID: 10,
								Chat:      TelegramChat{ID: allowedChatID},
								Text:      "system monitor",
							},
						},
					},
				}
				json.NewEncoder(w).Encode(resp)
				return
			}
			// Subsequent calls return empty updates
			resp := GetUpdatesResponse{Ok: true, Result: []TelegramUpdate{}}
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/bot"+botToken+"/sendMessage" {
			atomic.AddInt32(&sendMessageReceived, 1)
			var req SendMessageRequest
			json.NewDecoder(r.Body).Decode(&req)
			if req.ChatID != allowedChatID {
				t.Errorf("SendMessage received wrong chat ID: %d", req.ChatID)
			}
			if req.Text != "Status OK" {
				t.Errorf("SendMessage received wrong text: %s", req.Text)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"ok": true}`))
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	// Override BaseURL to point to mock server
	oldBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = oldBaseURL }()

	ctx, cancel := context.WithCancel(context.Background())

	handler := func(intent string) (string, error) {
		if intent == "system monitor" {
			return "Status OK", nil
		}
		return "", fmt.Errorf("unknown")
	}

	go StartListener(ctx, botToken, allowedChatID, handler)

	// Wait for updates to process
	time.Sleep(300 * time.Millisecond)
	cancel() // graceful shutdown

	if atomic.LoadInt32(&sendMessageReceived) == 0 {
		t.Fatalf("Expected SendMessage to be invoked by StartListener")
	}
}
