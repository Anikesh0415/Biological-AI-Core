package downloader_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"planner_bot/pkg/downloader"
)

// helper to calculate SHA-256 hex string
func sha256Hex(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

// helper to assemble in-memory zip archives
func createTestZip(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatalf("failed to create zip entry %s: %v", name, err)
		}
		if _, err := w.Write(content); err != nil {
			t.Fatalf("failed to write zip entry content %s: %v", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatalf("failed to close zip writer: %v", err)
	}
	return buf.Bytes()
}

// helper to emulate HTTP Range requests
func handleRangeRequest(w http.ResponseWriter, r *http.Request, data []byte) {
	rangeHeader := r.Header.Get("Range")
	if rangeHeader == "" {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
		return
	}

	var start int64
	n, err := fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
	if err != nil || n != 1 || start < 0 || start >= int64(len(data)) {
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
		return
	}

	end := int64(len(data)) - 1
	w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", int64(len(data))-start))
	w.WriteHeader(http.StatusPartialContent)
	_, _ = w.Write(data[start:])
}

// -----------------------------------------------------------------------------
// Test 1: Full Download (HTTP 200)
// -----------------------------------------------------------------------------
func TestEnsureModel_FullDownload_HTTP200(t *testing.T) {
	payload := bytes.Repeat([]byte("MOCK_GGUF_MODEL_DATA_1234567890"), 50) // 1500 bytes
	expectedHash := sha256Hex(payload)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "test-model.gguf",
		URL:          server.URL + "/test-model.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	ctx := context.Background()
	modelPath, err := dl.EnsureModel(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed: %v", err)
	}

	expectedPath := filepath.Join(cacheDir, cfg.Name)
	if modelPath != expectedPath {
		t.Errorf("expected path %s, got %s", expectedPath, modelPath)
	}

	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("failed to read downloaded file: %v", err)
	}
	if !bytes.Equal(content, payload) {
		t.Errorf("downloaded content mismatch")
	}

	partPath := modelPath + ".part"
	if _, err := os.Stat(partPath); !os.IsNotExist(err) {
		t.Errorf("part file still exists at %s", partPath)
	}
}

// -----------------------------------------------------------------------------
// Test 2: Resumed Download with Range Header (HTTP 206)
// -----------------------------------------------------------------------------
func TestEnsureModel_ResumedDownload_HTTP206(t *testing.T) {
	payload := bytes.Repeat([]byte("RESUMABLE_STREAM_PAYLOAD_CHUNK_!"), 64) // 2048 bytes
	expectedHash := sha256Hex(payload)
	splitPoint := 600

	var rangeRequested atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHdr := r.Header.Get("Range")
		if rangeHdr != "" {
			rangeRequested.Store(true)
		}
		handleRangeRequest(w, r, payload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "resume-model.gguf",
		URL:          server.URL + "/resume-model.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	// Seed partial .part file with first 600 bytes
	partPath := filepath.Join(cacheDir, cfg.Name+".part")
	if err := os.WriteFile(partPath, payload[:splitPoint], 0644); err != nil {
		t.Fatalf("failed to seed part file: %v", err)
	}

	ctx := context.Background()
	modelPath, err := dl.EnsureModel(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed on resume: %v", err)
	}

	if !rangeRequested.Load() {
		t.Errorf("expected Range header to be requested from server")
	}

	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("failed to read resumed file: %v", err)
	}
	if !bytes.Equal(content, payload) {
		t.Errorf("resumed content does not match full payload")
	}
}

// -----------------------------------------------------------------------------
// Test 3: Resume Fallback when Server Ignores Range (HTTP 200)
// -----------------------------------------------------------------------------
func TestEnsureModel_ResumeServerIgnoresRange_HTTP200(t *testing.T) {
	payload := []byte("NEW_FULL_PAYLOAD_WHEN_SERVER_IGNORES_RANGE_HEADER_123")
	expectedHash := sha256Hex(payload)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server ignores Range header and returns full HTTP 200
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "ignore-range.gguf",
		URL:          server.URL + "/ignore-range.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	// Stale part file with garbage data
	partPath := filepath.Join(cacheDir, cfg.Name+".part")
	_ = os.WriteFile(partPath, []byte("STALE_GARBAGE_BYTES_"), 0644)

	ctx := context.Background()
	modelPath, err := dl.EnsureModel(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed: %v", err)
	}

	content, err := os.ReadFile(modelPath)
	if err != nil {
		t.Fatalf("failed to read completed file: %v", err)
	}
	if !bytes.Equal(content, payload) {
		t.Errorf("expected clean overwrite with payload, got %q", string(content))
	}
}

