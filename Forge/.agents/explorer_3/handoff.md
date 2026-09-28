# Handoff Report — Explorer 3: Live Progress HUD Overlay (R3) & E2E Testing Infrastructure

**Author**: Explorer 3 (`teamwork_preview_explorer`)  
**Working Directory**: `E:/AIF_Project/.agents/explorer_3`  
**Target Repository**: `E:/AIF_Project`  
**Date**: 2026-08-10  

---

## 1. Observation

### 1.1 Existing Notification System Inspection
Direct inspection of `E:/AIF_Project/notify.ps1` (lines 1–12):
```powershell
Add-Type -AssemblyName System.Windows.Forms
$balloon = New-Object System.Windows.Forms.NotifyIcon
$balloon.Icon = [System.Drawing.SystemIcons]::Information
$balloon.BalloonTipIcon = "Info"
$balloon.BalloonTipText = "Failed to plan action."
$balloon.BalloonTipTitle = "Forge Error"
$balloon.Visible = $true
$balloon.ShowBalloonTip(2000)
Start-Sleep -Seconds 3
$balloon.Dispose()
```

Direct inspection of `E:/AIF_Project/main.go` (lines 283–300):
```go
func notifyUser(title, message string) {
	ps1 := fmt.Sprintf(`
Add-Type -AssemblyName System.Windows.Forms
$balloon = New-Object System.Windows.Forms.NotifyIcon
...
`, message, title)
	os.WriteFile("notify.ps1", []byte(ps1), 0644)
	cmd := exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", "notify.ps1")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Start() // run asynchronously
}
```

Direct inspection of `E:/AIF_Project/pkg/recorder/hook.go` (lines 20–37):
Contains duplicate inline generation of `notify.ps1` using System.Windows.Forms.NotifyIcon.

Direct inspection of `E:/AIF_Project/input.ps1` (lines 2–26):
Demonstrates existing WPF XAML UI design patterns in Forge OS (`Add-Type -AssemblyName PresentationFramework`, `[Windows.Markup.XamlReader]::Load`, `AllowsTransparency="True"`, `Topmost="True"`).

### 1.2 Identified Limitations of Current `notify.ps1`
1. **Forms NotifyIcon Toast vs WPF Overlay**: `NotifyIcon` generates standard Windows OS balloon toasts. It cannot act as a custom WPF HUD overlay on top of screen content.
2. **Disjoint Spawning**: Each call to `notifyUser()` overwrites `notify.ps1` and executes a new `powershell.exe` process (`cmd.Start()`). Sequential progress updates (e.g. `[1/3] Step 1`, `[2/3] Step 2`) create multiple disjoint notification popups in Windows Action Center instead of updating a single on-screen HUD window.
3. **No Stateful Progress**: No progress bar, step tracking, or multi-line state retention between process invocations.

---

## 2. Logic Chain

### 2.1 Refactoring R3: Single-Instance WPF Live Progress HUD Overlay
To satisfy R3 ("Update existing notify.ps1 PowerShell WPF script to handle multi-line or stateful progress updates... without crashing or spawning multiple disjoint windows"):

1. **WPF XAML Overlay UI**:
   - `notify.ps1` will be converted to a modern WPF XAML window using `PresentationFramework` (matching `input.ps1` style).
   - UI layout: Semi-transparent dark background (`#E6181825`), top-right screen positioning (`Topmost="True"`), title label (`TextBlock`), multi-line status message (`TextBlock` with `TextWrapping="Wrap"`), and dynamic progress bar (`ProgressBar`).

2. **Single-Instance Enforcement via NamedPipe IPC**:
   - IPC Pipe Name: `\\.\pipe\ForgeHUD_Pipe`.
   - **Invocation Logic**:
     - When `notify.ps1 -Title "..." -Message "..." -Step X -Total Y` runs:
     - **Client Mode Check**: The script attempts to connect to `\\.\pipe\ForgeHUD_Pipe` with a 200ms timeout.
     - **If Pipe Exists (HUD Window Already Active)**: The script converts parameters to JSON (`{"Title":..., "Message":..., "Step":..., "Total":..., "Close":...}`), writes to the pipe, flushes, and **exits immediately (<50ms runtime)**. No second window is created!
     - **If Pipe Connection Fails (No Active HUD)**: The script initializes the WPF Window and starts a `NamedPipeServerStream` background thread.
     - The background thread listens for incoming JSON payloads and calls `$window.Dispatcher.Invoke(...)` to update `TextBlock.Text` and `ProgressBar.Value` smoothly in place.

3. **Go Integration (`main.go` & `pkg/recorder/hook.go`)**:
   - Go's `notifyUser(title, message)` function can pass parameters directly:
     `exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", "notify.ps1", "-Title", title, "-Message", message)`
   - Go can also pass optional `-Step` and `-Total` arguments when executing orchestrator loops.

