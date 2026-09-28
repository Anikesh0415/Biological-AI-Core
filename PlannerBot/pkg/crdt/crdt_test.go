package crdt

import (
	"path/filepath"
	"testing"
	"time"

	"planner_bot/pkg/store"
)

func TestResolveConflictNewerWins(t *testing.T) {
	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Hour)

	local := &store.Reminder{
		ID:        "r1",
		Task:      "Original task",
		UpdatedAt: t0,
		Version:   1,
	}

	incomingNewer := &store.Reminder{
		ID:        "r1",
		Task:      "Updated task",
		UpdatedAt: t1,
		Version:   2,
	}

	if !ResolveConflict(local, incomingNewer) {
		t.Fatal("incoming newer timestamp must win")
	}

	if ResolveConflict(incomingNewer, local) {
		t.Fatal("local newer timestamp must reject older incoming")
	}
}

func TestResolveConflictTombstoneWinsOnTie(t *testing.T) {
	t0 := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)

	localActive := &store.Reminder{
		ID:        "r1",
		Task:      "Buy milk",
		UpdatedAt: t0,
		Version:   1,
		Deleted:   false,
	}

	incomingDeleted := &store.Reminder{
		ID:        "r1",
		Task:      "Buy milk",
		UpdatedAt: t0,
		Version:   1,
		Deleted:   true,
	}

	if !ResolveConflict(localActive, incomingDeleted) {
		t.Fatal("incoming tombstone must win over active when timestamps and versions are tied")
	}

	if ResolveConflict(incomingDeleted, localActive) {
		t.Fatal("incoming active must NOT overwrite existing tombstone on tie")
	}
}

func TestMergeReminders(t *testing.T) {
	t0 := time.Date(2025, 1, 1, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(1 * time.Hour)
	t2 := t0.Add(2 * time.Hour)

	local := map[string]*store.Reminder{
		"r1": {ID: "r1", Task: "Keep local (newer)", UpdatedAt: t2},
		"r2": {ID: "r2", Task: "Local old", UpdatedAt: t0},
	}

	incoming := []*store.Reminder{
		{ID: "r1", Task: "Remote old (should ignore)", UpdatedAt: t1},
		{ID: "r2", Task: "Remote new (should update)", UpdatedAt: t1},
		{ID: "r3", Task: "Brand new remote", UpdatedAt: t1},
	}

	merged, res := MergeReminders(local, incoming)

	if res.Added != 1 {
		t.Fatalf("expected 1 added, got %d", res.Added)
	}
	if res.Updated != 1 {
		t.Fatalf("expected 1 updated, got %d", res.Updated)
	}
	if res.Ignored != 1 {
		t.Fatalf("expected 1 ignored, got %d", res.Ignored)
	}

	if merged["r1"].Task != "Keep local (newer)" {
		t.Errorf("r1 task mismatch: %q", merged["r1"].Task)
	}
	if merged["r2"].Task != "Remote new (should update)" {
		t.Errorf("r2 task mismatch: %q", merged["r2"].Task)
	}
	if merged["r3"].Task != "Brand new remote" {
		t.Errorf("r3 task mismatch: %q", merged["r3"].Task)
	}
}

func TestApplyDeltaToStore(t *testing.T) {
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "crdt_test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}
	defer s.Close()

	now := time.Now().UTC()
	_ = s.Save(&store.Reminder{
		ID:        "test-1",
		Task:      "Task on Laptop",
		UpdatedAt: now,
		Version:   1,
	})

	// Simulate Phone sending delta:
	// 1. Updating test-1 with a newer timestamp
	// 2. Adding test-2 from Phone
	phoneDelta := []*store.Reminder{
		{
			ID:        "test-1",
			Task:      "Task edited on Phone",
			UpdatedAt: now.Add(10 * time.Minute),
			Version:   2,
		},
		{
			ID:        "test-2",
			Task:      "New task from Phone",
			UpdatedAt: now.Add(5 * time.Minute),
			Version:   1,
		},
	}

	res, err := ApplyDeltaToStore(s, phoneDelta)
	if err != nil {
		t.Fatalf("ApplyDeltaToStore failed: %v", err)
	}

	if res.Added != 1 || res.Updated != 1 {
		t.Fatalf("expected 1 added and 1 updated, got %+v", res)
	}

	r1, err := s.Get("test-1")
	if err != nil {
		t.Fatalf("failed to get test-1: %v", err)
	}
	if r1.Task != "Task edited on Phone" {
		t.Errorf("expected updated task, got %q", r1.Task)
	}

	r2, err := s.Get("test-2")
	if err != nil {
		t.Fatalf("failed to get test-2: %v", err)
	}
	if r2.Task != "New task from Phone" {
		t.Errorf("expected new task, got %q", r2.Task)
	}
}

func TestMergeFieldLevelConcurrentEditAndComplete(t *testing.T) {
	t0 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	tLaptop := t0.Add(5 * time.Minute)
	tPhone := t0.Add(10 * time.Minute)

	// Laptop marked it completed at 10:05
	laptopRecord := &store.Reminder{
		ID:        "sync-task",
		Task:      "Original task description",
		Fired:     true,
		FiredAt:   &tLaptop,
		UpdatedAt: tLaptop,
		Version:   2,
	}

	// Phone edited task title at 10:10 (without knowing it was marked completed yet)
	phoneRecord := &store.Reminder{
		ID:        "sync-task",
		Task:      "Refined task title by user",
		Fired:     false,
		UpdatedAt: tPhone,
		Version:   3,
	}

	// Merge Phone into Laptop
	merged, changed := MergeFieldLevel(laptopRecord, phoneRecord)
	if !changed {
		t.Fatal("expected merged state to have changed")
	}

	// Both the new title AND the completion status MUST be preserved
	if merged.Task != "Refined task title by user" {
		t.Errorf("expected updated task title, got %q", merged.Task)
	}
	if !merged.Fired {
		t.Errorf("expected Fired to remain true (completion monotonicity preserved)")
	}
	if merged.FiredAt == nil || !merged.FiredAt.Equal(tLaptop) {
		t.Errorf("expected FiredAt timestamp to be preserved")
	}
}

