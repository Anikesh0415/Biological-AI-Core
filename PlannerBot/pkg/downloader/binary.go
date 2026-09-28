package downloader

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// HealthCheckFunc verifies whether an executable is functional.
type HealthCheckFunc func(ctx context.Context, binPath string) error

// DefaultHealthCheck executes the binary with `--version` and expects exit code 0.
func DefaultHealthCheck(ctx context.Context, binPath string) error {
	ctxTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctxTimeout, binPath, "--version")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("health check failed for %s: %w, output: %s", binPath, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// verifyBinary checks health using the configured strategy.
func (d *Downloader) verifyBinary(ctx context.Context, cfg BinaryConfig, binPath string) error {
	if cfg.SkipHealthCheck || d.SkipHealthCheck {
		return nil
	}
	if os.Getenv("PLANNER_SKIP_HEALTH_CHECK") == "1" {
		return nil
	}
	if d.HealthChecker != nil {
		return d.HealthChecker(ctx, binPath)
	}
	return DefaultHealthCheck(ctx, binPath)
}

// EnsureBinary locates an existing healthy llama-server binary across resolution tiers,
// or automatically downloads and extracts the default prebuilt binary.
func (d *Downloader) EnsureBinary(ctx context.Context) (string, error) {
	return d.EnsureBinaryWithConfig(ctx, d.BinaryConfig)
}

// EnsureBinaryWithConfig locates or downloads the binary using the provided BinaryConfig.
func (d *Downloader) EnsureBinaryWithConfig(ctx context.Context, cfg BinaryConfig) (string, error) {
	if cfg.Name == "" {
		cfg.Name = defaultBinaryName()
	}
	if cfg.getURL() == "" && cfg.Name == DefaultBinaryConfig.Name {
		cfg.URL = DefaultBinaryConfig.URL
		cfg.ExpectedSize = DefaultBinaryConfig.ExpectedSize
		cfg.SHA256 = DefaultBinaryConfig.SHA256
	}

	cacheBinDir, err := d.resolveBinaryCacheDir(cfg)
	if err != nil {
		return "", fmt.Errorf("resolve binary cache dir: %w", err)
	}

	// Tier 1: Explicit file override via $PLANNER_SERVER_PATH
	if envPath := os.Getenv("PLANNER_SERVER_PATH"); envPath != "" {
		cleanPath := filepath.Clean(envPath)
		fi, err := os.Stat(cleanPath)
		if err != nil || fi.IsDir() {
			return "", fmt.Errorf("PLANNER_SERVER_PATH %q does not exist or is a directory: %w", envPath, err)
		}
		if err := d.verifyBinary(ctx, cfg, cleanPath); err != nil {
			return "", fmt.Errorf("PLANNER_SERVER_PATH %q failed health check: %w", envPath, err)
		}
		return cleanPath, nil
	}

	// Tier 2: Explicit directory override via $PLANNER_BIN_DIR
	if envDir := os.Getenv("PLANNER_BIN_DIR"); envDir != "" {
		candidate := filepath.Join(envDir, cfg.Name)
		if fi, err := os.Stat(candidate); err == nil && !fi.IsDir() {
			if err := d.verifyBinary(ctx, cfg, candidate); err == nil {
				return candidate, nil
			}
		}
	}

	// Tier 3: System PATH lookup
	if lookPath, err := exec.LookPath(cfg.Name); err == nil {
		if err := d.verifyBinary(ctx, cfg, lookPath); err == nil {
			return lookPath, nil
		}
	}
	if runtime.GOOS == "windows" && !strings.HasSuffix(strings.ToLower(cfg.Name), ".exe") {
		if lookPath, err := exec.LookPath(cfg.Name + ".exe"); err == nil {
			if err := d.verifyBinary(ctx, cfg, lookPath); err == nil {
				return lookPath, nil
			}
		}
	}

	// Tier 4: Project-local ./bin directory
	localCandidate := filepath.Join("bin", cfg.Name)
	if fi, err := os.Stat(localCandidate); err == nil && !fi.IsDir() {
		if err := d.verifyBinary(ctx, cfg, localCandidate); err == nil {
			if abs, err := filepath.Abs(localCandidate); err == nil {
				return abs, nil
			}
			return localCandidate, nil
		}
	}

	// Tier 5: Cache directory candidate
	cacheCandidate := filepath.Join(cacheBinDir, cfg.Name)
	if fi, err := os.Stat(cacheCandidate); err == nil && !fi.IsDir() {
		if err := d.verifyBinary(ctx, cfg, cacheCandidate); err == nil {
			return cacheCandidate, nil
		}
	}

	// Tier 6: Automated Download & Bootstrapping
	targetBinary := filepath.Join(cacheBinDir, cfg.Name)
	unlock, err := globalLocker.lock(ctx, targetBinary)
	if err != nil {
		return "", fmt.Errorf("lock binary bootstrap: %w", err)
	}
	defer unlock()

	// Double-checked validation: check Tier 5 candidate again under lock
	if fi, err := os.Stat(targetBinary); err == nil && !fi.IsDir() {
		if err := d.verifyBinary(ctx, cfg, targetBinary); err == nil {
			return targetBinary, nil
		}
	}

	if err := os.MkdirAll(cacheBinDir, 0755); err != nil {
		return "", fmt.Errorf("create binary cache dir: %w", err)
	}

	if err := d.bootstrapBinary(ctx, cfg, cacheBinDir); err != nil {
		return "", fmt.Errorf("bootstrap binary: %w", err)
	}

	if fi, err := os.Stat(targetBinary); err != nil || fi.IsDir() {
		return "", fmt.Errorf("bootstrapped binary missing at %s: %w", targetBinary, err)
	}

	if err := d.verifyBinary(ctx, cfg, targetBinary); err != nil {
		return "", fmt.Errorf("health verification failed on bootstrapped binary: %w", err)
	}

	return targetBinary, nil
}

// resolveBinaryCacheDir establishes the target folder for binaries and companion DLLs.
func (d *Downloader) resolveBinaryCacheDir(cfg BinaryConfig) (string, error) {
	subdir := cfg.BinarySubdir
	if subdir == "" {
		subdir = "bin"
	}

	if d.CacheDir != "" {
		return filepath.Join(d.CacheDir, subdir), nil
	}

	if envDir := os.Getenv("PLANNER_BIN_DIR"); envDir != "" {
		return filepath.Abs(envDir)
	}

	userCache, err := os.UserCacheDir()
	if err != nil {
		home, hErr := os.UserHomeDir()
		if hErr != nil {
			return "", fmt.Errorf("failed to obtain user cache or home dir: %w", err)
		}
		return filepath.Join(home, ".cache", "planner_bot", subdir), nil
	}

	return filepath.Join(userCache, "planner_bot", subdir), nil
}

// bootstrapBinary downloads the release zip archive and extracts binaries and DLLs into destDir.
func (d *Downloader) bootstrapBinary(ctx context.Context, cfg BinaryConfig, destDir string) error {
	zipPath := filepath.Join(destDir, fmt.Sprintf("llama-bin-%d-%d.zip", os.Getpid(), time.Now().UnixNano()))

	// Ensure any stale zip is cleared before download
	_ = os.Remove(zipPath)

	downloadURL := cfg.getURL()
	if downloadURL == "" {
		return fmt.Errorf("no download URL provided for binary %s", cfg.Name)
	}

	if err := d.downloadWithRetry(ctx, downloadURL, zipPath, cfg.ExpectedSize, cfg.SHA256); err != nil {
		return fmt.Errorf("download binary archive: %w", err)
	}
	defer os.Remove(zipPath) // Clean up zip archive after extraction

	if err := extractBinaryAndDLLs(zipPath, destDir, cfg.Name); err != nil {
		return fmt.Errorf("extract binary and DLLs: %w", err)
	}

	return nil
}

// extractBinaryAndDLLs unzips the target executable and all runtime DLLs into destDir.
func extractBinaryAndDLLs(zipPath, destDir, targetBinName string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open zip %s: %w", zipPath, err)
	}
	defer r.Close()

	cleanDest := filepath.Clean(destDir)
	targetLower := strings.ToLower(targetBinName)

	for _, f := range r.File {
		// Zip Slip protection: reject directory traversal, illegal root paths, and drive/volume specifiers (e.g. C:evil.exe)
		if filepath.IsAbs(f.Name) || strings.HasPrefix(f.Name, "/") || strings.HasPrefix(f.Name, "\\") || filepath.VolumeName(f.Name) != "" || strings.Contains(f.Name, ":") {
			return fmt.Errorf("illegal zip path escaping destination: %s", f.Name)
		}

		targetPath := filepath.Join(cleanDest, f.Name)
		cleanTarget := filepath.Clean(targetPath)
		if cleanTarget != cleanDest && !strings.HasPrefix(cleanTarget, cleanDest+string(filepath.Separator)) {
			return fmt.Errorf("illegal zip path escaping destination: %s", f.Name)
		}

		// Skip directory records early: we only extract target files and companion DLLs
		if f.FileInfo().IsDir() {
			continue
		}

		name := filepath.Base(f.Name)
		nameLower := strings.ToLower(name)

		// Filter: extract target binary and all runtime DLLs
		isTarget := nameLower == targetLower
		isDLL := strings.HasSuffix(nameLower, ".dll")
		if !isTarget && !isDLL {
			continue
		}

		destPath := filepath.Join(cleanDest, name)
		if err := extractSingleZipEntry(f, destPath); err != nil {
			return fmt.Errorf("extract %s: %w", name, err)
		}
	}

	return nil
}

// extractSingleZipEntry writes an individual zip entry safely to destPath.
func extractSingleZipEntry(f *zip.File, destPath string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, copyErr := io.Copy(outFile, rc)
	closeErr := outFile.Close() // Explicit handle closure is required for Windows

	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
