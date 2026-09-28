package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"planner_bot/pkg/llm"
	"planner_bot/pkg/store"
)

// DaySummary captures the state of tasks for a given day.
type DaySummary struct {
	Date           time.Time      `json:"date"`
	CompletedCount int            `json:"completed_count"`
	PendingCount   int            `json:"pending_count"`
	MissedCount    int            `json:"missed_count"`
	CompletedTasks []string       `json:"completed_tasks"`
	PendingTasks   []string       `json:"pending_tasks"`
	AnalysisText   string         `json:"analysis_text"`
	Suggestions    []TomorrowTask `json:"suggestions"`
	Win            string         `json:"win,omitempty"`
	Friction       string         `json:"friction,omitempty"`
	Keystone       string         `json:"keystone,omitempty"`
	ElasticScore   float64        `json:"elastic_score"` // James Clear Habit Consistency (0 - 100)
}

// TomorrowTask represents a suggested action item following the Rule of Three.
type TomorrowTask struct {
	Task          string `json:"task"`
	SuggestedTime string `json:"suggested_time"`
	Reason        string `json:"reason"`
	Category      string `json:"category,omitempty"` // "Keystone Priority", "Quick Win", "Recovery"
}

// Analyzer inspects daily task history, detects patterns, and generates insights and schedule recommendations.
type Analyzer struct {
	store *store.Store
	llm   *llm.Server
	mu    sync.RWMutex
}

// New creates a new day analyzer.
func New(s *store.Store, l *llm.Server) *Analyzer {
	return &Analyzer{
		store: s,
		llm:   l,
	}
}

// SetLLM dynamically attaches or updates the LLM server once loaded.
func (a *Analyzer) SetLLM(l *llm.Server) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.llm = l
}

// AnalyzeDay gathers today's task metrics and queries the local LLM for 3-Part Stoic debrief and schedule recommendations.
func (a *Analyzer) AnalyzeDay(ctx context.Context, targetDate time.Time) (*DaySummary, error) {
	if a.store == nil {
		return nil, fmt.Errorf("analyzer: store is required")
	}

	allReminders, err := a.store.ListAll()
	if err != nil {
		return nil, fmt.Errorf("analyzer: list reminders: %w", err)
	}

	startOfDay := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, targetDate.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	summary := &DaySummary{
		Date:           startOfDay,
		CompletedTasks: make([]string, 0),
		PendingTasks:   make([]string, 0),
		Suggestions:    make([]TomorrowTask, 0),
	}

	for _, r := range allReminders {
		onDate := (r.FireAt.After(startOfDay) && r.FireAt.Before(endOfDay)) ||
			(r.FiredAt != nil && r.FiredAt.After(startOfDay) && r.FiredAt.Before(endOfDay))

		if !onDate {
			continue
		}

		if r.Fired {
			summary.CompletedCount++
			summary.CompletedTasks = append(summary.CompletedTasks, r.Task)
		} else if r.Missed {
			summary.MissedCount++
			summary.PendingTasks = append(summary.PendingTasks, r.Task+" (missed)")
		} else {
			summary.PendingCount++
			summary.PendingTasks = append(summary.PendingTasks, r.Task)
		}
	}

	// Calculate James Clear Elastic Consistency Score (0 - 100)
	total := summary.CompletedCount + summary.PendingCount + summary.MissedCount
	if total == 0 {
		summary.ElasticScore = 100.0
	} else {
		rate := float64(summary.CompletedCount) / float64(total) * 100.0
		summary.ElasticScore = rate
	}

	// If LLM is available, generate 3-Part Stoic reflection and Rule of Three suggestions
	a.mu.RLock()
	activeLLM := a.llm
	a.mu.RUnlock()

	if activeLLM != nil {
		analysis, suggestions, win, friction, keystone, err := a.generateAIInsights(ctx, summary, targetDate)
		if err == nil {
			summary.AnalysisText = analysis
			summary.Suggestions = suggestions
			summary.Win = win
			summary.Friction = friction
			summary.Keystone = keystone
			return summary, nil
		}
	}

	// Fallback heuristic analysis if offline or LLM unavailable
	summary.AnalysisText = a.generateHeuristicAnalysis(summary)
	summary.Suggestions = a.generateHeuristicSuggestions(summary)
	summary.Win = "Committed to daily task execution and logging"
	if summary.PendingCount+summary.MissedCount > 0 {
		summary.Friction = fmt.Sprintf("%d tasks deferred or interrupted", summary.PendingCount+summary.MissedCount)
	} else {
		summary.Friction = "Clean execution with zero pending backlogs"
	}
	summary.Keystone = "Protect a 90-minute morning focus sprint tomorrow"

	return summary, nil
}

func (a *Analyzer) generateHeuristicAnalysis(s *DaySummary) string {
	total := s.CompletedCount + s.PendingCount + s.MissedCount
	if total == 0 {
		return "You had no scheduled tasks today. A great day to rest, reflect, and plan ahead for tomorrow."
	}

	rate := float64(s.CompletedCount) / float64(total) * 100
	if rate >= 80 {
		return fmt.Sprintf("Outstanding productivity. You completed %d of %d tasks (%.0f%%) with strong execution consistency.", s.CompletedCount, total, rate)
	} else if rate >= 50 {
		return fmt.Sprintf("Solid effort today. You finished %d of %d tasks (%.0f%%). Let's rebalance remaining tasks into protected focus blocks tomorrow.", s.CompletedCount, total, rate)
	}
	return fmt.Sprintf("Hectic day with %d missed or pending tasks out of %d. We apply compassionate schedule rebalancing: prune non-essentials and focus purely on your primary keystone tomorrow.", s.PendingCount+s.MissedCount, total)
}

