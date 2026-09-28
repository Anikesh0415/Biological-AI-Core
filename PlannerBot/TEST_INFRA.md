# E2E Test Infra: Planner Bot

## Test Philosophy
- Opaque-box, requirement-driven derived directly from `ORIGINAL_REQUEST.md`.
- Verifies that any natural language reminder request is reliably extracted into strictly valid JSON matching `{"task": string, "time": string}`.
- Methodology: 4-Tier Hierarchical Testing (Category-Partition, Boundary Value Analysis, Pairwise Combinations, Real-World Workload Scenarios).

## Feature Inventory Coverage
| # | Feature | Requirement Source | Tier 1 | Tier 2 | Tier 3 | Tier 4 |
|---|---------|-------------------|:------:|:------:|:------:|:------:|
| F1 | Natural Language to JSON Extraction | ORIGINAL_REQUEST §R1 | 5 | 5 | ✓ | ✓ |
| F2 | Local LLM & GBNF Grammar Enforcement | ORIGINAL_REQUEST §R2 | 5 | 5 | ✓ | ✓ |
| F3 | Automated Model Download & Cache | ORIGINAL_REQUEST §R2 | 2 | 2 | ✓ | ✓ |
| F4 | JSON Unmarshal & Strict Parsing | ORIGINAL_REQUEST Acceptance | 5 | 5 | ✓ | ✓ |

## Test Architecture
- **Location**: `test/e2e_test.go` and `test/adversarial_test.go`
- **Invocation**: `go test -v -timeout 10m ./test/...`
- **Verification Channel**:
  1. `json.Unmarshal([]byte(rawOutput), &target)` MUST succeed with zero errors.
  2. Extracted `target.Task` and `target.Time` MUST be non-empty strings for valid reminder prompts.
  3. Ground truth semantic alignment against benchmark dataset.

## Benchmark Test Suite (Tiers 1-4)

### Tier 1: Feature Coverage (Core Happy Path)
1. `"remind me to call mom at 7pm"` -> Task: `"call mom"`, Time: `"7pm"`
2. `"remind me tomorrow at 5pm to buy milk"` -> Task: `"buy milk"`, Time: `"tomorrow at 5pm"`
3. `"schedule a dentist appointment for next Monday at 10:30am"` -> Task: `"dentist appointment"`, Time: `"next Monday at 10:30am"`
4. `"set an alarm for 6:00am"` -> Task: `"alarm"`, Time: `"6:00am"`
5. `"remind me to submit the quarterly tax report by Friday 5pm"` -> Task: `"submit the quarterly tax report"`, Time: `"Friday 5pm"`

### Tier 2: Boundary & Corner Cases
1. Relative short duration: `"remind me in 15 minutes to turn off the oven"` -> Task: `"turn off the oven"`, Time: `"in 15 minutes"`
2. Noon boundary: `"lunch with David at 12:00pm tomorrow"` -> Task: `"lunch with David"`, Time: `"12:00pm tomorrow"`
3. Midnight boundary: `"server maintenance at 12:00am midnight"` -> Task: `"server maintenance"`, Time: `"12:00am midnight"`
4. Inverted syntax (time first): `"At 8pm tonight, remind me to take medicine"` -> Task: `"take medicine"`, Time: `"8pm tonight"`
5. Punctuation & quotes: `"don't forget: pick up 'Project Plan' package at 3:15pm"` -> Task: `"pick up 'Project Plan' package"`, Time: `"3:15pm"`

### Tier 3: Cross-Feature Combinations
1. Date + Relative Time: `"remind me on October 24th at 4pm to renew vehicle registration"` -> Task: `"renew vehicle registration"`, Time: `"October 24th at 4pm"`
2. Multi-word action + Day offset: `"remind me day after tomorrow at 9am to review pull request 104"` -> Task: `"review pull request 104"`, Time: `"day after tomorrow at 9am"`
3. Casual / Colloquial expression: `"hey planner, gotta grab laundry in 45 mins"` -> Task: `"grab laundry"`, Time: `"in 45 mins"`

### Tier 4: Real-World Application Scenarios
1. Flight check-in: `"check in for flight AA1234 exactly 24 hours before 3pm tomorrow"` -> Task: `"check in for flight AA1234"`, Time: `"24 hours before 3pm tomorrow"`
2. Team sync: `"weekly engineering sync on Thursdays at 2pm"` -> Task: `"weekly engineering sync"`, Time: `"Thursdays at 2pm"`
3. Medicine schedule: `"take antibiotic pill with water at 8:00am and 8:00pm daily"` -> Task: `"take antibiotic pill with water"`, Time: `"8:00am and 8:00pm daily"`

## Coverage Thresholds
- Tier 1: ≥5 test cases
- Tier 2: ≥5 test cases
- Tier 3: ≥3 combination test cases
- Tier 4: ≥3 realistic application scenarios
- **Total Minimum**: ≥16 verified scenarios
