# Forge OS Test Infrastructure & E2E Suite Plan

## 1. Overview & Test Architecture
The Forge OS test infrastructure establishes a requirement-driven, 4-Tier test suite to validate all newly introduced features in Forge OS:
- **R1: Telegram Remote Control**
- **R2: Offline Voice Push-to-Talk**
- **R3: Live Progress HUD Overlay**

The test architecture relies on opaque-box verification, isolating network traffic to verify offline compliance, using `httptest.Server` for API mocks, and leveraging native OS process inspection for WPF UI single-instance verification.

### Testing Tiers
| Tier | Scope | Framework / Mechanism | Target Features |
|------|-------|-----------------------|-----------------|
| Tier 1 | Unit & Component Verification | Go `testing` package | Update parsing, JSON pipe schema, Chat ID whitelist logic |
| Tier 2 | Integration & Pipeline Verification | Go `httptest.Server` & NamedPipe clients | `DispatchIntent` routing, Telegram API interaction, NamedPipe IPC |
| Tier 3 | End-to-End Requirement Verification | Go `testing` + PowerShell scripts | Complete R1, R2, R3 workflow validation |
| Tier 4 | Hardening & Environmental Isolation | Custom HTTP Transport & Win32 Process Audit | Zero-network outbound enforcement (R2), Single-instance WPF HUD process count (R3) |

---

## 2. Feature Inventory & Test Mapping

| Req ID | Feature Name | Test File | Test Case | Description & Authoritative Verification |
|--------|--------------|-----------|-----------|------------------------------------------|
| R1.1 | Telegram Update Polling | `tests/e2e/r1_telegram_e2e_test.go` | `TestTelegramBot_GetUpdates` | Verifies `httptest.Server` `/getUpdates` response deserialization and payload parsing. |
| R1.2 | Chat Authorization Whitelist | `tests/e2e/r1_telegram_e2e_test.go` | `TestTelegramBot_AuthorizationWhitelist` | Validates authorized Chat IDs trigger intent handling while unauthorized Chat IDs are rejected. |
| R1.3 | Telegram Intent Routing & Response | `tests/e2e/r1_telegram_e2e_test.go` | `TestTelegramBot_IntentDispatchAndResponse` | Confirms received text forwards to `DispatchIntent` and outbound response posts to `/sendMessage`. |
| R2.1 | Offline Speech Transcription Pipeline | `tests/e2e/r2_voice_e2e_test.go` | `TestVoice_OfflineTranscriptionPipeline` | Verifies Windows SAPI offline dictation worker transcribes audio and routes text to `DispatchIntent`. |
| R2.2 | Zero Outbound Network Requests | `tests/e2e/r2_voice_e2e_test.go` | `TestVoice_NetworkIsolationEnforcement` | Intercepts HTTP transport during speech transcription to assert zero external network calls. |
| R3.1 | Single-Instance WPF HUD Process | `tests/e2e/r3_hud_e2e_test.ps1` | `Test-HUD-SingleInstance` | Verifies process count remains exactly 1 when issuing sequential progress updates (`[1/3]`, `[2/3]`, `[3/3]`). |
| R3.2 | NamedPipe IPC Communication | `tests/e2e/r3_hud_e2e_test.ps1` | `Test-HUD-NamedPipeIPC` | Validates JSON payloads sent over `\\.\pipe\ForgeHUD_Pipe` correctly update UI state without window recreation. |

---

## 3. Detailed Test Design

### R1. Telegram Remote Control (`tests/e2e/r1_telegram_e2e_test.go`)
- **Mock Server**: Spawns `httptest.NewServer` simulating Telegram Bot API.
- **Endpoints**:
  - `POST/GET /bot<token>/getUpdates`: Returns JSON updates containing test chat messages.
  - `POST /bot<token>/sendMessage`: Captures outbound responses from Forge OS to verify expected response text and `chat_id`.
- **Test Scenarios**:
  1. **Authorized Chat**: Send message from whitelisted Chat ID (e.g., `12345678`), verify `DispatchIntent` execution and `/sendMessage` payload response.
  2. **Unauthorized Chat**: Send message from untrusted Chat ID (e.g., `99999999`), verify rejection and zero `/sendMessage` calls.
  3. **Malformed Update Payload**: Handle corrupt or empty updates gracefully without application panic.

### R2. Offline Voice Push-to-Talk (`tests/e2e/r2_voice_e2e_test.go`)
- **Offline Transcription**: Invokes SAPI speech recognition pipeline to transcribe speech.
- **Network Isolation Enforcement**: Installs a custom `http.RoundTripper` monitoring `http.DefaultClient` or custom HTTP transport to ensure NO TCP/HTTP network connections are established during voice recognition.
- **Test Scenarios**:
  1. **Transcription Dispatch**: Mock/execute voice transcription, verify output string is dispatched to `DispatchIntent`.
  2. **Strict Network Isolation**: Monitor all network outbound calls during transcription execution. Fail test if any outbound request occurs.

### R3. Live Progress HUD Overlay (`tests/e2e/r3_hud_e2e_test.ps1`)
- **Single-Instance WPF Verification**: Issues sequential updates (`[1/3] Step 1`, `[2/3] Step 2`, `[3/3] Step 3`) via `notify.ps1` and NamedPipe IPC `\\.\pipe\ForgeHUD_Pipe`.
- **Test Scenarios**:
  1. **Process Count Audit**: Queries `Get-Process` before and during sequential updates to verify no extra `powershell` window processes are spawned.
  2. **NamedPipe Payload Verification**: Sends JSON state updates and verifies pipe connection handles state transitions smoothly.
  3. **Close Command Verification**: Sends `Close` flag via IPC and confirms process terminates cleanly.

---

## 4. Execution & Verification Commands

To execute the test suite:

```powershell
# 1. Run Go E2E Integration Tests (R1, R2)
go test -v ./tests/e2e/...

# 2. Run PowerShell HUD Process Management E2E Test (R3)
powershell.exe -ExecutionPolicy Bypass -File ./tests/e2e/r3_hud_e2e_test.ps1
```
