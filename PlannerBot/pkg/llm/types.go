package llm

// CompletionRequest is the JSON body sent to llama-server's /completion endpoint.
type CompletionRequest struct {
	Prompt      string  `json:"prompt"`
	Grammar     string  `json:"grammar,omitempty"`
	NPredict    int     `json:"n_predict,omitempty"`
	Temperature   float64  `json:"temperature"`
	TopP          float64  `json:"top_p,omitempty"`
	TopK          int      `json:"top_k,omitempty"`
	RepeatPenalty float64  `json:"repeat_penalty,omitempty"`
	Stream        bool     `json:"stream"`
	CachePrompt   bool     `json:"cache_prompt"`
	Stop          []string `json:"stop,omitempty"`
}

// CompletionResponse is the JSON body received from llama-server's /completion endpoint.
type CompletionResponse struct {
	Content string `json:"content"`
	Stop    bool   `json:"stop"`
}

// HealthResponse represents the /health endpoint response.
type HealthResponse struct {
	Status string `json:"status"`
}
