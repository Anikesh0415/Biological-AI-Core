package vision

import (
	"bytes"
	"context"
	"fmt"
	"image/png"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/kbinani/screenshot"
)

func CaptureAndAnalyze() (string, error) {
	// 1. Take Screenshot
	bounds := screenshot.GetDisplayBounds(0)
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return "", fmt.Errorf("failed to capture screen: %w", err)
	}

	screenshotFile := "temp_screenshot.png"
	file, err := os.Create(screenshotFile)
	if err != nil {
		return "", fmt.Errorf("failed to create screenshot file: %w", err)
	}
	if err := png.Encode(file, img); err != nil {
		file.Close()
		return "", fmt.Errorf("failed to encode png: %w", err)
	}
	file.Close()
	defer os.Remove(screenshotFile)

	// 2. Try SmolVLM-256M first (Fastest)
	fmt.Println("Trying SmolVLM-256M vision pass...")
	smolResult, err := analyzeWithSmolVLM(screenshotFile)

	// Consider it a failure if there's an error, or if output is extremely short/meaningless
	if err == nil && len(strings.TrimSpace(smolResult)) > 10 {
		return smolResult, nil
	}

	fmt.Println("SmolVLM failed or gave empty output. Falling back to Moondream2...")

	// 3. Fallback to Moondream2 (More robust for complex UI layout)
	return analyzeWithMoondream(screenshotFile)
}

func analyzeWithSmolVLM(imagePath string) (string, error) {
	// Using standard llama-cli (or llama-llava-cli) for SmolVLM
	// Ensure the user downloads this model to models/smolvlm-256m-instruct.gguf
	llamaExe := `E:\AIF_Project\llama.cpp\build\bin\Release\llama-cli.exe`
	modelPath := `E:\AIF_Project\models\smolvlm-256m-instruct.gguf`

	// Using context to enforce a strict timeout for the fast-pass
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	prompt := `Describe the screen layout and give coordinate hints for interactive elements.`
	cmd := exec.CommandContext(ctx, llamaExe,
		"-m", modelPath,
		"--image", imagePath,
		"-p", prompt,
		"-c", "2048",
		"--temp", "0.1",
		"--no-display-prompt",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// Separate stdout (actual vision output) from stderr (llama warnings/banner)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("SmolVLM execution failed: %v", err)
	}

	return strings.TrimSpace(stdout.String()), nil
}

func analyzeWithMoondream(imagePath string) (string, error) {
	llavaExe := `E:\AIF_Project\llama.cpp\build\bin\Release\llama-mtmd-cli.exe`
	modelPath := `E:\AIF_Project\models\moondream2-text-model-f16.gguf`
	mmprojPath := `E:\AIF_Project\models\moondream2-mmproj-f16.gguf`

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()

	prompt := `Describe the screen layout and give coordinate hints for interactive elements.`
	cmd := exec.CommandContext(ctx, llavaExe,
		"-m", modelPath,
		"--mmproj", mmprojPath,
		"--image", imagePath,
		"-p", prompt,
		"--chat-template", "chatml",
		"-c", "4096",
		"--temp", "0.1",
	)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// Separate stdout (actual vision output) from stderr (llama warnings/banner)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("Moondream execution failed: %v\nstderr: %s", err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}
