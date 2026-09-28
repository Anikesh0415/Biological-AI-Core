package extractor

import (
	"context"
	"errors"
	"testing"

	"planner_bot/pkg/grammar"
)

func TestNewEngineRequiresPathsWhenDownloadDisabled(t *testing.T) {
	cfg := Config{
		ModelPath:      "",
		ServerBinary:   "",
		DownloadIfNone: false,
	}

	_, err := NewEngine(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when ModelPath and ServerBinary are empty with DownloadIfNone=false")
	}
	t.Logf("correctly got error: %v", err)
}

func TestNewEngineRequiresModelPath(t *testing.T) {
	cfg := Config{
		ModelPath:      "",
		ServerBinary:   "/some/binary",
		DownloadIfNone: false,
	}

	_, err := NewEngine(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when ModelPath is empty with DownloadIfNone=false")
	}
	t.Logf("correctly got error: %v", err)
}

func TestNewEngineRequiresServerBinary(t *testing.T) {
	cfg := Config{
		ModelPath:      "/some/model.gguf",
		ServerBinary:   "",
		DownloadIfNone: false,
	}

	_, err := NewEngine(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error when ServerBinary is empty with DownloadIfNone=false")
	}
	t.Logf("correctly got error: %v", err)
}

func TestExtractReminderEmptyInput(t *testing.T) {
	// Create a minimal engine (server is nil, but we're testing input validation)
	e := &Engine{
		server:     nil,
		grammarStr: grammar.GetGrammar(),
	}

	_, err := e.ExtractReminder(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	if !errors.Is(err, grammar.ErrEmptyQuery) {
		t.Logf("got error: %v (may wrap ErrEmptyQuery)", err)
	}
}

func TestExtractReminderWhitespaceInput(t *testing.T) {
	e := &Engine{
		server:     nil,
		grammarStr: grammar.GetGrammar(),
	}

	_, err := e.ExtractReminder(context.Background(), "   \n\t  ")
	if err == nil {
		t.Fatal("expected error for whitespace-only input")
	}
	t.Logf("correctly got error: %v", err)
}

func TestReminderRawJSON(t *testing.T) {
	r := &Reminder{
		Task:    "call mom",
		Time:    "7pm",
		rawJSON: `{"task":"call mom","time":"7pm"}`,
	}

	if got := r.RawJSON(); got != `{"task":"call mom","time":"7pm"}` {
		t.Errorf("RawJSON() = %q, want %q", got, `{"task":"call mom","time":"7pm"}`)
	}
}

func TestReminderRawJSONEmpty(t *testing.T) {
	r := &Reminder{
		Task: "test",
		Time: "now",
	}

	if got := r.RawJSON(); got != "" {
		t.Errorf("RawJSON() = %q, want empty string", got)
	}
}

func TestConfigDefaults(t *testing.T) {
	cfg := Config{}
	if cfg.Threads != 0 {
		t.Errorf("expected zero Threads default, got %d", cfg.Threads)
	}
	if cfg.ContextSize != 0 {
		t.Errorf("expected zero ContextSize default, got %d", cfg.ContextSize)
	}
	if cfg.DownloadIfNone {
		t.Error("expected DownloadIfNone to be false by default")
	}
}
