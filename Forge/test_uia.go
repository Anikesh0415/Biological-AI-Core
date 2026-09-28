package main

import (
	"encoding/json"
	"fmt"
	"forge/pkg/uia"
)

func main() {
	jsonStr, err := uia.DumpUI()
	if err != nil {
		fmt.Printf("DumpUI err: %v\n", err)
		return
	}
	fmt.Printf("DumpUI returned %d bytes\n", len(jsonStr))
	
	var elements []uia.Element
	if err := json.Unmarshal([]byte(jsonStr), &elements); err != nil {
		fmt.Printf("JSON Unmarshal ERROR: %v\n", err)
		
		// Find where the error is
		if syntaxErr, ok := err.(*json.SyntaxError); ok {
			start := max(0, int(syntaxErr.Offset)-50)
			end := min(len(jsonStr), int(syntaxErr.Offset)+50)
			fmt.Printf("Context around offset %d: \n...%s...\n", syntaxErr.Offset, jsonStr[start:end])
		}
	} else {
		fmt.Printf("JSON Unmarshal SUCCESS! Found %d elements\n", len(elements))
	}
}

func max(a, b int) int { if a > b { return a } else { return b } }
func min(a, b int) int { if a < b { return a } else { return b } }
