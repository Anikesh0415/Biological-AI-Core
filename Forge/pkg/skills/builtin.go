package skills

import (
	"fmt"
	"forge/pkg/executor"
	"strings"
)

// InitBuiltinSkills registers universal dynamic skills directly in Go, removing the dependency on Python.
func InitBuiltinSkills() {
	// Universal Apps & Openers
	openApps := []string{
		"notepad", "calculator", "paint", "cmd", "powershell", "task manager",
		"control panel", "snipping tool", "spotify", "discord", "whatsapp",
		"telegram", "slack", "teams", "zoom", "brave", "chrome", "edge", "firefox",
	}

	for _, app := range openApps {
		Register(&DynamicSkill{
			SkillName: fmt.Sprintf("open %s", app),
			Actions: []executor.Action{
				{Type: "key", Key: "win"},
				{Type: "sleep", Ms: 800},
				{Type: "type", Text: app},
				{Type: "sleep", Ms: 800},
				{Type: "key", Key: "enter"},
			},
		})
	}

	// Universal Parameterized Macros
	universalMacros := []struct {
		intent  string
		actions []executor.Action
	}{
		{
			intent: "send {message} to {contact} on whatsapp",
			actions: []executor.Action{
				{Type: "key", Key: "win"},
				{Type: "sleep", Ms: 800},
				{Type: "type", Text: "whatsapp"},
				{Type: "sleep", Ms: 800},
				{Type: "key", Key: "enter"},
				{Type: "sleep", Ms: 4000},
				{Type: "key", Key: "ctrl+f"},
				{Type: "sleep", Ms: 800},
				{Type: "type", Text: "{contact}"},
				{Type: "sleep", Ms: 800},
				{Type: "key", Key: "enter"},
				{Type: "sleep", Ms: 800},
				{Type: "type", Text: "{message}"},
				{Type: "sleep", Ms: 500},
				{Type: "key", Key: "enter"},
			},
		},
		{
			intent: "play {song} on spotify",
			actions: []executor.Action{
				{Type: "key", Key: "win"},
				{Type: "sleep", Ms: 800},
				{Type: "type", Text: "spotify"},
				{Type: "sleep", Ms: 800},
				{Type: "key", Key: "enter"},
				{Type: "sleep", Ms: 4000},
				{Type: "key", Key: "ctrl+l"},
				{Type: "sleep", Ms: 800},
				{Type: "type", Text: "{song}"},
				{Type: "sleep", Ms: 800},
				{Type: "key", Key: "enter"},
				{Type: "sleep", Ms: 1000},
				{Type: "key", Key: "tab"},
				{Type: "sleep", Ms: 100},
				{Type: "key", Key: "enter"},
			},
		},
		{
			intent: "browse {site}",
			actions: []executor.Action{
				{Type: "key", Key: "win+r"},
				{Type: "sleep", Ms: 800},
				{Type: "type", Text: "brave https://{site}"},
				{Type: "sleep", Ms: 800},
				{Type: "key", Key: "enter"},
			},
		},
	}

	for _, m := range universalMacros {
		Register(&DynamicSkill{
			SkillName: m.intent,
			Actions:   m.actions,
		})
	}
}

// UniversalSearchSkill handles any "search {query} on {site}" dynamically in pure Go!
type UniversalSearchSkill struct{}

func init() {
	Register(&UniversalSearchSkill{})
}

func (s *UniversalSearchSkill) Name() string {
	return "UniversalSearchSkill"
}

func (s *UniversalSearchSkill) Match(intent string) bool {
	lower := strings.ToLower(intent)
	return strings.HasPrefix(lower, "search ") && strings.Contains(lower, " on ")
}

func (s *UniversalSearchSkill) Execute(intent string) error {
	lower := strings.ToLower(intent)
	// Extract query and site: "search <query> on <site>"
	parts := strings.Split(lower, " on ")
	if len(parts) < 2 {
		return fmt.Errorf("invalid search format")
	}

	query := strings.TrimPrefix(parts[0], "search ")
	site := parts[1]

	// Map common site names to search URLs
	searchUrlMap := map[string]string{
		"youtube":       "https://www.youtube.com/results?search_query=",
		"amazon":        "https://www.amazon.com/s?k=",
		"ebay":          "https://www.ebay.com/sch/i.html?_nkw=",
		"reddit":        "https://www.reddit.com/search/?q=",
		"wikipedia":     "https://en.wikipedia.org/wiki/Special:Search?search=",
		"github":        "https://github.com/search?q=",
		"stackoverflow": "https://stackoverflow.com/search?q=",
		"google":        "https://www.google.com/search?q=",
		"chatgpt":       "https://chatgpt.com/?q=",
		"claude":        "https://claude.ai/new?q=",
		"twitter":       "https://twitter.com/search?q=",
		"x":             "https://twitter.com/search?q=",
		"instagram":     "https://www.instagram.com/",
		"spotify":       "https://open.spotify.com/search/",
	}

	targetUrl := fmt.Sprintf("https://www.%s.com/search?q=%s", site, query)
	if baseUrl, exists := searchUrlMap[site]; exists {
		targetUrl = baseUrl + query
	}

	actions := []executor.Action{
		{Type: "key", Key: "win+r"},
		{Type: "sleep", Ms: 800},
		{Type: "type", Text: "brave " + targetUrl},
		{Type: "sleep", Ms: 800},
		{Type: "key", Key: "enter"},
	}

	executor.ExecutePlan(actions)
	return nil
}
