package downloader

import (
	"runtime"
)

// ModelConfig defines a GGUF model download source, expected byte size, and verification checksum.
type ModelConfig struct {
	// Name is the target filename on disk (e.g. "qwen2.5-0.5b-instruct-q4_k_m.gguf").
	Name string

	// URL is the remote HTTP(S) download endpoint. Supports standard redirects.
	URL string

	// ExpectedSize is the exact byte size of the completed file.
	// If ExpectedSize <= 0, size validation is bypassed (useful for mock tests).
	ExpectedSize int64

	// SHA256 is the hexadecimal SHA-256 checksum of the completed file.
	// If empty, hash verification is bypassed.
	SHA256 string
}

// ProgressFunc defines the signature for download progress monitoring callbacks.
// current: bytes downloaded so far.
// total: expected total bytes (or -1 if unknown).
// percent: completion percentage [0.0 - 100.0].
type ProgressFunc func(current, total int64, percent float64)

// DefaultModel is the primary recommended model for Planner Bot (Qwen2.5-0.5B-Instruct Q4_K_M).
// Verified against official Hugging Face repository metadata (size: 491,400,032 bytes).
var DefaultModel = ModelConfig{
	Name:         "qwen2.5-0.5b-instruct-q4_k_m.gguf",
	URL:          "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf",
	ExpectedSize: 491400032,
	SHA256:       "74a4da8c9fdbcd15bd1f6d01d621410d31c6fc00986f5eb687824e7b93d7a9db",
}

// FallbackModel is the secondary lightweight model (SmolLM2-135M-Instruct Q4_K_M).
// Ideal for low-bandwidth environments or minimal test suites (~105 MB).
var FallbackModel = ModelConfig{
	Name:         "smollm2-135m-instruct-q4_k_m.gguf",
	URL:          "https://huggingface.co/bartowski/SmolLM2-135M-Instruct-GGUF/resolve/main/SmolLM2-135M-Instruct-Q4_K_M.gguf",
	ExpectedSize: 105454432,
	SHA256:       "2e8040ceae7815abe0dcb3540b9995eaa1fa0d2ca9e797d0a635ae4433c68c2d",
}

// BartowskiQwenModel provides the community-quantized Qwen2.5-0.5B-Instruct alternative (~398 MB).
var BartowskiQwenModel = ModelConfig{
	Name:         "Qwen2.5-0.5B-Instruct-Q4_K_M.gguf",
	URL:          "https://huggingface.co/bartowski/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/Qwen2.5-0.5B-Instruct-Q4_K_M.gguf",
	ExpectedSize: 397808192,
	SHA256:       "6eb923e7d26e9cea28811e1a8e852009b21242fb157b26149d3b188f3a8c8653",
}

// BinaryConfig specifies parameters for acquiring and validating the llama.cpp server binary.
type BinaryConfig struct {
	// Name specifies the executable filename ("llama-server.exe" on Windows, "llama-server" on UNIX).
	Name string

	// URL is the HTTP(S) URL pointing to the prebuilt zip release.
	URL string

	// ArchiveURL is an alias for URL supported for flexible configuration.
	ArchiveURL string

	// ExpectedSize is the expected archive size in bytes (<= 0 to bypass size check).
	ExpectedSize int64

	// SHA256 is the hexadecimal SHA-256 checksum of the archive (empty to bypass check).
	SHA256 string

	// BinarySubdir specifies the subfolder within CacheDir (defaults to "bin").
	BinarySubdir string

	// SkipHealthCheck disables the active probe (--version) check (useful for mock unit tests).
	SkipHealthCheck bool
}

// getURL returns the download endpoint from either URL or ArchiveURL.
func (cfg BinaryConfig) getURL() string {
	if cfg.URL != "" {
		return cfg.URL
	}
	return cfg.ArchiveURL
}

// DefaultBinaryConfig defines the official verified prebuilt Windows llama.cpp binary release (b10941).
var DefaultBinaryConfig = BinaryConfig{
	Name:         defaultBinaryName(),
	URL:          "https://github.com/ggml-org/llama.cpp/releases/download/b10941/llama-b10941-bin-win-cpu-x64.zip",
	ExpectedSize: 18426335,
	SHA256:       "033ab72aa6fc69059e7529affa383b93201b612abbe72ea39bba103560a81cc8",
	BinarySubdir: "bin",
}

func defaultBinaryName() string {
	if runtime.GOOS == "windows" {
		return "llama-server.exe"
	}
	return "llama-server"
}
