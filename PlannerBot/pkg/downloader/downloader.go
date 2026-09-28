package downloader

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// keyLock tracks reference counts and synchronization for a canonical path.
type keyLock struct {
	refCount int
	sem      chan struct{}
}

// pathLocker manages process-wide mutual exclusion for files by canonical path.
type pathLocker struct {
	mu    sync.Mutex
	locks map[string]*keyLock
}

var globalLocker = &pathLocker{
	locks: make(map[string]*keyLock),
}

// canonicalPathKey returns a normalized, case-insensitive (on Windows) absolute path string.
func canonicalPathKey(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = filepath.Clean(path)
	}
	clean := filepath.Clean(abs)
	if runtime.GOOS == "windows" {
		return strings.ToLower(clean)
	}
	return clean
}

// lock acquires mutual exclusion on the specified file path, respecting context cancellation.
func (pl *pathLocker) lock(ctx context.Context, path string) (func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	key := canonicalPathKey(path)

	pl.mu.Lock()
	kl, exists := pl.locks[key]
	if !exists {
		kl = &keyLock{
			sem: make(chan struct{}, 1),
		}
		pl.locks[key] = kl
	}
	kl.refCount++
	pl.mu.Unlock()

	select {
	case kl.sem <- struct{}{}:
		var once sync.Once
		unlock := func() {
			once.Do(func() {
				<-kl.sem
				pl.mu.Lock()
				kl.refCount--
				if kl.refCount == 0 {
					delete(pl.locks, key)
				}
				pl.mu.Unlock()
			})
		}
		return unlock, nil

	case <-ctx.Done():
		pl.mu.Lock()
		kl.refCount--
		if kl.refCount == 0 {
			delete(pl.locks, key)
		}
		pl.mu.Unlock()
		return nil, ctx.Err()
	}
}

// Downloader manages model caching, acquisition, and verification.
type Downloader struct {
	CacheDir        string
	HTTPClient      *http.Client
	BaseRetryDelay  time.Duration
	MaxRetries      int
	OnProgress      ProgressFunc
	BinaryConfig    BinaryConfig
	HealthChecker   HealthCheckFunc
	SkipHealthCheck bool
}

// Option configures a Downloader instance.
type Option func(*Downloader)

// WithHTTPClient overrides the HTTP client used for downloads.
func WithHTTPClient(client *http.Client) Option {
	return func(d *Downloader) {
		d.HTTPClient = client
	}
}

// WithRetryDelay configures the initial exponential backoff delay.
func WithRetryDelay(delay time.Duration) Option {
	return func(d *Downloader) {
		d.BaseRetryDelay = delay
	}
}

// WithMaxRetries configures the maximum download attempts.
func WithMaxRetries(retries int) Option {
	return func(d *Downloader) {
		d.MaxRetries = retries
	}
}

// WithProgress attaches a progress callback to the downloader.
func WithProgress(fn ProgressFunc) Option {
	return func(d *Downloader) {
		d.OnProgress = fn
	}
}

// WithBinaryConfig sets the binary configuration for llama-server.
func WithBinaryConfig(cfg BinaryConfig) Option {
	return func(d *Downloader) {
		d.BinaryConfig = cfg
	}
}

// WithHealthChecker provides a custom binary health checking function.
func WithHealthChecker(fn HealthCheckFunc) Option {
	return func(d *Downloader) {
		d.HealthChecker = fn
	}
}

// WithSkipHealthCheck controls whether binary health verification (--version) is skipped.
func WithSkipHealthCheck(skip bool) Option {
	return func(d *Downloader) {
		d.SkipHealthCheck = skip
	}
}

