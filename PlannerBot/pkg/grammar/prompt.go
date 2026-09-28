package grammar

import (
	"errors"
	"fmt"
	"strings"
)

// Sentinel errors for prompt construction.
var (
	// ErrEmptyQuery is returned when the input query is empty or whitespace-only.
	ErrEmptyQuery = errors.New("grammar: query cannot be empty")

	// ErrQueryTooLong is returned when the input query exceeds MaxQueryLength.
	ErrQueryTooLong = errors.New("grammar: query exceeds maximum length")
)

// Constants for prompt formatting and validation constraints.
const (
	// MaxQueryLength defines the upper byte limit for user reminder queries.
	MaxQueryLength = 1000

	// Special ChatML markers.
	imStart = "<|im_start|>"
	imEnd   = "<|im_end|>"
)

// SystemPrompt defines the authoritative extraction instructions and few-shot exemplars.
const SystemPrompt = `You are a precise reminder extraction assistant.
Extract the reminder task and the time from the user input into a JSON object with keys "task" and "time".
Follow these rules:
1. "task": The actionable reminder task with trigger phrases (such as "remind me to", "don't forget to", "please", "set an alarm for") removed.
2. "time": The exact temporal expression (date, clock time, duration, or relative time). If no time is specified, set "time" to "".
3. If no actionable task is specified, set "task" to "".
4. Output ONLY the JSON object. Do not include markdown blocks, notes, or additional text.

Examples:
User: remind me to call mom at 7pm
Assistant: {"task": "call mom", "time": "7pm"}

User: remind me tomorrow at 5pm to buy milk
Assistant: {"task": "buy milk", "time": "tomorrow at 5pm"}

User: at 8pm tonight, remind me to take medicine
Assistant: {"task": "take medicine", "time": "8pm tonight"}

User: water the garden plants
Assistant: {"task": "water the garden plants", "time": ""}`

// BuildPrompt constructs a ChatML prompt formatted for small instruction-tuned models like Qwen2.5-0.5B-Instruct.
// It complies with the PROJECT.md interface contract: func BuildPrompt(userQuery string) string.
// If the input is invalid (empty, whitespace-only, or oversized), it returns an empty string.
// For callers requiring explicit errors, use ValidateAndBuildPrompt.
func BuildPrompt(userQuery string) string {
	prompt, _ := ValidateAndBuildPrompt(userQuery)
	return prompt
}

// ValidateAndBuildPrompt sanitizes the query, validates constraints, and constructs the ChatML prompt.
// It returns ErrEmptyQuery if userQuery is empty or whitespace-only, and ErrQueryTooLong if it exceeds MaxQueryLength.
func ValidateAndBuildPrompt(userQuery string) (string, error) {
	sanitized := SanitizeQuery(userQuery)
	if len(sanitized) == 0 {
		return "", ErrEmptyQuery
	}
	if len(sanitized) > MaxQueryLength {
		return "", fmt.Errorf("%w: length %d exceeds max %d", ErrQueryTooLong, len(sanitized), MaxQueryLength)
	}

	var sb strings.Builder
	// Preallocate capacity: system prompt (~700) + query (~100) + markers (~100)
	sb.Grow(len(SystemPrompt) + len(sanitized) + 128)

	// System turn
	sb.WriteString(imStart)
	sb.WriteString("system\n")
	sb.WriteString(SystemPrompt)
	sb.WriteString("\n")
	sb.WriteString(imEnd)
	sb.WriteByte('\n')

	// User turn
	sb.WriteString(imStart)
	sb.WriteString("user\n")
	sb.WriteString(sanitized)
	sb.WriteString("\n")
	sb.WriteString(imEnd)
	sb.WriteByte('\n')

	// Assistant turn primer (ends strictly with newline, NO trailing space)
	sb.WriteString(imStart)
	sb.WriteString("assistant\n")

	return sb.String(), nil
}

