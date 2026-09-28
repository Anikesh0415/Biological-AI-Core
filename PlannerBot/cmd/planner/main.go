package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"planner_bot/pkg/analyzer"
	"planner_bot/pkg/crypto"
	"planner_bot/pkg/extractor"
	"planner_bot/pkg/grammar"
	"planner_bot/pkg/notifier"
	"planner_bot/pkg/p2p"
	"planner_bot/pkg/scheduler"
	"planner_bot/pkg/store"
	"planner_bot/pkg/timeparse"
	"planner_bot/pkg/ui"

	"github.com/google/uuid"
)

func main() {
	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║         🧠 Planner Bot v0.2.0               ║")
	fmt.Println("║   Your AI-powered local reminder assistant   ║")
	fmt.Println("║       with Encrypted P2P CRDT Sync           ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// --- Initialize Store ---
	fmt.Print("📦 Initializing database... ")
	db, err := store.New("")
	if err != nil {
		log.Fatalf("Failed to initialize store: %v", err)
	}
	defer db.Close()
	fmt.Println("✅")

	// --- Initialize Pairing Code ---
	userCode := getOrGeneratePairingCode()

	// --- Initialize P2P Sync Engine ---
	fmt.Print("🔗 Starting encrypted P2P sync engine... ")
	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "Device-" + uuid.New().String()[:4]
	}

	p2pEngine, err := p2p.NewEngine(p2p.EngineConfig{
		DeviceID:     "dev-" + uuid.New().String()[:8],
		DeviceName:   hostname,
		UserCode:     userCode,
		Store:        db,
		SyncInterval: 15 * time.Second,
	})
	if err != nil {
		log.Fatalf("Failed to initialize P2P engine: %v", err)
	}
	if err := p2pEngine.Start(ctx); err != nil {
		log.Printf("⚠️  P2P Engine start warning: %v", err)
	}
	defer p2pEngine.Close()
	fmt.Println("✅")
	fmt.Printf("   🔑 Pairing Code: %s\n", userCode)
	fmt.Printf("   📡 Sync Server: port %d (AES-256-GCM encrypted)\n", p2pEngine.Port())
	fmt.Println()

	// --- Initialize Notifier ---
	notify := notifier.New("Planner Bot")

	// --- Initialize Day Analyzer ---
	dayAnalyzer := analyzer.New(db, nil)

	// --- Initialize Web Dashboard Immediately ---
	fmt.Print("🌐 Starting desktop web dashboard... ")
	uiServer := ui.NewServer(ui.Config{
		Port:      8088, // Consistent port with automatic fallback if occupied
		Store:     db,
		Extractor: nil,  // Will attach dynamically once loaded!
		P2PEngine: p2pEngine,
		Analyzer:  dayAnalyzer,
	})
	if err := uiServer.Start(); err != nil {
		log.Printf("⚠️  UI Server start warning: %v", err)
	} else {
		defer uiServer.Close()
		fmt.Println("✅")
		fmt.Printf("   🖥️  Dashboard URL: %s\n", uiServer.URL())
		if os.Getenv("PLANNER_NO_BROWSER") == "" {
			uiServer.OpenBrowser()
		}
	}
	fmt.Println()

	// --- Initialize Scheduler ---
	fmt.Print("⏰ Starting reminder scheduler... ")
	sched := scheduler.New(db, 1*time.Minute, func(r *store.Reminder) {
		dueTime := r.FireAt.Local().Format("3:04 PM")
		if r.Missed {
			_ = notify.SendMissedReminder(r.Task, dueTime)
			fmt.Printf("\n⚠️  MISSED REMINDER: %s (was due at %s)\n> ", r.Task, dueTime)
		} else {
			_ = notify.SendReminder(r.Task, dueTime)
			fmt.Printf("\n🔔 REMINDER: %s (due: %s)\n> ", r.Task, dueTime)
		}
		// Push real-time alert to web dashboard
		uiServer.NotifyReminder(r)
	})
	if err := sched.Start(ctx); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}
	defer sched.Stop()
	fmt.Println("✅")

	// --- Initialize AI Engine Asynchronously ---
	var engine *extractor.Engine
	go func() {
		fmt.Println("🤖 Initializing AI engine...")
		eng, err := extractor.NewEngine(ctx, extractor.Config{
			DownloadIfNone: true,
			ContextSize:    8192,
		})
		if err == nil {
			engine = eng
			uiServer.SetExtractor(eng)
			dayAnalyzer.SetLLM(eng.Server())
			fmt.Println("✅ AI engine loaded, connected to Day Analyzer, and attached to UI server!")
		} else {
			fmt.Printf("⚠️  AI engine notice: %v (UI is running in fast-mode)\n", err)
		}
	}()

	fmt.Println("💬 Talk to me naturally! Examples:")
	fmt.Println("   • remind me to call mom at 7pm")
	fmt.Println("   • don't forget to buy milk tomorrow at 5pm")
	fmt.Println("   • schedule a dentist appointment for next Monday at 10:30am")
	fmt.Println()
	fmt.Println("Commands: 'list', 'analyze', 'sync', 'peers', 'code', 'web', 'delete <id>', 'clear', 'help', 'quit'")
	fmt.Println()

	// --- Handle Ctrl+C / Termination gracefully ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n👋 Goodbye! Your reminders are saved.")
		cancel()
		os.Exit(0)
	}()

	// --- Detect if running in Interactive Terminal vs Windows GUI mode ---
	fileInfo, err := os.Stdin.Stat()
	isTerminal := err == nil && (fileInfo.Mode()&os.ModeCharDevice) != 0

	if !isTerminal {
		// Running in Windows GUI mode (double clicked exe without console attached).
		// Keep the background server running until context cancellation or OS signal!
		<-ctx.Done()
		return
	}

	// --- Main Chat Loop (for interactive terminal) ---
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}

		lowerInput := strings.ToLower(input)
		switch {
		case lowerInput == "quit" || lowerInput == "exit" || lowerInput == "q":
			fmt.Println("👋 Goodbye! Your reminders are saved.")
			return
		case lowerInput == "help" || lowerInput == "h":
			printHelp(userCode)
		case lowerInput == "list" || lowerInput == "ls":
			listReminders(db)
		case lowerInput == "analyze":
			runDayAnalysis(ctx, dayAnalyzer)
		case lowerInput == "web":
			fmt.Printf("🖥️  Dashboard URL: %s\n", uiServer.URL())
			uiServer.OpenBrowser()
		case lowerInput == "clear":
			clearCompleted(db)
		case lowerInput == "peers":
			listPeers(p2pEngine)
		case lowerInput == "sync":
			triggerSync(ctx, p2pEngine)
		case lowerInput == "code":
			fmt.Printf("🔑 Your P2P Pairing Code is: %s\n", userCode)
			fmt.Println("   Use this code on your other devices (laptop, phone) to sync securely.")
		case strings.HasPrefix(lowerInput, "delete "):
			id := strings.TrimSpace(input[7:])
			deleteReminder(db, p2pEngine, id)
		default:
			handleAIInput(ctx, engine, db, p2pEngine, input)
		}
	}
}