// New initializes a Downloader instance with multi-tier cache directory defaults.
func New(cacheDir string, opts ...Option) (*Downloader, error) {
	resolvedDir, err := resolveCacheDir(cacheDir)
	if err != nil {
		return nil, fmt.Errorf("resolve cache dir: %w", err)
	}

	if err := os.MkdirAll(resolvedDir, 0755); err != nil {
		return nil, fmt.Errorf("create cache dir %q: %w", resolvedDir, err)
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   15 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   15 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	d := &Downloader{
		CacheDir: resolvedDir,
		HTTPClient: &http.Client{
			Transport: transport,
		},
		BaseRetryDelay: 2 * time.Second,
		MaxRetries:     3,
		BinaryConfig:   DefaultBinaryConfig,
	}

	for _, opt := range opts {
		opt(d)
	}

	return d, nil
}

// EnsureModel verifies if the configured model exists in any cache tier.
// If absent or corrupted, it downloads the model with resume support into CacheDir.
// Returns the absolute path to the validated .gguf file.
func (d *Downloader) EnsureModel(ctx context.Context, cfg ModelConfig) (string, error) {
	// 1. Check multi-tier cache locations before acquiring lock
	cachedPath, err := d.findExistingModel(cfg)
	if err != nil {
		return "", err
	}
	if cachedPath != "" {
		return cachedPath, nil
	}

	// 2. Target file in designated CacheDir
	finalPath := filepath.Join(d.CacheDir, cfg.Name)

	unlock, err := globalLocker.lock(ctx, finalPath)
	if err != nil {
		return "", fmt.Errorf("lock model download: %w", err)
	}
	defer unlock()

	// Double-check cache under lock
	cachedPath, err = d.findExistingModel(cfg)
	if err != nil {
		return "", err
	}
	if cachedPath != "" {
		return cachedPath, nil
	}

	// 3. Resilient download with retry and backoff
	if err := d.downloadWithRetry(ctx, cfg.URL, finalPath, cfg.ExpectedSize, cfg.SHA256); err != nil {
		return "", fmt.Errorf("model download failed for %s: %w", cfg.Name, err)
	}

	return finalPath, nil
}

// findExistingModel searches multi-tier locations in strict priority order.
func (d *Downloader) findExistingModel(cfg ModelConfig) (string, error) {
	// Tier 1: Environment variable direct file override ($PLANNER_MODEL_PATH)
	if envPath := os.Getenv("PLANNER_MODEL_PATH"); envPath != "" {
		fi, err := os.Stat(envPath)
		if err != nil || fi.IsDir() {
			return "", fmt.Errorf("PLANNER_MODEL_PATH %q does not exist or is a directory", envPath)
		}
		if cfg.ExpectedSize > 0 && fi.Size() != cfg.ExpectedSize {
			return "", fmt.Errorf("PLANNER_MODEL_PATH %q has size %d bytes, expected %d", envPath, fi.Size(), cfg.ExpectedSize)
		}
		if cfg.SHA256 != "" {
			if err := verifySHA256(envPath, cfg.SHA256); err != nil {
				return "", fmt.Errorf("PLANNER_MODEL_PATH %q checksum mismatch: %w", envPath, err)
			}
		}
		return envPath, nil
	}

	// Tier 2: Environment variable directory override ($PLANNER_MODEL_DIR)
	if envDir := os.Getenv("PLANNER_MODEL_DIR"); envDir != "" {
		candidate := filepath.Join(envDir, cfg.Name)
		if isValidCachedModel(candidate, cfg.ExpectedSize, cfg.SHA256, false) {
			return candidate, nil
		}
	}

	// Tier 3: Downloader CacheDir
	if d.CacheDir != "" {
		candidate := filepath.Join(d.CacheDir, cfg.Name)
		if isValidCachedModel(candidate, cfg.ExpectedSize, cfg.SHA256, true) {
			return candidate, nil
		}
	}

	// Tier 4: Project-local models directory (./models)
	localCandidate := filepath.Join("models", cfg.Name)
	if isValidCachedModel(localCandidate, cfg.ExpectedSize, cfg.SHA256, false) {
		if abs, err := filepath.Abs(localCandidate); err == nil {
			return abs, nil
		}
		return localCandidate, nil
	}

	// Tier 5: System user cache directory
	if userCache, err := os.UserCacheDir(); err == nil {
		sysCandidate := filepath.Join(userCache, "planner_bot", "models", cfg.Name)
		if isValidCachedModel(sysCandidate, cfg.ExpectedSize, cfg.SHA256, false) {
			return sysCandidate, nil
		}
	}

	return "", nil
}

// isValidCachedModel checks file existence, ensures it is not a directory, and validates size and hash.
func isValidCachedModel(path string, expectedSize int64, expectedSHA256 string, purgeIfInvalid bool) bool {
	fi, err := os.Stat(path)
	if err != nil || fi.IsDir() {
		return false
	}
	if expectedSize > 0 && fi.Size() != expectedSize {
		if purgeIfInvalid {
			_ = os.Remove(path)
		}
		return false
	}
	if expectedSHA256 != "" {
		if err := verifySHA256(path, expectedSHA256); err != nil {
			if purgeIfInvalid {
				_ = os.Remove(path)
			}
			return false
		}
	}
	return true
}

// resolveCacheDir establishes the target cache directory based on priority tiers.
func resolveCacheDir(explicitDir string) (string, error) {
	if explicitDir != "" {
		return filepath.Abs(explicitDir)
	}

	if envDir := os.Getenv("PLANNER_MODEL_DIR"); envDir != "" {
		return filepath.Abs(envDir)
	}

	// Check if local ./models directory exists
	if fi, err := os.Stat("models"); err == nil && fi.IsDir() {
		return filepath.Abs("models")
	}

	// Default to user cache directory
	userCache, err := os.UserCacheDir()
	if err != nil {
		home, hErr := os.UserHomeDir()
		if hErr != nil {
			return "", fmt.Errorf("user cache and home dir lookup failed: %w", err)
		}
		return filepath.Join(home, ".cache", "planner_bot", "models"), nil
	}

	return filepath.Join(userCache, "planner_bot", "models"), nil
}

// downloadWithRetry runs a download operation within an exponential backoff loop.
func (d *Downloader) downloadWithRetry(ctx context.Context, rawURL, finalPath string, expectedSize int64, expectedSHA256 string) error {
	maxRetries := d.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	backoff := d.BaseRetryDelay
	if backoff <= 0 {
		backoff = 2 * time.Second
	}

	var lastErr error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if isValidCachedModel(finalPath, expectedSize, expectedSHA256, false) {
			return nil
		}

		lastErr = d.downloadWithResume(ctx, rawURL, finalPath, expectedSize, expectedSHA256)
		if lastErr == nil {
			return nil
		}

		if ctx.Err() != nil {
			return ctx.Err()
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
			backoff *= 2
		}
	}

	return fmt.Errorf("download failed after %d attempts: %w", maxRetries, lastErr)
}

