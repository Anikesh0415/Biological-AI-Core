package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestStress_ProcessTelegramUpdate_MalformedJSON tests parser robustness against invalid/corrupt JSON.
func TestStress_ProcessTelegramUpdate_MalformedJSON(t *testing.T) {
	allowedChatID := int64(12345)
	dummyHandler := func(intent string) (string, error) {
		return "ok", nil
	}

	malformedInputs := []string{
		"",
		"{",
		"not json",
		`{"update_id": "not_an_int"}`,
		`{"message": "should_be_an_object"}`,
		`{"message": {"chat": "should_be_object"}}`,
		`{"message": {"chat": {"id": "invalid"}}}`,
		`[1, 2, 3]`,
		`{"update_id": 100, "message": null}`,
		`{"update_id": 101, "message": {"text": 12345}}`,
	}

	for i, input := range malformedInputs {
		t.Run(fmt.Sprintf("Malformed_%d", i), func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					t.Fatalf("Panic detected on malformed input #%d (%s): %v", i, input, r)
				}
			}()

			_, _, err := ProcessTelegramUpdateJSON([]byte(input), allowedChatID, dummyHandler)
			// Malformed input should either return an error or be safely unmarshaled/handled without panic.
			_ = err
		})
	}
}

// TestStress_ProcessTelegramUpdate_UnauthorizedChatIDs tests whitelist behavior across various Chat IDs.
func TestStress_ProcessTelegramUpdate_UnauthorizedChatIDs(t *testing.T) {
	dummyHandler := func(intent string) (string, error) {
		return "ok", nil
	}

	t.Run("Positive allowedChatID filters wrong IDs", func(t *testing.T) {
		allowedChatID := int64(987654)
		testIDs := []int64{0, 1, 987653, 987655, -987654, 9223372036854775807, -9223372036854775808}

		for _, badID := range testIDs {
			update := TelegramUpdate{
				UpdateID: 1,
				Message: &TelegramMessage{
					MessageID: 1,
					Chat:      TelegramChat{ID: badID},
					Text:      "test intent",
				},
			}
			resp, processed, err := ProcessTelegramUpdate(update, allowedChatID, dummyHandler)
			if err == nil {
				t.Errorf("Expected unauthorized error for chat ID %d when allowed is %d", badID, allowedChatID)
			}
			if processed {
				t.Errorf("Expected processed=false for unauthorized chat ID %d", badID)
			}
			if resp != "Unauthorized chat ID" {
				t.Errorf("Expected 'Unauthorized chat ID' response for bad ID %d, got '%s'", badID, resp)
			}
		}
	})

	t.Run("Negative allowedChatID (Group Chat)", func(t *testing.T) {
		allowedGroupChatID := int64(-1001987654321)
		updateAuthorized := TelegramUpdate{
			UpdateID: 1,
			Message: &TelegramMessage{
				MessageID: 1,
				Chat:      TelegramChat{ID: allowedGroupChatID},
				Text:      "group intent",
			},
		}
		_, processed, err := ProcessTelegramUpdate(updateAuthorized, allowedGroupChatID, dummyHandler)
		if err != nil || !processed {
			t.Fatalf("Failed to authorize valid negative group chat ID: err=%v, processed=%v", err, processed)
		}

		updateUnauthorized := TelegramUpdate{
			UpdateID: 2,
			Message: &TelegramMessage{
				MessageID: 2,
				Chat:      TelegramChat{ID: 12345},
				Text:      "group intent",
			},
		}
		_, processed, err = ProcessTelegramUpdate(updateUnauthorized, allowedGroupChatID, dummyHandler)
		if err == nil || processed {
			t.Fatalf("Unauthorized chat passed negative group chat whitelist")
		}
	})

	t.Run("allowedChatID == 0 behavior", func(t *testing.T) {
		// When allowedChatID is 0, does it allow any chat ID?
		update := TelegramUpdate{
			UpdateID: 1,
			Message: &TelegramMessage{
				MessageID: 1,
				Chat:      TelegramChat{ID: 55555},
				Text:      "open notepad",
			},
		}
		resp, processed, err := ProcessTelegramUpdate(update, 0, dummyHandler)
		if err != nil || !processed || resp != "ok" {
			t.Logf("Notice: allowedChatID=0 behavior -> resp='%s', processed=%v, err=%v", resp, processed, err)
		}
	})
}

// TestStress_ProcessTelegramUpdate_ConcurrentCalls tests concurrent calls for race conditions.
func TestStress_ProcessTelegramUpdate_ConcurrentCalls(t *testing.T) {
	allowedChatID := int64(100)
	var processedCount int64
	var unauthorizedCount int64

	handler := func(intent string) (string, error) {
		time.Sleep(1 * time.Millisecond) // simulate brief work
		return "processed: " + intent, nil
	}

	const numGoroutines = 50
	const requestsPerGoroutine = 20
	var wg sync.WaitGroup

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(gID int) {
			defer wg.Done()
			for r := 0; r < requestsPerGoroutine; r++ {
				chatID := allowedChatID
				if (gID+r)%2 == 1 {
					chatID = 999 // unauthorized
				}

				update := TelegramUpdate{
					UpdateID: int64(gID*1000 + r),
					Message: &TelegramMessage{
						MessageID: int64(r),
						Chat:      TelegramChat{ID: chatID},
						Text:      fmt.Sprintf("intent-%d-%d", gID, r),
					},
				}

				_, processed, err := ProcessTelegramUpdate(update, allowedChatID, handler)
				if processed && err == nil {
					atomic.AddInt64(&processedCount, 1)
				} else {
					atomic.AddInt64(&unauthorizedCount, 1)
				}
			}
		}(i)
	}

	wg.Wait()

	expectedTotal := int64(numGoroutines * requestsPerGoroutine)
	actualTotal := processedCount + unauthorizedCount
	if actualTotal != expectedTotal {
		t.Fatalf("Expected total requests %d, got %d (processed: %d, unauthorized: %d)",
			expectedTotal, actualTotal, processedCount, unauthorizedCount)
	}
}

