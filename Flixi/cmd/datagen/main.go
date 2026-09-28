package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatML struct {
	Messages []ChatMessage `json:"messages"`
}

var browsers = []string{"Brave", "Chrome", "Edge", "Firefox"}
var searchEngines = []string{"Google", "Bing", "DuckDuckGo"}
var topics = []string{"NCERT chapter on thermodynamics", "quantum mechanics PDF", "latest AI news", "how to bake a cake", "Python machine learning tutorial", "stock market trends", "history of the Roman Empire", "best workout routines"}
var editors = []string{"Notepad", "VS Code", "Sublime Text", "Word"}
var files = []string{"notes.txt", "project.py", "script.js", "todo.md", "draft.docx"}
var emails = []string{"boss@company.com", "friend@gmail.com", "teacher@school.edu"}

func randomElement(list []string) string {
	return list[rand.Intn(len(list))]
}

func generateWebToEmailScenario() []ChatML {
	var samples []ChatML
	for i := 0; i < 2000; i++ {
		browser := randomElement(browsers)
		topic := randomElement(topics)
		email := randomElement(emails)
		
		task := fmt.Sprintf("Search for '%s' on %s, copy the summary, and email it to %s", topic, browser, email)
		
		history := "none"
		uia := `[{"name": "Start", "x": 10, "y": 1080}, {"name": "Search Box", "x": 50, "y": 1080}]`
		screen := "Windows desktop"
		
		// Step 1: Open browser
		prompt1 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt1, `{"type":"key", "key":"win"}`))
		
		// Step 2: Type browser
		history += "\n- pressed win"
		uia = `[{"name": "Search Box", "x": 50, "y": 1000}, {"name": "Best match", "x": 100, "y": 200}]`
		screen = "Windows start menu open"
		prompt2 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt2, fmt.Sprintf(`{"type":"type", "text":"%s"}`, browser)))
		
		// Step 3: Enter
		history += fmt.Sprintf("\n- typed %s", browser)
		prompt3 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt3, `{"type":"key", "key":"enter"}`))
		
		// Step 4: Search
		history += "\n- pressed enter"
		uia = `[{"name": "Address and search bar", "x": 500, "y": 50}, {"name": "New Tab", "x": 100, "y": 10}]`
		screen = browser + " browser open on new tab"
		prompt4 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt4, `{"type":"click_element", "name":"Address and search bar"}`))
		
		// Step 5: Type query
		history += "\n- clicked Address and search bar"
		prompt5 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt5, fmt.Sprintf(`{"type":"type", "text":"%s summary"}`, topic)))
		
		// Step 6: Enter
		history += fmt.Sprintf("\n- typed %s summary", topic)
		prompt6 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt6, `{"type":"key", "key":"enter"}`))
		
		// Step 7: Select text
		history += "\n- pressed enter"
		uia = `[{"name": "Summary paragraph", "x": 300, "y": 400}, {"name": "Address and search bar", "x": 500, "y": 50}]`
		screen = "Search results page"
		prompt7 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt7, `{"type":"click_element", "name":"Summary paragraph"}`))
		
		// Step 8: Copy
		history += "\n- clicked Summary paragraph"
		prompt8 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt8, `{"type":"key", "key":"ctrl+c"}`))
		
		// Step 9: Open Email App
		history += "\n- pressed ctrl+c"
		prompt9 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt9, `{"type":"key", "key":"win"}`))
		
		// Step 10: Type Mail
		history += "\n- pressed win"
		uia = `[{"name": "Search Box", "x": 50, "y": 1000}]`
		screen = "Windows start menu open"
		prompt10 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt10, `{"type":"type", "text":"Mail"}`))
		
		// Step 11: Enter
		history += "\n- typed Mail"
		prompt11 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt11, `{"type":"key", "key":"enter"}`))
		
		// Step 12: New Email
		history += "\n- pressed enter"
		uia = `[{"name": "New mail", "x": 100, "y": 100}, {"name": "Inbox", "x": 50, "y": 200}]`
		screen = "Mail app open"
		prompt12 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt12, `{"type":"click_element", "name":"New mail"}`))
		
		// Step 13: Type Recipient
		history += "\n- clicked New mail"
		uia = `[{"name": "To", "x": 200, "y": 150}, {"name": "Subject", "x": 200, "y": 200}, {"name": "Body", "x": 200, "y": 300}]`
		screen = "New email window"
		prompt13 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt13, fmt.Sprintf(`{"type":"type", "text":"%s"}`, email)))
		
		// Step 14: Tab to Subject
		history += fmt.Sprintf("\n- typed %s", email)
		prompt14 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt14, `{"type":"key", "key":"tab"}`))
		
		// Step 15: Type Subject
		history += "\n- pressed tab"
		prompt15 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt15, fmt.Sprintf(`{"type":"type", "text":"%s info"}`, topic)))
		
		// Step 16: Tab to Body
		history += fmt.Sprintf("\n- typed %s info", topic)
		prompt16 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt16, `{"type":"key", "key":"tab"}`))
		
		// Step 17: Paste
		history += "\n- pressed tab"
		prompt17 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt17, `{"type":"key", "key":"ctrl+v"}`))
		
		// Step 18: Done
		history += "\n- pressed ctrl+v"
		prompt18 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt18, `{"type":"done"}`))
	}
	return samples
}

