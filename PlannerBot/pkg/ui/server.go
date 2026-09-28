package ui

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"planner_bot/pkg/analyzer"
	"planner_bot/pkg/extractor"
	"planner_bot/pkg/grammar"
	"planner_bot/pkg/p2p"
	"planner_bot/pkg/screentime"
	"planner_bot/pkg/store"
	"planner_bot/pkg/timeparse"

	"github.com/google/uuid"
)

//go:embed web/*
var webFS embed.FS

// Server hosts the embedded Web UI and REST API.
type Server struct {
	port           int
	store          *store.Store
	extractor      *extractor.Engine
	p2pEngine      *p2p.Engine
	analyzer       *analyzer.Analyzer
	httpServer     *http.Server
	listener       net.Listener
	eventClients   map[chan *store.Reminder]bool
	eventClientsMu sync.Mutex
	mu             sync.RWMutex
}

// Config configures the UI server.
type Config struct {
	Port      int
	Store     *store.Store
	Extractor *extractor.Engine
	P2PEngine *p2p.Engine
	Analyzer  *analyzer.Analyzer
}

// NewServer creates a new embedded UI server.
func NewServer(cfg Config) *Server {
	return &Server{
		port:         cfg.Port,
		store:        cfg.Store,
		extractor:    cfg.Extractor,
		p2pEngine:    cfg.P2PEngine,
		analyzer:     cfg.Analyzer,
		eventClients: make(map[chan *store.Reminder]bool),
	}
}

func (s *Server) Start() error {
	addr := fmt.Sprintf("127.0.0.1:%d", s.port)
	ln, err := net.Listen("tcp", addr)
	if err != nil && s.port != 0 {
		// Fallback to auto-assigned port if preferred port is in use
		ln, err = net.Listen("tcp", "127.0.0.1:0")
	}
	if err != nil {
		return fmt.Errorf("ui: listen: %w", err)
	}
	s.listener = ln
	s.port = ln.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()

	// Static asset handler
	distFS, err := fs.Sub(webFS, "web")
	if err != nil {
		return fmt.Errorf("ui: sub fs: %w", err)
	}
	mux.Handle("/", http.FileServer(http.FS(distFS)))

	// API endpoints
	mux.HandleFunc("/api/status", s.handleStatus)
	mux.HandleFunc("/api/chat", s.handleChat)
	mux.HandleFunc("/api/reminders", s.handleReminders)
	mux.HandleFunc("/api/reminders/", s.handleReminderByID)
	mux.HandleFunc("/api/reminders/due", s.handleDueReminders)
	mux.HandleFunc("/api/events", s.handleEvents)
	mux.HandleFunc("/api/peers", s.handlePeers)
	mux.HandleFunc("/api/sync", s.handleSync)
	mux.HandleFunc("/api/analyze", s.handleAnalyze)
	mux.HandleFunc("/api/screentime/analyze", s.handleScreentimeAnalyze)
	mux.HandleFunc("/api/briefing", s.handleBriefing)

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	go func() {
		_ = s.httpServer.Serve(ln)
	}()

	return nil
}

// Port returns the assigned listening port.
func (s *Server) Port() int {
	return s.port
}

// URL returns the web UI address.
func (s *Server) URL() string {
	return fmt.Sprintf("http://127.0.0.1:%d", s.port)
}

// OpenBrowser opens the local Web UI in the user's default browser.
func (s *Server) OpenBrowser() {
	url := s.URL()
	switch runtime.GOOS {
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		_ = exec.Command("open", url).Start()
	case "linux":
		_ = exec.Command("xdg-open", url).Start()
	}
}

// SetExtractor updates the AI engine dynamically once loaded in the background.
func (s *Server) SetExtractor(e *extractor.Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.extractor = e
	if s.analyzer != nil && e != nil {
		s.analyzer.SetLLM(e.Server())
	}
}

// NotifyReminder broadcasts a due or fired reminder to all connected SSE clients.
func (s *Server) NotifyReminder(r *store.Reminder) {
	if r == nil {
		return
	}
	s.eventClientsMu.Lock()
	defer s.eventClientsMu.Unlock()
	for ch := range s.eventClients {
		select {
		case ch <- r:
		default:
		}
	}
}

