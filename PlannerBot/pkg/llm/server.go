package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

// ServerConfig holds configuration for the managed llama-server process.
type ServerConfig struct {
	// BinaryPath is the absolute path to the llama-server executable.
	BinaryPath string

	// ModelPath is the absolute path to the GGUF model file.
	ModelPath string

	// Host is the bind address. Defaults to "127.0.0.1".
	Host string

	// Port is the TCP port. 0 means auto-assign a free port.
	Port int

	// Threads is the number of CPU threads for inference. Defaults to 4.
	Threads int

	// ContextSize is the context window size in tokens. Defaults to 2048.
	ContextSize int

	// GPULayers is the number of layers to offload to GPU. Defaults to 0 (CPU only).
	GPULayers int
}

// applyDefaults fills in zero-valued config fields with sensible defaults.
func (cfg *ServerConfig) applyDefaults() {
	if cfg.Host == "" {
		cfg.Host = "127.0.0.1"
	}
	if cfg.Threads <= 0 {
		cfg.Threads = 4
	}
	if cfg.ContextSize <= 0 {
		cfg.ContextSize = 8192
	}
}

// Server represents a running llama-server subprocess.
type Server struct {
	cmd       *exec.Cmd
	host      string
	port      int
	stdout    *bytes.Buffer
	stderr    *bytes.Buffer
	closeOnce sync.Once
	closeErr  error
}

// findFreePort discovers an available TCP port by briefly listening on :0.
func findFreePort() (int, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, fmt.Errorf("find free port: %w", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port, nil
}

// StartServer launches a llama-server subprocess and waits until it becomes healthy.
// The context controls the startup timeout (health polling). If the context is cancelled
// before the server becomes healthy, the process is killed and an error is returned.
func StartServer(ctx context.Context, cfg ServerConfig) (*Server, error) {
	cfg.applyDefaults()

	// Validate required fields
	if cfg.BinaryPath == "" {
		return nil, fmt.Errorf("llm: BinaryPath is required")
	}
	if cfg.ModelPath == "" {
		return nil, fmt.Errorf("llm: ModelPath is required")
	}

	// Verify binary exists
	if _, err := os.Stat(cfg.BinaryPath); err != nil {
		return nil, fmt.Errorf("llm: binary not found at %q: %w", cfg.BinaryPath, err)
	}

	// Verify model exists
	if _, err := os.Stat(cfg.ModelPath); err != nil {
		return nil, fmt.Errorf("llm: model not found at %q: %w", cfg.ModelPath, err)
	}

	// Auto-assign port if needed
	port := cfg.Port
	if port == 0 {
		var err error
		port, err = findFreePort()
		if err != nil {
			return nil, err
		}
	}

	// Build command arguments
	args := []string{
		"--model", cfg.ModelPath,
		"--host", cfg.Host,
		"--port", strconv.Itoa(port),
		"--threads", strconv.Itoa(cfg.Threads),
		"--ctx-size", strconv.Itoa(cfg.ContextSize),
		"--n-gpu-layers", strconv.Itoa(cfg.GPULayers),
	}

	cmd := exec.CommandContext(ctx, cfg.BinaryPath, args...)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("llm: failed to start server: %w\nstderr: %s", err, stderrBuf.String())
	}

	srv := &Server{
		cmd:    cmd,
		host:   cfg.Host,
		port:   port,
		stdout: &stdoutBuf,
		stderr: &stderrBuf,
	}

	// Poll /health until the server is ready
	if err := srv.waitForHealth(ctx); err != nil {
		// Kill the process if health check fails
		_ = srv.Close()
		return nil, fmt.Errorf("llm: server failed health check: %w\nstderr: %s", err, stderrBuf.String())
	}

	return srv, nil
}

// waitForHealth polls the /health endpoint until it returns {"status":"ok"} or the context expires.
func (s *Server) waitForHealth(ctx context.Context) error {
	healthURL := s.BaseURL() + "/health"
	client := &http.Client{Timeout: 2 * time.Second}

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	// Allow up to 120 seconds for the model to load
	deadline := time.Now().Add(120 * time.Second)

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled while waiting for server health: %w", ctx.Err())
		case <-ticker.C:
			if time.Now().After(deadline) {
				return fmt.Errorf("server health check timed out after 120 seconds")
			}

			// Check if process died
			if s.cmd.ProcessState != nil && s.cmd.ProcessState.Exited() {
				return fmt.Errorf("server process exited unexpectedly with code %d", s.cmd.ProcessState.ExitCode())
			}

			resp, err := client.Get(healthURL)
			if err != nil {
				continue // Server not yet accepting connections
			}

			var health HealthResponse
			decodeErr := json.NewDecoder(resp.Body).Decode(&health)
			_ = resp.Body.Close()

			if decodeErr != nil {
				continue
			}

			if health.Status == "ok" {
				return nil
			}
		}
	}
}

// BaseURL returns the HTTP base URL of the running server (e.g., "http://127.0.0.1:8080").
func (s *Server) BaseURL() string {
	return fmt.Sprintf("http://%s:%d", s.host, s.port)
}

// Addr returns the host:port address string.
func (s *Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

// Port returns the TCP port the server is listening on.
func (s *Server) Port() int {
	return s.port
}

// Stdout returns the captured stdout output from the server process.
func (s *Server) Stdout() string {
	return s.stdout.String()
}

// Stderr returns the captured stderr output from the server process.
func (s *Server) Stderr() string {
	return s.stderr.String()
}

// Close gracefully shuts down the llama-server process.
// It first sends an interrupt signal, waits up to 5 seconds, then force kills.
func (s *Server) Close() error {
	s.closeOnce.Do(func() {
		if s.cmd == nil || s.cmd.Process == nil {
			return
		}

		// Try graceful shutdown first
		done := make(chan error, 1)
		go func() {
			done <- s.cmd.Wait()
		}()

		// On Windows, os.Interrupt is not supported, so we go straight to Kill.
		// On Unix, we'd send SIGINT first then SIGKILL.
		_ = s.cmd.Process.Kill()

		select {
		case err := <-done:
			// Process exited (possibly with error from Kill, which is expected)
			_ = err
		case <-time.After(5 * time.Second):
			// Force kill if still alive
			_ = s.cmd.Process.Kill()
		}
	})
	return s.closeErr
}
