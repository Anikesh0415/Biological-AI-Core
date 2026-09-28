# Codebase Investigation Handoff Report — Explorer 1

## 1. Observation

Direct observations from examining the codebase at `E:/AIF_Project`:

### Codebase Architecture & Key Files
- **`main.go`**: Entry point for Forge OS.
  - Line 21-37: `main()` initializes `db.InitBrain()`, `recorder.StartHooks()`, `skills.InitBuiltinSkills()`, `skills.LoadLearnedSkills()`, and runs `handleSummon()` in a loop.
  - Line 39-108: `handleSummon()` writes and runs `input.ps1` (WPF text box prompt) to capture user intent.
  - Line 128: `matchedSkill := skills.MatchIntent(intent)` calls the skill routing engine. If matched, `matchedSkill.Execute(intent)` is run.
  - Line 137-149: Rule-based fallback for `"open <app>"` intents under 3 words.
  - Line 157-229: 15-step AI Orchestration loop using `vision.CaptureAndAnalyze()`, `uia.DumpUI()`, `planner.PlanActions()`, `checkSafeguards()`, and `executor.ExecutePlan()`.
  - Line 283-300: `notifyUser(title, message)` writes a temporary `notify.ps1` script that creates a `System.Windows.Forms.NotifyIcon` balloon tip and runs `powershell -ExecutionPolicy Bypass -File notify.ps1` asynchronously using `cmd.Start()`.

- **`pkg/skills/`**:
  - `skills.go`: Defines `Skill` interface (`Match`, `Execute`, `Name`), global `Registry []Skill`, and routing function `MatchIntent(intent string) Skill` (lines 24-52). Includes `FuzzyMatchWithScore` (lines 142-206) and `ExtractVariables` template parser (lines 208-293). `LoadLearnedSkills()` reads JSON definitions from `skills_db/*.json`.
  - `builtin.go`: `InitBuiltinSkills()` (lines 10-94) populates `Registry` with default app openers (`notepad`, `spotify`, etc.) and parameterized macros (`send {message} to {contact} on whatsapp`, `play {song} on spotify`, `browse {site}`). Defines `UniversalSearchSkill` for `"search <query> on <site>"`.
  - `browser_search.go`: Implements `BrowserSearchSkill` for Google / generic browser searches.

- **`pkg/planner/json_llm.go`**:
  - `PlanActions(intent, visionContext, uiaContext, history)` (lines 15-109): Formats ChatML prompt and invokes `E:\AIF_Project\llama.cpp\build\bin\Release\llama-cli.exe` with model `E:\AIF_Project\models\qwen2.5-0.5b-instruct-q4_k_m.gguf` on `temp_prompt.txt` to parse JSON action objects.

- **`pkg/executor/win32.go`**:
  - `ExecutePlan(actions []Action)` (lines 27-63): Translates structured actions (`move`, `click`, `click_element`, `type`, `key`, `sleep`, `browser_navigate`, `browser_click_dom`, `browser_type_dom`) to Windows `SendInput` API calls.

- **`pkg/recorder/hook.go`**:
  - `StartHooks()` (lines 182-253): Registers low-level Windows keyboard (`WH_KEYBOARD_LL`) and mouse (`WH_MOUSE_LL`) hooks via `SetWindowsHookExW`.
  - Line 99: Keyboard callback checks `kbd.VkCode == VK_R && isKeyDown(VK_CONTROL) && isKeyDown(VK_SHIFT)` (`Ctrl+Shift+R`) to toggle macro recording and save to `skills_db/learned_<timestamp>.json`.
  - Line 20-37: Duplicate definition of `notifyUser(title, message)`.

- **`pkg/uia/`**:
  - `uia.go`: `DumpUI()` (lines 53-90) compiles C# code `uia_dumper.cs` via `csc.exe` on-the-fly (if `uia_dumper.exe` missing) using `System.Windows.Automation` to export visible element bounds and names. `GetElementAtPoint(x, y)` (lines 113-142) queries element at mouse point.
  - `watcher.go`: `WaitForElement(query, timeoutMs)` (lines 93-129) polls UIA output with exact and fuzzy matching.

- **`pkg/vision/moondream.go`**:
  - `CaptureAndAnalyze()` (lines 14-58): Captures primary display screenshot via `github.com/kbinani/screenshot`, encodes to PNG, and invokes `llama-mtmd-cli.exe` with Moondream models (`moondream2-text-model-f16.gguf` / `moondream2-mmproj-f16.gguf`).

- **`pkg/db/brain.go`**:
  - `InitBrain()` (lines 13-32): Opens local SQLite database `brain.db` with table `preferences (key TEXT PRIMARY KEY, value TEXT)`.

- **`pkg/browser/cdp.go`**:
  - Chrome DevTools Protocol client using `github.com/chromedp/chromedp` to connect to Chrome remote debugging port 9222.

- **UI Scripts & Data**:
  - `notify.ps1`: 12-line PowerShell balloon notification script.
  - `skills_db/`: Directory containing 37 learned skill JSON files (e.g. `learned_ask_antigravity.json`, `learned_close_app.json`).

### Tool Commands & Test Environment Observations
- Command `$env:GOTOOLCHAIN="local"; E:\AIF_Project\go\bin\go.exe test ./...` returned:
  `go: go.mod requires go >= 1.26 (running go 1.22.5; GOTOOLCHAIN=local)`
- Local Go compiler bundled at `E:\AIF_Project\go\bin\go.exe` is **Go 1.22.5** (detected via `go version`).
- `go.mod` line 3 specifies `go 1.26`. Changing line 3 to `go 1.22` allows building with local Go binary without external network downloads.
- Standalone test script present at `test_planner.go` tests 59 iterations of `planner.PlanActions`.