// -----------------------------------------------------------------------------
// Test 4: Stale / Corrupt Range Reset (HTTP 416 -> Recovery)
// -----------------------------------------------------------------------------
func TestEnsureModel_ResumeRangeNotSatisfiable_HTTP416(t *testing.T) {
	payload := []byte("VALID_PAYLOAD_AFTER_416_RECOVERY")
	expectedHash := sha256Hex(payload)

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqNum := requestCount.Add(1)
		rangeHdr := r.Header.Get("Range")

		if reqNum == 1 && rangeHdr != "" {
			// First attempt sends range: return 416
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		// Subsequent attempt without range or after purge
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "range-416-model.gguf",
		URL:          server.URL + "/range-416-model.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	// Seed part file that causes range error
	partPath := filepath.Join(cacheDir, cfg.Name+".part")
	_ = os.WriteFile(partPath, []byte("STALE_OFFSET_BYTES_"), 0644)

	ctx := context.Background()
	modelPath, err := dl.EnsureModel(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed after 416: %v", err)
	}

	content, err := os.ReadFile(modelPath)
	if err != nil || !bytes.Equal(content, payload) {
		t.Errorf("failed to recover clean payload after 416")
	}
}

// -----------------------------------------------------------------------------
// Test 5: Existing Valid Cached File Bypass (Zero Network Calls)
// -----------------------------------------------------------------------------
func TestEnsureModel_ValidCacheBypass(t *testing.T) {
	payload := []byte("CACHED_GGUF_VALID_BYTES_1234567890")
	expectedHash := sha256Hex(payload)

	var requestCount atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount.Add(1)
		t.Errorf("mock server should NOT be called when cache is valid")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "cached-model.gguf",
		URL:          server.URL + "/cached-model.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	// Seed valid file into cacheDir
	cachedPath := filepath.Join(cacheDir, cfg.Name)
	if err := os.WriteFile(cachedPath, payload, 0644); err != nil {
		t.Fatalf("failed to seed cached file: %v", err)
	}

	ctx := context.Background()
	start := time.Now()
	resultPath, err := dl.EnsureModel(ctx, cfg)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("EnsureModel failed on cached file: %v", err)
	}
	if resultPath != cachedPath {
		t.Errorf("expected cached path %s, got %s", cachedPath, resultPath)
	}
	if count := requestCount.Load(); count != 0 {
		t.Errorf("expected 0 HTTP requests, got %d", count)
	}
	if elapsed > 100*time.Millisecond {
		t.Errorf("cache bypass took too long: %v (expected < 100ms)", elapsed)
	}
}

// -----------------------------------------------------------------------------
// Test 6: Corrupted / Size-Mismatched File Recovery
// -----------------------------------------------------------------------------
func TestEnsureModel_CorruptedCacheRecovery_SizeMismatch(t *testing.T) {
	validPayload := bytes.Repeat([]byte("PROPER_VALID_BYTES_"), 30) // 600 bytes
	expectedHash := sha256Hex(validPayload)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(validPayload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(validPayload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "re-download-model.gguf",
		URL:          server.URL + "/re-download-model.gguf",
		ExpectedSize: int64(len(validPayload)),
		SHA256:       expectedHash,
	}

	// Seed truncated corrupt file
	corruptedPath := filepath.Join(cacheDir, cfg.Name)
	_ = os.WriteFile(corruptedPath, []byte("TRUNCATED"), 0644)

	ctx := context.Background()
	resultPath, err := dl.EnsureModel(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed to recover corrupt file: %v", err)
	}

	content, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("failed to read recovered file: %v", err)
	}
	if !bytes.Equal(content, validPayload) {
		t.Errorf("recovered content mismatch")
	}
}

