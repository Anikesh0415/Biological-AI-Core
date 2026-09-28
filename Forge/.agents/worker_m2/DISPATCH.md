# DISPATCH — Worker M2 (Offline Voice Push-to-Talk R2)

You are Worker M2 (`teamwork_preview_worker`).
Working directory: `E:/AIF_Project/.agents/worker_m2`
Read `E:/AIF_Project/.agents/ORIGINAL_REQUEST.md` and `E:/AIF_Project/PROJECT.md`.

## Assignment
Implement Requirement R2: Offline Voice Push-to-Talk in `E:/AIF_Project`.

### Technical Requirements
1. **Offline Voice Package (`pkg/voice/voice.go` & `pkg/voice/voice_listen.ps1`)**:
   - Create PowerShell helper `pkg/voice/voice_listen.ps1` using `System.Speech.Recognition.SpeechRecognitionEngine` with `DictationGrammar` and `SetInputToDefaultAudioDevice()`. It listens for microphone input, transcribes speech offline, and prints the result to stdout.
   - Implement `CaptureAndTranscribe(timeoutSec int) (string, error)` in Go (`pkg/voice/voice.go`) which executes `voice_listen.ps1` asynchronously or with timeout.
   - Verify 100% offline operation — zero outbound HTTP network requests during transcription.
2. **Global Hotkey Registration (`Ctrl+Shift+V`)**:
   - Add `Ctrl+Shift+V` (`VK_V = 0x56` with `VK_CONTROL` and `VK_SHIFT`) hotkey handling to `pkg/recorder/hook.go` (or `pkg/voice`).
   - When triggered, invoke `CaptureAndTranscribe(...)` and forward the transcribed text to `main.go`'s `DispatchIntent(text)`.
3. **Unit & Isolation Tests (`pkg/voice/voice_test.go`)**:
   - Test `CaptureAndTranscribe` offline transcription function.
   - Test network isolation (assert zero HTTP client sockets / requests).
   - Test hotkey handler registration.
4. **Compiler Compatibility**:
   - Verify with `E:\AIF_Project\go\bin\go.exe test -v ./pkg/voice` and `E:\AIF_Project\go\bin\go.exe build -o forge.exe main.go`.

### MANDATORY INTEGRITY WARNING
DO NOT CHEAT. All implementations must be genuine. DO NOT hardcode test results, create dummy/facade implementations, or circumvent the intended task. A teamwork_preview_auditor will independently verify your work. Integrity violations WILL be detected and your work WILL be rejected.

### Required Deliverables
- Implementation files: `pkg/voice/voice.go`, `pkg/voice/voice_listen.ps1`, `pkg/voice/voice_test.go`, updates to `pkg/recorder/hook.go` / `main.go`.
- Run build and tests using `E:\AIF_Project\go\bin\go.exe`.
- Write complete handoff report to `E:/AIF_Project/.agents/worker_m2/handoff.md`.
