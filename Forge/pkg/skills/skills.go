package skills

import (
	"encoding/json"
	"fmt"
	"forge/pkg/executor"
	"os"
	"path/filepath"
	"strings"
)

type Skill interface {
	Match(intent string) bool
	Execute(intent string) error
	Name() string
}

var Registry []Skill

func Register(s Skill) {
	Registry = append(Registry, s)
}

func MatchIntent(intent string) Skill {
	var bestSkill Skill
	bestScore := 9999

	for _, s := range Registry {
		if ds, ok := s.(*DynamicSkill); ok {
			// First try variable extraction
			vars, match := ExtractVariables(intent, ds.SkillName)
			if match {
				// We can prioritize perfect matches with 0 penalty,
				// but for now if extraction succeeds, it's a very strong match.
				score := 0
				if len(vars) == 0 { // no variables extracted, fall back to pure fuzzy match score
					_, score = FuzzyMatchWithScore(intent, ds.SkillName)
				}
				if score < bestScore {
					bestScore = score
					bestSkill = s
				}
			}
		} else {
			// Advanced skill
			if s.Match(intent) {
				return s // Advanced skills always win if they match
			}
		}
	}
	return bestSkill
}

// Helper to check if a string contains all keywords
func ContainsAllKeywords(intent string, keywords ...string) bool {
	inputTokens := strings.Fields(strings.ToLower(intent))
	
	for _, kw := range keywords {
		kw = strings.ToLower(kw)
		found := false
		
		for _, iToken := range inputTokens {
			dist := levenshtein(iToken, kw)
			// Much stricter typos for exact keyword constraints in advanced skills
			allowedTypos := 0
			if len(kw) >= 5 {
				allowedTypos = 1
			}
			if dist <= allowedTypos {
				found = true
				break
			}
		}
		
		if !found {
			return false
		}
	}
	return true
}

type DynamicSkill struct {
	SkillName string            `json:"name"`
	Actions   []executor.Action `json:"actions"`
}