// -----------------------------------------------------------------------------
// Test 7: Checksum Hash Mismatch Deletion & Recovery
// -----------------------------------------------------------------------------
func TestEnsureModel_CorruptedCacheRecovery_HashMismatch(t *testing.T) {
	validPayload := []byte("REAL_PAYLOAD_WITH_ACCURATE_HASH_VALUES_123456789")
	expectedHash := sha256Hex(validPayload)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(validPayload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(validPayload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "hash-check-model.gguf",
		URL:          server.URL + "/hash-check-model.gguf",
		ExpectedSize: int64(len(validPayload)),
		SHA256:       expectedHash,
	}

	// Seed file with same size but wrong contents
	badPayload := bytes.Repeat([]byte("X"), len(validPayload))
	corruptedPath := filepath.Join(cacheDir, cfg.Name)
	_ = os.WriteFile(corruptedPath, badPayload, 0644)

	ctx := context.Background()
	resultPath, err := dl.EnsureModel(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed to recover hash-mismatched file: %v", err)
	}

	content, err := os.ReadFile(resultPath)
	if err != nil {
		t.Fatalf("failed to read recovered file: %v", err)
	}
	if !bytes.Equal(content, validPayload) {
		t.Errorf("content does not match genuine payload")
	}
}

// -----------------------------------------------------------------------------
// Test 8: Network Retry Logic with Exponential Backoff (HTTP 500 / 503)
// -----------------------------------------------------------------------------
func TestEnsureModel_NetworkRetryAndBackoff_Transient500(t *testing.T) {
	payload := []byte("RETRY_TEST_PAYLOAD_SUCCESS_123456")
	expectedHash := sha256Hex(payload)

	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		curr := attempts.Add(1)
		switch curr {
		case 1:
			w.WriteHeader(http.StatusInternalServerError) // 500
		case 2:
			w.WriteHeader(http.StatusServiceUnavailable) // 503
		default:
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write(payload)
		}
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond), downloader.WithMaxRetries(3))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "retry-model.gguf",
		URL:          server.URL + "/retry-model.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	ctx := context.Background()
	start := time.Now()
	resultPath, err := dl.EnsureModel(ctx, cfg)
	elapsed := time.Since(start)

	if err != nil {
		t.Fatalf("EnsureModel should succeed on attempt 3, failed: %v", err)
	}
	if attempts.Load() != 3 {
		t.Errorf("expected exactly 3 attempts, got %d", attempts.Load())
	}
	if elapsed > 500*time.Millisecond {
		t.Errorf("retries took too long with 1ms backoff: %v", elapsed)
	}

	content, err := os.ReadFile(resultPath)
	if err != nil || !bytes.Equal(content, payload) {
		t.Errorf("downloaded file corrupted or unreadable")
	}
}

// -----------------------------------------------------------------------------
// Test 9: Retry Exhaustion (Permanent HTTP 500)
// -----------------------------------------------------------------------------
func TestEnsureModel_RetryExhausted_Permanent500(t *testing.T) {
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond), downloader.WithMaxRetries(3))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "failing-model.gguf",
		URL:          server.URL + "/failing-model.gguf",
		ExpectedSize: 100,
	}

	ctx := context.Background()
	_, err = dl.EnsureModel(ctx, cfg)
	if err == nil {
		t.Fatalf("expected error after exhausted retries, got nil")
	}
	if attempts.Load() != 3 {
		t.Errorf("expected 3 retry attempts, got %d", attempts.Load())
	}
}

// -----------------------------------------------------------------------------
// Test 10: Context Cancellation During Download
// -----------------------------------------------------------------------------
func TestEnsureModel_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(1*time.Millisecond))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name: "cancel-model.gguf",
		URL:  server.URL + "/cancel-model.gguf",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	_, err = dl.EnsureModel(ctx, cfg)
	if err == nil {
		t.Fatalf("expected context deadline error, got nil")
	}
}

