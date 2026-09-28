# Forge OS 🚀

Forge OS is an incredibly fast, ultra-lightweight, zero-RAM GUI Agent for Windows that acts as the physical brain and hands for your PC.

Written entirely in **pure Go**, Forge achieves a true **Zero-RAM footprint** when idle. It combines real-time event-driven UI Automation (UIA), native Windows Hooks, local SQLite long-term memory, localized AI Planning, remote Telegram control, offline voice input, GBNF-constrained JSON parsing, an FSA state-machine safeguard engine, and a live glassmorphism HUD — all with zero cloud dependencies.

---

## Key Features ✨

- ⚡ **Zero-RAM Idle Footprint:** Pure Go binary — consumes effectively 0 MB of RAM while sleeping. Wakes instantly when summoned.
- 👁️ **Dual-Stage Vision Pipeline:** Uses SmolVLM-256M as a sub-second fast pass, seamlessly falling back to Moondream2 for complex GUI spatial analysis.
- 📐 **GBNF Grammar JSON Locking:** Enforces GGML Backus-Naur Form (`action.gbnf`) at the sampling layer, guaranteeing 100% syntactically valid JSON output from local LLMs.
- 🛡️ **Finite State Automata (FSA) Safeguards:** Tracks execution risk across action sequences over time (`Safe` → `Suspicious` → `Elevated` → `HighRisk`) to prevent malicious or accidental system actions.
- 📡 **Telegram Remote Control:** Send natural language commands to Forge from anywhere via your Telegram bot. Authenticates by Chat ID and replies with execution status.
- 🎙️ **Offline Voice Push-to-Talk (`Ctrl+Shift+V`):** Speak and Forge listens — 100% offline via Windows SAPI (Speech Recognition). Zero cloud calls, fully private.
- 🖥️ **Live Glassmorphism HUD Overlay:** A sleek bottom-right progress panel appears during multi-step runs — showing step numbers, teal progress fill, and live action status.
- 🔴 **Watch-and-Learn Macro Recorder (`Ctrl+Shift+R`):** Record workflows by performing them physically. Forge captures mouse clicks and keystrokes via Win32 hooks (`user32.dll`) and saves them as zero-hallucination `.json` skills.
- 🧠 **Embedded Local SQLite Brain (`brain.db`):** Pure Go SQLite engine (`modernc.org/sqlite`). Stores user preferences, macro mappings, and execution history.
- 🔍 **Pure-Go Fuzzy Matcher Engine:** Tokenized Levenshtein distance algorithm for instant (0ms) matching of 600+ built-in app shortcuts and macros.
- 🌐 **100% Local & Private:** No cloud subscriptions, no telemetry, no data harvesting.

---

## What's New in Version 2.83 🚀

### 🏝️ Dynamic Island UI (`main.go`)
- Completely redesigned the Forge summon overlay into a sleek, 46px tall macOS-style "Dynamic Island".
- Renders via native WPF/XAML injected via PowerShell, avoiding extra RAM consumption while maintaining a modern glassy aesthetic.

### 🧠 Specialized AI Fine-Tuning Pipeline (`finetune.ipynb` & `json_llm.go`)
- Added a seamless Unsloth-powered fine-tuning pipeline tailored for Google Colab to train `Qwen2.5-0.5B-Instruct` on user-recorded UI interactions.
- Hardcoded `forge_specialized.gguf` as the default model directly into the Go source, eliminating the need to set manual environment variables.

### 👁️ Dual-Stage Vision Pipeline (`pkg/vision/vision.go`)
- Integrated **SmolVLM-256M** as a hyper-fast first-pass vision model with a tight 10s execution timeout.
- Automatic fallback to **Moondream2** if SmolVLM returns an empty response or encounters complex GUI element layouts.

