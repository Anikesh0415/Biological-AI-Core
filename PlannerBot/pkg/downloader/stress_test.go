package downloader_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"planner_bot/pkg/downloader"
)

// helper to calculate SHA-256 hex string for stress tests
func computeSHA256(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// -----------------------------------------------------------------------------
// Challenge 1.1: Single Mid-Stream Disconnect & Resumed Range Download (HTTP 206)
// -----------------------------------------------------------------------------
func TestStress_InterruptedDownload_MidStreamDisconnectResume(t *testing.T) {
	// Generate deterministic 64 KB payload
	payload := make([]byte, 64*1024)
	for i := range payload {
		payload[i] = byte(i % 251)
	}
	expectedHash := computeSHA256(payload)
	splitPoint := 20 * 1024 // 20 KB

	var attempts atomic.Int32
	var rangeRequested atomic.Bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		att := attempts.Add(1)
		if att == 1 {
			// Attempt 1: Hijack connection and write partial data, then abruptly close TCP connection
			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatalf("server does not support hijacking")
			}
			conn, bufrw, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack failed: %v", err)
			}
			defer conn.Close()

			// Send HTTP 200 response with full Content-Length but close socket mid-body
			respHeader := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\nContent-Type: application/octet-stream\r\n\r\n", len(payload))
			_, _ = bufrw.WriteString(respHeader)
			_, _ = bufrw.Write(payload[:splitPoint])
			_ = bufrw.Flush()
			// Abrupt TCP socket closure simulating network drop / client disconnect midway
			_ = conn.Close()
			return
		}

		// Attempt 2: Downloader should have caught the error, kept 20 KB in .part, and sent Range header
		rangeHdr := r.Header.Get("Range")
		if rangeHdr != "" {
			rangeRequested.Store(true)
		}

		expectedRange := fmt.Sprintf("bytes=%d-", splitPoint)
		if rangeHdr != expectedRange {
			t.Errorf("expected Range header %q, got %q", expectedRange, rangeHdr)
		}

		// Serve remainder via HTTP 206
		handleRangeRequest(w, r, payload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(
		cacheDir,
		downloader.WithRetryDelay(5*time.Millisecond),
		downloader.WithMaxRetries(3),
	)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "interrupted-resume.gguf",
		URL:          server.URL + "/interrupted-resume.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	ctx := context.Background()
	finalPath, err := dl.EnsureModel(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed to complete interrupted download: %v", err)
	}

	if attempts.Load() != 2 {
		t.Errorf("expected exactly 2 attempts, got %d", attempts.Load())
	}
	if !rangeRequested.Load() {
		t.Errorf("expected Range header on resumed attempt")
	}

	downloadedBytes, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("failed to read completed file: %v", err)
	}
	if int64(len(downloadedBytes)) != cfg.ExpectedSize {
		t.Errorf("size mismatch: got %d, expected %d", len(downloadedBytes), cfg.ExpectedSize)
	}
	if !bytes.Equal(downloadedBytes, payload) {
		t.Errorf("payload byte mismatch after resume")
	}

	partPath := finalPath + ".part"
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Errorf("expected .part file to be removed after promotion")
	}
}

// -----------------------------------------------------------------------------
// Challenge 1.2: Multiple Consecutive Mid-Stream Disconnects with Incremental Resumes
// -----------------------------------------------------------------------------
func TestStress_MultipleMidStreamDisconnectsResume(t *testing.T) {
	payload := make([]byte, 100*1024) // 100 KB
	for i := range payload {
		payload[i] = byte((i * 13) % 256)
	}
	expectedHash := computeSHA256(payload)

	// Disconnect milestones: 25 KB, 55 KB, 85 KB, then complete at 100 KB
	milestones := []int{25 * 1024, 55 * 1024, 85 * 1024}

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		att := int(attempts.Add(1))
		if att <= len(milestones) {
			targetCutoff := milestones[att-1]

			rangeHdr := r.Header.Get("Range")
			var startOffset int = 0
			if rangeHdr != "" {
				_, _ = fmt.Sscanf(rangeHdr, "bytes=%d-", &startOffset)
			}

			hj, ok := w.(http.Hijacker)
			if !ok {
				t.Fatalf("hijack not supported")
			}
			conn, bufrw, err := hj.Hijack()
			if err != nil {
				t.Fatalf("hijack failed: %v", err)
			}
			defer conn.Close()

			if startOffset == 0 {
				respHeader := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n", len(payload))
				_, _ = bufrw.WriteString(respHeader)
			} else {
				respHeader := fmt.Sprintf("HTTP/1.1 206 Partial Content\r\nContent-Range: bytes %d-%d/%d\r\nContent-Length: %d\r\n\r\n",
					startOffset, len(payload)-1, len(payload), len(payload)-startOffset)
				_, _ = bufrw.WriteString(respHeader)
			}

			// Stream from startOffset up to targetCutoff, then abruptly close socket
			chunk := payload[startOffset:targetCutoff]
			_, _ = bufrw.Write(chunk)
			_ = bufrw.Flush()
			_ = conn.Close()
			return
		}

		// Final attempt: stream remainder cleanly
		handleRangeRequest(w, r, payload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(
		cacheDir,
		downloader.WithRetryDelay(2*time.Millisecond),
		downloader.WithMaxRetries(5),
	)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "multi-disconnect.gguf",
		URL:          server.URL + "/multi-disconnect.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	finalPath, err := dl.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed across multiple disconnects: %v", err)
	}

	if attempts.Load() != 4 {
		t.Errorf("expected 4 total attempts, got %d", attempts.Load())
	}

	data, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Errorf("multi-disconnect resumed file is corrupted")
	}
}

