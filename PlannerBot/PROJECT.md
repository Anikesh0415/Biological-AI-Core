# Project: Planner Bot — Go AI Extraction Engine with GBNF & llama.cpp

## Architecture
A modular, pure-Go extraction engine that leverages `llama.cpp` and GBNF grammar constraints to reliably extract natural language reminders into strictly valid JSON payloads (`{"task": string, "time": string}`).

```
+-------------------------------------------------------------------------------+
|                             Public Extractor API                              |
|          pkg/extractor.Engine.ExtractReminder(ctx, "call mom at 7pm")         |
+-------------------------------------------------------------------------------+
       |                                                 |
       v                                                 v
+-------------------------------+             +---------------------------------+
|     pkg/grammar & Prompt      |             |         pkg/downloader          |
| - reminder.gbnf (RFC 8259)    |             | - Auto-fetches Qwen2.5-0.5B GGUF|
| - Bounded whitespace space    |             | - Auto-fetches llama-server.exe |
| - ChatML few-shot template    |             | - Multi-tier caching & resume   |
+-------------------------------+             +---------------------------------+
       \                                                 /
        \                                               /
         v                                             v
       +-------------------------------------------------+
       |                  pkg/llm (Runtime)              |
       | - Managed llama-server.exe background process   |
       | - Dynamic port binding & /health probe polling  |
       | - HTTP client POST /completion with GBNF grammar|
       | - Clean lifecycle shutdown & Job Object cleanup |
       +-------------------------------------------------+
                                |
                                v
                Strict JSON: {"task": "...", "time": "..."}
                                |
                                v
                     json.Unmarshal Validation
```

## Feature Inventory
| # | Feature | Description | Milestone | Source |
|---|---------|-------------|-----------|--------|
| F1 | Bounded GBNF Grammar Engine | Authoritative GBNF grammar enforcing `{"task": string, "time": string}` with bounded whitespace and RFC 8259 escapes | M1 | spec_miner_survey |
| F2 | Prompt Engineering & ChatML Templates | ChatML prompt formatter with system instruction and few-shot examples for 0.5B-3B models | M1 | spec_miner_survey |
| F3 | Model Downloader & Cache Manager | Multi-tier cache resolution, resumable HTTP range streaming, and SHA-256/size verification for GGUF models | M2 | explorer_model_survey |
| F4 | Runtime Binary Bootstrapper | Automatic acquisition and caching of official prebuilt `llama-server.exe` / `llama-cli.exe` for Windows | M2 | explorer_env_survey |
| F5 | Managed Subprocess Engine | Pure Go lifecycle manager for `llama-server.exe` with dynamic port allocation, health checking, and graceful context teardown | M3 | explorer_env_survey |
| F6 | LLM Completion Client | HTTP client for `/completion` endpoint passing prompt and raw GBNF grammar string | M3 | explorer_env_survey |
| F7 | Reminder Extraction API | High-level `ExtractReminder(ctx, prompt)` API returning structured `Reminder` struct with input pre-validation | M4 | ORIGINAL_REQUEST §R1 |
| F8 | Strict JSON Output Verification | Guaranteed parseability via `json.Unmarshal` validating GBNF enforcement without external JSON pollution | M4 | ORIGINAL_REQUEST §R2 |
| F9 | Automated Go E2E Test Suite | Automated test runner verifying ≥5 distinct natural language reminder prompts end-to-end | M5 | ORIGINAL_REQUEST Acceptance Criteria |
| F10 | Adversarial & Edge Case Coverage | Robustness validation on relative dates, inverted word order, empty input, and injection resistance | M5 | spec_miner_survey |

