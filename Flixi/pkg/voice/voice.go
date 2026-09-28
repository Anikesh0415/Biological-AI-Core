package voice

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	intentHandler   func(string) (string, error)
	intentHandlerMu sync.RWMutex
	hotkeySequence  = "Ctrl+Shift+V"
	mockText        string
	mockTextMu      sync.RWMutex
)

// RegisterHotkeyHandler registers the intent handler function to be invoked when voice transcription completes.
func RegisterHotkeyHandler(handler func(string) (string, error)) {
	intentHandlerMu.Lock()
	defer intentHandlerMu.Unlock()
	intentHandler = handler
}

// SetIntentHandler is an alias for RegisterHotkeyHandler.
func SetIntentHandler(handler func(string) (string, error)) {
	RegisterHotkeyHandler(handler)
}

// GetIntentHandler returns the currently registered intent handler function.
func GetIntentHandler() func(string) (string, error) {
	intentHandlerMu.RLock()
	defer intentHandlerMu.RUnlock()
	return intentHandler
}

// IsHotkeyRegistered returns whether an intent handler is registered for voice hotkey.
func IsHotkeyRegistered() bool {
	intentHandlerMu.RLock()
	defer intentHandlerMu.RUnlock()
	return intentHandler != nil
}

// GetHotkeySequence returns the hotkey string representation ("Ctrl+Shift+V").
func GetHotkeySequence() string {
	return hotkeySequence
}

// SetMockTranscription sets a mock transcribed text for testing or simulation.
func SetMockTranscription(text string) {
	mockTextMu.Lock()
	defer mockTextMu.Unlock()
	mockText = text
}

func getMockTranscription() string {
	mockTextMu.RLock()
	defer mockTextMu.RUnlock()
	if mockText != "" {
		return mockText
	}
	return os.Getenv("FORGE_VOICE_MOCK_TEXT")
}

// getScriptPath locates voice_listen.ps1 across known directory paths.
func getScriptPath() string {
	candidates := []string{
		"pkg/voice/voice_listen.ps1",
		"./pkg/voice/voice_listen.ps1",
		"voice_listen.ps1",
		"../pkg/voice/voice_listen.ps1",
		"../../pkg/voice/voice_listen.ps1",
	}

	if exePath, err := os.Executable(); err == nil {
		exeDir := filepath.Dir(exePath)
		candidates = append(candidates, filepath.Join(exeDir, "pkg", "voice", "voice_listen.ps1"))
		candidates = append(candidates, filepath.Join(exeDir, "voice_listen.ps1"))
	}

	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "pkg", "voice", "voice_listen.ps1"))
		candidates = append(candidates, filepath.Join(wd, "voice_listen.ps1"))
	}

	for _, path := range candidates {
		if abs, err := filepath.Abs(path); err == nil {
			if info, err := os.Stat(abs); err == nil && !info.IsDir() {
				return abs
			}
		}
	}

	return "pkg/voice/voice_listen.ps1"
}

// CaptureAndTranscribe executes offline Windows SAPI dictation worker (voice_listen.ps1) with timeoutSec.
func CaptureAndTranscribe(timeoutSec int) (string, error) {
	return CaptureAndTranscribeWithContext(context.Background(), timeoutSec)
}

// CaptureAndTranscribeWithContext executes voice transcription with context and timeout.
func CaptureAndTranscribeWithContext(ctx context.Context, timeoutSec int) (string, error) {
	if mock := getMockTranscription(); mock != "" {
		mock = strings.TrimSpace(mock)
		if mock == "" {
			return "", fmt.Errorf("empty speech transcription")
		}
		return mock, nil
	}

	if timeoutSec <= 0 {
		timeoutSec = 5
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec+3)*time.Second)
	defer cancel()

	scriptPath := filepath.Join("pkg", "voice", "voice_listen.py")
	modelPath := filepath.Join("pkg", "voice", "model")

	// We pipe stderr to null so we don't get Vosk's debug logs in the output text
	cmd := exec.CommandContext(timeoutCtx, "python", scriptPath, "--model", modelPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("voice transcription failed: %w", err)
	}

	text := strings.TrimSpace(string(out))
	if text == "" {
		return "", fmt.Errorf("empty speech transcription")
	}

	return text, nil
}

// TriggerVoiceCaptureAndDispatch performs voice capture and dispatches transcribed text to registered intent handler.
func TriggerVoiceCaptureAndDispatch(timeoutSec int) (string, error) {
	text, err := CaptureAndTranscribe(timeoutSec)
	if err != nil {
		return "", err
	}

	handler := GetIntentHandler()
	if handler != nil {
		resp, err := handler(text)
		if err != nil {
			return text, fmt.Errorf("intent dispatch error: %w", err)
		}
		return resp, nil
	}

	return text, nil
}
