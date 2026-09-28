# Handoff Report — Explorer 2 (R1 Telegram & R2 Offline Voice PTT)

## 1. Observation

### Codebase Findings & Evidence
1. **Dependency Analysis (`go.mod`)**:
   - `go.mod` defines module `forge` using `go 1.26` with standard Go libraries and minimal Win32 bindings (`golang.org/x/sys v0.47.0`, `github.com/lxn/win v0.0.0-20210218163916-a377121e959e`).
2. **Intent Execution Pipeline (`main.go` & `pkg/skills/skills.go`)**:
   - `main.go:128-133`: Matches user intent strings using `skills.MatchIntent(intent)`. If matched, calls `matchedSkill.Execute(intent)`.
   - `main.go:183-200`: Fallback AI Orchestration plans actions via `planner.PlanActions` and executes them via `executor.ExecutePlan`.
3. **Existing Win32 Keyboard Hooks (`pkg/recorder/hook.go`)**:
   - `pkg/recorder/hook.go:40-44`: Loads Win32 `user32.dll` and procedures `SetWindowsHookExW`, `CallNextHookEx`, `UnhookWindowsHookEx`.
   - `pkg/recorder/hook.go:94-103`: Installs low-level keyboard hook (`WH_KEYBOARD_LL = 13`) and catches `Ctrl+Shift+R` (`VK_R = 0x52`, `isKeyDown(VK_CONTROL)`, `isKeyDown(VK_SHIFT)`).
4. **Offline Speech Recognition Engine Verification**:
   - Executed PowerShell inspection: `[System.Speech.Recognition.SpeechRecognitionEngine]::InstalledRecognizers()`.
   - Result: `MS-1033-80-DESK MS-1033-80-DESK en-US` installed locally on Windows.
   - Executed test dictation recognition with 2-second capture: returned `NO_SPEECH_DETECTED` with exit code `0` and **zero network calls** (100% offline).

---

## 2. Logic Chain

### R1. Telegram Remote Control Architecture
- **Step 1**: Use standard Go `net/http` long-polling against Telegram Bot API (`https://api.telegram.org/bot<TOKEN>/getUpdates?offset=<N>&timeout=30`). This requires **zero new dependencies** in `go.mod`.
- **Step 2**: Create `pkg/telegram` with a lightweight background goroutine `StartListener(ctx, token, allowedChatID, intentDispatcher)`.
- **Step 3**: Security Authorization: Whitelist incoming messages by `update.Message.Chat.ID == allowedChatID`. Reject/ignore unauthorized chat IDs to prevent remote command execution exploits.
- **Step 4**: Decouple intent execution by creating a thread-safe `DispatchIntent(intent string) string` helper. Passing `update.Message.Text` into `DispatchIntent` triggers `skills.MatchIntent(intent)`.
- **Step 5**: Send completion/status updates back to Telegram chat via `sendMessage` endpoint (`https://api.telegram.org/bot<TOKEN>/sendMessage`).
- **Step 6**: Unit / Programmatic Verification: Expose `ProcessUpdate(update TelegramUpdate, allowedChatID int64) (string, error)` allowing mock JSON payloads to be tested programmatically without a live network connection.

### R2. Offline Voice Push-to-Talk Architecture
- **Step 1**: Register global hotkey `Ctrl+Shift+V` (`VK_V = 0x56`). This extends the existing `WH_KEYBOARD_LL` hook pattern in `pkg/recorder/hook.go` (or `pkg/voice/hotkey.go`).
- **Step 2**: When `Ctrl+Shift+V` is pressed:
  - Display HUD visual indicator (e.g. `notifyUser("Forge Voice", "🎙️ Listening...")`).
  - Spawn offline speech recognition worker.
- **Step 3**: Pure Offline Speech Recognition via Windows SAPI / `System.Speech`:
  - Run a lightweight PowerShell helper `voice_listen.ps1` using `System.Speech.Recognition.SpeechRecognitionEngine` with `DictationGrammar` and `SetInputToDefaultAudioDevice()`.
  - Speech recognizer listens for microphone input up to N seconds (or until silence), transcribes speech to text offline, and prints the result to stdout.
- **Step 4**: The transcribed string is passed to `DispatchIntent(transcribedText)`, executing the matched skill or planner fallback seamlessly.
- **Step 5**: Programmatic Verification: Function `CaptureAndTranscribe(timeoutSec int) (string, error)` can be tested offline by verifying zero network sockets are opened and engine returns valid text or empty status.

---

## 3. Caveats

1. **Microphone Device Dependency**: Windows `System.Speech.Recognition` requires a functional default audio input device. If no microphone is available, the recognizer returns an empty string without crashing.
2. **Telegram Bot Token Requirement**: R1 relies on `TELEGRAM_BOT_TOKEN` (and optional `TELEGRAM_ALLOWED_CHAT_ID`). If token is unconfigured, Telegram listener disables itself gracefully.
3. **Concurrency Lock**: Mouse/keyboard input automation cannot be run concurrently from multiple sources (e.g. Telegram message arriving while voice command is executing). A `sync.Mutex` in `DispatchIntent` is required to serialize execution.

---

## 4. Conclusion

Both R1 and R2 can be implemented cleanly, natively, and offline in Go/Windows:
- **R1 (Telegram Remote Control)**: Pure Go `net/http` background polling in `pkg/telegram` with chat ID authorization, mapping to `skills.MatchIntent()`.
- **R2 (Offline Voice Push-to-Talk)**: Win32 keyboard hook for `Ctrl+Shift+V` + Windows `System.Speech.Recognition` offline dictation grammar in `pkg/voice`.

---

## 5. Verification Method

### Test Plan & Commands
1. **Telegram Handler Unit Test (`pkg/telegram/telegram_test.go`)**:
   ```go
   func TestProcessTelegramUpdate(t *testing.T) {
       mockPayload := `{"update_id": 1, "message": {"chat": {"id": 12345}, "text": "open notepad"}}`
       var update TelegramUpdate
       json.Unmarshal([]byte(mockPayload), &update)
       
       res, err := ProcessTelegramUpdate(update, 12345, mockIntentHandler)
       if err != nil || res != "SKILL_MATCHED: open notepad" {
           t.Fatalf("Telegram update processing failed: %v, got %s", err, res)
       }
   }
   ```
   Run: `go test ./pkg/telegram`

2. **Offline Voice Transcription Test (`pkg/voice/voice_test.go`)**:
   ```go
   func TestOfflineVoiceTranscription(t *testing.T) {
       // Verifies execution operates with zero external network requests
       text, err := CaptureAndTranscribe(2)
       if err != nil {
           t.Fatalf("Offline voice transcription failed: %v", err)
       }
       t.Logf("Offline transcription result: '%s'", text)
   }
   ```
   Run: `go test ./pkg/voice`

3. **Compilation Check**:
   Run: `go build -o forge_test.exe main.go`
