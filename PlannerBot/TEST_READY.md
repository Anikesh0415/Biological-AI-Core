# TEST_READY: E2E Acceptance Test Suite

**Published**: 2026-09-13  
**Status**: OPERATIONAL & READY FOR IMPLEMENTATION INTEGRATION  
**Test Suite Path**: `test/e2e_test.go`  
**Test Harness Package**: `test`  
**Target Public API**: `planner_bot/pkg/extractor` (`NewEngine`, `ExtractReminder`, `Close`)

---

## 1. Overview & Verification Philosophy

The E2E Acceptance Test Suite implements opaque-box, requirement-driven verification derived directly from `ORIGINAL_REQUEST.md` and `TEST_INFRA.md`. It validates that natural language reminder requests are reliably extracted into strictly valid JSON payloads (`{"task": string, "time": string}`) using `llama.cpp` with GBNF grammar constraints.

The suite satisfies all 4 verification channels:
1. **Strict JSON Parseability**: Guaranteed parseability via `json.Unmarshal` with zero syntax errors.
2. **Non-Empty String Guarantees**: Extracted `Task` and `Time` are confirmed non-empty.
3. **GBNF Grammar Cleanliness**: Ensures no raw JSON syntax (`{`, `}`, `"task":`, `"time":`), markdown blocks (` ``` `), or escape artifacts leak into field values.
4. **Semantic Ground Truth Alignment**: Validates domain keywords against benchmark test cases across all categories.

---

## 2. Test Execution Commands

### 2.1 Standard Automated Execution (Default with Auto-Download)
Runs the complete test suite. The engine will automatically download the test model (`Qwen2.5-0.5B-Instruct-Q4_K_M.gguf`) and `llama-server.exe` into cache if not already present:
```powershell
go test -v -timeout 15m ./test/...
```

### 2.2 Execution by Tier
Run individual tiers selectively:
```powershell
# Tier 1: Core Feature Coverage (Happy Path)
go test -v -run TestE2E_Tier1 ./test/...

# Tier 2: Boundary & Corner Cases
go test -v -run TestE2E_Tier2 ./test/...

# Tier 3: Cross-Feature Combinations
go test -v -run TestE2E_Tier3 ./test/...

# Tier 4: Real-World Scenarios
go test -v -run TestE2E_Tier4 ./test/...

# Consolidated Benchmark across all tiers with latency telemetry
go test -v -run TestE2E_AllTiers_ConsolidatedSummary ./test/...
```

### 2.3 Custom Environment Configuration
Point the test suite to existing local models or server binaries to bypass downloading:
```powershell
$env:PLANNER_BOT_MODEL = "C:\path\to\qwen2.5-0.5b-instruct-q4_k_m.gguf"
$env:PLANNER_BOT_SERVER = "C:\path\to\llama-server.exe"
go test -v -timeout 10m ./test/...
```

### 2.4 Graceful Skip Mode
Skip E2E execution during lightweight unit-test phases:
```powershell
$env:SKIP_E2E = "1"
go test -v ./test/...
```

---

## 3. Coverage Checklist & Verification Matrix

The test suite covers **20 distinct scenarios** across 4 tiers (exceeding the minimum requirement of 16):

| Tier | Test ID | Category | Natural Language Prompt | Expected Task | Expected Time |
| :---: | :--- | :--- | :--- | :--- | :--- |
| **Tier 1** | `T1_01_CallMom` | Basic Reminder | `"remind me to call mom at 7pm"` | `call mom` | `7pm` |
| **Tier 1** | `T1_02_TomorrowBuyMilk` | Time Before Task | `"remind me tomorrow at 5pm to buy milk"` | `buy milk` | `tomorrow at 5pm` |
| **Tier 1** | `T1_03_DentistAppointment` | Explicit Schedule Action | `"schedule a dentist appointment for next Monday at 10:30am"` | `dentist appointment` | `next Monday at 10:30am` |
| **Tier 1** | `T1_04_SetAlarm` | Alarm Prompt | `"set an alarm for 6:00am"` | `alarm` | `6:00am` |
| **Tier 1** | `T1_05_QuarterlyTaxReport` | Deadline by Day/Time | `"remind me to submit the quarterly tax report by Friday 5pm"` | `submit the quarterly tax report` | `Friday 5pm` |
| **Tier 1** | `T1_06_DryCleaningToday` | Today with Hour | `"remind me to pick up dry cleaning today at 6pm"` | `pick up dry cleaning` | `today at 6pm` |
| **Tier 2** | `T2_01_RelativeShortDuration` | Relative Duration | `"remind me in 15 minutes to turn off the oven"` | `turn off the oven` | `in 15 minutes` |
| **Tier 2** | `T2_02_NoonBoundary` | Noon Boundary | `"lunch with David at 12:00pm tomorrow"` | `lunch with David` | `12:00pm tomorrow` |
| **Tier 2** | `T2_03_MidnightBoundary` | Midnight Boundary | `"server maintenance at 12:00am midnight"` | `server maintenance` | `12:00am midnight` |
| **Tier 2** | `T2_04_InvertedSyntaxTimeFirst` | Inverted Word Order | `"At 8pm tonight, remind me to take medicine"` | `take medicine` | `8pm tonight` |
| **Tier 2** | `T2_05_PunctuationAndQuotes` | Quotes & Punctuation | `"don't forget: pick up 'Project Plan' package at 3:15pm"` | `pick up 'Project Plan' package` | `3:15pm` |
| **Tier 2** | `T2_06_RelativeHoursOffset` | Relative Hours | `"remind me in 2 hours to check the laundry"` | `check the laundry` | `in 2 hours` |
| **Tier 3** | `T3_01_DateAndRelativeTime` | Date + Relative Time | `"remind me on October 24th at 4pm to renew vehicle registration"` | `renew vehicle registration` | `October 24th at 4pm` |
| **Tier 3** | `T3_02_MultiWordActionDayOffset` | Compound Action + Offset | `"remind me day after tomorrow at 9am to review pull request 104"` | `review pull request 104` | `day after tomorrow at 9am` |
| **Tier 3** | `T3_03_CasualColloquialExpression`| Colloquial / Slang | `"hey planner, gotta grab laundry in 45 mins"` | `grab laundry` | `in 45 mins` |
| **Tier 3** | `T3_04_SpecificDateMorningTime` | Date + Morning Time | `"schedule car service on November 15 at 8:30am"` | `car service` | `November 15 at 8:30am` |
| **Tier 4** | `T4_01_FlightCheckin` | Airline Check-in Window | `"check in for flight AA1234 exactly 24 hours before 3pm tomorrow"` | `check in for flight AA1234` | `24 hours before 3pm tomorrow` |
| **Tier 4** | `T4_02_TeamSync` | Recurring Team Sync | `"weekly engineering sync on Thursdays at 2pm"` | `weekly engineering sync` | `Thursdays at 2pm` |
| **Tier 4** | `T4_03_MedicationSchedule` | Complex Regimen | `"take antibiotic pill with water at 8:00am and 8:00pm daily"` | `take antibiotic pill with water` | `8:00am and 8:00pm daily` |
| **Tier 4** | `T4_04_ClientMeetingWithPrep` | Meeting with Action Item| `"prepare slides and meet client on Wednesday at 11am"` | `prepare slides and meet client` | `Wednesday at 11am` |

---

## 4. Integration Contract Confirmation

The test suite conforms to the opaque-box contract specified in `PROJECT.md § Interface Contracts`:

```go
package extractor

type Reminder struct {
    Task string `json:"task"`
    Time string `json:"time"`
}

type Config struct {
    ModelPath      string
    ServerBinary   string
    DownloadIfNone bool
}

type Engine struct { ... }

func NewEngine(ctx context.Context, cfg Config) (*Engine, error)
func (e *Engine) Close() error
func (e *Engine) ExtractReminder(ctx context.Context, input string) (*Reminder, error)
```

The test runner will execute as soon as `pkg/extractor` is implemented by the Milestone 4 worker.