---

### 2.2 E2E Testing Infrastructure Requirements & Plan

To validate R1, R2, and R3 programmatically and end-to-end:

#### A. R1: Telegram Remote Control E2E Testing
- **Test File**: `tests/e2e/r1_telegram_e2e_test.go`
- **Mock Infrastructure**:
  - `httptest.Server` mocking Telegram Bot API endpoints (`/bot<token>/getUpdates`, `/bot<token>/sendMessage`).
- **Verification Flow**:
  1. Local mock server pushes JSON Update payload (`{"message": {"text": "open chrome"}}`).
  2. Goroutine handler fetches message -> calls `planner.PlanActions`.
  3. Assert `planner.PlanActions` receives text and generates expected `executor.Action` list.
  4. Assert mock server receives outbound response payload from `/sendMessage`.
- **Hermetic Guarantee**: Runs 100% offline using `httptest`.

#### B. R2: Offline Voice Push-to-Talk E2E Testing
- **Test File**: `tests/e2e/r2_voice_e2e_test.go`
- **Mock Infrastructure**:
  - Pre-recorded / synthetic PCM WAV audio buffer.
  - Network transport interceptor (`http.DefaultTransport` override) to assert zero outbound network requests during transcription.
- **Verification Flow**:
  1. Invoke voice recognition engine with test audio buffer.
  2. Assert output transcription text string.
  3. Assert zero network traffic generated (offline verification).
  4. Feed transcription into skill matcher and assert action generation.
  5. Verify hotkey state toggle (`Ctrl+Shift+V` listener state machine).

#### C. R3: Live Progress HUD Overlay E2E Testing
- **Test File**: `tests/scripts/test_hud_single_instance.ps1` & `tests/e2e/r3_hud_e2e_test.go`
- **Verification Flow**:
  1. Trigger Step 1: `powershell -File notify.ps1 -Title "Test" -Message "[1/3] Step 1"`
  2. Query `Get-Process powershell` / WPF windows -> assert **exactly 1** HUD process created.
  3. Trigger Step 2: `powershell -File notify.ps1 -Title "Test" -Message "[2/3] Step 2"`
  4. Assert process count **remains 1** (client connected to pipe and exited).
  5. Trigger Step 3: `powershell -File notify.ps1 -Title "Test" -Message "[3/3] Done" -Close`
  6. Assert clean exit and process cleanup within 3 seconds.

---

## 3. Caveats

1. **GUI Session Requirement**: WPF window rendering requires an active Windows interactive desktop session. In headless CI/CD build environments, the WPF UI render step can be bypassed or mocked while testing NamedPipe IPC message passing.
2. **NamedPipe Permissions**: NamedPipes operate under standard Windows user security contexts. Forge OS and `notify.ps1` run under the same user account.

---

## 4. Conclusion

- `notify.ps1` currently relies on outdated `System.Windows.Forms.NotifyIcon` balloon tips, causing disjoint toast popups on sequential calls.
- Upgrading `notify.ps1` to a WPF XAML overlay with NamedPipe IPC client/server logic completely resolves multi-line progress updates and single-instance window management for R3.
- E2E testing framework strategy provides 100% programmatic coverage for R1 (httptest mock Telegram), R2 (offline audio buffer + network isolation assertion), and R3 (PowerShell process count single-instance verification).

---

## 5. Verification Method

### 5.1 Programmatic Verification Commands
1. **PowerShell HUD Single-Instance Test**:
   ```powershell
   powershell -ExecutionPolicy Bypass -File notify.ps1 -Title "Forge E2E" -Message "[1/3] Step 1: Initializing"
   powershell -ExecutionPolicy Bypass -File notify.ps1 -Title "Forge E2E" -Message "[2/3] Step 2: Running Executor"
   powershell -ExecutionPolicy Bypass -File notify.ps1 -Title "Forge E2E" -Message "[3/3] Complete!" -Close
   ```
   *Expected Result*: A single HUD overlay appears in top-right, updates text and progress bar live in place without spawning additional windows, and closes after step 3.

2. **Go Test Infrastructure Suite**:
   ```powershell
   go test -v ./pkg/...
   ```
   *Note*: The root folder contains multiple standalone runner scripts with `main()` functions (`test_planner.go`, `test_run.go`, `test_yt.go`), so package tests are scoped under `./pkg/...` or `./tests/...`.
   *Expected Result*: All package unit and E2E integration tests pass cleanly.

### 5.2 Files to Inspect
- `E:/AIF_Project/notify.ps1`
- `E:/AIF_Project/main.go`
- `E:/AIF_Project/pkg/recorder/hook.go`
- `E:/AIF_Project/.agents/explorer_3/handoff.md`