// downloadWithResume executes an HTTP GET with Range header support, writes to .part, validates, and renames.
func (d *Downloader) downloadWithResume(ctx context.Context, rawURL, finalPath string, expectedSize int64, expectedSHA256 string) error {
	partPath := finalPath + ".part"

	var existingBytes int64 = 0
	if fi, err := os.Stat(partPath); err == nil {
		existingBytes = fi.Size()
		if expectedSize > 0 {
			if existingBytes > expectedSize {
				_ = os.Remove(partPath)
				existingBytes = 0
			} else if existingBytes == expectedSize {
				// Potential completed download that missed promotion
				if expectedSHA256 == "" || verifySHA256(partPath, expectedSHA256) == nil {
					if err := atomicRename(partPath, finalPath); err != nil {
						if isValidCachedModel(finalPath, expectedSize, expectedSHA256, false) {
							return nil
						}
						return err
					}
					return nil
				}
				_ = os.Remove(partPath)
				existingBytes = 0
			}
		}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	if existingBytes > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", existingBytes))
	}

	client := d.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	var file *os.File
	var totalExpected int64 = expectedSize

	switch resp.StatusCode {
	case http.StatusPartialContent: // 206
		file, err = os.OpenFile(partPath, os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return fmt.Errorf("open part file for append: %w", err)
		}
		if totalExpected <= 0 && resp.ContentLength > 0 {
			totalExpected = existingBytes + resp.ContentLength
		}

	case http.StatusOK: // 200
		existingBytes = 0
		file, err = os.OpenFile(partPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("create part file: %w", err)
		}
		if totalExpected <= 0 && resp.ContentLength > 0 {
			totalExpected = resp.ContentLength
		}

	case http.StatusRequestedRangeNotSatisfiable: // 416
		_ = os.Remove(partPath)
		return errors.New("range not satisfiable, purged partial file")

	default:
		return fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	// Stream to file with progress tracking
	pw := &progressWriter{
		file:       file,
		downloaded: existingBytes,
		total:      totalExpected,
		onProgress: d.OnProgress,
	}

	_, copyErr := io.Copy(pw, resp.Body)
	closeErr := file.Close() // Explicit handle closure is required for Windows

	if copyErr != nil {
		return fmt.Errorf("stream copy: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close part file: %w", closeErr)
	}

	// Verify byte size
	partInfo, err := os.Stat(partPath)
	if err != nil {
		return fmt.Errorf("stat completed part file: %w", err)
	}
	if expectedSize > 0 && partInfo.Size() != expectedSize {
		_ = os.Remove(partPath)
		return fmt.Errorf("size mismatch for %s: got %d bytes, expected %d", filepath.Base(finalPath), partInfo.Size(), expectedSize)
	}

	// Verify SHA-256 checksum if configured
	if expectedSHA256 != "" {
		if err := verifySHA256(partPath, expectedSHA256); err != nil {
			_ = os.Remove(partPath)
			return fmt.Errorf("checksum mismatch for %s: %w", filepath.Base(finalPath), err)
		}
	}

	// Atomically promote .part to destination
	if err := atomicRename(partPath, finalPath); err != nil {
		if isValidCachedModel(finalPath, expectedSize, expectedSHA256, false) {
			return nil
		}
		return fmt.Errorf("promote file: %w", err)
	}

	return nil
}

// atomicRename handles Windows file locking transients with unlink and retry backoff.
func atomicRename(src, dst string) error {
	_ = os.Remove(dst)

	const maxAttempts = 5
	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		lastErr = os.Rename(src, dst)
		if lastErr == nil {
			return nil
		}

		time.Sleep(time.Duration(20*attempt) * time.Millisecond)
		_ = os.Remove(dst)
	}

	return fmt.Errorf("atomic rename failed from %s to %s: %w", src, dst, lastErr)
}

// verifySHA256 reads and checks the file's SHA-256 hash.
func verifySHA256(filePath, expectedHex string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, f); err != nil {
		return err
	}

	actualHex := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(actualHex, expectedHex) {
		return fmt.Errorf("got %s, expected %s", actualHex, expectedHex)
	}
	return nil
}

// progressWriter intercepts write calls to calculate and dispatch progress metrics.
type progressWriter struct {
	mu         sync.Mutex
	file       *os.File
	downloaded int64
	total      int64
	onProgress ProgressFunc
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.file.Write(p)
	if err != nil {
		return n, err
	}

	pw.mu.Lock()
	pw.downloaded += int64(n)
	downloaded := pw.downloaded
	total := pw.total
	fn := pw.onProgress
	pw.mu.Unlock()

	if fn != nil && total > 0 {
		pct := float64(downloaded) / float64(total) * 100.0
		fn(downloaded, total, pct)
	}
	return n, nil
}
