package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// newTestStore creates a temporary BoltDB store for testing.
func newTestStore(t *testing.T) *Store {
	t.Helper()
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "test.db")

	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}

	t.Cleanup(func() {
		_ = s.Close()
	})

	return s
}

func TestNewStore(t *testing.T) {
	s := newTestStore(t)
	if s.Path() == "" {
		t.Error("store path should not be empty")
	}
	if _, err := os.Stat(s.Path()); err != nil {
		t.Errorf("store file should exist: %v", err)
	}
}

func TestNewStoreDefaultPath(t *testing.T) {
	// Test that empty path uses default (don't actually create it in CI)
	// Just verify the constructor logic doesn't panic
	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "sub", "dir", "test.db")
	s, err := New(dbPath)
	if err != nil {
		t.Fatalf("failed to create store with nested path: %v", err)
	}
	defer s.Close()

	if s.Path() != dbPath {
		t.Errorf("expected path %q, got %q", dbPath, s.Path())
	}
}

func TestSaveAndGet(t *testing.T) {
	s := newTestStore(t)

	r := &Reminder{
		ID:        "test-001",
		Task:      "call mom",
		FireAt:    time.Now().Add(1 * time.Hour).UTC(),
		CreatedAt: time.Now().UTC(),
	}

	if err := s.Save(r); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	got, err := s.Get("test-001")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if got.Task != "call mom" {
		t.Errorf("expected Task='call mom', got %q", got.Task)
	}
	if got.Fired {
		t.Error("expected Fired=false")
	}
}