## Milestones
| # | Name | Scope | Dependencies | Status |
|---|------|-------|-------------|--------|
| M1 | Grammar & Prompt Engine (`pkg/grammar`) | GBNF grammar definitions, validator, and ChatML prompt constructor | none | DONE |
| M2 | Model & Binary Downloader (`pkg/downloader`) | Resumable downloader for GGUF model and Windows llama.cpp binary with caching and verification | none | PLANNED |
| M3 | Managed llama.cpp Engine (`pkg/llm`) | Process supervision, port allocation, health monitoring, and HTTP completion client | M1, M2 | PLANNED |
| M4 | Core Extractor API (`pkg/extractor`) | Public API, input validation, JSON decoding, and unit integration | M3 | PLANNED |
| M5 | E2E Acceptance Test Suite & Adversarial Hardening (`test`) | Complete automated test suite (Tiers 1-4) passing all requirements, plus Tier 5 adversarial hardening | M4 | PLANNED |

## Code Layout
```
C:\Users\Anikesh\teamwork_projects\planner_bot\
├── go.mod
├── go.sum
├── pkg/
│   ├── grammar/
│   │   ├── reminder.gbnf         # Embedded authoritative GBNF grammar
│   │   ├── grammar.go            # Grammar loader & string constants
│   │   ├── prompt.go             # ChatML prompt template & builder
│   │   └── prompt_test.go        # Unit tests for prompt & grammar
│   ├── downloader/
│   │   ├── config.go             # Model & Binary configurations
│   │   ├── downloader.go         # Resumable HTTP download & cache resolver
│   │   └── downloader_test.go    # Unit tests with mock HTTP server
│   ├── llm/
│   │   ├── server.go             # Managed llama-server.exe process supervisor
│   │   ├── client.go             # HTTP /completion client with GBNF payload
│   │   ├── types.go              # Request / Response schemas
│   │   └── server_test.go        # Integration test for server lifecycle
│   └── extractor/
│       ├── reminder.go           # Reminder model & validation logic
│       ├── engine.go             # Public Engine struct & ExtractReminder API
│       └── engine_test.go        # Extractor unit tests
├── test/
│   ├── e2e_test.go               # Tier 1-4 Acceptance test suite (>=5 prompts)
│   └── adversarial_test.go       # Tier 5 Stress & adversarial tests
├── models/                       # Local model cache directory (.gitignore)
├── bin/                          # Local binary cache directory (.gitignore)
├── PROJECT.md                    # Single source of truth for project architecture
├── TEST_INFRA.md                 # E2E test plan and methodology
├── TEST_READY.md                 # Published when E2E test suite is operational
└── ORIGINAL_REQUEST.md           # Verbatim user specification
```

## Interface Contracts

### `pkg/grammar`
```go
package grammar

// ReminderGBNF contains the authoritative GBNF grammar string for {"task": string, "time": string}
const ReminderGBNF string

// BuildPrompt constructs a ChatML prompt formatted for small instruction-tuned models.
func BuildPrompt(userQuery string) string
```

### `pkg/downloader`
```go
package downloader

type ModelConfig struct {
    Name         string
    URL          string
    ExpectedSize int64
    SHA256       string
}

type Downloader struct {
    CacheDir string
}

func New(cacheDir string) (*Downloader, error)
func (d *Downloader) EnsureModel(ctx context.Context, cfg ModelConfig) (string, error)
func (d *Downloader) EnsureBinary(ctx context.Context) (string, error)
```

### `pkg/llm`
```go
package llm

type ServerConfig struct {
    BinaryPath string
    ModelPath  string
    Host       string
    Port       int    // 0 for auto-assign
    Threads    int
}

type Server struct { ... }

func StartServer(ctx context.Context, cfg ServerConfig) (*Server, error)
func (s *Server) Close() error
func (s *Server) BaseURL() string
func (s *Server) Complete(ctx context.Context, prompt, grammar string) (string, error)
```

### `pkg/extractor`
```go
package extractor

type Reminder struct {
    Task string `json:"task"`
    Time string `json:"time"`
}

type Config struct {
    ModelPath      string
    ServerBinary   string
    DownloadIfNone bool
}

type Engine struct { ... }

func NewEngine(ctx context.Context, cfg Config) (*Engine, error)
func (e *Engine) Close() error
func (e *Engine) ExtractReminder(ctx context.Context, input string) (*Reminder, error)
```