func generateCodeEditorScenario() []ChatML {
	var samples []ChatML
	for i := 0; i < 2000; i++ {
		editor := randomElement(editors)
		file := randomElement(files)
		
		task := fmt.Sprintf("Open %s, create a file called %s, type a hello world script, and save it to Documents", editor, file)
		
		history := "none"
		uia := `[{"name": "Start", "x": 10, "y": 1080}]`
		screen := "Windows desktop"
		
		// Step 1: Open editor
		prompt1 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt1, `{"type":"key", "key":"win"}`))
		
		// Step 2: Type editor
		history += "\n- pressed win"
		uia = `[{"name": "Search Box", "x": 50, "y": 1000}]`
		screen = "Windows start menu open"
		prompt2 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt2, fmt.Sprintf(`{"type":"type", "text":"%s"}`, editor)))
		
		// Step 3: Enter
		history += fmt.Sprintf("\n- typed %s", editor)
		prompt3 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt3, `{"type":"key", "key":"enter"}`))
		
		// Step 4: Type code
		history += "\n- pressed enter"
		uia = `[{"name": "Text Editor", "x": 400, "y": 400}, {"name": "File", "x": 50, "y": 20}]`
		screen = editor + " open"
		prompt4 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt4, `{"type":"type", "text":"print('hello world')"}`))
		
		// Step 5: Save
		history += "\n- typed print('hello world')"
		prompt5 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt5, `{"type":"key", "key":"ctrl+s"}`))
		
		// Step 6: Save As Dialog - File name
		history += "\n- pressed ctrl+s"
		uia = `[{"name": "File name:", "x": 300, "y": 400}, {"name": "Save", "x": 500, "y": 500}]`
		screen = "Save As dialog open"
		prompt6 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt6, fmt.Sprintf(`{"type":"type", "text":"%s"}`, file)))
		
		// Step 7: Enter
		history += fmt.Sprintf("\n- typed %s", file)
		prompt7 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt7, `{"type":"key", "key":"enter"}`))
		
		// Step 8: Done
		history += "\n- pressed enter"
		prompt8 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt8, `{"type":"done"}`))
	}
	return samples
}

