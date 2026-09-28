package skills

import (
	"fmt"
	"forge/pkg/executor"
	"strings"
)

type BrowserSearchSkill struct{}

func init() {
	Register(&BrowserSearchSkill{})
}

func (s *BrowserSearchSkill) Name() string {
	return "BrowserSearch"
}

func (s *BrowserSearchSkill) Match(intent string) bool {
	// Do not intercept if it's meant for youtube, spotify, or an already opened specific app
	if strings.Contains(intent, "youtube") || strings.Contains(intent, "spotify") || strings.Contains(intent, "opened") {
		return false
	}
	return ContainsAllKeywords(intent, "search", "for") || 
	       ContainsAllKeywords(intent, "google")
}

func (s *BrowserSearchSkill) Execute(intent string) error {
	fmt.Println("Executing Advanced Skill: BrowserSearch")
	
	// Extremely naive extraction for demonstration.
	// "search for cute cats" -> "cute cats"
	searchTerm := "GitHub" // fallback
	if idx := strings.Index(intent, "search for"); idx != -1 {
		searchTerm = strings.TrimSpace(intent[idx+10:])
	} else if idx := strings.Index(intent, "google"); idx != -1 {
		searchTerm = strings.TrimSpace(intent[idx+6:])
	}
	
	actions := []executor.Action{
		// 1. Open Brave (Default browser via run prompt)
		{Type: "key", Key: "win"},
		{Type: "sleep", Ms: 1000},
		{Type: "type", Text: "brave"},
		{Type: "sleep", Ms: 1000},
		{Type: "key", Key: "enter"},
		{Type: "sleep", Ms: 3000},
		
		// 2. Focus address bar (Ctrl+L/Alt+D equivalent, or just assuming new tab focuses it)
		// For safety, typing directly usually works on a new tab.
		{Type: "type", Text: searchTerm},
		{Type: "sleep", Ms: 500},
		{Type: "key", Key: "enter"},
	}
	
	executor.ExecutePlan(actions)
	return nil
}
