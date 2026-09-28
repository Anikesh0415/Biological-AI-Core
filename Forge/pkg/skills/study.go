package skills

import (
	"fmt"
	"forge/pkg/executor"
)

type StudyModeSkill struct{}

func init() {
	Register(&StudyModeSkill{})
}

func (s *StudyModeSkill) Name() string {
	return "StudyMode"
}

func (s *StudyModeSkill) Match(intent string) bool {
	return ContainsAllKeywords(intent, "study", "notion") || ContainsAllKeywords(intent, "study", "clock")
}

func (s *StudyModeSkill) Execute(intent string) error {
	fmt.Println("Executing Advanced Skill: StudyMode")
	
	actions := []executor.Action{
		// 1. Open Notion
		{Type: "key", Key: "win"},
		{Type: "sleep", Ms: 1000},
		{Type: "type", Text: "Notion"},
		{Type: "sleep", Ms: 1000},
		{Type: "key", Key: "enter"},
		{Type: "sleep", Ms: 3000}, // Wait for Notion to load
		
		// 2. Open Clock (Focus Sessions)
		{Type: "key", Key: "win"},
		{Type: "sleep", Ms: 1000},
		{Type: "type", Text: "Clock"},
		{Type: "sleep", Ms: 1000},
		{Type: "key", Key: "enter"},
		{Type: "sleep", Ms: 2000}, // Wait for Clock to load
		
		// Note: At this point, we could use UIA to find the exact "Start" button for focus sessions.
		// For now, we assume the user has Focus Sessions as the default tab in Clock, and we just tab to it or click it using UIA later.
	}
	
	executor.ExecutePlan(actions)
	return nil
}
