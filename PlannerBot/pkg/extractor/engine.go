package extractor

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"

	"planner_bot/pkg/downloader"
	"planner_bot/pkg/grammar"
	"planner_bot/pkg/llm"
	"planner_bot/pkg/screentime"
)

// Config holds configuration for the extraction engine.
type Config struct {
	// ModelPath is the absolute path to a .gguf model file.
	// If empty and DownloadIfNone is true, the model is auto-downloaded.
	ModelPath string

	// ServerBinary is the absolute path to the llama-server executable.
	// If empty and DownloadIfNone is true, the binary is auto-downloaded.
	ServerBinary string

	// DownloadIfNone enables automatic downloading of model and binary when not found.
	DownloadIfNone bool

	// CacheDir specifies the directory for downloaded models and binaries.
	// Defaults to the user cache directory if empty.
	CacheDir string

	// Threads is the number of CPU threads for inference. Defaults to 4.
	Threads int

	// ContextSize is the context window size in tokens. Defaults to 2048.
	ContextSize int
}

// Engine is the high-level extraction engine that manages the LLM server and
// translates natural language prompts into structured Reminder JSON.
type Engine struct {
	server     *llm.Server
	grammarStr string
	closeOnce  sync.Once
	closeErr   error
}

// NewEngine initializes the extraction engine by resolving (or downloading) the model
// and binary, booting the llama-server, and preparing the GBNF grammar.
func NewEngine(ctx context.Context, cfg Config) (*Engine, error) {
	modelPath := cfg.ModelPath
	binaryPath := cfg.ServerBinary

	// Auto-download model and binary if paths are not provided
	if modelPath == "" || binaryPath == "" {
		if !cfg.DownloadIfNone {
			return nil, fmt.Errorf("extractor: ModelPath and ServerBinary are required when DownloadIfNone is false")
		}

		dl, err := downloader.New(cfg.CacheDir)
		if err != nil {
			return nil, fmt.Errorf("extractor: create downloader: %w", err)
		}

		// Resolve model
		if modelPath == "" {
			var dlErr error
			modelPath, dlErr = dl.EnsureModel(ctx, downloader.DefaultModel)
			if dlErr != nil {
				// Try fallback model
				modelPath, dlErr = dl.EnsureModel(ctx, downloader.FallbackModel)
				if dlErr != nil {
					return nil, fmt.Errorf("extractor: failed to acquire model (tried default and fallback): %w", dlErr)
				}
			}
		}

		// Resolve binary
		if binaryPath == "" {
			var dlErr error
			binaryPath, dlErr = dl.EnsureBinary(ctx)
			if dlErr != nil {
				return nil, fmt.Errorf("extractor: failed to acquire llama-server binary: %w", dlErr)
			}
		}
	}

	// Apply defaults
	threads := cfg.Threads
	if threads <= 0 {
		threads = 4
	}
	ctxSize := cfg.ContextSize
	if ctxSize <= 0 {
		ctxSize = 8192
	}

	// Start the LLM server
	serverCfg := llm.ServerConfig{
		BinaryPath:  binaryPath,
		ModelPath:   modelPath,
		Host:        "127.0.0.1",
		Port:        0, // Auto-assign
		Threads:     threads,
		ContextSize: ctxSize,
		GPULayers:   0,
	}

	srv, err := llm.StartServer(ctx, serverCfg)
	if err != nil {
		return nil, fmt.Errorf("extractor: start LLM server: %w", err)
	}

	grammarStr := grammar.GetGrammar()

	return &Engine{
		server:     srv,
		grammarStr: grammarStr,
	}, nil
}

// ExtractReminder takes a natural language input string and returns a structured Reminder.
// The LLM output is constrained by GBNF grammar to guarantee valid JSON.
func (e *Engine) ExtractReminder(ctx context.Context, input string) (*Reminder, error) {
	// Build the ChatML prompt with validation
	prompt, err := grammar.ValidateAndBuildPrompt(input)
	if err != nil {
		return nil, fmt.Errorf("extractor: %w", err)
	}

	// Get completion from the LLM with GBNF grammar enforcement
	rawOutput, err := e.server.Complete(ctx, prompt, e.grammarStr)
	if err != nil {
		return nil, fmt.Errorf("extractor: LLM completion failed: %w", err)
	}

	rawOutput = strings.TrimSpace(rawOutput)
	if rawOutput == "" {
		return nil, fmt.Errorf("extractor: LLM returned empty output")
	}

	// Parse the JSON output into a Reminder struct
	var reminder Reminder
	if err := json.Unmarshal([]byte(rawOutput), &reminder); err != nil {
		return nil, fmt.Errorf("extractor: failed to parse LLM output as JSON: %w\nraw output: %s", err, rawOutput)
	}

	reminder.rawJSON = rawOutput

	return &reminder, nil
}

// ExtractReminderWithContext extracts a structured reminder utilizing conversational history and active tasks.
func (e *Engine) ExtractReminderWithContext(ctx context.Context, input string, history []grammar.HistoryTurn, tasks []grammar.TaskContext) (*Reminder, error) {
	prompt, err := grammar.BuildPromptWithContext(input, history, tasks)
	if err != nil {
		return nil, fmt.Errorf("extractor: %w", err)
	}

	rawOutput, err := e.server.Complete(ctx, prompt, e.grammarStr)
	if err != nil {
		return nil, fmt.Errorf("extractor: LLM completion failed: %w", err)
	}

	rawOutput = strings.TrimSpace(rawOutput)
	if rawOutput == "" {
		return nil, fmt.Errorf("extractor: LLM returned empty output")
	}

	var reminder Reminder
	if err := json.Unmarshal([]byte(rawOutput), &reminder); err != nil {
		return nil, fmt.Errorf("extractor: failed to parse LLM output as JSON: %w\nraw output: %s", err, rawOutput)
	}

	reminder.rawJSON = rawOutput
	return &reminder, nil
}