// -----------------------------------------------------------------------------
// Challenge 1.3: Interrupted Download Across Distinct Downloader Executions
// -----------------------------------------------------------------------------
func TestStress_InterruptedDownload_IndependentRunResume(t *testing.T) {
	payload := bytes.Repeat([]byte("INDEPENDENT_RUN_CHALLENGE_BLOCK_"), 1024) // 32 KB
	expectedHash := computeSHA256(payload)
	cutoff := 14 * 1024

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req := requests.Add(1)
		if req == 1 {
			// First run: disconnect midway
			hj, _ := w.(http.Hijacker)
			conn, bufrw, _ := hj.Hijack()
			defer conn.Close()

			respHeader := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n", len(payload))
			_, _ = bufrw.WriteString(respHeader)
			_, _ = bufrw.Write(payload[:cutoff])
			_ = bufrw.Flush()
			_ = conn.Close()
			return
		}

		// Second run: handle resume
		handleRangeRequest(w, r, payload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	cfg := downloader.ModelConfig{
		Name:         "distinct-run-model.gguf",
		URL:          server.URL + "/model.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	// First execution: with MaxRetries = 1, so it fails and terminates execution
	dl1, _ := downloader.New(cacheDir, downloader.WithMaxRetries(1), downloader.WithRetryDelay(1*time.Millisecond))
	_, err := dl1.EnsureModel(context.Background(), cfg)
	if err == nil {
		t.Fatalf("expected run 1 to fail on socket disconnect, got nil")
	}

	// Verify .part exists and has exactly cutoff bytes
	partPath := filepath.Join(cacheDir, cfg.Name+".part")
	fi, statErr := os.Stat(partPath)
	if statErr != nil {
		t.Fatalf("expected .part file to remain after failure: %v", statErr)
	}
	if fi.Size() != int64(cutoff) {
		t.Fatalf("expected .part file size %d, got %d", cutoff, fi.Size())
	}

	// Second execution: completely new Downloader instance resuming from disk
	dl2, _ := downloader.New(cacheDir, downloader.WithMaxRetries(2), downloader.WithRetryDelay(1*time.Millisecond))
	finalPath, err := dl2.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("expected run 2 to resume and succeed: %v", err)
	}

	data, err := os.ReadFile(finalPath)
	if err != nil || !bytes.Equal(data, payload) {
		t.Fatalf("resumed file does not match expected payload")
	}
}

// -----------------------------------------------------------------------------
// Challenge 2.1: Transient Server Errors (500, 502, 503) and Exponential Backoff
// -----------------------------------------------------------------------------
func TestStress_TransientErrors_500_502_503_BackoffRecovery(t *testing.T) {
	payload := []byte("VALID_MODEL_PAYLOAD_AFTER_CHAOS_TEST_OK!")
	expectedHash := computeSHA256(payload)

	var attempts atomic.Int32
	var mu sync.Mutex
	timestamps := make([]time.Time, 0, 4)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		att := attempts.Add(1)

		mu.Lock()
		timestamps = append(timestamps, time.Now())
		mu.Unlock()

		switch att {
		case 1:
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusInternalServerError) // 500
			_, _ = w.Write([]byte("<html><body><h1>500 Internal Server Error</h1></body></html>"))
		case 2:
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusBadGateway) // 502
			_, _ = w.Write([]byte("<html><body><h1>502 Bad Gateway</h1></body></html>"))
		case 3:
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusServiceUnavailable) // 503
			_, _ = w.Write([]byte("<html><body><h1>503 Service Unavailable</h1></body></html>"))
		default:
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(payload)
		}
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	baseDelay := 20 * time.Millisecond
	dl, err := downloader.New(
		cacheDir,
		downloader.WithRetryDelay(baseDelay),
		downloader.WithMaxRetries(4),
	)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "transient-errors-model.gguf",
		URL:          server.URL + "/model.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	finalPath, err := dl.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("EnsureModel should have recovered after 500, 502, 503: %v", err)
	}

	if attempts.Load() != 4 {
		t.Errorf("expected exactly 4 attempts, got %d", attempts.Load())
	}

	// Verify backoff progression: attempt intervals should increase exponentially
	mu.Lock()
	defer mu.Unlock()
	if len(timestamps) == 4 {
		gap1 := timestamps[1].Sub(timestamps[0])
		gap2 := timestamps[2].Sub(timestamps[1])
		gap3 := timestamps[3].Sub(timestamps[2])

		// gap1 >= baseDelay (20ms), gap2 >= 2*baseDelay (40ms), gap3 >= 4*baseDelay (80ms)
		if gap1 < 10*time.Millisecond {
			t.Errorf("gap1 too small: %v", gap1)
		}
		if gap2 < 20*time.Millisecond {
			t.Errorf("gap2 too small: %v", gap2)
		}
		if gap3 < 40*time.Millisecond {
			t.Errorf("gap3 too small: %v", gap3)
		}
	}

	// Crucial integrity check: ensure NO HTML error content leaked into final file
	data, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("failed to read final file: %v", err)
	}
	if strings.Contains(string(data), "<html>") || strings.Contains(string(data), "500 Internal Server Error") || strings.Contains(string(data), "502 Bad Gateway") || strings.Contains(string(data), "503 Service Unavailable") {
		t.Errorf("HTML error payload leaked into downloaded file!")
	}
	if !bytes.Equal(data, payload) {
		t.Errorf("final payload mismatch")
	}
}