func TestSaveEmptyID(t *testing.T) {
	s := newTestStore(t)

	r := &Reminder{ID: "", Task: "test"}
	err := s.Save(r)
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestGetNotFound(t *testing.T) {
	s := newTestStore(t)

	_, err := s.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent ID")
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)

	r := &Reminder{ID: "del-001", Task: "delete me", FireAt: time.Now().UTC(), CreatedAt: time.Now().UTC()}
	_ = s.Save(r)

	if err := s.Delete("del-001"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := s.Get("del-001")
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestListPending(t *testing.T) {
	s := newTestStore(t)

	// Save 3 reminders: 2 pending, 1 fired
	now := time.Now().UTC()
	_ = s.Save(&Reminder{ID: "p1", Task: "pending 1", FireAt: now.Add(1 * time.Hour), CreatedAt: now})
	_ = s.Save(&Reminder{ID: "p2", Task: "pending 2", FireAt: now.Add(2 * time.Hour), CreatedAt: now})
	_ = s.Save(&Reminder{ID: "f1", Task: "fired 1", FireAt: now.Add(-1 * time.Hour), CreatedAt: now, Fired: true})

	pending, err := s.ListPending()
	if err != nil {
		t.Fatalf("ListPending failed: %v", err)
	}
	if len(pending) != 2 {
		t.Errorf("expected 2 pending, got %d", len(pending))
	}
}

func TestListDue(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	_ = s.Save(&Reminder{ID: "d1", Task: "due 1", FireAt: now.Add(-10 * time.Minute), CreatedAt: now})
	_ = s.Save(&Reminder{ID: "d2", Task: "due 2", FireAt: now.Add(-5 * time.Minute), CreatedAt: now})
	_ = s.Save(&Reminder{ID: "nd1", Task: "not due", FireAt: now.Add(1 * time.Hour), CreatedAt: now})
	_ = s.Save(&Reminder{ID: "fd1", Task: "fired", FireAt: now.Add(-1 * time.Hour), CreatedAt: now, Fired: true})

	due, err := s.ListDue(now)
	if err != nil {
		t.Fatalf("ListDue failed: %v", err)
	}
	if len(due) != 2 {
		t.Errorf("expected 2 due reminders, got %d", len(due))
	}
}

func TestMarkFired(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	_ = s.Save(&Reminder{ID: "mf1", Task: "mark me", FireAt: now, CreatedAt: now})

	firedAt := now.Add(1 * time.Minute)
	if err := s.MarkFired("mf1", firedAt); err != nil {
		t.Fatalf("MarkFired failed: %v", err)
	}

	got, err := s.Get("mf1")
	if err != nil {
		t.Fatalf("Get after MarkFired failed: %v", err)
	}
	if !got.Fired {
		t.Error("expected Fired=true")
	}
	if got.FiredAt == nil {
		t.Error("expected FiredAt to be set")
	}
}

func TestMarkMissed(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	_ = s.Save(&Reminder{ID: "mm1", Task: "missed one", FireAt: now, CreatedAt: now})

	if err := s.MarkMissed("mm1"); err != nil {
		t.Fatalf("MarkMissed failed: %v", err)
	}

	got, err := s.Get("mm1")
	if err != nil {
		t.Fatalf("Get after MarkMissed failed: %v", err)
	}
	if !got.Missed {
		t.Error("expected Missed=true")
	}
}

func TestCount(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	_ = s.Save(&Reminder{ID: "c1", Task: "one", FireAt: now, CreatedAt: now})
	_ = s.Save(&Reminder{ID: "c2", Task: "two", FireAt: now, CreatedAt: now})
	_ = s.Save(&Reminder{ID: "c3", Task: "three", FireAt: now, CreatedAt: now})

	count, err := s.Count()
	if err != nil {
		t.Fatalf("Count failed: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count=3, got %d", count)
	}
}

func TestPurgeCompleted(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	_ = s.Save(&Reminder{ID: "pc1", Task: "pending", FireAt: now.Add(1 * time.Hour), CreatedAt: now})
	_ = s.Save(&Reminder{ID: "pc2", Task: "fired1", FireAt: now, CreatedAt: now, Fired: true})
	_ = s.Save(&Reminder{ID: "pc3", Task: "fired2", FireAt: now, CreatedAt: now, Fired: true})

	purged, err := s.PurgeCompleted()
	if err != nil {
		t.Fatalf("PurgeCompleted failed: %v", err)
	}
	if purged != 2 {
		t.Errorf("expected 2 purged, got %d", purged)
	}

	count, _ := s.Count()
	if count != 1 {
		t.Errorf("expected 1 remaining, got %d", count)
	}
}

func TestListAll(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	_ = s.Save(&Reminder{ID: "la1", Task: "one", FireAt: now, CreatedAt: now})
	_ = s.Save(&Reminder{ID: "la2", Task: "two", FireAt: now, CreatedAt: now, Fired: true})

	all, err := s.ListAll()
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 total, got %d", len(all))
	}
}

func TestOverwriteExisting(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	_ = s.Save(&Reminder{ID: "ow1", Task: "original", FireAt: now, CreatedAt: now})
	_ = s.Save(&Reminder{ID: "ow1", Task: "updated", FireAt: now, CreatedAt: now})

	got, err := s.Get("ow1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Task != "updated" {
		t.Errorf("expected Task='updated', got %q", got.Task)
	}
}

func TestChatHistory(t *testing.T) {
	s := newTestStore(t)

	now := time.Now().UTC()
	msg1 := &ChatMessage{
		ID:        "m1",
		Role:      "user",
		Content:   "Remind me to buy groceries at 5pm",
		Timestamp: now.Add(-2 * time.Minute),
	}
	msg2 := &ChatMessage{
		ID:        "m2",
		Role:      "assistant",
		Content:   "Got it! Set a reminder for buy groceries at 5:00 PM.",
		Timestamp: now.Add(-1 * time.Minute),
		Action:    "reminder_created",
	}
	msg3 := &ChatMessage{
		ID:        "m3",
		Role:      "user",
		Content:   "What did I just plan?",
		Timestamp: now,
	}

	if err := s.SaveChat(msg1); err != nil {
		t.Fatalf("SaveChat m1 failed: %v", err)
	}
	if err := s.SaveChat(msg2); err != nil {
		t.Fatalf("SaveChat m2 failed: %v", err)
	}
	if err := s.SaveChat(msg3); err != nil {
		t.Fatalf("SaveChat m3 failed: %v", err)
	}

	chats, err := s.ListRecentChats(10)
	if err != nil {
		t.Fatalf("ListRecentChats failed: %v", err)
	}
	if len(chats) != 3 {
		t.Fatalf("expected 3 chats, got %d", len(chats))
	}
	if chats[0].Content != msg1.Content {
		t.Errorf("expected first chat to be %q, got %q", msg1.Content, chats[0].Content)
	}
	if chats[2].Content != msg3.Content {
		t.Errorf("expected third chat to be %q, got %q", msg3.Content, chats[2].Content)
	}

	// Test limit
	recent2, err := s.ListRecentChats(2)
	if err != nil {
		t.Fatalf("ListRecentChats(2) failed: %v", err)
	}
	if len(recent2) != 2 {
		t.Fatalf("expected 2 chats, got %d", len(recent2))
	}
	if recent2[0].ID != "m2" || recent2[1].ID != "m3" {
		t.Errorf("expected m2 and m3, got %s and %s", recent2[0].ID, recent2[1].ID)
	}

	// Test ClearChats
	if err := s.ClearChats(); err != nil {
		t.Fatalf("ClearChats failed: %v", err)
	}
	afterClear, err := s.ListRecentChats(10)
	if err != nil {
		t.Fatalf("ListRecentChats after clear failed: %v", err)
	}
	if len(afterClear) != 0 {
		t.Errorf("expected 0 chats after clear, got %d", len(afterClear))
	}
}