// SanitizeQuery trims whitespace, normalizes newlines, strips null bytes, and neutralizes ChatML tokens.
func SanitizeQuery(input string) string {
	if len(input) == 0 {
		return ""
	}
	// Strip null bytes to prevent premature C-string termination
	cleaned := strings.ReplaceAll(input, "\x00", "")
	// Normalize CRLF to LF
	cleaned = strings.ReplaceAll(cleaned, "\r\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, "\r", "\n")
	// Trim surrounding whitespace
	cleaned = strings.TrimSpace(cleaned)
	if len(cleaned) == 0 {
		return ""
	}
	// Neutralize ChatML special token markers to prevent prompt injection
	cleaned = strings.ReplaceAll(cleaned, "<|im_start|>", "[im_start]")
	cleaned = strings.ReplaceAll(cleaned, "<|im_end|>", "[im_end]")
	cleaned = strings.ReplaceAll(cleaned, "<|", "< |")

	return cleaned
}

// HistoryTurn represents a conversational message from the user or assistant.
type HistoryTurn struct {
	Role    string // "user" or "assistant"
	Content string
}

// TaskContext represents an active or recent task for conversational grounding.
type TaskContext struct {
	Task      string
	TimeExpr  string
	Completed bool
}

// ConversationalSystemPrompt instructs the AI to behave as a personal offline planner.
const ConversationalSystemPrompt = `You are Planner Bot, a private, offline AI productivity assistant and daily planner running locally on the user's device.
You help the user plan their day, organize tasks, set reminders, and answer questions thoughtfully and authentically.

Rules:
1. Be natural, direct, helpful, and candid. Never repeat robotic canned greetings over and over.
2. Answer the user's questions directly and conversationally. If asked about your nature, honestly explain that you are Planner Bot, powered by a local lightweight language model running privately offline on their device.
3. If the user asks about their schedule or past tasks, answer accurately using the conversation history and active tasks below.
4. Do NOT use emojis. Use clean, humanistic formatting.`

// BuildConversationalPrompt constructs a multi-turn ChatML prompt with conversation memory and optional executive briefing.
func BuildConversationalPrompt(history []HistoryTurn, tasks []TaskContext, userMessage string, currentTimeStr string, briefingContext ...string) string {
	var sb strings.Builder
	// Preallocate generous buffer for 8K token context
	sb.Grow(4096)

	// System turn with context injection
	sb.WriteString(imStart)
	sb.WriteString("system\n")
	sb.WriteString(ConversationalSystemPrompt)
	sb.WriteString("\n\n")

	if len(briefingContext) > 0 && briefingContext[0] != "" {
		sb.WriteString(briefingContext[0])
		sb.WriteString("\n\n")
	}

	if currentTimeStr != "" {
		sb.WriteString("Current Time: ")
		sb.WriteString(currentTimeStr)
		sb.WriteString("\n\n")
	}

	if len(tasks) > 0 {
		sb.WriteString("Active Tasks & Agenda:\n")
		for _, t := range tasks {
			status := "pending"
			if t.Completed {
				status = "completed"
			}
			if t.TimeExpr != "" {
				sb.WriteString(fmt.Sprintf("- %s (scheduled: %s) [%s]\n", t.Task, t.TimeExpr, status))
			} else {
				sb.WriteString(fmt.Sprintf("- %s [%s]\n", t.Task, status))
			}
		}
		sb.WriteString("\n")
	}

	sb.WriteString(imEnd)
	sb.WriteByte('\n')

	// Append recent conversation turns
	for _, turn := range history {
		role := turn.Role
		if role != "user" && role != "assistant" {
			role = "user"
		}
		cleanContent := SanitizeQuery(turn.Content)
		if cleanContent == "" {
			continue
		}
		sb.WriteString(imStart)
		sb.WriteString(role)
		sb.WriteByte('\n')
		sb.WriteString(cleanContent)
		sb.WriteString("\n")
		sb.WriteString(imEnd)
		sb.WriteByte('\n')
	}

	// Current user turn
	sanitizedQuery := SanitizeQuery(userMessage)
	if sanitizedQuery != "" {
		sb.WriteString(imStart)
		sb.WriteString("user\n")
		sb.WriteString(sanitizedQuery)
		sb.WriteString("\n")
		sb.WriteString(imEnd)
		sb.WriteByte('\n')
	}

	// Primer for assistant response
	sb.WriteString(imStart)
	sb.WriteString("assistant\n")

	return sb.String()
}

// BuildPromptWithContext constructs a reminder extraction prompt with history and tasks context.
func BuildPromptWithContext(userQuery string, history []HistoryTurn, tasks []TaskContext) (string, error) {
	sanitized := SanitizeQuery(userQuery)
	if len(sanitized) == 0 {
		return "", ErrEmptyQuery
	}

	var sb strings.Builder
	sb.Grow(len(SystemPrompt) + len(sanitized) + 1024)

	// System turn with extraction instructions + context
	sb.WriteString(imStart)
	sb.WriteString("system\n")
	sb.WriteString(SystemPrompt)

	if len(tasks) > 0 || len(history) > 0 {
		sb.WriteString("\n\nContext for resolving co-references:")
		if len(tasks) > 0 {
			sb.WriteString("\nRecent Tasks:\n")
			for _, t := range tasks {
				sb.WriteString(fmt.Sprintf("- %s (at %s)\n", t.Task, t.TimeExpr))
			}
		}
		if len(history) > 0 {
			sb.WriteString("Recent Discussion:\n")
			for _, h := range history {
				sb.WriteString(fmt.Sprintf("%s: %s\n", h.Role, SanitizeQuery(h.Content)))
			}
		}
	}

	sb.WriteString("\n")
	sb.WriteString(imEnd)
	sb.WriteByte('\n')

	// User turn
	sb.WriteString(imStart)
	sb.WriteString("user\n")
	sb.WriteString(sanitized)
	sb.WriteString("\n")
	sb.WriteString(imEnd)
	sb.WriteByte('\n')

	// Assistant turn primer
	sb.WriteString(imStart)
	sb.WriteString("assistant\n")

	return sb.String(), nil
}

