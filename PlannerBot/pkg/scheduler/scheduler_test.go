package scheduler

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"planner_bot/pkg/store"
)

func newTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.New(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("failed to create test store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestNewSchedulerDefaults(t *testing.T) {
	s := newTestStore(t)
	sc := New(s, 0, nil)

	if sc.interval != 1*time.Minute {
		t.Errorf("expected default interval=1m, got %v", sc.interval)
	}
	if sc.notify == nil {
		t.Error("expected default notify function")
	}
}

func TestStartAndStop(t *testing.T) {
	s := newTestStore(t)
	sc := New(s, 100*time.Millisecond, func(r *store.Reminder) {})

	ctx := context.Background()
	if err := sc.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !sc.IsRunning() {
		t.Error("expected scheduler to be running")
	}

	sc.Stop()

	if sc.IsRunning() {
		t.Error("expected scheduler to be stopped")
	}
}

func TestDoubleStartError(t *testing.T) {
	s := newTestStore(t)
	sc := New(s, 100*time.Millisecond, func(r *store.Reminder) {})

	ctx := context.Background()
	_ = sc.Start(ctx)
	defer sc.Stop()

	err := sc.Start(ctx)
	if err == nil {
		t.Fatal("expected error on double start")
	}
}

func TestSchedulerFiresDueReminder(t *testing.T) {
	s := newTestStore(t)

	var mu sync.Mutex
	var firedTasks []string

	sc := New(s, 50*time.Millisecond, func(r *store.Reminder) {
		mu.Lock()
		firedTasks = append(firedTasks, r.Task)
		mu.Unlock()
	})

	now := time.Now().UTC()
	_ = s.Save(&store.Reminder{
		ID:        "fire-me",
		Task:      "test task",
		FireAt:    now.Add(-1 * time.Minute), // Already due
		CreatedAt: now,
	})

	ctx := context.Background()
	_ = sc.Start(ctx)

	// Wait for at least one tick
	time.Sleep(200 * time.Millisecond)
	sc.Stop()

	mu.Lock()
	defer mu.Unlock()

	if len(firedTasks) == 0 {
		t.Fatal("expected at least one reminder to fire")
	}
	if firedTasks[0] != "test task" {
		t.Errorf("expected task='test task', got %q", firedTasks[0])
	}

	// Verify it was marked as fired in the store
	r, err := s.Get("fire-me")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if !r.Fired {
		t.Error("expected reminder to be marked as fired")
	}
}

func TestSchedulerSkipsFutureReminder(t *testing.T) {
	s := newTestStore(t)

	fired := false
	sc := New(s, 50*time.Millisecond, func(r *store.Reminder) {
		fired = true
	})

	now := time.Now().UTC()
	_ = s.Save(&store.Reminder{
		ID:        "future",
		Task:      "future task",
		FireAt:    now.Add(1 * time.Hour), // Not due yet
		CreatedAt: now,
	})

	ctx := context.Background()
	_ = sc.Start(ctx)
	time.Sleep(200 * time.Millisecond)
	sc.Stop()

	if fired {
		t.Error("future reminder should not have fired")
	}
}

func TestForceCheck(t *testing.T) {
	s := newTestStore(t)

	var fired bool
	sc := New(s, 1*time.Hour, func(r *store.Reminder) {
		fired = true
	})

	now := time.Now().UTC()
	_ = s.Save(&store.Reminder{
		ID:        "force",
		Task:      "forced check",
		FireAt:    now.Add(-1 * time.Minute),
		CreatedAt: now,
	})

	// Don't start the scheduler, just force a check
	sc.ForceCheck()

	if !fired {
		t.Error("ForceCheck should have fired the due reminder")
	}
}

func TestContextCancellation(t *testing.T) {
	s := newTestStore(t)
	sc := New(s, 50*time.Millisecond, func(r *store.Reminder) {})

	ctx, cancel := context.WithCancel(context.Background())
	_ = sc.Start(ctx)

	// Cancel the context
	cancel()
	time.Sleep(200 * time.Millisecond)

	// The scheduler should have stopped due to context cancellation
	sc.Stop() // Should not hang
}