func (s *Server) registerEventClient(ch chan *store.Reminder) {
	s.eventClientsMu.Lock()
	defer s.eventClientsMu.Unlock()
	s.eventClients[ch] = true
}

func (s *Server) unregisterEventClient(ch chan *store.Reminder) {
	s.eventClientsMu.Lock()
	defer s.eventClientsMu.Unlock()
	delete(s.eventClients, ch)
	close(ch)
}

// handleEvents provides Server-Sent Events (SSE) for real-time reminder alerts and updates.
func (s *Server) handleEvents(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientCh := make(chan *store.Reminder, 16)
	s.registerEventClient(clientCh)
	defer s.unregisterEventClient(clientCh)

	// Send initial heartbeat
	fmt.Fprintf(w, ": connected\n\n")
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case rem, ok := <-clientCh:
			if !ok {
				return
			}
			data, err := json.Marshal(rem)
			if err == nil {
				fmt.Fprintf(w, "event: reminder_due\ndata: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}

// handleDueReminders returns reminders that are currently due or were recently fired (past 5 minutes).
func (s *Server) handleDueReminders(w http.ResponseWriter, r *http.Request) {
	now := time.Now().UTC()
	all, err := s.store.ListAll()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var due []*store.Reminder
	for _, rem := range all {
		if rem.Deleted {
			continue
		}
		// Currently due and not fired
		if !rem.Fired && !rem.FireAt.IsZero() && !rem.FireAt.After(now) {
			due = append(due, rem)
			continue
		}
		// Recently fired in the last 5 minutes (for catchup on page load)
		if rem.Fired && rem.FiredAt != nil && now.Sub(*rem.FiredAt) < 5*time.Minute {
			due = append(due, rem)
		}
	}

	if due == nil {
		due = []*store.Reminder{}
	}
	writeJSON(w, due)
}

// Close gracefully stops the UI server.
func (s *Server) Close() error {
	if s.httpServer != nil {
		return s.httpServer.Close()
	}
	return nil
}

// handleStatus returns current backend diagnostics.
func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	peers := s.p2pEngine.Peers()
	s.mu.RLock()
	aiOnline := s.extractor != nil
	s.mu.RUnlock()

	status := map[string]interface{}{
		"version":      "0.2.0",
		"pairing_code": s.p2pEngine.PairingCode(),
		"p2p_port":     s.p2pEngine.Port(),
		"peers_count":  len(peers),
		"ai_online":    aiOnline,
	}
	writeJSON(w, status)
}

// handleReminders handles GET (list) and POST (create/extract) requests.
func (s *Server) handleReminders(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		reminders, err := s.store.ListAll()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, reminders)

	case http.MethodPost:
		var req struct {
			Input string `json:"input"`
			Task  string `json:"task"`
			Time  string `json:"time"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		now := time.Now()
		task := req.Task
		timeExpr := req.Time

		// If natural language input was sent, pass through AI engine
		if req.Input != "" && s.extractor != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			extracted, err := s.extractor.ExtractReminder(ctx, req.Input)
			if err == nil && extracted != nil {
				task = extracted.Task
				timeExpr = extracted.Time
			}
		}

		if task == "" {
			task = req.Input
		}

		fireAt, err := timeparse.Parse(timeExpr, now)
		if err != nil {
			// Fallback: default to +1 hour if time was unparseable
			fireAt = now.Add(1 * time.Hour)
		}

		id := uuid.New().String()[:8]
		reminder := &store.Reminder{
			ID:               id,
			Task:             task,
			FireAt:           fireAt.UTC(),
			CreatedAt:        now.UTC(),
			UpdatedAt:        now.UTC(),
			OriginalTimeExpr: timeExpr,
			Version:          1,
		}

		if err := s.store.Save(reminder); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Trigger background P2P sync
		go func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = s.p2pEngine.SyncNow(syncCtx)
		}()

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, reminder)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handleReminderByID handles PUT (edit) and DELETE requests.
func (s *Server) handleReminderByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/reminders/")
	if id == "" {
		http.Error(w, "Missing ID", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodPut:
		// Edit reminder
		var req struct {
			Task     string `json:"task"`
			TimeExpr string `json:"time"`
			Fired    *bool  `json:"fired"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}

		existing, err := s.store.Get(id)
		if err != nil {
			http.Error(w, "Reminder not found", http.StatusNotFound)
			return
		}

		if req.Task != "" {
			existing.Task = req.Task
		}
		if req.TimeExpr != "" {
			parsedTime, err := timeparse.Parse(req.TimeExpr, time.Now())
			if err == nil {
				existing.FireAt = parsedTime.UTC()
				existing.OriginalTimeExpr = req.TimeExpr
			}
		}
		if req.Fired != nil {
			existing.Fired = *req.Fired
			if *req.Fired {
				now := time.Now().UTC()
				existing.FiredAt = &now
			}
		}

		existing.UpdatedAt = time.Now().UTC()
		existing.Version++

		if err := s.store.Save(existing); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Trigger background P2P sync
		go func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = s.p2pEngine.SyncNow(syncCtx)
		}()

		writeJSON(w, existing)

	case http.MethodDelete:
		if err := s.store.SoftDelete(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Trigger background P2P sync
		go func() {
			syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			_, _ = s.p2pEngine.SyncNow(syncCtx)
		}()

		w.WriteHeader(http.StatusNoContent)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// handlePeers lists all discovered P2P devices on the local Wi-Fi.
func (s *Server) handlePeers(w http.ResponseWriter, r *http.Request) {
	peers := s.p2pEngine.Peers()
	writeJSON(w, peers)
}

// handleSync forces an immediate sync with peers.
func (s *Server) handleSync(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	changes, err := s.p2pEngine.SyncNow(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, map[string]interface{}{
		"status":         "success",
		"changes_synced": changes,
	})
}

// handleAnalyze triggers day reflection and tomorrow's suggestions.
func (s *Server) handleAnalyze(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	summary, err := s.analyzer.AnalyzeDay(ctx, time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, summary)
}

// handleBriefing returns the current executive state briefing.
func (s *Server) handleBriefing(w http.ResponseWriter, r *http.Request) {
	briefing, err := s.store.SynthesizeExecutiveBriefing(time.Now())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, briefing)
}

// handleScreentimeAnalyze processes screentime OCR text or image payloads,
// runs the neuro-symbolic habit engine, queries the AI habit coach, and returns structured insights.
func (s *Server) handleScreentimeAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OCRText string `json:"ocr_text"`
	}

	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(10 << 20); err == nil {
			req.OCRText = r.FormValue("ocr_text")
		}
	} else {
		_ = json.NewDecoder(r.Body).Decode(&req)
	}

	ocrText := strings.TrimSpace(req.OCRText)
	if ocrText == "" {
		writeJSON(w, map[string]interface{}{
			"reply":  "No readable text was found in the image. Please make sure the screenshot clearly shows your screen time and app usage names, or enter your usage directly in chat (e.g. 'screentime: Instagram 2h, YouTube 1h').",
			"action": "message",
		})
		return
	}

	report := screentime.ParseScreentimeText(ocrText)
	if len(report.Apps) == 0 && report.TotalTime == 0 {
		writeJSON(w, map[string]interface{}{
			"reply":  "Could not identify specific applications and durations in the image. Ensure the screenshot clearly shows app names and times (e.g. 'VS Code 2h 30m, YouTube 1h').",
			"action": "message",
		})
		return
	}

	// Save user message to chat history
	userChat := &store.ChatMessage{
		ID:        uuid.New().String()[:8],
		Role:      "user",
		Content:   fmt.Sprintf("Screentime Audit: %s total (%s distraction)", report.TotalTimeStr, report.DistractionTimeStr),
		Timestamp: time.Now().UTC(),
		Action:    "screentime",
	}
	_ = s.store.SaveChat(userChat)

	s.mu.RLock()
	ext := s.extractor
	s.mu.RUnlock()

	var coachReply string
	if ext != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
		defer cancel()
		advice, err := ext.AnalyzeScreentime(ctx, report)
		if err == nil && strings.TrimSpace(advice) != "" {
			coachReply = advice
		}
	}

	if coachReply == "" {
		coachReply = fmt.Sprintf("Your screentime audit shows %s of device activity today with a Focus Ratio of %.1f%%. Your top attention sink is %s (Dopamine Trap Score: %d/100). If continued daily, this equals approximately %d hours lost to distraction each year. I suggest scheduling a 45-minute focus sprint tomorrow to reclaim your momentum.",
			report.TotalTimeStr, report.FocusRatio, report.TopDopamineSink, report.DopamineTrapScore, report.AnnualLostHours)
	}

	// Save assistant response
	botChat := &store.ChatMessage{
		ID:        uuid.New().String()[:8],
		Role:      "assistant",
		Content:   coachReply,
		Timestamp: time.Now().UTC(),
		Action:    "screentime",
	}
	_ = s.store.SaveChat(botChat)

	writeJSON(w, map[string]interface{}{
		"reply":        coachReply,
		"action":       "screentime",
		"data":         report,
		"coach_advice": coachReply,
	})
}

