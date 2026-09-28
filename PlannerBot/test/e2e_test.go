package test

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"planner_bot/pkg/extractor"
)

// TestCase defines the structure for acceptance test scenarios across Tiers 1-4.
type TestCase struct {
	ID             string
	Tier           int
	Category       string
	Prompt         string
	ExpectedTask   string
	ExpectedTime   string
	TaskKeywords   []string
	TimeKeywords   []string
	NegativeTokens []string
}

// Global shared engine instance to avoid repeatedly booting llama-server for each test case.
var (
	sharedEngine     *extractor.Engine
	sharedEngineErr  error
	sharedEngineOnce sync.Once
)

// getTestEngine returns a shared extractor.Engine or skips the test if configured.
func getTestEngine(t *testing.T) *extractor.Engine {
	t.Helper()

	// Graceful skip support via environment variable
	if os.Getenv("SKIP_E2E") == "1" || strings.ToLower(os.Getenv("SKIP_E2E")) == "true" {
		t.Skip("Skipping E2E test suite: SKIP_E2E environment variable is set")
	}

	sharedEngineOnce.Do(func() {
		// Allow up to 10 minutes for model/binary acquisition and server warmup on cold starts
		initCtx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		skipDownload := os.Getenv("PLANNER_BOT_SKIP_DOWNLOAD") == "1" ||
			strings.ToLower(os.Getenv("PLANNER_BOT_SKIP_DOWNLOAD")) == "true"

		cfg := extractor.Config{
			ModelPath:      os.Getenv("PLANNER_BOT_MODEL"),
			ServerBinary:   os.Getenv("PLANNER_BOT_SERVER"),
			DownloadIfNone: !skipDownload,
		}

		t.Logf("Initializing Extractor Engine (ModelPath=%q, ServerBinary=%q, DownloadIfNone=%v)...",
			cfg.ModelPath, cfg.ServerBinary, cfg.DownloadIfNone)

		sharedEngine, sharedEngineErr = extractor.NewEngine(initCtx, cfg)
	})

	if sharedEngineErr != nil {
		t.Fatalf("Failed to initialize extractor.Engine: %v", sharedEngineErr)
	}

	if sharedEngine == nil {
		t.Fatalf("Extractor engine is nil after initialization")
	}

	return sharedEngine
}

// TestMain manages the global lifecycle of the LLM server engine during test runs.
func TestMain(m *testing.M) {
	exitCode := m.Run()

	if sharedEngine != nil {
		_ = sharedEngine.Close()
	}

	os.Exit(exitCode)
}

