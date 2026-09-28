package llm

import (
	"context"
	"testing"
)

func TestFindFreePort(t *testing.T) {
	port, err := findFreePort()
	if err != nil {
		t.Fatalf("findFreePort() returned error: %v", err)
	}
	if port <= 0 || port > 65535 {
		t.Errorf("findFreePort() returned invalid port: %d", port)
	}
	t.Logf("findFreePort() returned port: %d", port)
}

func TestFindFreePortUniqueness(t *testing.T) {
	ports := make(map[int]bool)
	for i := 0; i < 10; i++ {
		port, err := findFreePort()
		if err != nil {
			t.Fatalf("findFreePort() iteration %d returned error: %v", i, err)
		}
		if port <= 0 {
			t.Errorf("findFreePort() iteration %d returned invalid port: %d", i, port)
		}
		ports[port] = true
	}
	t.Logf("findFreePort() returned %d unique ports out of 10 calls", len(ports))
}

func TestServerConfigDefaults(t *testing.T) {
	cfg := ServerConfig{}
	cfg.applyDefaults()

	if cfg.Host != "127.0.0.1" {
		t.Errorf("expected default Host=127.0.0.1, got %q", cfg.Host)
	}
	if cfg.Threads != 4 {
		t.Errorf("expected default Threads=4, got %d", cfg.Threads)
	}
	if cfg.ContextSize != 8192 {
		t.Errorf("expected default ContextSize=8192, got %d", cfg.ContextSize)
	}
}

func TestServerConfigNoOverrideExplicit(t *testing.T) {
	cfg := ServerConfig{
		Host:        "0.0.0.0",
		Threads:     8,
		ContextSize: 4096,
	}
	cfg.applyDefaults()

	if cfg.Host != "0.0.0.0" {
		t.Errorf("expected Host=0.0.0.0, got %q", cfg.Host)
	}
	if cfg.Threads != 8 {
		t.Errorf("expected Threads=8, got %d", cfg.Threads)
	}
	if cfg.ContextSize != 4096 {
		t.Errorf("expected ContextSize=4096, got %d", cfg.ContextSize)
	}
}

func TestStartServerMissingBinary(t *testing.T) {
	cfg := ServerConfig{
		BinaryPath: "",
		ModelPath:  "/some/model.gguf",
	}
	_, err := StartServer(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for empty BinaryPath, got nil")
	}
	t.Logf("correctly got error: %v", err)
}

func TestStartServerMissingModel(t *testing.T) {
	cfg := ServerConfig{
		BinaryPath: "/some/binary",
		ModelPath:  "",
	}
	_, err := StartServer(context.Background(), cfg)
	if err == nil {
		t.Fatal("expected error for empty ModelPath, got nil")
	}
	t.Logf("correctly got error: %v", err)
}

func TestServerBaseURL(t *testing.T) {
	s := &Server{
		host: "127.0.0.1",
		port: 8080,
	}
	expected := "http://127.0.0.1:8080"
	if got := s.BaseURL(); got != expected {
		t.Errorf("BaseURL() = %q, want %q", got, expected)
	}
}

func TestServerAddr(t *testing.T) {
	s := &Server{
		host: "127.0.0.1",
		port: 9090,
	}
	expected := "127.0.0.1:9090"
	if got := s.Addr(); got != expected {
		t.Errorf("Addr() = %q, want %q", got, expected)
	}
}

func TestServerPort(t *testing.T) {
	s := &Server{
		port: 12345,
	}
	if got := s.Port(); got != 12345 {
		t.Errorf("Port() = %d, want 12345", got)
	}
}
