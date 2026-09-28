package screentime

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// AppCategory represents the behavioral classification of an application.
type AppCategory string

const (
	CategoryDopamineSink   AppCategory = "Dopamine Trap"
	CategoryProductive     AppCategory = "Productive"
	CategoryEntertainment  AppCategory = "Entertainment"
	CategoryUtility        AppCategory = "Utility"
	CategoryUnknown        AppCategory = "Other"
)

// AppUsage represents a single application's usage duration and classification.
type AppUsage struct {
	App            string        `json:"app"`
	Duration       time.Duration `json:"duration"`
	DurationStr    string        `json:"duration_str"`
	Category       AppCategory   `json:"category"`
	IsDopamineTrap bool          `json:"is_dopamine_trap"`
	Percentage     float64       `json:"percentage"`
}

// ScreentimeReport captures the comprehensive parsed metrics of a screentime screenshot or log.
type ScreentimeReport struct {
	TotalTime          time.Duration `json:"total_time"`
	TotalTimeStr       string        `json:"total_time_str"`
	ProductiveTime     time.Duration `json:"productive_time"`
	ProductiveTimeStr  string        `json:"productive_time_str"`
	DistractionTime    time.Duration `json:"distraction_time"`
	DistractionTimeStr string        `json:"distraction_time_str"`
	DopamineTrapScore  int           `json:"dopamine_trap_score"` // 0 - 100
	FocusRatio         float64       `json:"focus_ratio"`          // Percentage productive (0 - 100)
	AnnualLostHours    int           `json:"annual_lost_hours"`    // Projected yearly hours spent on dopamine sinks
	Apps               []AppUsage    `json:"apps"`
	TopDopamineSink    string        `json:"top_dopamine_sink"`
	Pickups            int           `json:"pickups,omitempty"`
	Notifications      int           `json:"notifications,omitempty"`
}

// CognitiveAudit bundles the screentime report with AI-generated habit coaching advice.
type CognitiveAudit struct {
	Report            *ScreentimeReport `json:"report"`
	AICoachAdvice     string            `json:"ai_coach_advice"`
	SuggestedTask     string            `json:"suggested_task"`
	SuggestedTimeExpr string            `json:"suggested_time_expr"`
}

// Known app classifications database (lightweight, zero RAM overhead)
var knownCategories = map[string]AppCategory{
	// Dopamine Traps (Algorithmic infinite-scroll feeds)
	"instagram":   CategoryDopamineSink,
	"tiktok":      CategoryDopamineSink,
	"youtube":     CategoryDopamineSink,
	"reels":       CategoryDopamineSink,
	"shorts":      CategoryDopamineSink,
	"reddit":      CategoryDopamineSink,
	"twitter":     CategoryDopamineSink,
	"x":           CategoryDopamineSink,
	"facebook":    CategoryDopamineSink,
	"snapchat":    CategoryDopamineSink,
	"threads":     CategoryDopamineSink,
	"twitch":      CategoryDopamineSink,
	"pinterest":   CategoryDopamineSink,

	// Entertainment
	"netflix":     CategoryEntertainment,
	"spotify":     CategoryEntertainment,
	"prime video": CategoryEntertainment,
	"disney":      CategoryEntertainment,
	"hulu":        CategoryEntertainment,
	"steam":       CategoryEntertainment,
	"roblox":      CategoryEntertainment,
	"candy crush": CategoryEntertainment,
	"pubg":        CategoryEntertainment,
	"genshin":     CategoryEntertainment,

	// Productive / Deep Work / Learning
	"vs code":     CategoryProductive,
	"vscode":      CategoryProductive,
	"terminal":    CategoryProductive,
	"github":      CategoryProductive,
	"slack":       CategoryProductive,
	"notion":      CategoryProductive,
	"obsidian":    CategoryProductive,
	"docs":        CategoryProductive,
	"sheets":      CategoryProductive,
	"figma":       CategoryProductive,
	"linear":      CategoryProductive,
	"anki":        CategoryProductive,
	"duolingo":    CategoryProductive,
	"kindle":      CategoryProductive,
	"books":       CategoryProductive,
	"coursera":    CategoryProductive,
	"planner":     CategoryProductive,

	// Utilities / Comms
	"whatsapp":    CategoryUtility,
	"telegram":    CategoryUtility,
	"messages":    CategoryUtility,
	"phone":       CategoryUtility,
	"gmail":       CategoryUtility,
	"mail":        CategoryUtility,
	"outlook":     CategoryUtility,
	"calendar":    CategoryUtility,
	"notes":       CategoryUtility,
	"settings":    CategoryUtility,
	"maps":        CategoryUtility,
	"uber":        CategoryUtility,
	"calculator":  CategoryUtility,
	"clock":       CategoryUtility,
}