func (s *DynamicSkill) Name() string {
	return s.SkillName
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
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
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

func FuzzyMatchWithScore(input, target string) (bool, int) {
	input = strings.ToLower(input)
	target = strings.ToLower(target)
	
	if input == target {
		return true, 0
	}
	if strings.Contains(input, target) && len(input) == len(target) {
		return true, 0
	}

	inputTokens := strings.Fields(input)
	targetTokens := strings.Fields(target)

	matchCount := 0
	totalScore := 0
	
	for _, tToken := range targetTokens {
		bestDist := 999
		for _, iToken := range inputTokens {
			dist := levenshtein(iToken, tToken)
			if dist < bestDist {
				bestDist = dist
			}
		}
		
		allowedTypos := 0
		if len(tToken) >= 5 {
			allowedTypos = 1
		}
		if len(tToken) >= 8 {
			allowedTypos = 2
		}
		
		if bestDist <= allowedTypos {
			matchCount++
			totalScore += bestDist
		} else {
			totalScore += 100 // penalty for missing word
		}
	}

	// Calculate a penalty for extra words in the input that didn't match anything
	extraWords := len(inputTokens) - matchCount
	if extraWords > 0 {
		totalScore += extraWords * 10 
	}

	// Must match all target tokens to be considered a match
	isMatch := (matchCount == len(targetTokens) && len(targetTokens) > 0)
	
	// Ensure that extra words don't allow a 1-word target to greedily consume a multi-word input
	// unless the score is exceptionally good. 
	if isMatch && extraWords > 0 && len(targetTokens) == 1 {
		// If input is "open spotify" (2 tokens) and target is "openai" (1 token)
		// totalScore = distance("open", "openai") + 1*10 = 2 + 10 = 12.
		// We can add a strict check: if it's a 1-word target but input has >1 word, it shouldn't match
		// unless it's explicitly containing the target.
		if !strings.Contains(input, target) {
			isMatch = false
		}
	}
	
	return isMatch, totalScore
}

func ExtractVariables(input, template string) (map[string]string, bool) {
	vars := make(map[string]string)
	input = strings.ToLower(input)
	template = strings.ToLower(template)
	
	// A basic regex-free extraction:
	// If template is "play {song} on {app}" and input is "play blinding lights on spotify"
	// We split template by "{", then "}" to find the constants and variables.
	
	if !strings.Contains(template, "{") {
		match, _ := FuzzyMatchWithScore(input, template)
		return vars, match
	}

	// This is a naive heuristic extractor. For a robust approach, we parse the template:
	// Find all static parts of the template.
	staticParts := []string{}
	varNames := []string{}
	
	remainingTemplate := template
	for {
		startIdx := strings.Index(remainingTemplate, "{")
		if startIdx == -1 {
			if remainingTemplate != "" {
				staticParts = append(staticParts, remainingTemplate)
			}
			break
		}
		
		staticParts = append(staticParts, remainingTemplate[:startIdx])
		remainingTemplate = remainingTemplate[startIdx+1:]
		
		endIdx := strings.Index(remainingTemplate, "}")
		if endIdx == -1 {
			break
		}
		varNames = append(varNames, remainingTemplate[:endIdx])
		remainingTemplate = remainingTemplate[endIdx+1:]
	}
	
	// Now try to extract from input
	remainingInput := input
	for i, staticPart := range staticParts {
		// Fuzzy search for the static part to handle typos
		// For simplicity, we just use strings.Index for exact bounds right now,
		// but since input might have typos, a robust extractor might need Levenshtein alignment.
		// For now, exact bounds.
		
		staticPart = strings.TrimSpace(staticPart)
		if staticPart == "" {
			continue
		}
		
		idx := strings.Index(remainingInput, staticPart)
		if idx == -1 {
			return nil, false
		}
		
		// If this isn't the first static part, the text before this part belongs to the previous variable
		if i > 0 && len(varNames) >= i {
			varValue := strings.TrimSpace(remainingInput[:idx])
			vars[varNames[i-1]] = varValue
		} else if i == 0 && idx > 0 {
			// If first static part doesn't start at 0, we can't extract (unless template starts with var)
			if len(varNames) > 0 && template[0] == '{' {
				vars[varNames[0]] = strings.TrimSpace(remainingInput[:idx])
			} else {
				return nil, false
			}
		}
		
		remainingInput = remainingInput[idx+len(staticPart):]
	}
	
	// If there's a trailing variable
	if len(varNames) > len(staticParts)-1 {
		if template[0] != '{' || len(varNames) > 1 {
			vars[varNames[len(varNames)-1]] = strings.TrimSpace(remainingInput)
		}
	} else if len(remainingInput) > 0 {
		// If there's trailing input but no trailing variable, it's not a perfect match
		// However, we can be forgiving.
	}

	return vars, true
}

func (s *DynamicSkill) Match(intent string) bool {
	// We no longer use this directly in MatchIntent, but keep it for interface satisfaction
	_, match := ExtractVariables(intent, s.SkillName)
	return match
}

func (s *DynamicSkill) Execute(intent string) error {
	fmt.Printf("Executing Learned Skill: %s\n", s.SkillName)
	
	vars, match := ExtractVariables(intent, s.SkillName)
	
	// Duplicate the actions to inject variables safely
	var injectedActions []executor.Action
	for _, act := range s.Actions {
		injectedAct := act
		if match && len(vars) > 0 {
			if injectedAct.Type == "type" {
				for k, v := range vars {
					injectedAct.Text = strings.ReplaceAll(injectedAct.Text, "{"+k+"}", v)
				}
			}
			if injectedAct.Name != "" {
				for k, v := range vars {
					injectedAct.Name = strings.ReplaceAll(injectedAct.Name, "{"+k+"}", v)
				}
			}
		}
		injectedActions = append(injectedActions, injectedAct)
	}
	
	executor.ExecutePlan(injectedActions)
	return nil
}

func LoadLearnedSkills() {
	dbPath := "skills_db"
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		os.Mkdir(dbPath, 0755)
		return
	}

	files, err := os.ReadDir(dbPath)
	if err != nil {
		fmt.Printf("Error reading skills_db: %v\n", err)
		return
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			content, err := os.ReadFile(filepath.Join(dbPath, file.Name()))
			if err != nil {
				continue
			}

			var skill DynamicSkill
			if err := json.Unmarshal(content, &skill); err == nil {
				Register(&skill)
				fmt.Printf("Loaded learned skill: %s\n", skill.SkillName)
			}
		}
	}
}