func runDayAnalysis(ctx context.Context, an *analyzer.Analyzer) {
	fmt.Println("📊 Analyzing your daily productivity...")
	summary, err := an.AnalyzeDay(ctx, time.Now())
	if err != nil {
		fmt.Printf("❌ Error analyzing day: %v\n", err)
		return
	}

	fmt.Println("─────────────────────────────────────────")
	fmt.Printf("📈 Completed: %d | ⏳ Pending: %d | ⚠️ Missed: %d\n\n",
		summary.CompletedCount, summary.PendingCount, summary.MissedCount)
	fmt.Printf("💡 Reflection: %s\n\n", summary.AnalysisText)

	if len(summary.Suggestions) > 0 {
		fmt.Println("📅 Suggested Agenda for Tomorrow:")
		for i, s := range summary.Suggestions {
			fmt.Printf("   %d. [%s] %s (%s)\n", i+1, s.SuggestedTime, s.Task, s.Reason)
		}
	}
	fmt.Println("─────────────────────────────────────────")
}

func handleAIInput(ctx context.Context, engine *extractor.Engine, db *store.Store, p2pEngine *p2p.Engine, input string) {
	if engine == nil {
		fmt.Println("⏳ AI engine is still initializing. Please try again in a few seconds.")
		return
	}

	aiCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	now := time.Now()

	// Persist user chat turn
	userMsg := &store.ChatMessage{
		ID:        uuid.New().String()[:8],
		Role:      "user",
		Content:   input,
		Timestamp: now.UTC(),
	}
	_ = db.SaveChat(userMsg)

	// Fetch recent chat context and tasks
	recentChats, _ := db.ListRecentChats(15)
	var historyTurns []grammar.HistoryTurn
	for _, c := range recentChats {
		if c.ID == userMsg.ID {
			continue
		}
		historyTurns = append(historyTurns, grammar.HistoryTurn{
			Role:    c.Role,
			Content: c.Content,
		})
	}

	allReminders, _ := db.ListAll()
	var taskContexts []grammar.TaskContext
	for _, r := range allReminders {
		if r.Deleted {
			continue
		}
		taskContexts = append(taskContexts, grammar.TaskContext{
			Task:      r.Task,
			TimeExpr:  r.OriginalTimeExpr,
			Completed: r.Fired,
		})
	}

	// 1. Try Context-Aware Reminder Extraction
	reminder, err := engine.ExtractReminderWithContext(aiCtx, input, historyTurns, taskContexts)
	if err == nil && reminder != nil && reminder.Task != "" {
		fireAt, err := timeparse.Parse(reminder.Time, now)
		if err != nil {
			fireAt = now.Add(1 * time.Hour)
		}

		id := uuid.New().String()[:8]
		r := &store.Reminder{
			ID:               id,
			Task:             reminder.Task,
			FireAt:           fireAt.UTC(),
			CreatedAt:        now.UTC(),
			UpdatedAt:        now.UTC(),
			OriginalTimeExpr: reminder.Time,
			Version:          1,
		}

		if err := db.Save(r); err != nil {
			fmt.Printf("❌ Failed to save reminder: %v\n", err)
			return
		}

		reply := fmt.Sprintf("Got it! I'll remind you to %q at %s", reminder.Task, fireAt.Local().Format("Mon Jan 2, 3:04 PM"))
		fmt.Printf("✅ %s\n", reply)
		fmt.Printf("   [ID: %s | Contextual AI extraction]\n", id)

		_ = db.SaveChat(&store.ChatMessage{
			ID:        uuid.New().String()[:8],
			Role:      "assistant",
			Content:   reply,
			Timestamp: time.Now().UTC(),
			Action:    "reminder_created",
		})

		// Automatically push to connected peers in background
		go func() {
			syncCtx, sCancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer sCancel()
			_, _ = p2pEngine.SyncNow(syncCtx)
		}()
		return
	}

	// 2. Conversational reasoning crawling previous data
	currentTimeStr := now.Format("Monday, Jan 2, 2006 at 3:04 PM")
	reply, err := engine.GenerateChatReply(aiCtx, historyTurns, taskContexts, input, currentTimeStr)
	if err == nil && strings.TrimSpace(reply) != "" {
		fmt.Printf("🤖 %s\n", reply)
		_ = db.SaveChat(&store.ChatMessage{
			ID:        uuid.New().String()[:8],
			Role:      "assistant",
			Content:   reply,
			Timestamp: time.Now().UTC(),
			Action:    "chat",
		})
		return
	}

	// 3. Fallback
	fallback := "I'm ready! You can ask me to set a reminder, check your schedule, or analyze your day."
	fmt.Printf("💬 %s\n", fallback)
	_ = db.SaveChat(&store.ChatMessage{
		ID:        uuid.New().String()[:8],
		Role:      "assistant",
		Content:   fallback,
		Timestamp: time.Now().UTC(),
		Action:    "general",
	})
}