// runTestCase performs opaque-box verification on a single prompt.
func runTestCase(t *testing.T, tc TestCase) {
	t.Helper()

	engine := getTestEngine(t)

	// Context with timeout per prompt execution
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	startTime := time.Now()
	reminder, err := engine.ExtractReminder(ctx, tc.Prompt)
	latency := time.Since(startTime)

	if err != nil {
		t.Fatalf("[%s] ExtractReminder returned an unexpected error for prompt %q: %v", tc.ID, tc.Prompt, err)
	}

	if reminder == nil {
		t.Fatalf("[%s] ExtractReminder returned nil Reminder struct for prompt %q", tc.ID, tc.Prompt)
	}

	t.Logf("[%s] Latency: %v | Prompt: %q | Extracted: Task=%q, Time=%q",
		tc.ID, latency, tc.Prompt, reminder.Task, reminder.Time)

	// --- Verification 1: Strict JSON Parseability (R2 & Acceptance Criteria) ---
	// Confirm that the extracted Reminder serializes to and unmarshals from valid JSON.
	marshaledJSON, err := json.Marshal(reminder)
	if err != nil {
		t.Fatalf("[%s] Failed to marshal Reminder struct to JSON: %v", tc.ID, err)
	}

	var parsedMap map[string]interface{}
	if err := json.Unmarshal(marshaledJSON, &parsedMap); err != nil {
		t.Fatalf("[%s] json.Unmarshal failed on reminder payload: %v | Payload: %s", tc.ID, err, string(marshaledJSON))
	}

	if _, exists := parsedMap["task"]; !exists {
		t.Errorf("[%s] Strict JSON validation failed: 'task' key missing in parsed JSON: %s", tc.ID, string(marshaledJSON))
	}
	if _, exists := parsedMap["time"]; !exists {
		t.Errorf("[%s] Strict JSON validation failed: 'time' key missing in parsed JSON: %s", tc.ID, string(marshaledJSON))
	}

	// Verify direct unmarshal if raw output provider is implemented
	type rawOutputProvider interface {
		RawJSON() string
	}
	if rop, ok := any(reminder).(rawOutputProvider); ok && rop.RawJSON() != "" {
		var rawMap map[string]interface{}
		if err := json.Unmarshal([]byte(rop.RawJSON()), &rawMap); err != nil {
			t.Errorf("[%s] Direct json.Unmarshal on raw LLM output failed: %v | Raw: %s", tc.ID, err, rop.RawJSON())
		}
	}

	// --- Verification 2: Non-empty task and time strings (R1) ---
	taskClean := strings.TrimSpace(reminder.Task)
	timeClean := strings.TrimSpace(reminder.Time)

	if taskClean == "" {
		t.Errorf("[%s] Extracted Task is empty for prompt: %q", tc.ID, tc.Prompt)
	}
	if timeClean == "" {
		t.Errorf("[%s] Extracted Time is empty for prompt: %q", tc.ID, tc.Prompt)
	}

	// --- Verification 3: GBNF Grammar Enforcement & Cleanliness ---
	// Output must NOT leak JSON syntax, delimiters, or Markdown code blocks inside field values.
	forbiddenDelimiters := []string{"```", "{", "}", "\"task\":", "\"time\":", `\n`}
	forbiddenDelimiters = append(forbiddenDelimiters, tc.NegativeTokens...)

	for _, token := range forbiddenDelimiters {
		if strings.Contains(reminder.Task, token) {
			t.Errorf("[%s] Task field contains forbidden delimiter/syntax token %q: %q", tc.ID, token, reminder.Task)
		}
		if strings.Contains(reminder.Time, token) {
			t.Errorf("[%s] Time field contains forbidden delimiter/syntax token %q: %q", tc.ID, token, reminder.Time)
		}
	}

	// --- Verification 4: Semantic Ground Truth Alignment ---
	taskLower := strings.ToLower(taskClean)
	for _, kw := range tc.TaskKeywords {
		if !strings.Contains(taskLower, strings.ToLower(kw)) {
			t.Errorf("[%s] Semantic alignment failure: Task %q missing expected keyword %q (expected: %q)",
				tc.ID, reminder.Task, kw, tc.ExpectedTask)
		}
	}

	timeLower := strings.ToLower(timeClean)
	for _, kw := range tc.TimeKeywords {
		if !strings.Contains(timeLower, strings.ToLower(kw)) {
			t.Errorf("[%s] Semantic alignment failure: Time %q missing expected keyword %q (expected: %q)",
				tc.ID, reminder.Time, kw, tc.ExpectedTime)
		}
	}
}

// =============================================================================
// Tier 1: Feature Coverage (Core Happy Path Prompts)
// =============================================================================

var tier1TestCases = []TestCase{
	{
		ID:           "T1_01_CallMom",
		Tier:         1,
		Category:     "Feature Coverage - Basic Reminder",
		Prompt:       "remind me to call mom at 7pm",
		ExpectedTask: "call mom",
		ExpectedTime: "7pm",
		TaskKeywords: []string{"call", "mom"},
		TimeKeywords: []string{"7", "pm"},
	},
	{
		ID:           "T1_02_TomorrowBuyMilk",
		Tier:         1,
		Category:     "Feature Coverage - Time Before Task",
		Prompt:       "remind me tomorrow at 5pm to buy milk",
		ExpectedTask: "buy milk",
		ExpectedTime: "tomorrow at 5pm",
		TaskKeywords: []string{"buy", "milk"},
		TimeKeywords: []string{"tomorrow", "5"},
	},
	{
		ID:           "T1_03_DentistAppointment",
		Tier:         1,
		Category:     "Feature Coverage - Explicit Schedule Action",
		Prompt:       "schedule a dentist appointment for next Monday at 10:30am",
		ExpectedTask: "dentist appointment",
		ExpectedTime: "next Monday at 10:30am",
		TaskKeywords: []string{"dentist", "appointment"},
		TimeKeywords: []string{"monday", "10:30"},
	},
	{
		ID:           "T1_04_SetAlarm",
		Tier:         1,
		Category:     "Feature Coverage - Alarm Prompt",
		Prompt:       "set an alarm for 6:00am",
		ExpectedTask: "alarm",
		ExpectedTime: "6:00am",
		TaskKeywords: []string{"alarm"},
		TimeKeywords: []string{"6:00"},
	},
	{
		ID:           "T1_05_QuarterlyTaxReport",
		Tier:         1,
		Category:     "Feature Coverage - Deadline by Day/Time",
		Prompt:       "remind me to submit the quarterly tax report by Friday 5pm",
		ExpectedTask: "submit the quarterly tax report",
		ExpectedTime: "Friday 5pm",
		TaskKeywords: []string{"tax", "report"},
		TimeKeywords: []string{"friday", "5pm"},
	},
	{
		ID:           "T1_06_DryCleaningToday",
		Tier:         1,
		Category:     "Feature Coverage - Today with Hour",
		Prompt:       "remind me to pick up dry cleaning today at 6pm",
		ExpectedTask: "pick up dry cleaning",
		ExpectedTime: "today at 6pm",
		TaskKeywords: []string{"dry", "cleaning"},
		TimeKeywords: []string{"today", "6pm"},
	},
}