// -----------------------------------------------------------------------------
// Challenge 2.2: Interrupted Download Followed by 502/503 Errors, Then Resume
// -----------------------------------------------------------------------------
func TestStress_MidStreamDisconnect_Then_502_503_Then_206Resume(t *testing.T) {
	payload := make([]byte, 50*1024)
	for i := range payload {
		payload[i] = byte(i % 199)
	}
	expectedHash := computeSHA256(payload)
	cutoff := 15 * 1024

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		att := attempts.Add(1)
		switch att {
		case 1:
			// Midstream drop
			hj, _ := w.(http.Hijacker)
			conn, bufrw, _ := hj.Hijack()
			defer conn.Close()

			respHeader := fmt.Sprintf("HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n", len(payload))
			_, _ = bufrw.WriteString(respHeader)
			_, _ = bufrw.Write(payload[:cutoff])
			_ = bufrw.Flush()
			_ = conn.Close()
		case 2:
			// Server goes down: 502
			w.WriteHeader(http.StatusBadGateway)
			_, _ = w.Write([]byte("502 Bad Gateway"))
		case 3:
			// Server still overloaded: 503
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("503 Service Unavailable"))
		default:
			// Server recovers: satisfies Range request
			handleRangeRequest(w, r, payload)
		}
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, _ := downloader.New(
		cacheDir,
		downloader.WithRetryDelay(5*time.Millisecond),
		downloader.WithMaxRetries(5),
	)

	cfg := downloader.ModelConfig{
		Name:         "disconnect-chaos.gguf",
		URL:          server.URL + "/model.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	finalPath, err := dl.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed in disconnect+502/503 combo: %v", err)
	}

	if attempts.Load() != 4 {
		t.Errorf("expected exactly 4 attempts, got %d", attempts.Load())
	}

	data, err := os.ReadFile(finalPath)
	if err != nil || !bytes.Equal(data, payload) {
		t.Errorf("corrupted data after disconnect + 502/503 + resume recovery")
	}
}

