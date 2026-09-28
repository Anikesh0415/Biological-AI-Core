package main

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
)

func main() {
	query := "judas"
	url := "https://www.youtube.com/results?search_query=" + query
	resp, err := http.Get(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading body:", err)
		return
	}

	re := regexp.MustCompile(`"videoId":"([a-zA-Z0-9_-]{11})"`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) > 1 {
		fmt.Println("Found Video ID:", matches[1])
		fmt.Println("URL: https://www.youtube.com/watch?v=" + matches[1])
	} else {
		fmt.Println("No video ID found.")
	}
}