// ClassifyApp identifies the behavioral category of an application name.
func ClassifyApp(appName string) AppCategory {
	lower := strings.ToLower(strings.TrimSpace(appName))
	for k, cat := range knownCategories {
		if strings.Contains(lower, k) {
			return cat
		}
	}
	return CategoryUnknown
}

// ParseScreentimeText extracts structured screentime metrics from OCR or pasted text feeds.
// Supports iOS Screen Time, Android Digital Wellbeing, and Windows Screen Time formats.
func ParseScreentimeText(raw string) *ScreentimeReport {
	report := &ScreentimeReport{
		Apps: make([]AppUsage, 0),
	}

	// Support comma or semicolon delimited inputs (e.g., "Instagram 2h, YouTube 1h")
	normalized := raw
	if !strings.Contains(raw, "\n") && (strings.Contains(raw, ",") || strings.Contains(raw, ";")) {
		normalized = strings.ReplaceAll(normalized, ",", "\n")
		normalized = strings.ReplaceAll(normalized, ";", "\n")
	}

	lines := strings.Split(normalized, "\n")
	var cleanedLines []string
	for _, l := range lines {
		t := strings.TrimSpace(l)
		if t != "" {
			cleanedLines = append(cleanedLines, t)
		}
	}

	totalRegex := regexp.MustCompile(`(?i)(?:screen\s*time|daily\s*average|total|today)[\s:]*(?:(\d+)\s*(?:h|hr|hrs|hours?))?\s*(?:(\d+)\s*(?:m|min|mins|minutes?))?`)
	pickupsRegex := regexp.MustCompile(`(?i)(?:pickups?|unlocks?)[\s:]*(\d+)|(\d+)\s*(?:pickups?|unlocks?)`)
	notifsRegex := regexp.MustCompile(`(?i)(?:notifications?)[\s:]*(\d+)|(\d+)\s*(?:notifications?)`)

	// 1. First pass: look for explicit total screen time header
	for _, line := range cleanedLines {
		if m := totalRegex.FindStringSubmatch(line); m != nil && (m[1] != "" || m[2] != "") {
			h, _ := strconv.Atoi(m[1])
			mins, _ := strconv.Atoi(m[2])
			if h > 0 || mins > 0 {
				dur := time.Duration(h)*time.Hour + time.Duration(mins)*time.Minute
				if dur > report.TotalTime {
					report.TotalTime = dur
				}
			}
		}
		if m := pickupsRegex.FindStringSubmatch(line); m != nil {
			if m[1] != "" {
				report.Pickups, _ = strconv.Atoi(m[1])
			} else if len(m) > 2 && m[2] != "" {
				report.Pickups, _ = strconv.Atoi(m[2])
			}
		}
		if m := notifsRegex.FindStringSubmatch(line); m != nil {
			if m[1] != "" {
				report.Notifications, _ = strconv.Atoi(m[1])
			} else if len(m) > 2 && m[2] != "" {
				report.Notifications, _ = strconv.Atoi(m[2])
			}
		}
	}

	// 2. Second pass: look for app + duration associations
	appEntryRegex := regexp.MustCompile(`(?i)^([a-zA-Z0-9\s\.\_\-]+?)[\s\-:–]+(\d+\s*(?:h|hr|hrs|hours?))?\s*(\d+\s*(?:m|min|mins|minutes?))$`)

	for i := 0; i < len(cleanedLines); i++ {
		line := cleanedLines[i]

		// Check single line format: "Instagram 2h 30m"
		if match := appEntryRegex.FindStringSubmatch(line); len(match) > 3 {
			appName := strings.TrimSpace(match[1])
			dur := parseDurationString(match[2] + " " + match[3])
			if isValidApp(appName) && dur > 0 {
				report.addApp(appName, dur)
				continue
			}
		}

		// Check two-line format:
		// Line i: App Name
		// Line i+1: Duration
		if i+1 < len(cleanedLines) {
			dur := parseDurationString(cleanedLines[i+1])
			if dur > 0 && isValidApp(line) {
				report.addApp(line, dur)
				i++ // skip duration line
				continue
			}
		}
	}

	// If no explicit total was captured, sum all parsed apps
	var sumApps time.Duration
	for _, app := range report.Apps {
		sumApps += app.Duration
	}
	if report.TotalTime < sumApps {
		report.TotalTime = sumApps
	}

	// Compute behavioral metrics
	report.finalizeMetrics()
	return report
}