---

## 2. Logic Chain

1. **Telegram Remote Control (R1) Integration**:
   - *Observation*: `main.go` initializes subsystem components in `main()` (lines 29-32) and processes user intents through `skills.MatchIntent(intent)` (line 128) or `planner.PlanActions(...)` (line 183).
   - *Reasoning*: A background Goroutine can be spawned inside `main()` (or via a `pkg/remote` package initialization) that long-polls the Telegram Bot API (`https://api.telegram.org/bot<TOKEN>/getUpdates`) or uses a lightweight Telegram library. Received text messages can be fed directly to `skills.MatchIntent(intent)` or `handleSummon()`'s intent pipeline. Results can be formatted and sent back as Telegram chat responses.

2. **Offline Voice Push-to-Talk (R2) Integration**:
   - *Observation*: `pkg/recorder/hook.go` (lines 40-137) installs a global Windows low-level keyboard hook `WH_KEYBOARD_LL` via `SetWindowsHookExW`. It currently detects `Ctrl+Shift+R` (`VK_R` with `VK_CONTROL` and `VK_SHIFT`).
   - *Reasoning*: The existing hook thread provides the exact infrastructure needed for global hotkeys. Adding a check for `kbd.VkCode == 0x56` (`VK_V`) when `Ctrl` and `Shift` are held (`Ctrl+Shift+V`) will trigger Push-to-Talk. Offline transcription can be executed using Windows SAPI (`System.Speech.Recognition.SpeechRecognitionEngine` via C#/PowerShell or SAPI COM interfaces in Go) without requiring network connectivity. The transcribed text string is then dispatched to `skills.MatchIntent(transcribedText)`.

3. **Live Progress HUD Overlay (R3) Integration**:
   - *Observation*: `notifyUser` in `main.go` (line 283) and `pkg/recorder/hook.go` (line 20) writes `notify.ps1` and runs `powershell -ExecutionPolicy Bypass -File notify.ps1` to display Windows balloon tips. The script disposes the notification icon after 3 seconds.
   - *Reasoning*: Balloon notifications are ephemeral, non-customizable, and do not support multi-line stateful progress updates (e.g. `[1/3] Step 1`). Updating `notify.ps1` to create a persistent translucent WPF window (similar to `input.ps1` in `main.go` lines 43-65) that accepts piped progress strings or IPC messages will allow real-time progress cues (`[1/3] Step 1`, `[2/3] Step 2`) without window flashing or spawning disjoint process instances.

4. **Build and Test Verification**:
   - *Observation*: `go.mod` declares `go 1.26`, while local binary is Go 1.22.5 (`E:\AIF_Project\go\bin\go.exe`).
   - *Reasoning*: Updating `go.mod` to `go 1.22` (or ensuring `GOTOOLCHAIN` compatibility) allows compilation with the bundled Go compiler via `go build -o forge.exe main.go`. Unit test files (`*_test.go`) can be added to packages for programmatic acceptance verification.

---

## 3. Caveats

- **No Existing `*_test.go` Files in `pkg/`**: Current tests exist as standalone Go scripts (`test_planner.go`, `test_yt.go`) in the repository root rather than standard Go package test files (`pkg/skills/skills_test.go`).
- **Telegram Bot Token Requirement**: Telegram integration requires a valid Bot API Token provided via environment variable or SQLite `preferences` table in `brain.db`.
- **Windows SAPI Language Pack Assumption**: Offline voice transcription using Windows SAPI assumes Windows Speech Recognition components / language packs (English) are installed on the host OS.
- **WPF Interop Overhead**: `notify.ps1` currently spawns a new PowerShell host process per notification call. Stateful progress updates will require either a long-running background PowerShell process reading stdin/named pipe or native WPF/Win32 overlay window.

---

## 4. Conclusion

The Forge OS codebase is well-structured in Go with modular packages (`pkg/skills`, `pkg/planner`, `pkg/executor`, `pkg/recorder`, `pkg/uia`, `pkg/vision`, `pkg/db`).

- **R1 Telegram Control** can be added as a concurrent goroutine (`pkg/telegram` or `pkg/remote`) connecting to `skills.MatchIntent(text)`.
- **R2 Offline Voice PTT** can be hooked into `pkg/recorder/hook.go` under hotkey `Ctrl+Shift+V` calling Windows SAPI speech recognition entirely offline.
- **R3 Live Progress HUD Overlay** can replace `notify.ps1` with a stateful WPF progress window that receives sequential step updates (`[1/3] Step 1`, `[2/3] Step 2`).

All component interface points are clean and accessible.

---

## 5. Verification Method

### 1. Build Verification
Run the following build command using the bundled Go compiler:
```powershell
E:\AIF_Project\go\bin\go.exe build -v -o forge.exe main.go
```
*Expected Result*: Binary `forge.exe` compiles without errors.

### 2. Package Unit Testing
Run package unit tests:
```powershell
E:\AIF_Project\go\bin\go.exe test ./...
```
*Expected Result*: All package tests execute and pass (`ok`).

### 3. Planner Integration Testing
Run the existing planner test harness:
```powershell
E:\AIF_Project\go\bin\go.exe run test_planner.go
```
*Expected Result*: Generates 59 mock plan test iterations and reports completion score.

### 4. Invalidation Conditions
- Compilation failure when running `go build`.
- Inability to bind `Ctrl+Shift+V` in `pkg/recorder/hook.go`.
- Inability to route string payload from Telegram handler or Voice PTT to `skills.MatchIntent()`.
