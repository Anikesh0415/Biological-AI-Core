package extractor

// Reminder represents the extracted structured data from a natural language reminder prompt.
type Reminder struct {
	Task    string `json:"task"`
	Time    string `json:"time"`
	rawJSON string // private: stores the raw LLM output for debugging
}

// RawJSON returns the raw JSON string from the LLM output (for test verification).
func (r *Reminder) RawJSON() string {
	return r.rawJSON
}
