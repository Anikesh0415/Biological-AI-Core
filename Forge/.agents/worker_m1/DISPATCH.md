# DISPATCH — Worker M1 (Telegram Remote Control R1)

You are Worker M1 (`teamwork_preview_worker`).
Working directory: `E:/AIF_Project/.agents/worker_m1`
Read `E:/AIF_Project/.agents/ORIGINAL_REQUEST.md` and `E:/AIF_Project/PROJECT.md`.

## Assignment
Implement Requirement R1: Telegram Remote Control in `E:/AIF_Project`.

### Technical Requirements
1. **Centralized Intent Dispatcher**:
   - In `main.go`, implement a thread-safe `DispatchIntent(intent string) (string, error)` function protected by a `sync.Mutex`.
   - `DispatchIntent` invokes `skills.MatchIntent(intent)`. If matched, executes the skill and returns execution status. If unmatched, falls back to `planner.PlanActions` / `handleSummon` flow or returns skill matching error.
2. **Telegram Module (`pkg/telegram/telegram.go`)**:
   - Implement `StartListener(ctx context.Context, botToken string, allowedChatID int64, intentHandler func(string) (string, error))` as a lightweight background Goroutine using standard Go `net/http` long-polling (`/getUpdates?offset=X&timeout=30`).
   - Parse incoming JSON updates (`TelegramUpdate`, `TelegramMessage`, `TelegramChat`).
   - Whitelist authorization: Verify `message.Chat.ID == allowedChatID`. Reject/ignore unauthorized chat IDs.
   - Forward `message.Text` to `intentHandler` (`DispatchIntent`).
   - Send response string back to Telegram chat using `sendMessage` API (`https://api.telegram.org/bot<TOKEN>/sendMessage`).
   - Graceful shutdown when context is cancelled or `botToken` is empty.
3. **Unit Tests (`pkg/telegram/telegram_test.go`)**:
   - Implement `TestProcessTelegramUpdate(t *testing.T)` using mock JSON payloads to test authorization, message parsing, and skill routing programmatically without live network calls.
4. **Compiler Compatibility**:
   - Check `go.mod` and ensure compatibility with local Go compiler (`E:\AIF_Project\go\bin\go.exe test ./pkg/telegram`).

### MANDATORY INTEGRITY WARNING
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

### Required Deliverables
- Code implementation in `pkg/telegram/telegram.go`, `pkg/telegram/telegram_test.go`, and `main.go`.
- Run build (`E:\AIF_Project\go\bin\go.exe build -o forge.exe main.go`) and tests (`E:\AIF_Project\go\bin\go.exe test ./pkg/telegram`).
- Write complete handoff report to `E:/AIF_Project/.agents/worker_m1/handoff.md`.
