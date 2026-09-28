package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// defaultCompletionTimeout is applied when the caller's context has no deadline.
const defaultCompletionTimeout = 60 * time.Second

// Complete sends a prompt to the llama-server /completion endpoint with an optional
// GBNF grammar constraint, and returns the generated text content.
func (s *Server) Complete(ctx context.Context, prompt, grammar string) (string, error) {
	return s.CompleteWithOptions(ctx, CompletionRequest{
		Prompt:      prompt,
		Grammar:     grammar,
		NPredict:    512,
		Temperature: 0.0,
		Stream:      false,
		CachePrompt: true,
		Stop:        []string{"<|im_end|>", "<|endoftext|>"},
	})
}

// CompleteWithOptions sends a fully customized CompletionRequest to llama-server.
func (s *Server) CompleteWithOptions(ctx context.Context, reqBody CompletionRequest) (string, error) {
	// Apply default timeout if context has no deadline
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, defaultCompletionTimeout)
		defer cancel()
	}

	if reqBody.NPredict <= 0 {
		reqBody.NPredict = 512
	}
	if len(reqBody.Stop) == 0 {
		reqBody.Stop = []string{"<|im_end|>", "<|endoftext|>"}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("llm: marshal completion request: %w", err)
	}

	url := s.BaseURL() + "/completion"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return "", fmt.Errorf("llm: create completion request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llm: completion request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("llm: completion returned HTTP %d: %s", resp.StatusCode, string(body))
	}

	var completionResp CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return "", fmt.Errorf("llm: decode completion response: %w", err)
	}

	return strings.TrimSpace(completionResp.Content), nil
}
