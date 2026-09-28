package downloader_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"planner_bot/pkg/downloader"
)

// TestChallenge_ZipSlip_PathTraversal strictly asserts that malicious path traversal entries
// such as `../../evil.exe`, `../evil.exe`, backslash variants, absolute paths, etc.
// are strictly rejected by the binary extractor and do not escape into parent directories.
func TestChallenge_ZipSlip_PathTraversal(t *testing.T) {
	traversalVectors := []struct {
		name      string
		entryPath string
	}{
		{name: "parent_two_levels", entryPath: "../../evil.exe"},
		{name: "parent_one_level", entryPath: "../evil.exe"},
		{name: "backslash_parent", entryPath: "..\\..\\evil.exe"},
		{name: "nested_parent_traversal", entryPath: "nested/../../evil.exe"},
		{name: "nested_backslash_traversal", entryPath: "nested\\..\\..\\evil.exe"},
		{name: "unix_root_escape", entryPath: "/evil.exe"},
		{name: "windows_root_escape", entryPath: "\\evil.exe"},
		{name: "windows_abs_path", entryPath: "C:\\Windows\\System32\\evil.exe"},
		{name: "windows_drive_relative", entryPath: "C:evil.exe"},
	}

	for _, tc := range traversalVectors {
		t.Run(tc.name, func(t *testing.T) {
			evilZip := createTestZip(t, map[string][]byte{
				tc.entryPath: []byte("MALICIOUS_EXECUTABLE_PAYLOAD"),
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
			res, err := dl.EnsureBinary(ctx)
			if err == nil {
				t.Fatalf("SECURITY FLAW: Zip Slip vector %q (%s) was NOT rejected! Result: %s", tc.name, tc.entryPath, res)
			}
			t.Logf("Vector %q (%s) successfully rejected: %v", tc.name, tc.entryPath, err)

			// Assert no file was written anywhere outside the designated directory
			for _, testPath := range []string{
				filepath.Join(cacheDir, "evil.exe"),
				filepath.Join(filepath.Dir(cacheDir), "evil.exe"),
			} {
				if _, statErr := os.Stat(testPath); statErr == nil {
					t.Fatalf("CRITICAL SECURITY VULNERABILITY: file was written to %s outside destination!", testPath)
				}
			}
		})
	}
}

// TestChallenge_ConcurrentEnsureModel challenges the downloader under concurrent goroutines
// downloading the same model into the same cache directory.
// Verifies whether Windows file locking errors (ERROR_SHARING_VIOLATION, ERROR_ACCESS_DENIED)
// or file promotion races occur.
func TestChallenge_ConcurrentEnsureModel(t *testing.T) {
	payload := bytes.Repeat([]byte("CONCURRENT_DOWNLOAD_CHUNK_DATA_1234567890_"), 1000) // ~43 KB
	expectedHash := sha256Hex(payload)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond) // inject slight latency to force concurrent overlap
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(payload)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(payload)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	cfg := downloader.ModelConfig{
		Name:         "concurrent-model.gguf",
		URL:          server.URL + "/concurrent-model.gguf",
		ExpectedSize: int64(len(payload)),
		SHA256:       expectedHash,
	}

	const concurrency = 8
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	results := make([]string, concurrency)
	startGate := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			dl, err := downloader.New(cacheDir, downloader.WithRetryDelay(10*time.Millisecond))
			if err != nil {
				errs[idx] = err
				return
			}
			<-startGate

			res, err := dl.EnsureModel(context.Background(), cfg)
			errs[idx] = err
			results[idx] = res
		}(i)
	}

	close(startGate)
	wg.Wait()

	var fileLockErrors []string
	var failCount int
	for i, err := range errs {
		if err != nil {
			failCount++
			errStr := err.Error()
			t.Logf("Goroutine %d failed: %v", i, err)
			lower := strings.ToLower(errStr)
			if strings.Contains(lower, "sharing violation") ||
				strings.Contains(lower, "access is denied") ||
				strings.Contains(lower, "used by another process") ||
				strings.Contains(lower, "cannot find the file specified") {
				fileLockErrors = append(fileLockErrors, fmt.Sprintf("Goroutine %d: %s", i, errStr))
			}
		} else {
			t.Logf("Goroutine %d succeeded: %s", i, results[i])
		}
	}

	if len(fileLockErrors) > 0 {
		t.Errorf("EMPIRICAL FINDING: Windows file lock / race errors observed in EnsureModel:\n%s", strings.Join(fileLockErrors, "\n"))
	}

	if failCount > 0 {
		t.Fatalf("%d out of %d concurrent EnsureModel goroutines failed!", failCount, concurrency)
	}

	finalPath := filepath.Join(cacheDir, cfg.Name)
	data, err := os.ReadFile(finalPath)
	if err != nil {
		t.Fatalf("failed to read final file: %v", err)
	}
	if !bytes.Equal(data, payload) {
		t.Fatalf("final file content corrupted")
	}
}

// TestChallenge_ConcurrentEnsureBinary challenges the downloader under concurrent goroutines
// downloading and extracting the binary archive into the same cache directory.
// Verifies whether Windows file locking errors (ERROR_SHARING_VIOLATION, ERROR_ACCESS_DENIED) occur.
func TestChallenge_ConcurrentEnsureBinary(t *testing.T) {
	exeContent := []byte("MOCK_LLAMA_SERVER_EXECUTABLE_CONTENT")
	dllContent := []byte("MOCK_LLAMA_DLL_CONTENT")

	zipBytes := createTestZip(t, map[string][]byte{
		"llama-server.exe": exeContent,
		"llama.dll":        dllContent,
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)
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

	const concurrency = 8
	var wg sync.WaitGroup
	errs := make([]error, concurrency)
	results := make([]string, concurrency)
	startGate := make(chan struct{})

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			dl, err := downloader.New(cacheDir,
				downloader.WithBinaryConfig(binConfig),
				downloader.WithSkipHealthCheck(true),
				downloader.WithRetryDelay(10*time.Millisecond),
			)
			if err != nil {
				errs[idx] = err
				return
			}
			<-startGate

			res, err := dl.EnsureBinary(context.Background())
			errs[idx] = err
			results[idx] = res
		}(i)
	}

	close(startGate)
	wg.Wait()

	var fileLockErrors []string
	var failCount int
	for i, err := range errs {
		if err != nil {
			failCount++
			errStr := err.Error()
			t.Logf("Goroutine %d failed: %v", i, err)
			lower := strings.ToLower(errStr)
			if strings.Contains(lower, "sharing violation") ||
				strings.Contains(lower, "access is denied") ||
				strings.Contains(lower, "used by another process") {
				fileLockErrors = append(fileLockErrors, fmt.Sprintf("Goroutine %d: %s", i, errStr))
			}
		} else {
			t.Logf("Goroutine %d succeeded: %s", i, results[i])
		}
	}

	if len(fileLockErrors) > 0 {
		t.Errorf("EMPIRICAL FINDING: Windows file lock errors observed in EnsureBinary:\n%s", strings.Join(fileLockErrors, "\n"))
	}

	if failCount > 0 {
		t.Fatalf("%d out of %d concurrent EnsureBinary goroutines failed!", failCount, concurrency)
	}
}
