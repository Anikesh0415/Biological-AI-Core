# Original User Request

## 2026-09-13T15:49:20Z

# Teamwork Project Prompt — Draft

> Status: Launched.
> Goal: Craft prompt → get user approval → delegate to teamwork_preview
> Requested team: The full team (Multiple agents for research, builds, ops)

Build a Go-based AI extraction engine that integrates with llama.cpp. It must use GBNF (Grammar Enforcement) to parse natural language user prompts (e.g. "remind me to call mom at 7pm") into strictly structured JSON payloads for a reminder system.

Working directory: ~/teamwork_projects/planner_bot
Integrity mode: development

## Requirements

### R1. Natural Language to JSON Extraction
The engine must accept a natural language string representing a reminder request and return a structured JSON object containing the `task` (string) and `time` (string).

### R2. Local LLM Integration with Grammar Enforcement
The engine must interface with a local LLM using `llama.cpp` Go bindings (or a direct execution wrapper). It must apply a GBNF grammar during inference to guarantee the output is strictly parseable JSON. The system should automatically download a small quantized test model (e.g., Phi-3 or Qwen GGUF) for its internal tests if one is not provided.

## Acceptance Criteria

### Functional
- [ ] An automated Go test suite (`go test`) exists and passes without human intervention.
- [ ] The test suite verifies that at least 5 different natural language prompts (e.g., "remind me tomorrow at 5pm to buy milk") are correctly extracted into valid JSON objects.
- [ ] The system successfully loads a local GGUF model and performs inference.
- [ ] The raw string output from the LLM is confirmed to be strictly valid JSON via `json.Unmarshal`, proving that the GBNF grammar enforcement is working correctly.
