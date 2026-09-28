package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// BaseURL allows overriding the Telegram API endpoint URL for testing.
var BaseURL = "https://api.telegram.org"

type TelegramUpdate struct {
	UpdateID int64            `json:"update_id"`
	Message  *TelegramMessage `json:"message"`
}

type TelegramMessage struct {
	MessageID int64         `json:"message_id"`
	From      *TelegramUser `json:"from"`
	Chat      TelegramChat  `json:"chat"`
	Text      string        `json:"text"`
	Date      int64         `json:"date"`
}

type TelegramUser struct {
	ID        int64  `json:"id"`
	IsBot     bool   `json:"is_bot"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
}

type TelegramChat struct {
	ID        int64  `json:"id"`
	Type      string `json:"type"`
	Title     string `json:"title"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type GetUpdatesResponse struct {
	Ok          bool             `json:"ok"`
	Result      []TelegramUpdate `json:"result"`
	Description string           `json:"description"`
}

type SendMessageRequest struct {
	ChatID int64  `json:"chat_id"`
	Text   string `json:"text"`
}

type SendMessageResponse struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
}

// ProcessTelegramUpdate parses and validates a Telegram update against allowedChatID
// and forwards authorized messages to intentHandler.
func ProcessTelegramUpdate(update TelegramUpdate, allowedChatID int64, intentHandler func(string) (string, error)) (string, bool, error) {
	if update.Message == nil {
		return "", false, nil
	}

	if allowedChatID != 0 && update.Message.Chat.ID != allowedChatID {
		return "Unauthorized chat ID", false, fmt.Errorf("unauthorized chat ID: %d", update.Message.Chat.ID)
	}

	text := strings.TrimSpace(update.Message.Text)
	if text == "" {
		return "Empty intent text", true, nil
	}

	if intentHandler == nil {
		return "", true, fmt.Errorf("no intent handler configured")
	}

	res, err := intentHandler(text)
	if err != nil {
		return fmt.Sprintf("Error executing intent: %v", err), true, err
	}

	return res, true, nil
}

// ProcessTelegramUpdateJSON parses raw JSON update data and processes it via ProcessTelegramUpdate.
func ProcessTelegramUpdateJSON(rawJSON []byte, allowedChatID int64, intentHandler func(string) (string, error)) (string, bool, error) {
	var update TelegramUpdate
	if err := json.Unmarshal(rawJSON, &update); err != nil {
		return "", false, fmt.Errorf("failed to unmarshal update JSON: %w", err)
	}
	return ProcessTelegramUpdate(update, allowedChatID, intentHandler)
}

// SendMessage sends a text response to a Telegram chat ID using the bot API.
func SendMessage(ctx context.Context, client *http.Client, botToken string, chatID int64, text string) error {
	if botToken == "" {
		return fmt.Errorf("empty botToken")
	}
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	apiURL := fmt.Sprintf("%s/bot%s/sendMessage", BaseURL, botToken)
	reqBody := SendMessageRequest{
		ChatID: chatID,
		Text:   text,
	}
	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal sendMessage request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("sendMessage HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sendMessage returned HTTP status %d", resp.StatusCode)
	}
	return nil
}

// StartListener runs a background polling loop to fetch Telegram updates and process them.
func StartListener(ctx context.Context, botToken string, allowedChatID int64, intentHandler func(string) (string, error)) {
	if botToken == "" {
		return
	}

	client := &http.Client{
		Timeout: 45 * time.Second,
	}

	var offset int64 = 0

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		url := fmt.Sprintf("%s/bot%s/getUpdates?offset=%d&timeout=30", BaseURL, botToken, offset)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}

		resp, err := client.Do(req)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}

		var updateResp GetUpdatesResponse
		err = json.NewDecoder(resp.Body).Decode(&updateResp)
		resp.Body.Close()

		if err != nil || !updateResp.Ok {
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}

		for _, update := range updateResp.Result {
			if update.UpdateID >= offset {
				offset = update.UpdateID + 1
			}

			respText, processed, _ := ProcessTelegramUpdate(update, allowedChatID, intentHandler)
			if processed && update.Message != nil && update.Message.Chat.ID != 0 {
				_ = SendMessage(ctx, client, botToken, update.Message.Chat.ID, respText)
			}
		}
	}
}
