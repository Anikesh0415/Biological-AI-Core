# Flixi 🚀
### The Ultimate Local AI Automation OS for Consumers

Written entirely in **pure Go**, Flixi (formerly Forge) achieves a true **Zero-RAM footprint** when idle. It combines real-time event-driven UI Automation (UIA), native Windows Hooks, local SQLite long-term memory (scaling up to 80 million macros!), localized AI Planning, offline Vosk voice input, multi-step Intent Chaining, and a live glassmorphism HUD — all with zero cloud dependencies.

---

## Key Features ✨

- ⚡ **Zero-RAM Idle Footprint:** Pure Go binary — consumes effectively 0 MB of RAM while sleeping. Wakes instantly when summoned.
- 🗄️ **O(1) SQLite Macro Engine:** Scaled and battle-tested to instantly route against an 80,000,000+ action database (`app_actions.db`) locally without breaking a sweat.
- 🎙️ **Local Python Vosk AI Voice Engine (`Ctrl+Shift+V`):** Speak and Flixi listens — completely offline via a powerful local Vosk acoustic model. Zero cloud calls, fully private.
- 🔗 **Mega-Macro Intent Chaining:** Seamlessly slice and stitch multiple intents together (e.g., `"open youtube then search judas on spotify then open discord"`).
- 👁️ **Dual-Stage Vision Pipeline:** Uses SmolVLM-256M as a sub-second fast pass, seamlessly falling back to Moondream2 for complex GUI spatial analysis.
- 📐 **GBNF Grammar JSON Locking:** Enforces GGML Backus-Naur Form at the sampling layer, guaranteeing 100% syntactically valid JSON output from local LLMs.
- 🖥️ **Live Glassmorphism HUD Overlay:** A sleek bottom-right progress panel appears during multi-step runs — showing step numbers, progress fills, and live action status.

---

## What's New in Version 3.0 🚀

### 🗄️ 80-Million Action SQLite Database
- Replaced the volatile RAM-heavy in-memory mapping with a pure `modernc.org/sqlite` database integration.
- Stress-tested with an 80,000,000 synthetic action generator, proving Flixi can handle a macro for every conceivable application on the internet instantly in O(1) time.

### 🎙️ Vosk AI Voice Overhaul
- Completely ripped out the legacy Windows SAPI dictation engine.
- Replaced with a fully offline, high-accuracy Python Vosk AI transcription pipeline.

### 🔗 Multi-Step "Mega Macro" Intent Chaining
- Completely overhauled `skills.go` to support dynamic NLP intent slicing.
- Intelligently merges Go-based logical skills (like `UniversalSearchSkill`) into raw JSON arrays dynamically to create massive, unbreakable workflow sequences on the fly.

### 🌐 Universal Parameterization
- Eliminated hardcoded string mapping (e.g. 19 fixed `openApps`).
- Flixi now utilizes a flexible `open {app}` regex structure, allowing it to dynamically interact with the Windows Start Menu and launch any arbitrary application.

---

## Architecture 🏗️

```
         +----------------------------------------------+
         |  Input Sources                               |
         |  ┌─────────────┐ ┌───────────┐ ┌─────────┐  |
         |  │ WPF Prompt  │ │ Telegram  │ │ Vosk AI │  |
         |  │ (keyboard)  │ │ (remote)  │ │ (Voice) │  |
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
    [Skill Matched (O(1))]           [No Skill Match]
               │                             │
               v                             v
   +───────────────────+         +──────────────────────+
   │ 80-Million SQLite │         │ GBNF-Locked Qwen     │
   │ Macro Engine      │         │ 0.5B Planner         │
   +─────────┬─────────+         +──────────┬───────────+
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
- Go 1.20+ 
- Python 3.10+ (for Vosk Voice Engine)

### 2. Required Models
Download the following models into their respective directories:
- **Vosk Voice Model:** Extract to `pkg/voice/model/`
- **Qwen 2.5 (0.5B):** `models/qwen2.5-0.5b-instruct-q4_k_m.gguf`
- **Moondream2 (Deep Vision):** `models/moondream2-text-model-f16.gguf` & `models/moondream2-mmproj-f16.gguf`

### 3. Build & Run
```bash
# Clone the repository
git clone https://github.com/Anikesh0415/Flixi.git
cd Flixi

# Build executable
go build -ldflags="-H windowsgui" -o flixi.exe main.go

# Launch Flixi
.\flixi.exe
```
