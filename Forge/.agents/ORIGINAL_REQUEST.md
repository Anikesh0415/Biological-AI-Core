# Original User Request

## 2026-08-10T16:10:19Z

# Teamwork Project Prompt — Draft

> Status: Ready for launch — awaiting user approval.
> Goal: Craft prompt → get user approval → delegate to teamwork_preview

Add Telegram remote control, offline Voice Push-to-Talk, and a live progress HUD overlay to Forge OS (a Go-based desktop automation agent).

Working directory: E:/AIF_Project
Integrity mode: development

## Requirements

### R1. Telegram Remote Control
Implement a lightweight background Goroutine in Go that connects to the Telegram Bot API. It should receive remote text messages and map them to existing Forge skills/intents.

### R2. Offline Voice Push-to-Talk
Implement a global hotkey listener (e.g., `Ctrl+Shift+V`) that captures microphone input and transcribes it to text entirely offline (e.g., using Windows SAPI or a lightweight local library). The transcribed text should be sent to the existing planner/skill matcher.

### R3. Live Progress HUD Overlay
Update the existing `notify.ps1` PowerShell WPF script to handle multi-line or stateful progress updates (e.g., `[1/3] Step 1`, `[2/3] Step 2`). It should provide a visual cue to the user about current agent activity.

## Acceptance Criteria

### Verification
- [ ] Programmatic: Calling the Telegram handler function with a mock message payload correctly triggers the skill matching logic.
- [ ] Programmatic: The Voice transcription function operates correctly without external network requests (pure offline).
- [ ] Programmatic: Running `notify.ps1` with sequential progress updates successfully updates the UI text without crashing or spawning multiple disjoint windows.
