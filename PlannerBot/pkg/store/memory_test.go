package store

import (
	"path/filepath"
	"testing"
	"time"
)

func TestExecutiveBriefingAndMemory(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test_memory.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}
	defer s.Close()

	// Use fixed reference midday time so late night test execution never crosses midnight
	refTime := time.Date(2026, 9, 14, 12, 0, 0, 0, time.UTC)

	// 1. Test User Profile
	prof, err := s.GetUserProfile()
	if err != nil {
		t.Fatalf("GetUserProfile failed: %v", err)
	}
	if prof.PeakFocusHours == "" {
		t.Errorf("expected default peak focus hours, got empty")
	}

	prof.PrimeObjective = "Ship PlannerBot v0.3.0"
	if err := s.SaveUserProfile(prof); err != nil {
		t.Fatalf("SaveUserProfile failed: %v", err)
	}

	updatedProf, _ := s.GetUserProfile()
	if updatedProf.PrimeObjective != "Ship PlannerBot v0.3.0" {
		t.Errorf("expected updated objective, got %s", updatedProf.PrimeObjective)
	}

	// 2. Add some test reminders
	yesterday := refTime.AddDate(0, 0, -1)
	rYesterdayDone := &Reminder{
		ID:        "y-done",
		Task:      "Complete sprint review",
		FireAt:    yesterday.UTC(),
		Fired:     true,
		CreatedAt: yesterday.UTC(),
		Version:   1,
	}
	_ = s.Save(rYesterdayDone)

	rTodayPending := &Reminder{
		ID:        "t-pend",
		Task:      "Deep architecture sprint",
		FireAt:    refTime.Add(2 * time.Hour).UTC(), // 14:00 today
		Fired:     false,
		CreatedAt: refTime.UTC(),
		Version:   1,
	}
	_ = s.Save(rTodayPending)

	// 3. Test SynthesizeExecutiveBriefing
	briefing, err := s.SynthesizeExecutiveBriefing(refTime)
	if err != nil {
		t.Fatalf("SynthesizeExecutiveBriefing failed: %v", err)
	}

	if briefing.YesterdayDone != 1 {
		t.Errorf("expected yesterday done = 1, got %d", briefing.YesterdayDone)
	}
	if len(briefing.TodayPending) != 1 {
		t.Errorf("expected today pending = 1, got %d", len(briefing.TodayPending))
	}
	if briefing.PromptInjection == "" {
		t.Errorf("expected non-empty prompt injection")
	}

	// 4. Test Memory Node
	node := &MemoryNode{
		ID:        "w37",
		Period:    "2026-W37",
		Summary:   "High velocity sprint on edge LLM and P2P CRDT sync",
		KeyThemes: []string{"Cognitive Engine", "Local First", "Zero Cloud"},
		FocusAvg:  78.5,
	}
	if err := s.SaveMemoryNode(node); err != nil {
		t.Fatalf("SaveMemoryNode failed: %v", err)
	}

	nodes, err := s.ListMemoryNodes()
	if err != nil {
		t.Fatalf("ListMemoryNodes failed: %v", err)
	}
	if len(nodes) != 1 || nodes[0].Period != "2026-W37" {
		t.Errorf("unexpected nodes list: %+v", nodes)
	}
}
