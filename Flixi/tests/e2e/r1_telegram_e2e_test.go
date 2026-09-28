package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// Telegram API Data Structures as per R1 specification
type TelegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
}

type TelegramChat struct {
	ID   int64  `json:"id"`
	Type string `json:"type"`
}

type TelegramMessage struct {
	MessageID int          `json:"message_id"`
	From      *TelegramUser `json:"from,omitempty"`
	Chat      TelegramChat `json:"chat"`
	Text      string       `json:"text"`
	Date      int64        `json:"date"`
}

type TelegramUpdate struct {
	UpdateID int              `json:"update_id"`
	Message  *TelegramMessage `json:"message,omitempty"`
}

type TelegramGetUpdatesResponse struct {
	OK     bool             `json:"ok"`
	Result []TelegramUpdate `json:"result"`
}

type TelegramSendMessagePayload struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type TelegramSendMessageResponse struct {
	OK     bool             `json:"ok"`
	Result TelegramMessage `json:"result"`
}

// TelegramBotEngine encapsulates R1 Telegram remote control logic for testing
type TelegramBotEngine struct {
	Token          string
	AllowedChatIDs map[int64]bool
	BaseURL        string
	HTTPClient     *http.Client
	IntentHandler  func(intent string) (string, error)
	mu             sync.Mutex
}

func NewTelegramBotEngine(token string, allowedIDs []int64, baseURL string, handler func(intent string) (string, error)) *TelegramBotEngine {
	whitelist := make(map[int64]bool)
	for _, id := range allowedIDs {
		whitelist[id] = true
	}
	return &TelegramBotEngine{
		Token:          token,
		AllowedChatIDs: whitelist,
		BaseURL:        baseURL,
		HTTPClient:     &http.Client{Timeout: 5 * time.Second},
		IntentHandler:  handler,
	}
}

// IsChatAuthorized checks if chat_id is in whitelist
func (b *TelegramBotEngine) IsChatAuthorized(chatID int64) bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.AllowedChatIDs[chatID]
}

// ProcessUpdate handles a single Telegram update
func (b *TelegramBotEngine) ProcessUpdate(update TelegramUpdate) (*TelegramSendMessagePayload, error) {
	if update.Message == nil || update.Message.Text == "" {
		return nil, fmt.Errorf("empty message or text")
	}

	chatID := update.Message.Chat.ID
	if !b.IsChatAuthorized(chatID) {
		return nil, fmt.Errorf("unauthorized chat ID: %d", chatID)
	}

	intentText := update.Message.Text
	var respText string
	var err error

	if b.IntentHandler != nil {
		respText, err = b.IntentHandler(intentText)
		if err != nil {
			respText = fmt.Sprintf("Error executing intent: %v", err)
		}
	} else {
		respText = fmt.Sprintf("Processed intent: %s", intentText)
	}

	payload := &TelegramSendMessagePayload{
		ChatID: chatID,
		Text:   respText,
	}

	// Send outbound reply to Telegram API
	err = b.SendTelegramMessage(payload)
	if err != nil {
		return payload, fmt.Errorf("failed to send message payload: %w", err)
	}

	return payload, nil
}

// SendTelegramMessage posts a sendMessage request to Telegram API
func (b *TelegramBotEngine) SendTelegramMessage(payload *TelegramSendMessagePayload) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/bot%s/sendMessage", b.BaseURL, b.Token)
	resp, err := b.HTTPClient.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("telegram API returned status: %d", resp.StatusCode)
	}

	return nil
}

// Polling simulation: fetches updates from mock server
func (b *TelegramBotEngine) FetchUpdates() ([]TelegramUpdate, error) {
	url := fmt.Sprintf("%s/bot%s/getUpdates", b.BaseURL, b.Token)
	resp, err := b.HTTPClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp TelegramGetUpdatesResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	return apiResp.Result, nil
}

// --- E2E TEST SUITES FOR R1 ---

func TestTelegramBot_GetUpdates(t *testing.T) {
	mockUpdates := []TelegramUpdate{
		{
			UpdateID: 1001,
			Message: &TelegramMessage{
				MessageID: 1,
				Chat:      TelegramChat{ID: 12345, Type: "private"},
				Text:      "open notepad",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/botTEST_TOKEN/getUpdates" {
			t.Errorf("Unexpected path: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TelegramGetUpdatesResponse{
			OK:     true,
			Result: mockUpdates,
		})
	}))
	defer server.Close()

	bot := NewTelegramBotEngine("TEST_TOKEN", []int64{12345}, server.URL, nil)
	updates, err := bot.FetchUpdates()
	if err != nil {
		t.Fatalf("FetchUpdates failed: %v", err)
	}

	if len(updates) != 1 {
		t.Fatalf("Expected 1 update, got %d", len(updates))
	}
	if updates[0].Message.Text != "open notepad" {
		t.Errorf("Expected message text 'open notepad', got '%s'", updates[0].Message.Text)
	}
}

