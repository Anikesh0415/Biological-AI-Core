package ui

import (
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	"planner_bot/pkg/analyzer"
	"planner_bot/pkg/p2p"
	"planner_bot/pkg/store"
)

func TestUIServerBootAndAPI(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "ui_test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}
	defer s.Close()

	p2pEngine, err := p2p.NewEngine(p2p.EngineConfig{
		DeviceID: "ui-test-device",
		UserCode: "PLAN-UI-TEST",
		Store:    s,
	})
	if err != nil {
		t.Fatalf("p2p.NewEngine failed: %v", err)
	}
	defer p2pEngine.Close()

	an := analyzer.New(s, nil)

	server := NewServer(Config{
		Port:      0, // Auto-assign port
		Store:     s,
		P2PEngine: p2pEngine,
		Analyzer:  an,
	})

	if err := server.Start(); err != nil {
		t.Fatalf("server.Start failed: %v", err)
	}
	defer server.Close()

	if server.Port() <= 0 {
		t.Fatalf("expected positive port, got %d", server.Port())
	}

	// 1. Test Static HTML Index
	resp, err := http.Get(server.URL() + "/")
	if err != nil {
		t.Fatalf("GET / failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK from index, got %d", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	if !strings.Contains(string(body), "Planner Bot") {
		t.Errorf("expected HTML to contain 'Planner Bot', got %s", string(body[:200]))
	}

	// 2. Test /api/status JSON endpoint
	respStatus, err := http.Get(server.URL() + "/api/status")
	if err != nil {
		t.Fatalf("GET /api/status failed: %v", err)
	}
	defer respStatus.Body.Close()

	var statusMap map[string]interface{}
	if err := json.NewDecoder(respStatus.Body).Decode(&statusMap); err != nil {
		t.Fatalf("decode status JSON failed: %v", err)
	}

	if statusMap["pairing_code"] != "PLAN-UI-TEST" {
		t.Errorf("expected pairing_code='PLAN-UI-TEST', got %v", statusMap["pairing_code"])
	}
}

func TestIsReminderIntent(t *testing.T) {
	// Conversational queries that should NOT be reminders
	conversational := []string{
		"i am asking who developed antigravity",
		"who is antigravity",
		"who created you",
		"what is qwen",
		"how to code in rust",
		"why is the sky blue",
		"tell me about yourself",
		"in brief tell me how you work",
		"what can you do",
	}
	for _, q := range conversational {
		if isReminderIntent(q) {
			t.Errorf("expected isReminderIntent(%q) = false, got true", q)
		}
	}

	// Legitimate reminder requests that SHOULD be reminders
	reminders := []string{
		"remind me to call mom at 5pm",
		"set a reminder for 5 minutes",
		"set an alarm for 7:30 am",
		"schedule meeting tomorrow at 10am",
		"don't forget to buy milk tonight",
		"remind me in 10 minutes",
	}
	for _, r := range reminders {
		if !isReminderIntent(r) {
			t.Errorf("expected isReminderIntent(%q) = true, got false", r)
		}
	}
}

func TestCleanExtractedTask(t *testing.T) {
	tests := []struct {
		task     string
		rawTime  string
		expected string
	}{
		{"set an alarm for", "5 minutes", "Quick Reminder (5 minutes)"},
		{"5 minutes", "5 minutes", "Quick Reminder (5 minutes)"},
		{"set a reminder for", "10m", "Quick Reminder (10m)"},
		{"remind me to call mom", "7pm", "call mom"},
		{"call mom", "7pm", "call mom"},
		{"", "15 minutes", "Quick Reminder (15 minutes)"},
		{"alarm", "30 minutes", "Quick Reminder (30 minutes)"},
		{"study for English exam", "tomorrow at 4pm", "study for English exam"},
	}

	for _, tt := range tests {
		got := cleanExtractedTask(tt.task, tt.rawTime, "")
		if got != tt.expected {
			t.Errorf("cleanExtractedTask(%q, %q) = %q, expected %q", tt.task, tt.rawTime, got, tt.expected)
		}
	}
}