// -----------------------------------------------------------------------------
// Test 11: Zip Extraction and Binary Resolution (`llama-server.exe` + DLLs)
// -----------------------------------------------------------------------------
func TestEnsureBinary_ZipExtractionAndResolution(t *testing.T) {
	exeContent := []byte("MOCK_LLAMA_SERVER_EXECUTABLE_BINARY_BYTES")
	llamaDll := []byte("MOCK_LLAMA_DLL_BYTES")
	ggmlDll := []byte("MOCK_GGML_DLL_BYTES")
	ompDll := []byte("MOCK_LIBOMP_DLL_BYTES")

	zipBytes := createTestZip(t, map[string][]byte{
		"llama-server.exe": exeContent,
		"llama.dll":        llamaDll,
		"ggml.dll":         ggmlDll,
		"libomp.dll":       ompDll,
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
	binConfig := downloader.BinaryConfig{
		Name:            "llama-server.exe",
		ArchiveURL:      server.URL + "/llama-bin.zip",
		BinarySubdir:    "bin",
		SkipHealthCheck: true,
	}

	dl, err := downloader.New(cacheDir, downloader.WithBinaryConfig(binConfig), downloader.WithSkipHealthCheck(true))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	ctx := context.Background()

	// 1. First invocation: downloads and extracts zip
	binaryPath, err := dl.EnsureBinary(ctx)
	if err != nil {
		t.Fatalf("EnsureBinary failed on initial download: %v", err)
	}

	expectedBinPath := filepath.Join(cacheDir, "bin", "llama-server.exe")
	if binaryPath != expectedBinPath {
		t.Errorf("expected binary path %s, got %s", expectedBinPath, binaryPath)
	}

	// Verify all companion DLLs are placed alongside the executable
	for _, dllName := range []string{"llama.dll", "ggml.dll", "libomp.dll"} {
		dllPath := filepath.Join(cacheDir, "bin", dllName)
		if fi, err := os.Stat(dllPath); err != nil || fi.Size() == 0 {
			t.Errorf("expected runtime DLL %s to exist in bin dir, err: %v", dllName, err)
		}
	}

	// 2. Second invocation: cache hit, zero additional downloads
	secondPath, err := dl.EnsureBinary(ctx)
	if err != nil {
		t.Fatalf("EnsureBinary failed on second call: %v", err)
	}
	if secondPath != binaryPath {
		t.Errorf("expected same binary path on cache hit")
	}
	if hits := serverHits.Load(); hits != 1 {
		t.Errorf("expected exactly 1 server hit (cached hit on call 2), got %d", hits)
	}
}

// -----------------------------------------------------------------------------
// Test 12: Zip Slip Security Protection
// -----------------------------------------------------------------------------
func TestEnsureBinary_ZipSlipProtection(t *testing.T) {
	evilZip := createTestZip(t, map[string][]byte{
		"../evil.exe": []byte("MALICIOUS_PAYLOAD"),
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/zip")
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(evilZip)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(evilZip)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	binConfig := downloader.BinaryConfig{
		Name:            "llama-server.exe",
		ArchiveURL:      server.URL + "/evil.zip",
		BinarySubdir:    "bin",
		SkipHealthCheck: true,
	}

	dl, err := downloader.New(cacheDir, downloader.WithBinaryConfig(binConfig), downloader.WithSkipHealthCheck(true))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	ctx := context.Background()
	_, err = dl.EnsureBinary(ctx)
	if err == nil {
		t.Fatalf("expected Zip Slip traversal attempt to fail with security error, got nil")
	}
}

// -----------------------------------------------------------------------------
// Test 13: Environment Variable Overrides ($PLANNER_SERVER_PATH)
// -----------------------------------------------------------------------------
func TestEnsureBinary_EnvironmentOverride(t *testing.T) {
	tempDir := t.TempDir()
	dummyServer := filepath.Join(tempDir, "mock-server.exe")
	if err := os.WriteFile(dummyServer, []byte("CUSTOM_SERVER_EXE"), 0755); err != nil {
		t.Fatalf("failed to write dummy server: %v", err)
	}

	t.Setenv("PLANNER_SERVER_PATH", dummyServer)

	dl, err := downloader.New(t.TempDir(), downloader.WithSkipHealthCheck(true))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	ctx := context.Background()
	res, err := dl.EnsureBinary(ctx)
	if err != nil {
		t.Fatalf("EnsureBinary with env override failed: %v", err)
	}
	if res != dummyServer {
		t.Errorf("expected %s from PLANNER_SERVER_PATH, got %s", dummyServer, res)
	}
}

// -----------------------------------------------------------------------------
// Test 14: Progress Reporting Callback Verification
// -----------------------------------------------------------------------------
func TestEnsureModel_ProgressReporting(t *testing.T) {
	payload := bytes.Repeat([]byte("0123456789"), 100) // 1000 bytes

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	var callCount atomic.Int32
	var finalPercent atomic.Int64

	progressFn := func(current, total int64, pct float64) {
		callCount.Add(1)
		finalPercent.Store(int64(pct))
	}

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir, downloader.WithProgress(progressFn))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "progress-model.gguf",
		URL:          server.URL + "/progress-model.gguf",
		ExpectedSize: int64(len(payload)),
	}

	ctx := context.Background()
	_, err = dl.EnsureModel(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed: %v", err)
	}

	if callCount.Load() == 0 {
		t.Errorf("expected progress callback to be called at least once")
	}
	if finalPercent.Load() != 100 {
		t.Errorf("expected final progress to be 100%%, got %d%%", finalPercent.Load())
	}
}

// -----------------------------------------------------------------------------
// Test 15: Genuine Subprocess Health Check Execution (--version)
// -----------------------------------------------------------------------------
func TestVerifyBinaryHealth_LiveSubprocessExecution(t *testing.T) {
	tempDir := t.TempDir()
	sourceFile := filepath.Join(tempDir, "main.go")
	exeFile := filepath.Join(tempDir, "mock_llama_server.exe")

	src := []byte(`package main

import (
	"fmt"
	"os"
)

func main() {
	for _, arg := range os.Args[1:] {
		if arg == "--version" {
			fmt.Println("version: 0.4.0-test (mock)")
			os.Exit(0)
		}
	}
	os.Exit(1)
}
`)
	if err := os.WriteFile(sourceFile, src, 0644); err != nil {
		t.Fatalf("failed to write source: %v", err)
	}

	// Compile the real mock binary using the Go toolchain
	cmd := exec.Command("go", "build", "-o", exeFile, sourceFile)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("failed to compile test binary: %v, output: %s", err, string(out))
	}

	ctx := context.Background()

	// 1. Positive test: genuine executable returning exit code 0 on --version
	if err := downloader.DefaultHealthCheck(ctx, exeFile); err != nil {
		t.Errorf("expected DefaultHealthCheck to pass on genuine binary, got error: %v", err)
	}

	// 2. Negative test: corrupt non-executable file failing execution
	dummyFile := filepath.Join(tempDir, "dummy_corrupt.exe")
	if err := os.WriteFile(dummyFile, []byte("NOT_A_VALID_PE_FILE"), 0755); err != nil {
		t.Fatalf("failed to write dummy file: %v", err)
	}
	if err := downloader.DefaultHealthCheck(ctx, dummyFile); err == nil {
		t.Errorf("expected DefaultHealthCheck to fail on invalid dummy file, got nil")
	}
}

// -----------------------------------------------------------------------------
// Test 16: Preset Model and Binary Configurations Sanity
// -----------------------------------------------------------------------------
func TestPresets_Sanity(t *testing.T) {
	if downloader.DefaultModel.Name == "" || downloader.DefaultModel.URL == "" {
		t.Errorf("DefaultModel is missing Name or URL")
	}
	if downloader.DefaultModel.ExpectedSize != 491400032 {
		t.Errorf("DefaultModel ExpectedSize should be 491400032, got %d", downloader.DefaultModel.ExpectedSize)
	}
	if len(downloader.DefaultModel.SHA256) != 64 {
		t.Errorf("DefaultModel SHA256 hex length should be 64, got %d", len(downloader.DefaultModel.SHA256))
	}

	if downloader.FallbackModel.Name == "" || downloader.FallbackModel.URL == "" {
		t.Errorf("FallbackModel is missing Name or URL")
	}
	if downloader.FallbackModel.ExpectedSize != 105454432 {
		t.Errorf("FallbackModel ExpectedSize should be 105454432, got %d", downloader.FallbackModel.ExpectedSize)
	}
	if len(downloader.FallbackModel.SHA256) != 64 {
		t.Errorf("FallbackModel SHA256 hex length should be 64, got %d", len(downloader.FallbackModel.SHA256))
	}

	if downloader.DefaultBinaryConfig.Name == "" || downloader.DefaultBinaryConfig.URL == "" {
		t.Errorf("DefaultBinaryConfig is missing Name or URL")
	}
}

// -----------------------------------------------------------------------------
// Test 17: Pre-completed .part File Direct Promotion
// -----------------------------------------------------------------------------
func TestEnsureModel_AlreadyCompletePartFilePromotesWithoutNetwork(t *testing.T) {
	payload := []byte("PRE_COMPLETED_PART_FILE_DATA_123456789")
	expectedHash := sha256Hex(payload)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("server should NOT be queried when .part is already complete")
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dl, err := downloader.New(cacheDir)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "already-done.gguf",
		URL:          server.URL + "/already-done.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	// Write fully complete .part file
	partPath := filepath.Join(cacheDir, cfg.Name+".part")
	if err := os.WriteFile(partPath, payload, 0644); err != nil {
		t.Fatalf("failed to write part file: %v", err)
	}

	ctx := context.Background()
	finalPath, err := dl.EnsureModel(ctx, cfg)
	if err != nil {
		t.Fatalf("EnsureModel failed: %v", err)
	}

	expectedFinal := filepath.Join(cacheDir, cfg.Name)
	if finalPath != expectedFinal {
		t.Errorf("expected final path %s, got %s", expectedFinal, finalPath)
	}

	content, err := os.ReadFile(finalPath)
	if err != nil || !bytes.Equal(content, payload) {
		t.Errorf("promoted content mismatch")
	}
}

// -----------------------------------------------------------------------------
// Test 18: Model Environment Overrides ($PLANNER_MODEL_PATH & $PLANNER_MODEL_DIR)
// -----------------------------------------------------------------------------
func TestEnsureModel_EnvironmentOverrides(t *testing.T) {
	tempDir := t.TempDir()
	validPayload := []byte("ENV_OVERRIDE_MODEL_BYTES_12345")
	validHash := sha256Hex(validPayload)

	validModelPath := filepath.Join(tempDir, "custom-model.gguf")
	if err := os.WriteFile(validModelPath, validPayload, 0644); err != nil {
		t.Fatalf("failed to write valid model: %v", err)
	}

	dl, err := downloader.New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	cfg := downloader.ModelConfig{
		Name:         "custom-model.gguf",
		URL:          "http://unreachable-host.local/model.gguf",
		ExpectedSize: int64(len(validPayload)),
		SHA256:       validHash,
	}

	// 1. Valid $PLANNER_MODEL_PATH
	t.Setenv("PLANNER_MODEL_PATH", validModelPath)
	res, err := dl.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("EnsureModel with PLANNER_MODEL_PATH failed: %v", err)
	}
	if res != validModelPath {
		t.Errorf("expected %s, got %s", validModelPath, res)
	}

	// 2. Corrupt $PLANNER_MODEL_PATH (wrong size)
	t.Setenv("PLANNER_MODEL_PATH", validModelPath)
	badSizeCfg := cfg
	badSizeCfg.ExpectedSize = 99999
	_, err = dl.EnsureModel(context.Background(), badSizeCfg)
	if err == nil {
		t.Fatalf("expected error on corrupt PLANNER_MODEL_PATH size, got nil")
	}

	// 3. Valid $PLANNER_MODEL_DIR
	t.Setenv("PLANNER_MODEL_PATH", "")
	t.Setenv("PLANNER_MODEL_DIR", tempDir)
	resDir, err := dl.EnsureModel(context.Background(), cfg)
	if err != nil {
		t.Fatalf("EnsureModel with PLANNER_MODEL_DIR failed: %v", err)
	}
	if resDir != validModelPath {
		t.Errorf("expected %s, got %s", validModelPath, resDir)
	}
}