func parseDurationString(s string) time.Duration {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	hRegex := regexp.MustCompile(`(?i)(\d+)\s*(?:h|hr|hrs|hours?)`)
	mRegex := regexp.MustCompile(`(?i)(\d+)\s*(?:m|min|mins|minutes?)`)

	var total time.Duration
	if hm := hRegex.FindStringSubmatch(s); len(hm) > 1 {
		h, _ := strconv.Atoi(hm[1])
		total += time.Duration(h) * time.Hour
	}
	if mm := mRegex.FindStringSubmatch(s); len(mm) > 1 {
		m, _ := strconv.Atoi(mm[1])
		total += time.Duration(m) * time.Minute
	}
	return total
}

func isValidApp(name string) bool {
	clean := strings.TrimSpace(strings.ToLower(name))
	clean = strings.TrimRight(clean, ":-–— ")
	if len(clean) < 2 || len(clean) > 35 {
		return false
	}
	// Filter out common UI labels that are not app names
	ignores := []string{
		"daily screen time", "screen time", "daily average", "pickups", "notifications", "today",
		"yesterday", "last 7 days", "limit", "most used", "categories", "show more",
		"see all", "settings", "digital wellbeing", "dashboard", "total", "daily", "average",
	}
	for _, ign := range ignores {
		if clean == ign || strings.HasPrefix(clean, ign) || strings.Contains(clean, "screen time") {
			return false
		}
	}
	return true
}

func (r *ScreentimeReport) addApp(name string, dur time.Duration) {
	name = strings.TrimSpace(name)
	category := ClassifyApp(name)
	isDopamine := category == CategoryDopamineSink

	// Check if already added
	for idx, a := range r.Apps {
		if strings.EqualFold(a.App, name) {
			r.Apps[idx].Duration += dur
			r.Apps[idx].DurationStr = FormatDuration(r.Apps[idx].Duration)
			return
		}
	}

	r.Apps = append(r.Apps, AppUsage{
		App:            name,
		Duration:       dur,
		DurationStr:    FormatDuration(dur),
		Category:       category,
		IsDopamineTrap: isDopamine,
	})
}