// handleChat processes conversational inputs, persists chat history to BoltDB,
// crawls previous context within the 8,192 token window, extracts reminders, and generates replies.
func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Return recent conversation history
		chats, err := s.store.ListRecentChats(50)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if chats == nil {
			chats = []*store.ChatMessage{}
		}
		writeJSON(w, chats)
		return

	case http.MethodDelete:
		// Clear chat history
		if err := s.store.ClearChats(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, map[string]string{"status": "cleared"})
		return

	case http.MethodPost:
		var req struct {
			Message string `json:"message"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.Message) == "" {
			http.Error(w, "Message is required", http.StatusBadRequest)
			return
		}

		msg := strings.TrimSpace(req.Message)
		now := time.Now()

		// Save user message immediately to BoltDB chat history
		userChat := &store.ChatMessage{
			ID:        uuid.New().String()[:8],
			Role:      "user",
			Content:   msg,
			Timestamp: now.UTC(),
		}
		_ = s.store.SaveChat(userChat)

		// Helper to save bot response to BoltDB and send HTTP response
		respondAndSave := func(reply string, action string, data interface{}) {
			botChat := &store.ChatMessage{
				ID:        uuid.New().String()[:8],
				Role:      "assistant",
				Content:   reply,
				Timestamp: time.Now().UTC(),
				Action:    action,
			}
			_ = s.store.SaveChat(botChat)
			writeJSON(w, map[string]interface{}{
				"reply":  reply,
				"action": action,
				"data":   data,
			})
		}

		lower := strings.ToLower(msg)

		// Screentime analysis intent
		if strings.Contains(lower, "screentime") || strings.Contains(lower, "screen time") || strings.Contains(lower, "digital wellbeing") || strings.Contains(lower, "app usage") {
			report := screentime.ParseScreentimeText(msg)
			if report.TotalTime == 0 && len(report.Apps) == 0 {
				if strings.Contains(lower, "demo") || strings.Contains(lower, "sample") {
					report = screentime.ParseScreentimeText("Daily Screen Time: 5h 45m\nPickups: 84\nNotifications: 112\n\nInstagram 2h 30m\nYouTube 1h 15m\nVS Code 1h 20m\nWhatsApp 25m\nReddit 15m")
				} else {
					respondAndSave("To audit your screentime, upload or paste a screenshot with the image button, or type your usage directly (e.g. 'screentime: Instagram 2h 30m, VS Code 3h').", "message", nil)
					return
				}
			}
			s.mu.RLock()
			ext := s.extractor
			s.mu.RUnlock()

			var coachReply string
			if ext != nil {
				ctx, cancel := context.WithTimeout(r.Context(), 45*time.Second)
				defer cancel()
				advice, err := ext.AnalyzeScreentime(ctx, report)
				if err == nil && strings.TrimSpace(advice) != "" {
					coachReply = advice
				}
			}
			if coachReply == "" {
				coachReply = fmt.Sprintf("Your screentime audit indicates %s of total usage today with a Focus Ratio of %.1f%%. Top attention sink: %s (Dopamine Trap Score: %d/100). If this continues daily, it totals ~%d hours lost to distraction each year. I suggest scheduling a 45-minute focus block.",
					report.TotalTimeStr, report.FocusRatio, report.TopDopamineSink, report.DopamineTrapScore, report.AnnualLostHours)
			}
			respondAndSave(coachReply, "screentime", report)
			return
		}

		// Executive Briefing intent
		if strings.Contains(lower, "briefing") || strings.Contains(lower, "morning brief") || strings.Contains(lower, "executive brief") || strings.Contains(lower, "daily brief") {
			briefing, err := s.store.SynthesizeExecutiveBriefing(now)
			if err == nil && briefing != nil {
				reply := fmt.Sprintf("%s! Here is your Executive State Briefing for %s:\n\n%s", briefing.Greeting, briefing.Date, briefing.PromptInjection)
				respondAndSave(reply, "briefing", briefing)
				return
			}
		}

		// Socratic Goal Decomposition intent
		if isGoalDecompositionIntent(lower) {
			s.mu.RLock()
			ext := s.extractor
			s.mu.RUnlock()
			if ext != nil {
				ctx, cancel := context.WithTimeout(r.Context(), 35*time.Second)
				defer cancel()
				cleanGoal := cleanGoalPrompt(msg)
				breakdown, err := ext.DecomposeGoal(ctx, cleanGoal)
				if err == nil && breakdown != nil {
					reply := fmt.Sprintf("Strategic 3-Step Milestone Roadmap for **%s**:\n\n%s\n\nI recommend scheduling Step 1 right now to build execution momentum.", breakdown.Goal, breakdown.Overview)
					respondAndSave(reply, "goal_breakdown", breakdown)
					return
				}
			}
		}

		// 1. Day Analysis intent
		if (strings.Contains(lower, "analyze") || strings.Contains(lower, "summary") || strings.Contains(lower, "how was my day") || strings.Contains(lower, "how did i do") || strings.Contains(lower, "debrief") || strings.Contains(lower, "reflection")) && !strings.Contains(lower, "screen") {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()
			summary, err := s.analyzer.AnalyzeDay(ctx, time.Now())
			if err != nil {
				respondAndSave("I ran into an issue analyzing today: "+err.Error(), "general", nil)
				return
			}
			reply := fmt.Sprintf("Daily Stoic Productivity Debrief\n\n%s", summary.AnalysisText)
			respondAndSave(reply, "analysis", summary)
			return
		}

		// 2. List reminders intent (robust query detector)
		if isReminderQueryIntent(lower) {
			pending, err := s.store.ListPending()
			if err != nil {
				respondAndSave("Couldn't fetch reminders: "+err.Error(), "general", nil)
				return
			}
			if len(pending) == 0 {
				respondAndSave("You don't have any pending reminders scheduled right now. Enjoy your free time!", "list", pending)
				return
			}
			reply := fmt.Sprintf("Here are your %d pending reminder(s):", len(pending))
			respondAndSave(reply, "list", pending)
			return
		}

		// 3. Delete / Remove reminder intent via chat (e.g. "delete 06204499" or "remove task buy groceries")
		if strings.HasPrefix(lower, "delete ") || strings.HasPrefix(lower, "remove ") {
			target := strings.TrimSpace(msg[strings.Index(msg, " ")+1:])
			targetLower := strings.ToLower(target)
			// Remove common stop words like "task" or "reminder"
			targetClean := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(targetLower, "reminder"), "task"))

			all, _ := s.store.ListAll()
			var deletedItem *store.Reminder
			for _, item := range all {
				if item.Deleted {
					continue
				}
				if strings.EqualFold(item.ID, target) || strings.EqualFold(strings.ToLower(item.Task), targetClean) || strings.Contains(strings.ToLower(item.Task), targetClean) {
					if err := s.store.SoftDelete(item.ID); err == nil {
						deletedItem = item
						break
					}
				}
			}
			if deletedItem != nil {
				respondAndSave(fmt.Sprintf("Deleted reminder: %q", deletedItem.Task), "message", nil)
				return
			}
			respondAndSave(fmt.Sprintf("I couldn't find an active reminder matching %q.", target), "message", nil)
			return
		}

		// 4. Pairing Code intent
		if strings.Contains(lower, "pairing code") || strings.Contains(lower, "pair") || strings.Contains(lower, "sync code") || lower == "code" {
			code := s.p2pEngine.PairingCode()
			reply := fmt.Sprintf("Your encrypted P2P pairing code is %s.\n\nEnter this code on your mobile app to link both devices securely over local Wi-Fi.", code)
			respondAndSave(reply, "pairing", map[string]string{"pairing_code": code})
			return
		}

		// Retrieve past conversation history & current tasks to crawl
		recentChats, _ := s.store.ListRecentChats(20)
		var historyTurns []grammar.HistoryTurn
		for _, c := range recentChats {
			// Skip the message we just saved so it's not duplicated
			if c.ID == userChat.ID {
				continue
			}
			historyTurns = append(historyTurns, grammar.HistoryTurn{
				Role:    c.Role,
				Content: c.Content,
			})
		}

		allReminders, _ := s.store.ListAll()
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

		s.mu.RLock()
		ext := s.extractor
		s.mu.RUnlock()

		if ext != nil {
			ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
			defer cancel()

			// 4. Try Context-Aware Reminder Extraction only if reminder intent detected
			if isReminderIntent(msg) {
				extracted, err := ext.ExtractReminderWithContext(ctx, msg, historyTurns, taskContexts)
				if err == nil && extracted != nil && extracted.Time != "" {
					cleanTask := cleanExtractedTask(extracted.Task, extracted.Time, msg)
					fireAt, err := timeparse.Parse(extracted.Time, now)
					if err != nil {
						fireAt = now.Add(1 * time.Hour)
					}
					id := uuid.New().String()[:8]
					reminder := &store.Reminder{
						ID:               id,
						Task:             cleanTask,
						FireAt:           fireAt.UTC(),
						CreatedAt:        now.UTC(),
						UpdatedAt:        now.UTC(),
						OriginalTimeExpr: extracted.Time,
						Version:          1,
					}
					_ = s.store.Save(reminder)
					go func() {
						syncCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
						defer cancel()
						_, _ = s.p2pEngine.SyncNow(syncCtx)
					}()

					timeFmt := fireAt.Local().Format("Mon, Jan 2 at 3:04 PM")
					reply := fmt.Sprintf("Got it! I've set a reminder for %q at %s.", cleanTask, timeFmt)
					respondAndSave(reply, "reminder_created", reminder)
					return
				}
			}

			// 5. Conversational Inference Crawling Previous Data & Executive Context
			currentTimeStr := now.Format("Monday, Jan 2, 2006 at 3:04 PM")
			briefing, _ := s.store.SynthesizeExecutiveBriefing(now)
			var briefingContext string
			if briefing != nil {
				briefingContext = briefing.PromptInjection
			}
			chatReply, err := ext.GenerateChatReply(ctx, historyTurns, taskContexts, msg, currentTimeStr, briefingContext)
			if err == nil && strings.TrimSpace(chatReply) != "" {
				respondAndSave(chatReply, "chat", nil)
				return
			}
		}

		// 6. Friendly conversational fallback
		fallback := "I'm ready! You can ask me to set a reminder (e.g. 'remind me to call mom at 7pm'), analyze your day ('analyze my day'), plan a project ('plan: build portfolio'), or view your schedule ('what are my tasks?')."
		respondAndSave(fallback, "general", nil)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(data)
}

func isGoalDecompositionIntent(lower string) bool {
	if strings.HasPrefix(lower, "goal:") || strings.HasPrefix(lower, "project:") || strings.HasPrefix(lower, "plan:") {
		return true
	}
	triggers := []string{
		"decompose", "break down", "roadmap for", "how do i build", "how can i build",
		"plan my project", "steps to", "milestones for", "milestone roadmap",
	}
	for _, t := range triggers {
		if strings.Contains(lower, t) {
			return true
		}
	}
	return false
}

func cleanGoalPrompt(msg string) string {
	prefixes := []string{"goal:", "project:", "plan:", "break down", "decompose", "roadmap for", "how do i build", "how can i build", "plan my project"}
	lower := strings.ToLower(msg)
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			return strings.TrimSpace(msg[len(p):])
		}
	}
	return strings.TrimSpace(msg)
}

func isReminderQueryIntent(lower string) bool {
	lower = strings.TrimSpace(lower)
	if lower == "list" || lower == "ls" || lower == "tasks" || lower == "agenda" || lower == "reminders" || lower == "pending" {
		return true
	}
	isQuery := strings.Contains(lower, "show") ||
		strings.Contains(lower, "list") ||
		strings.Contains(lower, "what") ||
		strings.Contains(lower, "view") ||
		strings.Contains(lower, "check") ||
		strings.Contains(lower, "get") ||
		strings.Contains(lower, "display") ||
		strings.Contains(lower, "see")

	hasTarget := strings.Contains(lower, "reminder") ||
		strings.Contains(lower, "task") ||
		strings.Contains(lower, "agenda") ||
		strings.Contains(lower, "schedule") ||
		strings.Contains(lower, "todo") ||
		strings.Contains(lower, "to do") ||
		strings.Contains(lower, "pending")

	return isQuery && hasTarget
}

var timePatternRegex = regexp.MustCompile(`(?i)(\b\d{1,2}(:\d{2})?\s*(am|pm)\b|\b\d+\s*(minutes?|mins?|hours?|hrs?|seconds?|secs?)\b|\bat\s+\d{1,2}(:\d{2})?\b|\bin\s+\d+\s*(minutes?|mins?|hours?|hrs?)\b)`)

func isReminderIntent(msg string) bool {
	lower := strings.ToLower(strings.TrimSpace(msg))
	if isReminderQueryIntent(lower) {
		return false
	}
	if strings.HasPrefix(lower, "delete ") || strings.HasPrefix(lower, "remove ") {
		return false
	}

	// Check if this is an informational / conversational question that should not be treated as a reminder
	questionPrefixes := []string{
		"who is", "who are", "who was", "who developed", "who created", "who built", "who made",
		"what is", "what was", "what are", "what can you", "what do you",
		"why is", "why are", "why does", "why do",
		"how to", "how do", "how can", "how does", "how is", "how dumb",
		"tell me", "explain", "help me understand", "can you tell", "i have a question",
		"i am asking", "im asking", "i asked", "nah ", "bro ", "broh ",
	}
	hasExplicitScheduleWord := strings.Contains(lower, "remind me") ||
		strings.Contains(lower, "schedule a") ||
		strings.Contains(lower, "schedule me") ||
		strings.Contains(lower, "set an alarm") ||
		strings.Contains(lower, "set a reminder") ||
		strings.Contains(lower, "set reminder")

	for _, cp := range questionPrefixes {
		if strings.HasPrefix(lower, cp) && !hasExplicitScheduleWord {
			return false
		}
	}

	triggers := []string{
		"remind", "reminder", "schedule", "don't forget", "dont forget",
		"set an alarm", "set alarm", "alarm for", "alert me", "todo", "appointment", "meeting",
	}
	for _, t := range triggers {
		if strings.Contains(lower, t) {
			return true
		}
	}

	// Strict time pattern matching (avoids matching "I am" or bare words)
	if timePatternRegex.MatchString(lower) {
		return true
	}

	dateWords := []string{
		"tomorrow", "tonight", "today at", "next week", "next monday",
		"next tuesday", "next wednesday", "next thursday", "next friday", "next saturday", "next sunday",
		"o'clock",
	}
	for _, dw := range dateWords {
		if strings.Contains(lower, dw) {
			return true
		}
	}
	return false
}

func cleanExtractedTask(task, rawTime, originalMsg string) string {
	t := strings.TrimSpace(task)
	lower := strings.ToLower(t)

	// Strip common redundant prefixes left by the LLM
	prefixes := []string{
		"set an alarm for", "set alarm for", "set a reminder for", "set reminder for",
		"set an alarm to", "set a reminder to", "set reminder to",
		"remind me to", "remind me for", "remind me about", "remind me",
		"reminder to", "reminder for", "alarm for", "alarm to",
	}
	for _, p := range prefixes {
		if strings.HasPrefix(lower, p) {
			t = strings.TrimSpace(t[len(p):])
			lower = strings.ToLower(t)
		}
	}

	// Remove trailing/leading prepositions
	for _, w := range []string{"to ", "for ", "about "} {
		if strings.HasPrefix(lower, w) {
			t = strings.TrimSpace(t[len(w):])
			lower = strings.ToLower(t)
		}
	}

	// If empty, identical to the time expression, or purely a generic token, give it a clean descriptive title
	if t == "" || strings.EqualFold(t, strings.TrimSpace(rawTime)) || lower == "alarm" || lower == "reminder" || lower == "timer" || lower == "task" {
		if rawTime != "" {
			return fmt.Sprintf("Quick Reminder (%s)", rawTime)
		}
		return "Quick Reminder"
	}

	return t
}