func listReminders(db *store.Store) {
	pending, err := db.ListPending()
	if err != nil {
		fmt.Printf("❌ Error listing reminders: %v\n", err)
		return
	}

	if len(pending) == 0 {
		fmt.Println("📭 No pending reminders.")
		return
	}

	fmt.Printf("\n📋 Pending Reminders (%d):\n", len(pending))
	fmt.Println("─────────────────────────────────────────")
	for i, r := range pending {
		status := "⏳"
		if r.Missed {
			status = "⚠️"
		}
		fmt.Printf("  %s %d. [%s] %s — %s\n",
			status,
			i+1,
			r.ID,
			r.Task,
			r.FireAt.Local().Format("Mon Jan 2, 3:04 PM"),
		)
	}
	fmt.Println()
}

func deleteReminder(db *store.Store, p2pEngine *p2p.Engine, id string) {
	if err := db.SoftDelete(id); err != nil {
		fmt.Printf("❌ Failed to delete reminder %s: %v\n", id, err)
		return
	}
	fmt.Printf("🗑️  Deleted reminder [%s] (tombstone will sync to peers).\n", id)

	go func() {
		syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = p2pEngine.SyncNow(syncCtx)
	}()
}

func listPeers(p2pEngine *p2p.Engine) {
	peers := p2pEngine.Peers()
	if len(peers) == 0 {
		fmt.Println("🔍 No peers discovered on the local network yet.")
		fmt.Println("   Make sure your other device is on the same Wi-Fi with the same pairing code.")
		return
	}

	fmt.Printf("\n🌐 Discovered Peers (%d):\n", len(peers))
	fmt.Println("─────────────────────────────────────────")
	for i, p := range peers {
		fmt.Printf("  📱 %d. %s (%s:%d) — Last seen: %s\n",
			i+1, p.Name, p.IP, p.Port, p.LastSeen.Format("15:04:05"))
	}
	fmt.Println()
}

