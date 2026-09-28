package skills

import (
	"fmt"
	"forge/pkg/executor"
	"io"
	"net/http"
	"regexp"
	"strings"
)

type YouTubeSearchSkill struct{}

func init() {
	Register(&YouTubeSearchSkill{})
}

func (s *YouTubeSearchSkill) Name() string {
	return "YouTubeSearch"
}

func (s *YouTubeSearchSkill) Match(intent string) bool {
	if strings.Contains(intent, "opened") || strings.Contains(intent, "this") || 
	   strings.Contains(intent, "click") || strings.Contains(intent, "screen") {
		return false
	}
	
	return ContainsAllKeywords(intent, "youtube", "search") || 
	       ContainsAllKeywords(intent, "play", "youtube")
}

func (s *YouTubeSearchSkill) Execute(intent string) error {
	fmt.Println("Executing Advanced Skill: YouTubeSearch")
	
	searchTerm := "Sarthak Goswami" // fallback
	if idx := strings.Index(intent, "search for"); idx != -1 {
		searchTerm = strings.TrimSpace(intent[idx+10:])
	} else if idx := strings.Index(intent, "play"); idx != -1 {
		part := strings.TrimSpace(intent[idx+4:])
		searchTerm = strings.TrimPrefix(part, "on youtube")
		searchTerm = strings.TrimSuffix(searchTerm, "on youtube")
		searchTerm = strings.TrimSpace(searchTerm)
	}
	
	safeSearch := strings.ReplaceAll(searchTerm, " ", "+")
	targetUrl := "https://www.youtube.com/results?search_query=" + safeSearch
	
	// If it's a "play" command, we want to reliably hit the first video.
	if strings.Contains(intent, "play") {
		fmt.Printf("Fetching search results for: %s\n", searchTerm)
		resp, err := http.Get(targetUrl)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			re := regexp.MustCompile(`"videoId":"([a-zA-Z0-9_-]{11})"`)
			matches := re.FindStringSubmatch(string(body))
			if len(matches) > 1 {
				targetUrl = "https://www.youtube.com/watch?v=" + matches[1]
				fmt.Printf("Found direct video ID: %s\n", matches[1])
			}
		}
	}
	
	actions := []executor.Action{
		{Type: "key", Key: "win+r"},
		{Type: "sleep", Ms: 1000},
		{Type: "type", Text: "brave " + targetUrl},
		{Type: "sleep", Ms: 1000},
		{Type: "key", Key: "enter"},
	}
	
	executor.ExecutePlan(actions)
	return nil
}