// -----------------------------------------------------------------------------
// Test 19: Binary Environment Overrides ($PLANNER_BIN_DIR & Invalid Server Path)
// -----------------------------------------------------------------------------
func TestEnsureBinary_BinDirAndInvalidServerPath(t *testing.T) {
	tempDir := t.TempDir()
	binName := "llama-server.exe"
	mockExe := filepath.Join(tempDir, binName)
	if err := os.WriteFile(mockExe, []byte("MOCK_EXE"), 0755); err != nil {
		t.Fatalf("failed to write mock exe: %v", err)
	}

	dl, err := downloader.New(t.TempDir(), downloader.WithSkipHealthCheck(true))
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	// 1. $PLANNER_BIN_DIR override
	t.Setenv("PLANNER_BIN_DIR", tempDir)
	res, err := dl.EnsureBinary(context.Background())
	if err != nil {
		t.Fatalf("EnsureBinary with PLANNER_BIN_DIR failed: %v", err)
	}
	if res != mockExe {
		t.Errorf("expected %s, got %s", mockExe, res)
	}

	// 2. Invalid $PLANNER_SERVER_PATH pointing to non-existent file
	t.Setenv("PLANNER_SERVER_PATH", filepath.Join(tempDir, "does-not-exist.exe"))
	_, err = dl.EnsureBinary(context.Background())
	if err == nil {
		t.Fatalf("expected error for non-existent PLANNER_SERVER_PATH, got nil")
	}
}