// TestStress_StartListener_RapidUpdates verifies long-polling under high volume of updates.
func TestStress_StartListener_RapidUpdates(t *testing.T) {
	allowedChatID := int64(42)
	botToken := "RAPID_TEST_TOKEN"

	const totalBatchUpdates = 200
	var updateCounter int32
	var sendMessageCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/bot"+botToken+"/getUpdates" {
			current := atomic.AddInt32(&updateCounter, 1)
			w.Header().Set("Content-Type", "application/json")
			if current == 1 {
				// Return 200 updates in first batch
				results := make([]TelegramUpdate, totalBatchUpdates)
				for i := 0; i < totalBatchUpdates; i++ {
					results[i] = TelegramUpdate{
						UpdateID: int64(i + 1),
						Message: &TelegramMessage{
							MessageID: int64(100 + i),
							Chat:      TelegramChat{ID: allowedChatID},
							Text:      fmt.Sprintf("intent #%d", i+1),
						},
					}
				}
				resp := GetUpdatesResponse{Ok: true, Result: results}
				_ = json.NewEncoder(w).Encode(resp)
				return
			}
			// Subsequent calls return empty list
			resp := GetUpdatesResponse{Ok: true, Result: []TelegramUpdate{}}
			_ = json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/bot"+botToken+"/sendMessage" {
			atomic.AddInt32(&sendMessageCount, 1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok": true}`))
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	oldBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = oldBaseURL }()

	ctx, cancel := context.WithCancel(context.Background())

	handler := func(intent string) (string, error) {
		return "done: " + intent, nil
	}

	go StartListener(ctx, botToken, allowedChatID, handler)

	// Allow time to process rapid updates
	time.Sleep(500 * time.Millisecond)
	cancel()

	sent := atomic.LoadInt32(&sendMessageCount)
	if sent != totalBatchUpdates {
		t.Fatalf("Expected %d SendMessage calls for rapid updates, got %d", totalBatchUpdates, sent)
	}
}

// TestStress_StartListener_ServerErrors tests listener resiliency against HTTP 500 / corrupt server responses.
func TestStress_StartListener_ServerErrors(t *testing.T) {
	allowedChatID := int64(42)
	botToken := "ERROR_TEST_TOKEN"
	var callCount int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := atomic.AddInt32(&callCount, 1)
		w.Header().Set("Content-Type", "application/json")
		switch count % 3 {
		case 1:
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(`{"ok": false, "description": "Internal server error"}`))
		case 2:
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`INVALID_JSON_PAYLOAD`))
		default:
			w.WriteHeader(http.StatusOK)
			resp := GetUpdatesResponse{Ok: true, Result: []TelegramUpdate{}}
			_ = json.NewEncoder(w).Encode(resp)
		}
	}))
	defer server.Close()

	oldBaseURL := BaseURL
	BaseURL = server.URL
	defer func() { BaseURL = oldBaseURL }()

	ctx, cancel := context.WithTimeout(context.Background(), 2500*time.Millisecond)
	defer cancel()

	handler := func(intent string) (string, error) {
		return "ok", nil
	}

	// Listener should not crash or panic on 500s or invalid JSON
	doneChan := make(chan struct{})
	go func() {
		StartListener(ctx, botToken, allowedChatID, handler)
		close(doneChan)
	}()

	select {
	case <-doneChan:
		// Successfully exited on context timeout without hanging or panicking
	case <-time.After(4 * time.Second):
		t.Fatalf("StartListener failed to exit after context cancellation")
	}

	if atomic.LoadInt32(&callCount) == 0 {
		t.Fatalf("Expected listener to make HTTP requests to mock server")
	}
}

// TestStress_SendMessage_Errors tests SendMessage error handling paths.
func TestStress_SendMessage_Errors(t *testing.T) {
	ctx := context.Background()

	t.Run("Empty botToken", func(t *testing.T) {
		err := SendMessage(ctx, nil, "", 123, "test")
		if err == nil {
			t.Fatalf("Expected error for empty botToken")
		}
	})

	t.Run("Canceled context", func(t *testing.T) {
		cancCtx, cancel := context.WithCancel(context.Background())
		cancel()
		err := SendMessage(cancCtx, nil, "token", 123, "test")
		if err == nil {
			t.Fatalf("Expected error for canceled context")
		}
	})

	t.Run("HTTP Non-200 Response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"ok": false, "description": "Forbidden"}`))
		}))
		defer server.Close()

		oldBaseURL := BaseURL
		BaseURL = server.URL
		defer func() { BaseURL = oldBaseURL }()

		err := SendMessage(ctx, nil, "token", 123, "test")
		if err == nil {
			t.Fatalf("Expected error on HTTP 403 response")
		}
	})
}
