package uia

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Element struct {
	Name string `json:"name"`
	Type string `json:"type"`
	X    int    `json:"x"`
	Y    int    `json:"y"`
}

// levenshtein computes the Levenshtein distance between two strings
func levenshtein(s1, s2 string) int {
	lenS1 := len(s1)
	lenS2 := len(s2)
	if lenS1 == 0 {
		return lenS2
	}
	if lenS2 == 0 {
		return lenS1
	}
	matrix := make([][]int, lenS1+1)
	for i := range matrix {
		matrix[i] = make([]int, lenS2+1)
	}
	for i := 0; i <= lenS1; i++ {
		matrix[i][0] = i
	}
	for j := 0; j <= lenS2; j++ {
		matrix[0][j] = j
	}
	for i := 1; i <= lenS1; i++ {
		for j := 1; j <= lenS2; j++ {
			cost := 1
			if s1[i-1] == s2[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				matrix[i][j-1]+1,
				matrix[i-1][j-1]+cost,
			)
		}
	}
	return matrix[lenS1][lenS2]
}

func min(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}

func fuzzyMatch(input, target string) bool {
	input = strings.ToLower(input)
	target = strings.ToLower(target)
	if input == target || strings.Contains(input, target) {
		return true
	}
	inputTokens := strings.Fields(input)
	targetTokens := strings.Fields(target)
	matchCount := 0
	for _, tToken := range targetTokens {
		bestDist := 999
		for _, iToken := range inputTokens {
			dist := levenshtein(iToken, tToken)
			if dist < bestDist {
				bestDist = dist
			}
		}
		allowedTypos := 1
		if len(tToken) > 5 {
			allowedTypos = 2
		}
		if bestDist <= allowedTypos {
			matchCount++
		}
	}
	return matchCount == len(targetTokens) && len(targetTokens) > 0
}

// WaitForElement polls the UI until an element matching the query is found or timeout is reached.
func WaitForElement(query string, timeoutMs int) (*Element, error) {
	start := time.Now()
	timeout := time.Duration(timeoutMs) * time.Millisecond
	
	// Fast polling loop
	for time.Since(start) < timeout {
		jsonStr, err := DumpUI()
		if err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		
		var elements []Element
		if err := json.Unmarshal([]byte(jsonStr), &elements); err != nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}
		
		// First pass: exact substring match
		for _, el := range elements {
			if strings.Contains(strings.ToLower(el.Name), strings.ToLower(query)) {
				return &el, nil
			}
		}
		
		// Second pass: fuzzy match
		for _, el := range elements {
			if fuzzyMatch(el.Name, query) {
				return &el, nil
			}
		}
		
		time.Sleep(500 * time.Millisecond)
	}
	
	return nil, fmt.Errorf("timeout waiting for element: %s", query)
}