func TestTelegramBot_AuthorizationWhitelist(t *testing.T) {
	sentMessages := make([]TelegramSendMessagePayload, 0)
	var sentMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/botTEST_TOKEN/sendMessage" {
			var payload TelegramSendMessagePayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
				sentMu.Lock()
				sentMessages = append(sentMessages, payload)
				sentMu.Unlock()
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TelegramSendMessageResponse{OK: true})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	authorizedChatID := int64(987654321)
	unauthorizedChatID := int64(111222333)

	bot := NewTelegramBotEngine("TEST_TOKEN", []int64{authorizedChatID}, server.URL, func(intent string) (string, error) {
		return "Handled: " + intent, nil
	})

	// Test 1: Authorized Chat
	authUpdate := TelegramUpdate{
		UpdateID: 2001,
		Message: &TelegramMessage{
			MessageID: 10,
			Chat:      TelegramChat{ID: authorizedChatID},
			Text:      "search cats",
		},
	}
	payload, err := bot.ProcessUpdate(authUpdate)
	if err != nil {
		t.Fatalf("ProcessUpdate failed for authorized chat: %v", err)
	}
	if payload.Text != "Handled: search cats" {
		t.Errorf("Expected response 'Handled: search cats', got '%s'", payload.Text)
	}

	// Test 2: Unauthorized Chat
	unauthUpdate := TelegramUpdate{
		UpdateID: 2002,
		Message: &TelegramMessage{
			MessageID: 11,
			Chat:      TelegramChat{ID: unauthorizedChatID},
			Text:      "delete files",
		},
	}
	_, err = bot.ProcessUpdate(unauthUpdate)
	if err == nil {
		t.Fatalf("Expected error for unauthorized chat ID %d, but got none", unauthorizedChatID)
	}

	sentMu.Lock()
	defer sentMu.Unlock()
	if len(sentMessages) != 1 {
		t.Errorf("Expected exactly 1 sent message (only from authorized chat), got %d", len(sentMessages))
	}
	if len(sentMessages) > 0 && sentMessages[0].ChatID != authorizedChatID {
		t.Errorf("Expected sent message ChatID %d, got %d", authorizedChatID, sentMessages[0].ChatID)
	}
}

func TestTelegramBot_IntentDispatchAndResponse(t *testing.T) {
	var capturedPayload TelegramSendMessagePayload
	var payloadMu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/botTOKEN123/sendMessage" {
			payloadMu.Lock()
			_ = json.NewDecoder(r.Body).Decode(&capturedPayload)
			payloadMu.Unlock()

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(TelegramSendMessageResponse{OK: true})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	var dispatchedIntent string
	var dispatchMu sync.Mutex

	mockDispatchIntent := func(intent string) (string, error) {
		dispatchMu.Lock()
		dispatchedIntent = intent
		dispatchMu.Unlock()
		return fmt.Sprintf("Success: Executed '%s'", intent), nil
	}

	bot := NewTelegramBotEngine("TOKEN123", []int64{55555}, server.URL, mockDispatchIntent)

	update := TelegramUpdate{
		UpdateID: 3001,
		Message: &TelegramMessage{
			MessageID: 100,
			Chat:      TelegramChat{ID: 55555},
			Text:      "open browser",
		},
	}

	_, err := bot.ProcessUpdate(update)
	if err != nil {
		t.Fatalf("Failed to process update: %v", err)
	}

	dispatchMu.Lock()
	if dispatchedIntent != "open browser" {
		t.Errorf("Expected dispatched intent 'open browser', got '%s'", dispatchedIntent)
	}
	dispatchMu.Unlock()

	payloadMu.Lock()
	if capturedPayload.ChatID != 55555 {
		t.Errorf("Expected outbound payload ChatID 55555, got %d", capturedPayload.ChatID)
	}
	if capturedPayload.Text != "Success: Executed 'open browser'" {
		t.Errorf("Expected payload text 'Success: Executed \\'open browser\\'', got '%s'", capturedPayload.Text)
	}
	payloadMu.Unlock()
}

func TestTelegramBot_MalformedUpdateHandling(t *testing.T) {
	bot := NewTelegramBotEngine("TOKEN123", []int64{100}, "http://localhost", nil)

	// Case 1: Empty message
	emptyUpdate := TelegramUpdate{UpdateID: 4001, Message: nil}
	_, err := bot.ProcessUpdate(emptyUpdate)
	if err == nil {
		t.Errorf("Expected error for empty message, got nil")
	}

	// Case 2: Message with empty text
	noTextUpdate := TelegramUpdate{
		UpdateID: 4002,
		Message: &TelegramMessage{
			MessageID: 2,
			Chat:      TelegramChat{ID: 100},
			Text:      "",
		},
	}
	_, err = bot.ProcessUpdate(noTextUpdate)
	if err == nil {
		t.Errorf("Expected error for empty message text, got nil")
	}
}

func TestTelegramBot_ConcurrentUpdates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(TelegramSendMessageResponse{OK: true})
	}))
	defer server.Close()

	var counter int
	var counterMu sync.Mutex

	mockDispatch := func(intent string) (string, error) {
		counterMu.Lock()
		counter++
		counterMu.Unlock()
		return "OK", nil
	}

	bot := NewTelegramBotEngine("TOKEN123", []int64{777}, server.URL, mockDispatch)

	var wg sync.WaitGroup
	numRoutines := 10
	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			update := TelegramUpdate{
				UpdateID: 5000 + idx,
				Message: &TelegramMessage{
					MessageID: idx,
					Chat:      TelegramChat{ID: 777},
					Text:      fmt.Sprintf("concurrent test %d", idx),
				},
			}
			_, _ = bot.ProcessUpdate(update)
		}(i)
	}

	wg.Wait()

	counterMu.Lock()
	defer counterMu.Unlock()
	if counter != numRoutines {
		t.Errorf("Expected counter %d, got %d", numRoutines, counter)
	}
}
