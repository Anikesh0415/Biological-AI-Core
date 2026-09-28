# DISPATCH — E2E Test Writer (Milestone M4)

You are the E2E Test Writer (`teamwork_preview_test_writer`).
Working directory: `E:/AIF_Project/.agents/test_writer_m4`
Read `E:/AIF_Project/.agents/ORIGINAL_REQUEST.md` and `E:/AIF_Project/PROJECT.md`.

## Assignment
Design and implement the opaque-box, requirement-driven E2E test suite for Forge OS (R1 Telegram Remote Control, R2 Offline Voice Push-to-Talk, R3 Live Progress HUD Overlay).

### Tasks
1. Create `TEST_INFRA.md` at project root (`E:/AIF_Project/TEST_INFRA.md`) following the standard template (Feature Inventory, Test Architecture, 4-Tier Test Cases).
2. Create test directory `E:/AIF_Project/tests/e2e/`.
3. Implement `tests/e2e/r1_telegram_e2e_test.go`:
   - Uses `httptest.Server` to mock Telegram Bot API (`/getUpdates`, `/sendMessage`).
   - Verifies Telegram update processing, chat ID authorization whitelist, intent matching, and outbound response payload.
4. Implement `tests/e2e/r2_voice_e2e_test.go`:
   - Verifies offline transcription pipeline and asserts zero outbound network requests during voice execution.
5. Implement `tests/e2e/r3_hud_e2e_test.ps1`:
   - Verifies single-instance WPF HUD process management when sequential progress updates (`[1/3]`, `[2/3]`, `[3/3]`) are issued.
6. Create `TEST_READY.md` at project root (`E:/AIF_Project/TEST_READY.md`) summarizing test coverage across Tiers 1-4 once test suite is ready.

### Required Deliverables
- `TEST_INFRA.md`
- `tests/e2e/r1_telegram_e2e_test.go`
- `tests/e2e/r2_voice_e2e_test.go`
- `tests/e2e/r3_hud_e2e_test.ps1`
- `TEST_READY.md`
- Handoff report in `E:/AIF_Project/.agents/test_writer_m4/handoff.md`.