func generateSettingsScenario() []ChatML {
	var samples []ChatML
	settings := []string{"Bluetooth", "Wi-Fi", "Display", "Sound", "Windows Update"}
	for i := 0; i < 1000; i++ {
		setting := randomElement(settings)
		
		task := fmt.Sprintf("Open Settings and turn off %s", setting)
		
		history := "none"
		uia := `[{"name": "Start", "x": 10, "y": 1080}]`
		screen := "Windows desktop"
		
		prompt1 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt1, `{"type":"key", "key":"win"}`))
		
		history += "\n- pressed win"
		uia = `[{"name": "Search Box", "x": 50, "y": 1000}]`
		screen = "Windows start menu open"
		prompt2 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt2, `{"type":"type", "text":"Settings"}`))
		
		history += "\n- typed Settings"
		prompt3 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt3, `{"type":"key", "key":"enter"}`))
		
		history += "\n- pressed enter"
		uia = `[{"name": "Find a setting", "x": 300, "y": 50}]`
		screen = "Settings app open"
		prompt4 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt4, `{"type":"click_element", "name":"Find a setting"}`))
		
		history += "\n- clicked Find a setting"
		prompt5 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt5, fmt.Sprintf(`{"type":"type", "text":"%s"}`, setting)))
		
		history += fmt.Sprintf("\n- typed %s", setting)
		prompt6 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt6, `{"type":"key", "key":"enter"}`))
		
		history += "\n- pressed enter"
		uia = fmt.Sprintf(`[{"name": "%s toggle", "x": 400, "y": 200}]`, setting)
		screen = setting + " settings page"
		prompt7 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt7, fmt.Sprintf(`{"type":"click_element", "name":"%s toggle"}`, setting)))
		
		history += fmt.Sprintf("\n- clicked %s toggle", setting)
		prompt8 := buildPrompt(task, history, uia, screen)
		samples = append(samples, buildSample(prompt8, `{"type":"done"}`))
	}
	return samples
}

func buildPrompt(task, history, uia, screen string) string {
	return fmt.Sprintf(`TASK: %s

DONE SO FAR:
%s

SCREEN ELEMENTS:
%s

SCREEN DESCRIPTION: %s

INSTRUCTIONS:
- STRONGLY PREFER KEYBOARD OVER MOUSE! It is much more reliable.
- To open ANY app (like clock, browser, etc): You MUST output {"type":"key", "key":"win"}.
- If you already pressed win, output {"type":"type", "text":"app name"} next. Do NOT press win again.
- After typing the app name, output {"type":"key", "key":"enter"}.
- DO NOT click taskbar items (like Start, Search, or Clock) to open apps! It will fail.
- Output {"type":"done"} when the task is fully complete.

EXAMPLE OF OPENING AN APP:
Task: open notepad
Step 1 Output: {"type":"key", "key":"win"}
Step 2 Output: {"type":"type", "text":"notepad"}
Step 3 Output: {"type":"key", "key":"enter"}`, task, history, uia, screen)
}

func buildSample(prompt, action string) ChatML {
	sysMsg := ChatMessage{Role: "system", Content: "You are a Windows PC automation agent. You output ONE JSON action for the next step. No text, just JSON.\nAvailable action types: click_element, type, key, sleep, done."}
	userMsg := ChatMessage{Role: "user", Content: prompt}
	astMsg := ChatMessage{Role: "assistant", Content: action}
	
	return ChatML{Messages: []ChatMessage{sysMsg, userMsg, astMsg}}
}

func main() {
	rand.Seed(time.Now().UnixNano())
	
	var allSamples []ChatML
	allSamples = append(allSamples, generateWebToEmailScenario()...)
	allSamples = append(allSamples, generateCodeEditorScenario()...)
	allSamples = append(allSamples, generateSettingsScenario()...)
	
	// Shuffle samples
	rand.Shuffle(len(allSamples), func(i, j int) {
		allSamples[i], allSamples[j] = allSamples[j], allSamples[i]
	})
	
	file, err := os.Create("training_data.jsonl")
	if err != nil {
		fmt.Printf("Error creating file: %v\n", err)
		return
	}
	defer file.Close()
	
	for _, sample := range allSamples {
		b, _ := json.Marshal(sample)
		file.WriteString(string(b) + "\n")
	}
	
	fmt.Printf("Successfully generated %d multi-step complex training examples in training_data.jsonl\n", len(allSamples))
}
