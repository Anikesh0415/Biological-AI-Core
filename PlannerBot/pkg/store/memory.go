package store

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
)

var (
	bucketMemoryNodes = []byte("memory_nodes")
	bucketUserProfile = []byte("user_profile")
)

// MemoryNode represents a consolidated weekly or thematic knowledge node.
type MemoryNode struct {
	ID        string    `json:"id"`
	Period    string    `json:"period"` // e.g. "2026-W37"
	Summary   string    `json:"summary"`
	KeyThemes []string  `json:"key_themes"`
	FocusAvg  float64   `json:"focus_avg"`
	CreatedAt time.Time `json:"created_at"`
}

// UserProfile holds distilled behavioral patterns, chronotype focus windows, and core objectives.
type UserProfile struct {
	PrimeObjective string    `json:"prime_objective"`
	PeakFocusHours string    `json:"peak_focus_hours"` // e.g. "09:00 - 11:30 AM"
	TopDistraction string    `json:"top_distraction"`
	TotalTasksDone int       `json:"total_tasks_done"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// ExecutiveBriefing bundles the morning cognitive snapshot for UI and prompt injection.
type ExecutiveBriefing struct {
	Date            string      `json:"date"`
	Greeting        string      `json:"greeting"`
	YesterdayDone   int         `json:"yesterday_done"`
	YesterdayMissed int         `json:"yesterday_missed"`
	TodayPending    []*Reminder `json:"today_pending"`
	OverdueCount    int         `json:"overdue_count"`
	PrimeObjective  string      `json:"prime_objective"`
	PeakFocusHours  string      `json:"peak_focus_hours"`
	PromptInjection string      `json:"prompt_injection"`
}

// SaveMemoryNode stores a consolidated weekly memory node.
func (s *Store) SaveMemoryNode(node *MemoryNode) error {
	if node.ID == "" {
		node.ID = fmt.Sprintf("node-%d", time.Now().Unix())
	}
	if node.CreatedAt.IsZero() {
		node.CreatedAt = time.Now().UTC()
	}

	data, err := json.Marshal(node)
	if err != nil {
		return fmt.Errorf("store: marshal memory node: %w", err)
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketMemoryNodes)
		if b == nil {
			return fmt.Errorf("store: memory_nodes bucket not found")
		}
		return b.Put([]byte(node.ID), data)
	})
}

// ListMemoryNodes returns all consolidated memory nodes in chronological order.
func (s *Store) ListMemoryNodes() ([]*MemoryNode, error) {
	var nodes []*MemoryNode
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketMemoryNodes)
		if b == nil {
			return nil
		}
		return b.ForEach(func(k, v []byte) error {
			var node MemoryNode
			if err := json.Unmarshal(v, &node); err == nil {
				nodes = append(nodes, &node)
			}
			return nil
		})
	})
	return nodes, err
}

// GetUserProfile retrieves the distilled user profile, or returns defaults if uninitialized.
func (s *Store) GetUserProfile() (*UserProfile, error) {
	var prof UserProfile
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketUserProfile)
		if b == nil {
			return nil
		}
		data := b.Get([]byte("current"))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &prof)
	})
	if err != nil {
		return nil, err
	}

	if prof.PeakFocusHours == "" {
		prof.PeakFocusHours = "09:00 - 11:30 AM"
	}
	if prof.PrimeObjective == "" {
		prof.PrimeObjective = "Master deep focus and maintain high execution consistency"
	}
	return &prof, nil
}

// SaveUserProfile saves or updates the user profile.
func (s *Store) SaveUserProfile(prof *UserProfile) error {
	prof.UpdatedAt = time.Now().UTC()
	data, err := json.Marshal(prof)
	if err != nil {
		return fmt.Errorf("store: marshal user profile: %w", err)
	}
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketUserProfile)
		if b == nil {
			return fmt.Errorf("store: user_profile bucket not found")
		}
		return b.Put([]byte("current"), data)
	})
}

// SynthesizeExecutiveBriefing builds an executive briefing snapshot from active database state.
func (s *Store) SynthesizeExecutiveBriefing(now time.Time) (*ExecutiveBriefing, error) {
	all, err := s.ListAll()
	if err != nil {
		return nil, err
	}

	loc := now.Location()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	todayEnd := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, loc)
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	yesterdayEnd := todayStart.Add(-time.Nanosecond)

	var yesterdayDone, yesterdayMissed, overdueCount int
	var todayPending []*Reminder

	for _, r := range all {
		if r.Deleted {
			continue
		}

		fireLocal := r.FireAt.In(loc)

		// Check yesterday
		if fireLocal.After(yesterdayStart) && fireLocal.Before(yesterdayEnd) {
			if r.Fired {
				yesterdayDone++
			} else {
				yesterdayMissed++
			}
		}

		// Check overdue (scheduled in past, not fired)
		if !r.Fired && fireLocal.Before(now) {
			overdueCount++
		}

		// Check today's pending
		if !r.Fired && fireLocal.After(todayStart) && fireLocal.Before(todayEnd) {
			todayPending = append(todayPending, r)
		}
	}

	prof, _ := s.GetUserProfile()
	if prof == nil {
		prof = &UserProfile{
			PeakFocusHours: "09:00 - 11:30 AM",
			PrimeObjective: "Deep focus and steady execution",
		}
	}

	greeting := "Good morning"
	hour := now.Hour()
	if hour >= 12 && hour < 17 {
		greeting = "Good afternoon"
	} else if hour >= 17 {
		greeting = "Good evening"
	}

	// Build injection text (~200-300 tokens)
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("[EXECUTIVE BRIEFING — %s]\n", now.Format("Monday, Jan 2")))
	sb.WriteString(fmt.Sprintf("- Yesterday Performance: %d tasks completed, %d missed.\n", yesterdayDone, yesterdayMissed))
	sb.WriteString(fmt.Sprintf("- Today's Scheduled Focus: %d tasks pending.\n", len(todayPending)))
	if len(todayPending) > 0 {
		earliest := todayPending[0].FireAt.In(loc).Format("3:04 PM")
		sb.WriteString(fmt.Sprintf("  • Next action at %s: \"%s\"\n", earliest, todayPending[0].Task))
	}
	if overdueCount > 0 {
		sb.WriteString(fmt.Sprintf("- Overdue Items: %d requiring compassionate rebalancing.\n", overdueCount))
	}
	sb.WriteString(fmt.Sprintf("- Biological Peak Focus Window: %s\n", prof.PeakFocusHours))
	sb.WriteString(fmt.Sprintf("- Keystone Objective: %s\n", prof.PrimeObjective))

	return &ExecutiveBriefing{
		Date:            now.Format("Monday, January 2"),
		Greeting:        greeting,
		YesterdayDone:   yesterdayDone,
		YesterdayMissed: yesterdayMissed,
		TodayPending:    todayPending,
		OverdueCount:    overdueCount,
		PrimeObjective:  prof.PrimeObjective,
		PeakFocusHours:  prof.PeakFocusHours,
		PromptInjection: sb.String(),
	}, nil
}