func (r *ScreentimeReport) finalizeMetrics() {
	if r.TotalTime == 0 {
		r.TotalTimeStr = "0m"
		return
	}

	// Sort apps descending by duration
	sort.Slice(r.Apps, func(i, j int) bool {
		return r.Apps[i].Duration > r.Apps[j].Duration
	})

	r.TotalTimeStr = FormatDuration(r.TotalTime)

	for i := range r.Apps {
		r.Apps[i].Percentage = (float64(r.Apps[i].Duration) / float64(r.TotalTime)) * 100
		if r.Apps[i].Category == CategoryProductive {
			r.ProductiveTime += r.Apps[i].Duration
		} else if r.Apps[i].IsDopamineTrap || r.Apps[i].Category == CategoryEntertainment {
			r.DistractionTime += r.Apps[i].Duration
		}
	}

	r.ProductiveTimeStr = FormatDuration(r.ProductiveTime)
	r.DistractionTimeStr = FormatDuration(r.DistractionTime)

	// Focus Ratio = Productive / Total
	if r.TotalTime > 0 {
		r.FocusRatio = (float64(r.ProductiveTime) / float64(r.TotalTime)) * 100
	}

	// Dopamine Trap Score: 0 (clean focus) to 100 (severe doomscrolling)
	distractionPct := float64(r.DistractionTime) / float64(r.TotalTime)
	hoursDistracted := r.DistractionTime.Hours()

	score := int(distractionPct*60.0 + (hoursDistracted/6.0)*40.0)
	if score > 100 {
		score = 100
	} else if score < 0 {
		score = 0
	}
	r.DopamineTrapScore = score

	// Annual Lost Hours = Distraction daily average * 365
	r.AnnualLostHours = int(r.DistractionTime.Hours() * 365)

	// Identify top dopamine sink
	for _, app := range r.Apps {
		if app.IsDopamineTrap {
			r.TopDopamineSink = fmt.Sprintf("%s (%s)", app.App, app.DurationStr)
			break
		}
	}
}

// FormatDuration formats time.Duration as clean humanistic text (e.g. "4h 25m" or "45m").
func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	} else if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

// ScreentimeCoachSystemPrompt defines the expert persona and rules for habit coaching.
const ScreentimeCoachSystemPrompt = `You are an encouraging AI productivity coach. Help the user improve their daily schedule based on their screen time.
Rules:
1. Be encouraging, constructive, and direct.
2. Note their highest screen time apps and contrast with productive time.
3. Recommend 1-2 realistic daily habits and suggest a focus block to schedule in Planner Bot.
4. Do not use emojis.`

// BuildPrompt constructs a ChatML prompt formatted with rich screentime facts.
func (r *ScreentimeReport) BuildPrompt() string {
	var sb strings.Builder
	sb.WriteString("<|im_start|>system\n")
	sb.WriteString(ScreentimeCoachSystemPrompt)
	sb.WriteString("\n<|im_end|>\n")

	sb.WriteString("<|im_start|>user\n")
	sb.WriteString("Here is my screen time summary for today:\n")
	sb.WriteString(fmt.Sprintf("- Total Screen Time: %s\n", r.TotalTimeStr))
	sb.WriteString(fmt.Sprintf("- Productive Time: %s (Focus Ratio: %.1f%%)\n", r.ProductiveTimeStr, r.FocusRatio))
	sb.WriteString(fmt.Sprintf("- Leisure Time: %s\n", r.DistractionTimeStr))
	if r.TopDopamineSink != "" {
		sb.WriteString(fmt.Sprintf("- Top Leisure App: %s\n", r.TopDopamineSink))
	}
	if r.Pickups > 0 {
		sb.WriteString(fmt.Sprintf("- Device Pickups: %d times\n", r.Pickups))
	}

	if len(r.Apps) > 0 {
		sb.WriteString("\nApp Breakdown:\n")
		limit := len(r.Apps)
		if limit > 6 {
			limit = 6
		}
		for i := 0; i < limit; i++ {
			app := r.Apps[i]
			sb.WriteString(fmt.Sprintf("- %s: %s (%.1f%% of screen time)\n", app.App, app.DurationStr, app.Percentage))
		}
	}

	sb.WriteString("\nPlease give me constructive advice to improve my productivity tomorrow and suggest a focus sprint to schedule.\n")
	sb.WriteString("<|im_end|>\n")
	sb.WriteString("<|im_start|>assistant\n")

	return sb.String()
}