// -----------------------------------------------------------------------------
// Test 20: Options (WithHTTPClient & WithHealthChecker)
// -----------------------------------------------------------------------------
func TestDownloader_OptionsCoverage(t *testing.T) {
	customClient := &http.Client{Timeout: 5 * time.Second}
	var healthCheckCalled atomic.Bool

	customHealthChecker := func(ctx context.Context, binPath string) error {
		healthCheckCalled.Store(true)
		return nil
	}

	tempDir := t.TempDir()
	binPath := filepath.Join(tempDir, "dummy-server.exe")
	_ = os.WriteFile(binPath, []byte("DUMMY"), 0755)

	t.Setenv("PLANNER_SERVER_PATH", binPath)

	dl, err := downloader.New(
		tempDir,
		downloader.WithHTTPClient(customClient),
		downloader.WithHealthChecker(customHealthChecker),
	)
	if err != nil {
		t.Fatalf("failed to create downloader: %v", err)
	}

	if dl.HTTPClient != customClient {
		t.Errorf("custom HTTP client not set")
	}

	res, err := dl.EnsureBinary(context.Background())
	if err != nil {
		t.Fatalf("EnsureBinary failed: %v", err)
	}
	if res != binPath {
		t.Errorf("expected %s, got %s", binPath, res)
	}
	if !healthCheckCalled.Load() {
		t.Errorf("expected custom health checker to be called")
	}
}

// -----------------------------------------------------------------------------
// Test 21: Cache Directory Resolution Fallbacks
// -----------------------------------------------------------------------------
func TestDownloader_ResolveCacheDirFallbacks(t *testing.T) {
	// Explicit cache dir
	explicit := t.TempDir()
	dl1, err := downloader.New(explicit)
	if err != nil || dl1.CacheDir != explicit {
		t.Errorf("expected CacheDir %s, got %s", explicit, dl1.CacheDir)
	}

	// PLANNER_MODEL_DIR override
	envDir := t.TempDir()
	t.Setenv("PLANNER_MODEL_DIR", envDir)
	dl2, err := downloader.New("")
	if err != nil || dl2.CacheDir != envDir {
		t.Errorf("expected CacheDir %s from PLANNER_MODEL_DIR, got %s", envDir, dl2.CacheDir)
	}
}

