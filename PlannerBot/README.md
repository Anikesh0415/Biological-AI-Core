# PlannerBot: Sovereign Cognitive Engine (v0.3.0)

> **Private, air-gapped, on-device AI daily planner and focus strategist designed to run on 7-year-old hardware with as little as 4GB RAM.**

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go)](https://go.dev)
[![Architecture](https://img.shields.io/badge/Architecture-Air--Gapped%20Local-emerald)](#)
[![RAM Footprint](https://img.shields.io/badge/RAM-~380MB-brightgreen)](#)
[![GitHub Release](https://img.shields.io/github/v/release/Anikesh0415/PlannerBot)](https://github.com/Anikesh0415/PlannerBot/releases)

PlannerBot is an autonomous, offline-first productivity system that integrates constrained on-device AI inference, natural language task scheduling, digital wellbeing audit, real-time alert chimes, and encrypted local P2P synchronization without relying on any external cloud services.

---

## 📥 Direct Downloads (No Go Compiler Required)

For regular users and friends who do **not** have the Go development environment installed:

- **[📦 Download Windows Package (ZIP)](https://github.com/Anikesh0415/PlannerBot/releases/download/v0.3.0/PlannerBot-v0.3.0-windows-amd64.zip)** *(Recommended: Includes `PlannerBot.exe` + `start.bat`)*
- **[🚀 Download Standalone Windows Executable (PlannerBot.exe)](https://github.com/Anikesh0415/PlannerBot/releases/download/v0.3.0/PlannerBot.exe)**
- **[🐧 Download Linux AMD64 Binary](https://github.com/Anikesh0415/PlannerBot/releases/download/v0.3.0/PlannerBot-linux-amd64)**

*Simply extract the ZIP and double-click `start.bat` (or run `PlannerBot.exe`)!*

---

## Key Features

### 1. Ultra-Lightweight On-Device Intelligence (<400MB RAM)
- Powered by a quantized **Qwen2.5-0.5B-Instruct (Q4_K_M)** model executed locally via an embedded `llama-server`.
- Uses **GBNF constrained decoding grammars** for guaranteed structured slot-filling (JSON extraction of dates, times, and actions) with conversational coaching.
- Runs smoothly even on low-end laptops and mobile phones with 4GB RAM.

### 2. Context-Aware Natural Language Scheduling
- Natural time parsing: *"remind me to call mom tomorrow at 5pm"*, *"set a timer for 15 minutes"*, *"review PR on Thursday at 2pm"*.
- Intelligent intent discrimination ensures general knowledge queries (*"who developed antigravity"*, *"how to code in rust"*) are answered conversationally without triggering accidental alarms.
- Automatic fallback for generic timers (*"set a reminder for 5 minutes"* &rarr; `"Quick Reminder (5 minutes)"`).

### 3. Real-Time Alert Modal & Synthesized Audio Chime
- Glassmorphic desktop modal triggers instantly when a scheduled reminder is due.
- Integrated Web Audio API bell chime ($G_5 \to C_6$ sine wave with exponential decay) and fallback HTML5 desktop notifications.
- 1-click actions: **Complete**, **Snooze 5m**, or **Dismiss**.

### 4. Screentime & Digital Wellbeing Coach
- On-device WebAssembly OCR (**Tesseract.js**) extracts app usage directly from screenshots pasted with `Ctrl+V` or uploaded via file dialog.
- Computes your **Focus Ratio**, identifies your top **Dopamine Trap**, and generates a candid habit debrief.

### 5. Stoic Evening Reflection & Socratic Roadmaps
- **3-Step Milestone Roadmap**: Type `plan: launch my portfolio` to get a structured 3-part execution roadmap following the Rule of Three.
- **Stoic Evening Debrief**: Type `analyze my day` for a 3-part reflection: *The Win*, *The Friction*, and *The Keystone* priority for tomorrow.

### 6. Encrypted P2P CRDT Synchronization
- Synchronizes schedules peer-to-peer over local Wi-Fi via UDP multicast discovery.
- End-to-end encrypted with **AES-256-GCM** using 12-character pairing codes (`PLAN-XXXX-XXXX`).
- Zero cloud servers or centralized accounts required.

---

## 🚀 One-Click Quickstart

### Windows
Double-click `start.bat` or run:
```cmd
start.bat
```
*What this script does:*
1. Automatically checks for and downloads the quantized Qwen2.5 GGUF model (~398MB) on first run.
2. Compiles `PlannerBot.exe` if needed.
3. Launches the sovereign engine and opens `http://127.0.0.1:8088` in your default browser.

### Linux / macOS
```bash
chmod +x start.sh
./start.sh
```

---

## 🔒 100% Privacy & Volunteer Training Data

All conversations, tasks, and reflections are permanently stored on your device in your local BoltDB ledger:
- **Windows**: `%USERPROFILE%\.planner_bot\reminders.db`
- **Linux / macOS**: `~/.planner_bot/reminders.db`

Your data **never leaves your machine**.

### Volunteering Chat Logs for AI Fine-Tuning
If you want to contribute to training and fine-tuning future open-source PlannerBot models:
1. Click the **"Volunteer Data"** button in the top navigation bar of the web interface.
2. PlannerBot will generate a local export: `plannerbot_volunteer_chat_YYYY-MM-DD.json`.
3. You can review the exported dialogue and voluntarily share it with the team to help train smarter small language models.

---

## Architecture Overview

```
                        +----------------------------+
                        |     Browser Web Client     |
                        |   (Single-Page App, SSE)   |
                        +--------------+-------------+
                                       | HTTP / SSE
                                       v
+-------------------------------------------------------------------------+
| PlannerBot Sovereign Engine (Go)                                        |
|                                                                         |
|  +---------------------+  +--------------------+  +------------------+  |
|  |   UI & REST API     |  |    CRDT Engine     |  |  Task Scheduler  |  |
|  | (Debounced Handler) |  |   (LWW Register)   |  |   (Cron Timer)   |  |
|  +----------+----------+  +---------+----------+  +--------+---------+  |
|             |                       |                      |            |
|             v                       v                      v            |
|  +-------------------------------------------------------------------+  |
|  |             BoltDB Persistent Storage (reminders.db)              |  |
|  +-------------------------------------------------------------------+  |
|             ^                                                           |
|             |                                                           |
|  +----------+----------+  +--------------------+  +------------------+  |
|  |  Extractor Engine   |  |   P2P Discovery    |  | Screentime OCR   |  |
|  | (GBNF Slot-Filling) |  | (AES-256 UDP Mesh) |  |  (Tesseract.js)  |  |
|  +----------+----------+  +--------------------+  +------------------+  |
+-------------|-----------------------------------------------------------+
              v
+------------------------------------+
| Local llama-server Process         |
| (Qwen2.5-0.5B-Instruct-Q4_K_M)     |
| Memory footprint: ~380MB RAM       |
+------------------------------------+
```

---

## Development & Testing

Run all unit and integration tests:
```bash
go test -v ./pkg/ui ./pkg/grammar ./pkg/store ./pkg/timeparse ./pkg/crdt ./pkg/screentime
```

Rebuild executable:
```bash
go build -o PlannerBot.exe ./cmd/planner
```

---

## License

MIT License. Developed for sovereign, private, on-device human focus.