// GenerateChatReply runs conversational inference over past dialogue and active tasks.
// Output is free-form conversational text (no GBNF constraint).
func (e *Engine) GenerateChatReply(ctx context.Context, history []grammar.HistoryTurn, tasks []grammar.TaskContext, userMessage string, currentTimeStr string, briefingContext ...string) (string, error) {
	prompt := grammar.BuildConversationalPrompt(history, tasks, userMessage, currentTimeStr, briefingContext...)

	req := llm.CompletionRequest{
		Prompt:        prompt,
		Temperature:   0.6,
		RepeatPenalty: 1.15,
		NPredict:      256,
		CachePrompt:   true,
		Stop:          []string{"<|im_end|>", "<|endoftext|>", "<|im_start|>"},
	}

	reply, err := e.server.CompleteWithOptions(ctx, req)
	if err != nil {
		return "", fmt.Errorf("extractor: generate chat reply: %w", err)
	}

	return strings.TrimSpace(reply), nil
}

// Server returns the underlying managed LLM server instance.
func (e *Engine) Server() *llm.Server {
	return e.server
}

// AnalyzeScreentime passes the parsed screentime metrics through the cognitive habit coach prompt.
func (e *Engine) AnalyzeScreentime(ctx context.Context, report *screentime.ScreentimeReport) (string, error) {
	if e.server == nil {
		return "", fmt.Errorf("extractor: llm server not initialized")
	}
	prompt := report.BuildPrompt()
	req := llm.CompletionRequest{
		Prompt:      prompt,
		Temperature: 0.35,
		NPredict:    320,
		CachePrompt: true,
		Stop:        []string{"<|im_end|>", "<|endoftext|>", "<|im_start|>"},
	}
	reply, err := e.server.CompleteWithOptions(ctx, req)
	if err != nil {
		return "", fmt.Errorf("extractor: screentime coach completion failed: %w", err)
	}
	return strings.TrimSpace(reply), nil
}

// GoalMilestone represents one step in a 3-step Socratic breakdown.
type GoalMilestone struct {
	Step        int    `json:"step"`
	Title       string `json:"title"`
	DurationEst string `json:"duration_est"`
	Rationale   string `json:"rationale"`
}

// GoalBreakdown encapsulates a 3-step project decomposition following the Rule of Three.
type GoalBreakdown struct {
	Goal       string          `json:"goal"`
	Overview   string          `json:"overview"`
	Milestones []GoalMilestone `json:"milestones"`
}

// DecomposeGoal breaks down a complex goal into exactly 3 actionable milestones with time estimates.
func (e *Engine) DecomposeGoal(ctx context.Context, goal string) (*GoalBreakdown, error) {
	if e.server == nil {
		return nil, fmt.Errorf("extractor: llm server not initialized")
	}

	prompt := fmt.Sprintf(`<|im_start|>system
You are a strategic productivity coach. Break the user's project goal into exactly 3 milestones:
- Step 1: Immediate low-friction entry point (25-45 mins)
- Step 2: Core deep work execution block (60-90 mins)
- Step 3: Polish, testing, or review (30-45 mins)
Output valid JSON only:
{
  "overview": "1 concise sentence summarizing execution strategy",
  "milestones": [
    {"step": 1, "title": "...", "duration_est": "30 mins", "rationale": "..."},
    {"step": 2, "title": "...", "duration_est": "90 mins", "rationale": "..."},
    {"step": 3, "title": "...", "duration_est": "45 mins", "rationale": "..."}
  ]
}
<|im_end|>
<|im_start|>user
Project Goal: %s
<|im_end|>
<|im_start|>assistant
`, goal)

	req := llm.CompletionRequest{
		Prompt:      prompt,
		Temperature: 0.25,
		NPredict:    320,
		CachePrompt: true,
		Stop:        []string{"<|im_end|>", "<|endoftext|>"},
	}

	reply, err := e.server.CompleteWithOptions(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("extractor: decompose goal: %w", err)
	}

	cleanJSON := extractJSONBlock(reply)
	var breakdown GoalBreakdown
	if err := json.Unmarshal([]byte(cleanJSON), &breakdown); err != nil {
		// Fallback heuristic 3-step breakdown
		return &GoalBreakdown{
			Goal:     goal,
			Overview: fmt.Sprintf("Action roadmap to execute \"%s\" in 3 focused steps.", goal),
			Milestones: []GoalMilestone{
				{Step: 1, Title: fmt.Sprintf("Outline core scope for %s", goal), DurationEst: "30 mins", Rationale: "Low friction entry point to gain immediate momentum"},
				{Step: 2, Title: fmt.Sprintf("Implement primary milestone for %s", goal), DurationEst: "90 mins", Rationale: "Protected deep focus block"},
				{Step: 3, Title: fmt.Sprintf("Review, test, and finalize %s", goal), DurationEst: "45 mins", Rationale: "Quality assurance and closure"},
			},
		}, nil
	}

	breakdown.Goal = goal
	return &breakdown, nil
}

func extractJSONBlock(s string) string {
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")
	if start != -1 && end != -1 && end > start {
		return s[start : end+1]
	}
	return s
}

// Close shuts down the underlying LLM server and releases all resources.
func (e *Engine) Close() error {
	e.closeOnce.Do(func() {
		if e.server != nil {
			e.closeErr = e.server.Close()
		}
	})
	return e.closeErr
}
