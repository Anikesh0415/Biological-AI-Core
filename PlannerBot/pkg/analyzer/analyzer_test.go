package analyzer

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"planner_bot/pkg/store"
)

func TestAnalyzeDayHeuristic(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "analyzer_test.db")
	s, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("store.New failed: %v", err)
	}
	defer s.Close()

	now := time.Now().UTC()
	today := time.Date(now.Year(), now.Month(), now.Day(), 12, 0, 0, 0, time.UTC)

	// Add 2 completed tasks
	_ = s.Save(&store.Reminder{
		ID:        "t1",
		Task:      "Submit physics homework",
		FireAt:    today.Add(-2 * time.Hour),
		Fired:     true,
		CreatedAt: today.Add(-5 * time.Hour),
	})
	_ = s.Save(&store.Reminder{
		ID:        "t2",
		Task:      "Workout at gym",
		FireAt:    today.Add(-1 * time.Hour),
		Fired:     true,
		CreatedAt: today.Add(-5 * time.Hour),
	})

	// Add 1 pending task
	_ = s.Save(&store.Reminder{
		ID:        "t3",
		Task:      "Read 20 pages of book",
		FireAt:    today.Add(2 * time.Hour),
		Fired:     false,
		CreatedAt: today.Add(-5 * time.Hour),
	})

	a := New(s, nil)
	summary, err := a.AnalyzeDay(context.Background(), today)
	if err != nil {
		t.Fatalf("AnalyzeDay failed: %v", err)
	}

	if summary.CompletedCount != 2 {
		t.Errorf("expected 2 completed, got %d", summary.CompletedCount)
	}
	if summary.PendingCount != 1 {
		t.Errorf("expected 1 pending, got %d", summary.PendingCount)
	}
	if len(summary.Suggestions) == 0 {
		t.Fatal("expected suggestions for tomorrow")
	}

	// Should carry forward unfinished task
	found := false
	for _, sugg := range summary.Suggestions {
		if strings.Contains(sugg.Task, "Read 20 pages") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'Read 20 pages' in tomorrow's suggestions: %+v", summary.Suggestions)
	}
}
