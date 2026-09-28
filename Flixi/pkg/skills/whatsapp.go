package skills

import (
	"fmt"
	"forge/pkg/executor"
)

type AIMessengerSkill struct{}

func init() {
	Register(&AIMessengerSkill{})
}

func (s *AIMessengerSkill) Name() string {
	return "AIMessenger"
}

func (s *AIMessengerSkill) Match(intent string) bool {
	return ContainsAllKeywords(intent, "gemini", "whatsapp", "letter") || 
	       ContainsAllKeywords(intent, "chatgpt", "whatsapp")
}

func (s *AIMessengerSkill) Execute(intent string) error {
	fmt.Println("Executing Advanced Skill: AIMessenger")
	
	// This would eventually extract the contact name and the prompt topic from the intent.
	// For now, we perform a generalized sequence.
	
	actions := []executor.Action{
		// 1. Open Gemini (Browser)
		{Type: "key", Key: "win"},
		{Type: "sleep", Ms: 1000},
		{Type: "type", Text: "gemini.google.com"},
		{Type: "sleep", Ms: 1000},
		{Type: "key", Key: "enter"},
		{Type: "sleep", Ms: 5000}, // Wait for browser
		
		// 2. Type Prompt
		{Type: "type", Text: "Write a short message to Balram asking how he is doing."},
		{Type: "sleep", Ms: 500},
		{Type: "key", Key: "enter"},
		{Type: "sleep", Ms: 10000}, // Wait for Gemini to generate (10s)
		
		// 3. Copy (Assuming Tab/Enter magic or UIA click later)
		// ... UIA click "Copy" button ...
		
		// 4. Open WhatsApp
		{Type: "key", Key: "win"},
		{Type: "sleep", Ms: 1000},
		{Type: "type", Text: "WhatsApp"},
		{Type: "sleep", Ms: 1000},
		{Type: "key", Key: "enter"},
		{Type: "sleep", Ms: 3000},
		
		// 5. Search Contact
		{Type: "type", Text: "Balram"},
		{Type: "sleep", Ms: 1000},
		{Type: "key", Key: "enter"},
		{Type: "sleep", Ms: 1000},
		
		// 6. Paste and Send
		// (Ctrl+V not natively implemented in typeText yet, we might need a ctrl modifier action)
	}
	
	executor.ExecutePlan(actions)
	return nil
}
