package planner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"

	"forge/pkg/executor"
)

func PlanActions(intent string, visionContext string, uiaContext string, history string) ([]executor.Action, error) {
	// Build a simplified, directive prompt optimized for the tiny 0.5B model.
	// Key design decisions:
	//   1. No full JSON examples in the prompt body — the GBNF grammar constrains output format
	//   2. Place the intent PROMINENTLY so the model sees it
	//   3. History in natural language (not JSON) — easier for small models
	//   4. Truncate UIA elements to avoid overwhelming the context
	//   5. Use a "marker" to find the model's actual output in the echoed conversation
	prompt := fmt.Sprintf(`<|im_start|>system
You are a Windows PC automation agent. You output ONE JSON action for the next step. No text, just JSON.
Available action types: click_element, type, key, sleep, done.
<|im_end|>
<|im_start|>user
TASK: %s

DONE SO FAR:
%s

SCREEN ELEMENTS:
%s

SCREEN DESCRIPTION: %s

INSTRUCTIONS:
- STRONGLY PREFER KEYBOARD OVER MOUSE! It is much more reliable.
- To open ANY app (like clock, browser, etc): You MUST output {"type":"key", "key":"win"}.
- If you already pressed win, output {"type":"type", "text":"app name"} next. Do NOT press win again.
- After typing the app name, output {"type":"key", "key":"enter"}.
- DO NOT click taskbar items (like Start, Search, or Clock) to open apps! It will fail.
- Output {"type":"done"} when the task is fully complete.

EXAMPLE OF OPENING AN APP:
Task: open notepad
Step 1 Output: {"type":"key", "key":"win"}
Step 2 Output: {"type":"type", "text":"notepad"}
Step 3 Output: {"type":"key", "key":"enter"}
<|im_end|>
<|im_start|>assistant
`, intent, history, truncateUIA(uiaContext, 15), truncateVision(visionContext))

	os.WriteFile("temp_prompt.txt", []byte(prompt), 0644)

	llamaExe := `E:\AIF_Project\llama.cpp\build\bin\Release\llama-cli.exe`
	modelPath := os.Getenv("FORGE_MODEL")
	if modelPath == "" {
		modelPath = `E:\AIF_Project\models\forge_specialized.gguf`
	}
	grammarFile := `E:\AIF_Project\pkg\planner\action.gbnf`

	// Use context timeout to prevent infinite hangs
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, llamaExe,
		"-m", modelPath,
		"-f", "temp_prompt.txt",
		"--grammar-file", grammarFile,
		"-c", "8192",
		"-n", "128",
		"--temp", "0.1",
		"--repeat-penalty", "1.2",
		"--no-display-prompt",
		"-st",                  // single-turn: generate one response then exit
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	cmd.Stdin = strings.NewReader("") // Pipe empty stdin as fallback EOF

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	rawOutput := stdout.String()

	// Strip ANSI escape codes
	ansiRe := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	rawOutput = ansiRe.ReplaceAllString(rawOutput, "")

	// In conversation mode, the prompt gets echoed. The model's actual output
	// comes AFTER the last "<|im_start|>assistant" marker. Extract only that part.
	response := rawOutput
	marker := "<|im_start|>assistant"
	if idx := strings.LastIndex(rawOutput, marker); idx >= 0 {
		response = rawOutput[idx+len(marker):]
	}
	response = strings.TrimSpace(response)

	fmt.Printf("Planner raw output length: %d bytes\n", len(rawOutput))
	fmt.Printf("Planner extracted response: %s\n", response)

	// With GBNF grammar, the response should be a clean JSON object
	var action executor.Action
	if parseErr := json.Unmarshal([]byte(response), &action); parseErr == nil {
		return []executor.Action{action}, nil
	}

	// Fallback: extract the LAST JSON object from the extracted response
	re := regexp.MustCompile(`\{[^{}]*\}`)
	matches := re.FindAllString(response, -1)
	for i := len(matches) - 1; i >= 0; i-- {
		if json.Unmarshal([]byte(matches[i]), &action) == nil {
			return []executor.Action{action}, nil
		}
	}

	// If we couldn't find JSON in the post-marker text, try the full output as last resort
	if response != rawOutput {
		allMatches := re.FindAllString(rawOutput, -1)
		for i := len(allMatches) - 1; i >= 0; i-- {
			if json.Unmarshal([]byte(allMatches[i]), &action) == nil {
				fmt.Println("Warning: extracted JSON from echoed output (pre-marker)")
				return []executor.Action{action}, nil
			}
		}
	}

	if err != nil {
		return nil, fmt.Errorf("llama execution failed (exit: %v)\nResponse: %s\nStderr length: %d", err, response, stderr.Len())
	}
	return nil, fmt.Errorf("no valid JSON action in response:\n%s", response)
}

// truncateUIA limits the UIA elements to the top N most relevant entries
// and filters out off-screen/empty-name elements.
func truncateUIA(uiaContext string, maxElements int) string {
	uiaContext = strings.TrimSpace(uiaContext)
	if !strings.HasPrefix(uiaContext, "[") {
		return uiaContext
	}

	var elements []map[string]interface{}
	if err := json.Unmarshal([]byte(uiaContext), &elements); err != nil {
		if len(uiaContext) > 2000 {
			return uiaContext[:2000] + "..."
		}
		return uiaContext
	}

	// Filter out elements with negative coordinates (off-screen), empty names, or taskbar buttons that distract the model
	var filtered []map[string]interface{}
	for _, el := range elements {
		name, _ := el["name"].(string)
		if name == "" {
			continue
		}
		
		// Remove temptation for the tiny model to click Start/Search instead of using Win key
		if name == "Start" || name == "Search" || name == "Type here to search" {
			continue
		}

		x, _ := el["x"].(float64)
		y, _ := el["y"].(float64)
		if x < 0 || y < 0 {
			continue
		}
		filtered = append(filtered, el)
	}

	if len(filtered) > maxElements {
		filtered = filtered[:maxElements]
	}

	result, err := json.Marshal(filtered)
	if err != nil {
		return uiaContext
	}
	return string(result)
}

// truncateVision keeps vision context short to save tokens for the tiny model.
func truncateVision(vision string) string {
	vision = strings.TrimSpace(vision)
	if len(vision) > 200 {
		return vision[:200] + "..."
	}
	return vision
}
