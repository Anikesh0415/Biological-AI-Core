# Project: Forge OS Feature Implementation

## Architecture
- Core: Go 1.22 desktop automation agent (`main.go`, `pkg/skills`, `pkg/planner`, `pkg/executor`).
- R1 Telegram Remote Control: `pkg/telegram` background Goroutine using standard `net/http` polling with authorized chat ID check and dispatching to centralized `DispatchIntent()`.
- R2 Offline Voice Push-to-Talk: Win32 keyboard hook (`Ctrl+Shift+V`) in `pkg/voice` / `pkg/recorder` + PowerShell Windows SAPI (`System.Speech.Recognition`) offline dictation worker dispatching to `DispatchIntent()`.
- R3 Live Progress HUD Overlay: Single-instance WPF XAML window + NamedPipe IPC server in `notify.ps1`, updated by `notifyUser()` in `main.go` and `pkg/recorder`.
- Centralized intent routing: `DispatchIntent(intent string) (string, error)` with `sync.Mutex` lock to serialize skill execution across UI, Telegram, and Voice.

## Feature Inventory
Every feature from the Survey phase is enumerated and assigned below:
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| 1 | Telegram Remote Listener & Security | Goroutine polling Telegram updates, verifying authorized Chat ID | M1 | Survey (R1) |
| 2 | Telegram Skill Dispatcher & Response | Forward text to `DispatchIntent` and send response back via Telegram API | M1 | Survey (R1) |
| 3 | Voice Hotkey Listener | Global hook for `Ctrl+Shift+V` hotkey in `pkg/voice` | M2 | Survey (R2) |
| 4 | Offline Speech Transcription | PowerShell Windows SAPI dictation recognition worker (100% network isolated) | M2 | Survey (R2) |
| 5 | Voice Intent Dispatcher | Send transcribed speech text to `DispatchIntent` | M2 | Survey (R2) |
| 6 | Single-Instance WPF HUD Overlay | WPF XAML overlay window in `notify.ps1` with NamedPipe IPC client/server | M3 | Survey (R3) |
| 7 | Multi-line & Stateful Progress UI | Update progress bar and text live in place for `[1/3] Step 1`, `[2/3] Step 2` | M3 | Survey (R3) |
| 8 | Dual-Track E2E Test Suite (Tiers 1-4) | Programmatic tests for R1 (httptest mock), R2 (offline audio + net isolation), R3 (single-instance process count) | M4 | E2E Testing Track |
| 9 | Final E2E Integration & Audit | 100% test pass verification and Forensic Auditor check | M5 | Implementation Final Milestone |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Telegram Remote Control (R1) | `pkg/telegram`, Telegram polling, chat whitelist, `ProcessTelegramUpdate` unit tests | none | DONE |
| M2 | Offline Voice PTT (R2) | `pkg/voice`, `Ctrl+Shift+V` hotkey, SAPI dictation script, `CaptureAndTranscribe` unit tests | M1 (DispatchIntent) | PLANNED |
| M3 | Live Progress HUD Overlay (R3) | `notify.ps1` WPF XAML + NamedPipe IPC, `notifyUser` Go integration, HUD unit/IPC tests | none | PLANNED |
| M4 | E2E Testing Suite (Dual-Track) | `tests/e2e/` (httptest mock Telegram, offline voice net-isolation, HUD process single-instance test) | M1, M2, M3 | DONE |
| M5 | Integration & Forensic Audit | Final E2E test execution, Tier 5 hardening, `teamwork_preview_auditor` verification | M1, M2, M3, M4 | PLANNED |

## Interface Contracts
### Main Intent Pipeline ↔ Feature Modules (R1, R2)
- Function: `DispatchIntent(intent string) (string, error)`
- Thread safety: `sync.Mutex` locked during skill match & execution.
- Input: intent string (e.g. "open notepad", "search cats on google")
- Output: status message string, error

### Go Subsystem ↔ Live Progress HUD (R3)
- PowerShell Invocation: `powershell.exe -ExecutionPolicy Bypass -File notify.ps1 -Title "<title>" -Message "<message>" [-Step X -Total Y] [-Close]`
- IPC Pipe: `\\.\pipe\ForgeHUD_Pipe` (JSON payload: `{"Title":"...", "Message":"...", "Step":1, "Total":3, "Close":false}`)

## Code Layout
- `main.go`: Entry point, `DispatchIntent` routing, `notifyUser` integration.
- `pkg/telegram/`: Telegram bot polling client, update parser, chat authorization.
- `pkg/voice/`: Hotkey hook handler (`Ctrl+Shift+V`), offline SAPI helper invocation (`voice_listen.ps1`), transcription parser.
- `notify.ps1`: Single-instance WPF XAML HUD overlay + NamedPipe IPC server/client.
- `tests/e2e/`: Requirement-driven E2E integration test suite.