func (a *Analyzer) generateHeuristicSuggestions(s *DaySummary) []TomorrowTask {
	var suggestions []TomorrowTask

	// Rule of Three:
	// 1. Keystone Priority
	primaryTask := "Deep Work: Keystone Sprint"
	if len(s.PendingTasks) > 0 {
		primaryTask = strings.TrimSuffix(s.PendingTasks[0], " (missed)")
	}
	suggestions = append(suggestions, TomorrowTask{
		Task:          primaryTask,
		SuggestedTime: "9:30 AM",
		Reason:        "Primary high-leverage focus block",
		Category:      "Keystone Priority",
	})

	// 2. Quick Win
	secondTask := "Quick Win: Triage and organize priority inbox"
	if len(s.PendingTasks) > 1 {
		secondTask = strings.TrimSuffix(s.PendingTasks[1], " (missed)")
	}
	suggestions = append(suggestions, TomorrowTask{
		Task:          secondTask,
		SuggestedTime: "2:00 PM",
		Reason:        "Low-friction momentum booster",
		Category:      "Quick Win",
	})

	// 3. Recovery / Reset
	suggestions = append(suggestions, TomorrowTask{
		Task:          "Evening Digital Detox & Wind-down",
		SuggestedTime: "8:00 PM",
		Reason:        "Protect evening sleep hygiene",
		Category:      "Recovery & Buffer",
	})

	return suggestions
}

func (a *Analyzer) generateAIInsights(ctx context.Context, s *DaySummary, targetDate time.Time) (string, []TomorrowTask, string, string, string, error) {
	hour := targetDate.Hour()
	isEvening := hour >= 17 || hour < 4

	toneDirective := "Tone: Crisp, direct, and analytical. Focus on momentum and high execution velocity. Do not use emojis."
	if isEvening {
		toneDirective = "Tone: Warm, philosophical, and reflective in the Stoic tradition. Help the user achieve mental clarity and restful closure. Do not use emojis."
	}

	prompt := fmt.Sprintf(`<|im_start|>system
You are a perceptive, candid AI productivity coach.
%s

Analyze the user's completed and pending tasks today using the 3-Part Stoic Debrief:
- "win": celebrate their highest-leverage accomplishment.
- "friction": diagnose where attention or execution lagged.
- "keystone": tomorrow's single most critical focus goal.

And provide 3 suggestions following the Rule of Three:
1. Keystone (morning deep work block)
2. Quick Win (afternoon momentum task)
3. Recovery (evening wind-down or buffer)

Output strictly valid JSON with this format:
{
  "win": "...",
  "friction": "...",
  "keystone": "...",
  "analysis": "2-3 sentences synthesizing the debrief with compassionate rebalancing",
  "suggestions": [
    {"task": "...", "suggested_time": "9:30 AM", "reason": "...", "category": "Keystone Priority"},
    {"task": "...", "suggested_time": "2:00 PM", "reason": "...", "category": "Quick Win"},
    {"task": "...", "suggested_time": "8:00 PM", "reason": "...", "category": "Recovery & Buffer"}
  ]
}
<|im_end|>
<|im_start|>user
Completed tasks today (%d): %s
Pending / missed tasks today (%d): %s
Please debrief my day.
<|im_end|>
<|im_start|>assistant
`, toneDirective, s.CompletedCount, strings.Join(s.CompletedTasks, ", "), s.PendingCount+s.MissedCount, strings.Join(s.PendingTasks, ", "))

	a.mu.RLock()
	srv := a.llm
	a.mu.RUnlock()
	if srv == nil {
		return "", nil, "", "", "", fmt.Errorf("analyzer: llm server is nil")
	}

	resp, err := srv.Complete(ctx, prompt, "")
	if err != nil {
		return "", nil, "", "", "", err
	}

	// Parse JSON
	type AIResponse struct {
		Win         string         `json:"win"`
		Friction    string         `json:"friction"`
		Keystone    string         `json:"keystone"`
		Analysis    string         `json:"analysis"`
		Suggestions []TomorrowTask `json:"suggestions"`
	}

	var aiResp AIResponse
	cleanJSON := extractJSONBlock(resp)
	if err := json.Unmarshal([]byte(cleanJSON), &aiResp); err != nil {
		return resp, a.generateHeuristicSuggestions(s), "Consistent execution effort today", "Task friction identified", "Protected morning focus block", nil
	}

	if aiResp.Win == "" {
		aiResp.Win = "Maintained task tracking discipline today"
	}
	if aiResp.Friction == "" {
		aiResp.Friction = "Managing energy across scattered task intervals"
	}
	if aiResp.Keystone == "" {
		aiResp.Keystone = "Protect tomorrow morning for high-priority deep focus"
	}

	return aiResp.Analysis, aiResp.Suggestions, aiResp.Win, aiResp.Friction, aiResp.Keystone, nil
}

func extractJSONBlock(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}