func TestE2E_Tier1_FeatureCoverage(t *testing.T) {
	for _, tc := range tier1TestCases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

// =============================================================================
// Tier 2: Boundary & Corner Cases
// =============================================================================

var tier2TestCases = []TestCase{
	{
		ID:           "T2_01_RelativeShortDuration",
		Tier:         2,
		Category:     "Boundary - Short Minute Duration",
		Prompt:       "remind me in 15 minutes to turn off the oven",
		ExpectedTask: "turn off the oven",
		ExpectedTime: "in 15 minutes",
		TaskKeywords: []string{"turn off", "oven"},
		TimeKeywords: []string{"15", "minute"},
	},
	{
		ID:           "T2_02_NoonBoundary",
		Tier:         2,
		Category:     "Boundary - Noon 12:00pm",
		Prompt:       "lunch with David at 12:00pm tomorrow",
		ExpectedTask: "lunch with David",
		ExpectedTime: "12:00pm tomorrow",
		TaskKeywords: []string{"lunch", "david"},
		TimeKeywords: []string{"12:00", "tomorrow"},
	},
	{
		ID:           "T2_03_MidnightBoundary",
		Tier:         2,
		Category:     "Boundary - Midnight 12:00am",
		Prompt:       "server maintenance at 12:00am midnight",
		ExpectedTask: "server maintenance",
		ExpectedTime: "12:00am midnight",
		TaskKeywords: []string{"server", "maintenance"},
		TimeKeywords: []string{"12:00"},
	},
	{
		ID:           "T2_04_InvertedSyntaxTimeFirst",
		Tier:         2,
		Category:     "Corner - Inverted Word Order",
		Prompt:       "At 8pm tonight, remind me to take medicine",
		ExpectedTask: "take medicine",
		ExpectedTime: "8pm tonight",
		TaskKeywords: []string{"medicine"},
		TimeKeywords: []string{"8pm", "tonight"},
	},
	{
		ID:           "T2_05_PunctuationAndQuotes",
		Tier:         2,
		Category:     "Corner - Quoted String & Colons",
		Prompt:       "don't forget: pick up 'Project Plan' package at 3:15pm",
		ExpectedTask: "pick up 'Project Plan' package",
		ExpectedTime: "3:15pm",
		TaskKeywords: []string{"project plan"},
		TimeKeywords: []string{"3:15"},
	},
	{
		ID:           "T2_06_RelativeHoursOffset",
		Tier:         2,
		Category:     "Boundary - Relative Hours Offset",
		Prompt:       "remind me in 2 hours to check the laundry",
		ExpectedTask: "check the laundry",
		ExpectedTime: "in 2 hours",
		TaskKeywords: []string{"check", "laundry"},
		TimeKeywords: []string{"2", "hour"},
	},
}

func TestE2E_Tier2_BoundaryAndCornerCases(t *testing.T) {
	for _, tc := range tier2TestCases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

// =============================================================================
// Tier 3: Cross-Feature Combinations
// =============================================================================

var tier3TestCases = []TestCase{
	{
		ID:           "T3_01_DateAndRelativeTime",
		Tier:         3,
		Category:     "Combination - Specific Date + Clock Time",
		Prompt:       "remind me on October 24th at 4pm to renew vehicle registration",
		ExpectedTask: "renew vehicle registration",
		ExpectedTime: "October 24th at 4pm",
		TaskKeywords: []string{"renew", "registration"},
		TimeKeywords: []string{"october", "4pm"},
	},
	{
		ID:           "T3_02_MultiWordActionDayOffset",
		Tier:         3,
		Category:     "Combination - Compound Action + Relative Day",
		Prompt:       "remind me day after tomorrow at 9am to review pull request 104",
		ExpectedTask: "review pull request 104",
		ExpectedTime: "day after tomorrow at 9am",
		TaskKeywords: []string{"review", "pull request"},
		TimeKeywords: []string{"after tomorrow", "9am"},
	},
	{
		ID:           "T3_03_CasualColloquialExpression",
		Tier:         3,
		Category:     "Combination - Colloquial / Slang Syntax",
		Prompt:       "hey planner, gotta grab laundry in 45 mins",
		ExpectedTask: "grab laundry",
		ExpectedTime: "in 45 mins",
		TaskKeywords: []string{"grab", "laundry"},
		TimeKeywords: []string{"45", "min"},
	},
	{
		ID:           "T3_04_SpecificDateMorningTime",
		Tier:         3,
		Category:     "Combination - Future Month + Morning Timestamp",
		Prompt:       "schedule car service on November 15 at 8:30am",
		ExpectedTask: "car service",
		ExpectedTime: "November 15 at 8:30am",
		TaskKeywords: []string{"car", "service"},
		TimeKeywords: []string{"november 15", "8:30"},
	},
}

func TestE2E_Tier3_CrossFeatureCombinations(t *testing.T) {
	for _, tc := range tier3TestCases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

// =============================================================================
// Tier 4: Real-World Scenarios
// =============================================================================

var tier4TestCases = []TestCase{
	{
		ID:           "T4_01_FlightCheckin",
		Tier:         4,
		Category:     "Real World - Airline Check-in Window",
		Prompt:       "check in for flight AA1234 exactly 24 hours before 3pm tomorrow",
		ExpectedTask: "check in for flight AA1234",
		ExpectedTime: "24 hours before 3pm tomorrow",
		TaskKeywords: []string{"check in", "flight", "AA1234"},
		TimeKeywords: []string{"tomorrow"},
	},
	{
		ID:           "T4_02_TeamSync",
		Tier:         4,
		Category:     "Real World - Recurring Weekly Team Sync",
		Prompt:       "weekly engineering sync on Thursdays at 2pm",
		ExpectedTask: "weekly engineering sync",
		ExpectedTime: "Thursdays at 2pm",
		TaskKeywords: []string{"engineering", "sync"},
		TimeKeywords: []string{"thursday", "2pm"},
	},
	{
		ID:           "T4_03_MedicationSchedule",
		Tier:         4,
		Category:     "Real World - Complex Daily Medication Regimen",
		Prompt:       "take antibiotic pill with water at 8:00am and 8:00pm daily",
		ExpectedTask: "take antibiotic pill with water",
		ExpectedTime: "8:00am and 8:00pm daily",
		TaskKeywords: []string{"antibiotic", "pill"},
		TimeKeywords: []string{"8:00"},
	},
	{
		ID:           "T4_04_ClientMeetingWithPrep",
		Tier:         4,
		Category:     "Real World - Meeting with Action Item",
		Prompt:       "prepare slides and meet client on Wednesday at 11am",
		ExpectedTask: "prepare slides and meet client",
		ExpectedTime: "Wednesday at 11am",
		TaskKeywords: []string{"slides", "client"},
		TimeKeywords: []string{"wednesday", "11am"},
	},
}

func TestE2E_Tier4_RealWorldScenarios(t *testing.T) {
	for _, tc := range tier4TestCases {
		tc := tc
		t.Run(tc.ID, func(t *testing.T) {
			runTestCase(t, tc)
		})
	}
}

// =============================================================================
// Consolidated Benchmark Test Across All Tiers
// =============================================================================

func TestE2E_AllTiers_ConsolidatedSummary(t *testing.T) {
	allTestCases := make([]TestCase, 0, len(tier1TestCases)+len(tier2TestCases)+len(tier3TestCases)+len(tier4TestCases))
	allTestCases = append(allTestCases, tier1TestCases...)
	allTestCases = append(allTestCases, tier2TestCases...)
	allTestCases = append(allTestCases, tier3TestCases...)
	allTestCases = append(allTestCases, tier4TestCases...)

	t.Logf("Running Consolidated Benchmark across %d scenarios across Tiers 1-4...", len(allTestCases))

	var totalLatency time.Duration
	passCount := 0

	for _, tc := range allTestCases {
		tc := tc
		start := time.Now()
		success := t.Run(tc.ID, func(subT *testing.T) {
			runTestCase(subT, tc)
		})
		latency := time.Since(start)
		totalLatency += latency

		if success {
			passCount++
		}
	}

	avgLatency := time.Duration(0)
	if len(allTestCases) > 0 {
		avgLatency = totalLatency / time.Duration(len(allTestCases))
	}

	t.Logf("=== Consolidated E2E Summary ===")
	t.Logf("Total Scenarios: %d", len(allTestCases))
	t.Logf("Passed:          %d", passCount)
	t.Logf("Total Time:      %v", totalLatency)
	t.Logf("Average Latency: %v/scenario", avgLatency)
}