### 📐 GBNF Grammar Schema Enforcement (`pkg/planner/action.gbnf`)
- Created GGML Backus-Naur Form grammar rules that mathematically force Qwen 0.5B to output *only* valid `executor.Action` JSON objects.
- Eliminated regex string cleanup and trailing-comma parsing failures.

### 🛡️ Context-Aware FSA Safeguard Engine (`main.go`)
- Replaced static keyword array checks with a pure-Go **Finite State Automaton (`SafeguardFSA`)**.
- Tracks state progression over time: opening terminal/browser transitions to `StateSuspicious`, typing elevates risk to `StateElevated`, and high-risk actions (e.g., `delete`, `pay`, `transfer`) escalate to `StateHighRisk`, prompting a native VBS user modal.

---

## Architecture 🏗️

```
         +----------------------------------------------+
         |  Input Sources                               |
         |  ┌─────────────┐ ┌───────────┐ ┌─────────┐  |
         |  │ WPF Prompt  │ │ Telegram  │ │  Voice  │  |
         |  │ (keyboard)  │ │ (remote)  │ │  SAPI   │  |
         |  └──────┬──────┘ └─────┬─────┘ └────┬────┘  |
         +─────────┼──────────────┼─────────────┼───────+
                   └──────────────┼─────────────┘
                                  │
                                  v
                   +─────────────────────────────+
                   │   DispatchIntent() Router   │
                   +──────────┬──────────────────+
                              │
               ┌──────────────┴──────────────┐
               │                             │
    [Skill Matched]                  [No Skill Match]
               │                             │
               v                             v
   +───────────────────+         +──────────────────────+
   │ Pure-Go Fuzzy     │         │ GBNF-Locked Qwen     │
   │ Macro Engine      │         │ 0.5B Planner         │
   │ (.json / 0ms)     │         +──────────┬───────────+
   +─────────┬─────────+                    │
             │                              │
             └──────────────┬───────────────┘
                            │
                            v
               +────────────────────────+
               │  FSA Safeguard Engine  │
               │  (Safe -> HighRisk)    │
               +────────────┬───────────+
                            │
                            v
               +────────────────────────+
               │   Live HUD Overlay     │
               │   (Step X / 15 bar)    │
               +────────────┬───────────+
                            │
                            v
               +────────────────────────+
               │  Dual Vision Pipeline  │
               │ (SmolVLM -> Moondream) │
               +────────────┬───────────+
                            │
                            v
               +────────────────────────+
               │  Native Win32 Executor │
               +────────────────────────+
```

---

## Getting Started 📥

### 1. Requirements
- Windows 10/11 (64-bit)
- Go 1.20+ (if building from source)

### 2. Required Models
Download the following `.gguf` files into the `models/` directory for dynamic fallback planning & vision:
- **Qwen 2.5 (0.5B):** `qwen2.5-0.5b-instruct-q4_k_m.gguf`
- **SmolVLM (Fast Vision):** `smolvlm-256m-instruct.gguf`
- **Moondream2 (Deep Vision):** `moondream2-text-model-f16.gguf` & `moondream2-mmproj-f16.gguf`

### 3. Build & Run
```bash
# Clone the repository
git clone https://github.com/Anikesh0415/Forge.git
cd Forge

# Build executable
go build -ldflags="-H windowsgui" -o forge.exe main.go

# Launch Forge
.\forge.exe
```

---

## How to Record a Macro 📹

1. Run `forge.exe`.
2. Press **`Ctrl + Shift + R`**. A HUD notification appears:  
   `🔴 Started recording macro... Press Ctrl+Shift+R to stop.`
3. Perform your task on your PC (click buttons, type text).
4. Press **`Ctrl + Shift + R`** again.  
   `✅ Saved macro: macro_...`
5. The macro is now saved in `skills_db/` and can be triggered anytime via natural language or fuzzy match!

---

## Disclaimer ⚠️

Forge has direct access to your physical mouse and keyboard. While targeted safeguards and FSA state monitoring protect against high-risk actions, always supervise the agent during dynamic AI execution.