// -----------------------------------------------------------------------------
// Challenge 3.1: Corrupt Truncated Cache Detection & Automatic Re-download
// -----------------------------------------------------------------------------
func TestStress_CorruptCache_TruncatedModelReDownloaded(t *testing.T) {
	validPayload := bytes.Repeat([]byte("GENUINE_TRUNCATED_TEST_PAYLOAD_CHUNK_"), 500) // ~18.5 KB
	expectedHash := computeSHA256(validPayload)

	var serverHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHits.Add(1)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(validPayload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(validPayload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	cfg := downloader.ModelConfig{
		Name:         "model-truncated.gguf",
		URL:          server.URL + "/model.gguf",
		ExpectedSize: int64(len(validPayload)),
		SHA256:       expectedHash,
	}

	// Seed corrupt truncated file in cacheDir
	cachedPath := filepath.Join(cacheDir, cfg.Name)
	if err := os.WriteFile(cachedPath, validPayload[:100], 0644); err != nil {
		t.Fatalf("failed to seed truncated file: %v", err)
	}

	dl, _ := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	finalPath, err := dl.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed on truncated cache recovery: %v", err)
	}

	if serverHits.Load() != 1 {
		t.Errorf("expected server to be queried to re-download truncated file")
	}

	data, err := os.ReadFile(finalPath)
	if err != nil || !bytes.Equal(data, validPayload) {
		t.Fatalf("failed to recover valid payload from truncated cache")
	}
}

// -----------------------------------------------------------------------------
// Challenge 3.2: Corrupt Hash Mismatch (Same-Size Bit-Flipped) Cache Recovery
// -----------------------------------------------------------------------------
func TestStress_CorruptCache_SameSizeBitFlippedReDownloaded(t *testing.T) {
	validPayload := make([]byte, 10000)
	for i := range validPayload {
		validPayload[i] = byte(i % 127)
	}
	expectedHash := computeSHA256(validPayload)

	var serverHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHits.Add(1)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(validPayload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(validPayload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	cfg := downloader.ModelConfig{
		Name:         "model-bitflip.gguf",
		URL:          server.URL + "/model.gguf",
		ExpectedSize: int64(len(validPayload)),
		SHA256:       expectedHash,
	}

	// Seed same-size file with bit-flipped bytes
	corrupted := make([]byte, len(validPayload))
	copy(corrupted, validPayload)
	corrupted[len(corrupted)/2] ^= 0xFF // Flip bits in the middle byte

	cachedPath := filepath.Join(cacheDir, cfg.Name)
	if err := os.WriteFile(cachedPath, corrupted, 0644); err != nil {
		t.Fatalf("failed to seed bit-flipped file: %v", err)
	}

	dl, _ := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	finalPath, err := dl.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed to recover bit-flipped corrupt cache: %v", err)
	}

	if serverHits.Load() != 1 {
		t.Errorf("expected server hit to re-download after hash failure")
	}

	data, err := os.ReadFile(finalPath)
	if err != nil || !bytes.Equal(data, validPayload) {
		t.Fatalf("recovered file does not match valid payload")
	}
}

// -----------------------------------------------------------------------------
// Challenge 3.3: Corrupted Prefix in .part File Automatically Purged and Recovered
// -----------------------------------------------------------------------------
func TestStress_CorruptPartFile_PrefixCorruptionPurgeAndRecover(t *testing.T) {
	validPayload := []byte("GENUINE_COMPLETE_PAYLOAD_FOR_CORRUPT_PART_PURGE_TEST_1234567890")
	expectedHash := computeSHA256(validPayload)
	prefixLen := 20

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		handleRangeRequest(w, r, validPayload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	cfg := downloader.ModelConfig{
		Name:         "corrupt-part.gguf",
		URL:          server.URL + "/corrupt-part.gguf",
		ExpectedSize: int64(len(validPayload)),
		SHA256:       expectedHash,
	}

	// Seed .part file with garbage bytes of length prefixLen
	partPath := filepath.Join(cacheDir, cfg.Name+".part")
	garbagePrefix := bytes.Repeat([]byte("X"), prefixLen)
	if err := os.WriteFile(partPath, garbagePrefix, 0644); err != nil {
		t.Fatalf("failed to write garbage part file: %v", err)
	}

	// Attempt 1 will request Range: bytes=20-, append remaining 44 bytes, but checksum will fail!
	// Downloader must purge .part on checksum fail, then Attempt 2 will download all 64 bytes from scratch!
	dl, _ := downloader.New(
		cacheDir,
		downloader.WithRetryDelay(1*time.Millisecond),
		downloader.WithMaxRetries(3),
	)

	finalPath, err := dl.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed to recover from corrupt .part file: %v", err)
	}

	if attempts.Load() != 2 {
		t.Errorf("expected attempt 1 to fail checksum, purge, and attempt 2 to succeed. Got attempts: %d", attempts.Load())
	}

	data, err := os.ReadFile(finalPath)
	if err != nil || !bytes.Equal(data, validPayload) {
		t.Fatalf("recovered file mismatch after corrupt .part purge")
	}
}

// -----------------------------------------------------------------------------
// Challenge 3.4: Corrupted Cached Binary Detected by Health Check and Re-Bootstrapped
// -----------------------------------------------------------------------------
func TestStress_CorruptBinaryCache_HealthCheckFailsAndBootstraps(t *testing.T) {
	// Create mock zip release with valid binary and DLLs
	validBin := []byte("HEALTHY_VALID_BINARY_BYTES")
	dll := []byte("HEALTHY_DLL_BYTES")
	zipBytes := createTestZip(t, map[string][]byte{
		"llama-server.exe": validBin,
		"llama.dll":        dll,
	})

	var serverHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHits.Add(1)
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(zipBytes)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(zipBytes)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	binDir := filepath.Join(cacheDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}

	// Seed corrupted/broken executable in binDir
	corruptBinPath := filepath.Join(binDir, "llama-server.exe")
	if err := os.WriteFile(corruptBinPath, []byte("BROKEN_CORRUPT_BINARY"), 0755); err != nil {
		t.Fatalf("failed to seed corrupt binary: %v", err)
	}

	// Custom health checker that considers any binary containing "BROKEN" as failing health check
	healthChecker := func(ctx context.Context, binPath string) error {
		data, err := os.ReadFile(binPath)
		if err != nil {
			return err
		}
		if bytes.Contains(data, []byte("BROKEN")) {
			return fmt.Errorf("health check failed: binary contains BROKEN marker")
		}
		return nil
	}

	binConfig := downloader.BinaryConfig{
		Name:         "llama-server.exe",
		ArchiveURL:   server.URL + "/llama-bin.zip",
		BinarySubdir: "bin",
	}

	dl, err := downloader.New(
		cacheDir,
		downloader.WithBinaryConfig(binConfig),
		downloader.WithHealthChecker(healthChecker),
	)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	finalBinPath, err := dl.EnsureBinary(context.Background())
	if err != nil {
		t.Fatalf("EnsureBinary failed to re-bootstrap corrupt binary: %v", err)
	}

	if serverHits.Load() != 1 {
		t.Errorf("expected server hit to download healthy zip after corrupt binary detected")
	}

	binContent, err := os.ReadFile(finalBinPath)
	if err != nil || !bytes.Equal(binContent, validBin) {
		t.Fatalf("bootstrapped binary does not match healthy content")
	}
}

// -----------------------------------------------------------------------------
// Challenge 3.5: Multi-Tier Cache Corruption Rejection ($PLANNER_MODEL_DIR)
// -----------------------------------------------------------------------------
func TestStress_CorruptCache_MultiTierDirRecovery(t *testing.T) {
	validPayload := []byte("GENUINE_TIER_OVERRIDE_PAYLOAD_12345")
	expectedHash := computeSHA256(validPayload)

	var serverHits atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverHits.Add(1)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(validPayload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(validPayload)
	}))
	defer server.Close()

	tempOverrideDir := t.TempDir()
	cacheDir := t.TempDir()

	cfg := downloader.ModelConfig{
		Name:         "tier-model.gguf",
		URL:          server.URL + "/tier-model.gguf",
		ExpectedSize: int64(len(validPayload)),
		SHA256:       expectedHash,
	}

	// Seed corrupt file in PLANNER_MODEL_DIR
	corruptPath := filepath.Join(tempOverrideDir, cfg.Name)
	_ = os.WriteFile(corruptPath, []byte("CORRUPT_TIER_DATA"), 0644)
	t.Setenv("PLANNER_MODEL_DIR", tempOverrideDir)

	dl, _ := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	finalPath, err := dl.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed: %v", err)
	}

	// Should reject corrupt PLANNER_MODEL_DIR candidate and download into cacheDir
	expectedCachePath := filepath.Join(cacheDir, cfg.Name)
	if finalPath != expectedCachePath {
		t.Errorf("expected fallback to cacheDir %s, got %s", expectedCachePath, finalPath)
	}
	if serverHits.Load() != 1 {
		t.Errorf("expected server hit to download valid file when tier candidate was corrupt")
	}

	data, err := os.ReadFile(finalPath)
	if err != nil || !bytes.Equal(data, validPayload) {
		t.Fatalf("final downloaded file corrupted")
	}
}
