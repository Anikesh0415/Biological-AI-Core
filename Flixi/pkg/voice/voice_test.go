package voice

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// NetworkMonitorTransport intercepts any HTTP requests to verify zero outbound network calls
type NetworkMonitorTransport struct {
	BaseTransport http.RoundTripper
	RequestCount  int64
	Requests      []*http.Request
	mu            sync.Mutex
}

func NewNetworkMonitorTransport() *NetworkMonitorTransport {
	return &NetworkMonitorTransport{
		BaseTransport: http.DefaultTransport,
		Requests:      make([]*http.Request, 0),
	}
}

func (n *NetworkMonitorTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	atomic.AddInt64(&n.RequestCount, 1)
	n.mu.Lock()
	n.Requests = append(n.Requests, req)
	n.mu.Unlock()
	return nil, fmt.Errorf("NETWORK VIOLATION: Outbound HTTP request blocked to %s", req.URL.String())
}

func TestCaptureAndTranscribe_MockInput(t *testing.T) {
	SetMockTranscription("open notepad")
	defer SetMockTranscription("")

	result, err := CaptureAndTranscribe(3)
	if err != nil {
		t.Fatalf("CaptureAndTranscribe failed: %v", err)
	}

	if result != "open notepad" {
		t.Errorf("Expected 'open notepad', got '%s'", result)
	}
}

func TestCaptureAndTranscribe_RealExecutionOrFallback(t *testing.T) {
	SetMockTranscription("") // ensure clean state
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	// CaptureAndTranscribeWithContext executes real voice_listen.ps1
	result, err := CaptureAndTranscribeWithContext(ctx, 1)
	if err != nil {
		t.Logf("CaptureAndTranscribe returned expected error during silence/no mic: %v", err)
	} else {
		t.Logf("CaptureAndTranscribe returned speech: %s", result)
	}
}

func TestVoice_NetworkIsolationEnforcement(t *testing.T) {
	monitor := NewNetworkMonitorTransport()
	oldTransport := http.DefaultTransport
	http.DefaultTransport = monitor
	defer func() {
		http.DefaultTransport = oldTransport
	}()

	SetMockTranscription("open calculator")
	defer SetMockTranscription("")

	result, err := CaptureAndTranscribe(2)
	if err != nil {
		t.Fatalf("CaptureAndTranscribe failed during isolation test: %v", err)
	}
	if result != "open calculator" {
		t.Fatalf("Unexpected result: %s", result)
	}

	reqCount := atomic.LoadInt64(&monitor.RequestCount)
	if reqCount != 0 {
		t.Fatalf("NETWORK ISOLATION VIOLATION: Detected %d outbound HTTP requests during voice transcription!", reqCount)
	}
}

func TestHotkeyHandlerRegistration(t *testing.T) {
	var dispatched string
	var mu sync.Mutex

	mockHandler := func(intent string) (string, error) {
		mu.Lock()
		dispatched = intent
		mu.Unlock()
		return "Executed: " + intent, nil
	}

	RegisterHotkeyHandler(mockHandler)
	defer RegisterHotkeyHandler(nil)

	if !IsHotkeyRegistered() {
		t.Errorf("Expected hotkey handler to be registered")
	}

	if seq := GetHotkeySequence(); seq != "Ctrl+Shift+V" {
		t.Errorf("Expected hotkey sequence 'Ctrl+Shift+V', got '%s'", seq)
	}

	handler := GetIntentHandler()
	if handler == nil {
		t.Fatalf("GetIntentHandler returned nil")
	}

	res, err := handler("open edge")
	if err != nil {
		t.Fatalf("Handler returned error: %v", err)
	}
	if res != "Executed: open edge" {
		t.Errorf("Unexpected handler result: %s", res)
	}

	mu.Lock()
	if dispatched != "open edge" {
		t.Errorf("Expected dispatched 'open edge', got '%s'", dispatched)
	}
	mu.Unlock()
}

func TestTriggerVoiceCaptureAndDispatch(t *testing.T) {
	var dispatched string
	mockHandler := func(intent string) (string, error) {
		dispatched = intent
		return "OK", nil
	}

	RegisterHotkeyHandler(mockHandler)
	defer RegisterHotkeyHandler(nil)

	SetMockTranscription("turn on bluetooth")
	defer SetMockTranscription("")

	resp, err := TriggerVoiceCaptureAndDispatch(2)
	if err != nil {
		t.Fatalf("TriggerVoiceCaptureAndDispatch failed: %v", err)
	}

	if resp != "OK" {
		t.Errorf("Expected response 'OK', got '%s'", resp)
	}

	if dispatched != "turn on bluetooth" {
		t.Errorf("Expected dispatched 'turn on bluetooth', got '%s'", dispatched)
	}
}

func TestEmptySpeechHandling(t *testing.T) {
	SetMockTranscription("   ")
	defer SetMockTranscription("")

	_, err := CaptureAndTranscribe(1)
	if err == nil {
		t.Errorf("Expected error for empty/whitespace mock speech, got nil")
	}
}