func triggerSync(ctx context.Context, p2pEngine *p2p.Engine) {
	peers := p2pEngine.Peers()
	if len(peers) == 0 {
		fmt.Println("📡 Scanning for peers... No peers discovered on Wi-Fi yet.")
		return
	}

	fmt.Printf("🔄 Syncing with %d peer(s)...\n", len(peers))
	syncCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	changes, err := p2pEngine.SyncNow(syncCtx)
	if err != nil {
		fmt.Printf("⚠️  Sync error: %v\n", err)
		return
	}

	fmt.Printf("✅ Sync complete! Applied %d mutation(s).\n", changes)
}

func clearCompleted(db *store.Store) {
	purged, err := db.PurgeCompleted()
	if err != nil {
		fmt.Printf("❌ Error clearing: %v\n", err)
		return
	}
	fmt.Printf("🗑️  Cleared %d completed reminder(s).\n", purged)
}

func printHelp(userCode string) {
	fmt.Println()
	fmt.Println("📖 Planner Bot Commands:")
	fmt.Println("─────────────────────────────────────────")
	fmt.Println("  Natural Language Reminders:")
	fmt.Println("    • remind me to call mom at 7pm")
	fmt.Println("    • buy groceries tomorrow at 5pm")
	fmt.Println("    • in 30 minutes, check the oven")
	fmt.Println()
	fmt.Println("  P2P Sync Commands:")
	fmt.Println("    sync        — Force immediate sync with all peers on Wi-Fi")
	fmt.Println("    peers       — List discovered devices on local network")
	fmt.Printf("    code        — Show your pairing code (%s)\n", userCode)
	fmt.Println()
	fmt.Println("  Daily Planning & Reflection:")
	fmt.Println("    analyze     — Review daily progress and get tomorrow's AI plan")
	fmt.Println("    web         — Open the Desktop Web Dashboard in your browser")
	fmt.Println()
	fmt.Println("  Reminder Management:")
	fmt.Println("    list / ls   — Show pending reminders")
	fmt.Println("    delete <id> — Delete reminder by ID (propagates to all devices)")
	fmt.Println("    clear       — Remove completed reminders")
	fmt.Println("    help / h    — Show this help message")
	fmt.Println("    quit / q    — Exit the app")
	fmt.Println()
}

func getOrGeneratePairingCode() string {
	if envCode := os.Getenv("PLANNER_PAIRING_CODE"); envCode != "" {
		return strings.TrimSpace(envCode)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return crypto.GeneratePairingCode()
	}

	configDir := filepath.Join(home, ".planner_bot")
	_ = os.MkdirAll(configDir, 0755)
	codeFile := filepath.Join(configDir, "pairing_code.txt")

	// Read existing code if present
	if data, err := os.ReadFile(codeFile); err == nil && len(strings.TrimSpace(string(data))) > 0 {
		return strings.TrimSpace(string(data))
	}

	// Generate new code and persist
	newCode := crypto.GeneratePairingCode()
	_ = os.WriteFile(codeFile, []byte(newCode), 0600)
	return newCode
}

func runManualMode(ctx context.Context, db *store.Store, sched *scheduler.Scheduler, notify *notifier.Notifier, p2pEngine *p2p.Engine) {
	dayAnalyzer := analyzer.New(db, nil)
	uiServer := ui.NewServer(ui.Config{
		Port:      8088,
		Store:     db,
		Extractor: nil,
		P2PEngine: p2pEngine,
		Analyzer:  dayAnalyzer,
	})
	if err := uiServer.Start(); err == nil {
		defer uiServer.Close()
		if os.Getenv("PLANNER_NO_BROWSER") == "" {
			uiServer.OpenBrowser()
		}
	}

	fmt.Println("📝 Manual mode: Type 'list', 'sync', 'peers', 'clear', or 'quit'")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\n\n👋 Goodbye!")
		os.Exit(0)
	}()

	fileInfo, err := os.Stdin.Stat()
	isTerminal := err == nil && (fileInfo.Mode()&os.ModeCharDevice) != 0
	if !isTerminal {
		<-ctx.Done()
		return
	}

	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			break
		}

		input := strings.TrimSpace(scanner.Text())
		lowerInput := strings.ToLower(input)
		switch {
		case lowerInput == "quit" || lowerInput == "exit" || lowerInput == "q":
			fmt.Println("👋 Goodbye!")
			return
		case lowerInput == "list" || lowerInput == "ls":
			listReminders(db)
		case lowerInput == "peers":
			listPeers(p2pEngine)
		case lowerInput == "sync":
			triggerSync(ctx, p2pEngine)
		case lowerInput == "help" || lowerInput == "h":
			printHelp(p2pEngine.PairingCode())
		default:
			fmt.Println("⚠️  AI engine not loaded. Commands: 'list', 'sync', 'peers', 'clear', 'quit'")
		}
	}
}
