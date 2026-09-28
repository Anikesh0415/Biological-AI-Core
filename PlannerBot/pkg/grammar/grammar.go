package grammar

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"strings"
)

//go:embed reminder.gbnf
var ReminderGBNF string

// RawReminderGBNF contains the authoritative GBNF grammar string as a compile-time constant.
const RawReminderGBNF = `# Production grammar for strict Reminder JSON extraction
# Schema: {"task": string, "time": string}

root ::= "{" space task-kv "," space time-kv space "}"

task-kv ::= "\"task\"" space ":" space string
time-kv ::= "\"time\"" space ":" space string

# String definition conforming to RFC 8259
string ::= "\"" char* "\""

# Character definition: unescaped Unicode (excluding control chars, quote, backslash)
# or valid JSON escape sequences (\", \\, \/, \b, \f, \n, \r, \t, \uXXXX)
char ::= [^"\\\x7F\x00-\x1F] | [\\] (["\\/bfnrt] | "u" [0-9a-fA-F]{4})

# Bounded whitespace rule: empty, single space, or bounded newline indentation
space ::= | " " | "\n"{1,2} [ \t]{0,20}
`

// PermissiveReminderGBNF contains the alternative GBNF grammar permitting either key ordering.
const PermissiveReminderGBNF = `# Permissive grammar allowing unordered task and time keys
root ::= "{" space ( task-kv "," space time-kv | time-kv "," space task-kv ) space "}"
task-kv ::= "\"task\"" space ":" space string
time-kv ::= "\"time\"" space ":" space string
string ::= "\"" char* "\""
char ::= [^"\\\x7F\x00-\x1F] | [\\] (["\\/bfnrt] | "u" [0-9a-fA-F]{4})
space ::= | " " | "\n"{1,2} [ \t]{0,20}
`

// utf8BOM is the 3-byte sequence for UTF-8 Byte Order Mark.
var utf8BOM = []byte{0xEF, 0xBB, 0xBF}

// HasUTF8BOM checks if the given byte slice starts with a UTF-8 BOM.
func HasUTF8BOM(data []byte) bool {
	return bytes.HasPrefix(data, utf8BOM)
}

// StripBOM removes the UTF-8 Byte Order Mark from the string if present.
func StripBOM(s string) string {
	return strings.TrimPrefix(s, "\xef\xbb\xbf")
}

// GetGrammar returns the authoritative GBNF grammar as a clean UTF-8 string without BOM.
// If ReminderGBNF is empty (e.g. during certain test harnesses), it falls back to RawReminderGBNF.
func GetGrammar() string {
	g := ReminderGBNF
	if len(strings.TrimSpace(g)) == 0 {
		g = RawReminderGBNF
	}
	return StripBOM(g)
}

// GetGrammarBytes returns the authoritative GBNF grammar as raw UTF-8 bytes without BOM.
func GetGrammarBytes() []byte {
	return []byte(GetGrammar())
}

// WriteTempGrammarFile creates a temporary .gbnf file containing the authoritative grammar
// encoded strictly in UTF-8 without BOM. It closes the file immediately to avoid Windows
// file-locking conflicts and returns the file path alongside a deferred cleanup function.
func WriteTempGrammarFile() (path string, cleanup func(), err error) {
	tmpFile, err := os.CreateTemp("", "reminder-*.gbnf")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temporary grammar file: %w", err)
	}

	path = tmpFile.Name()
	cleanup = func() {
		_ = os.Remove(path)
	}

	content := GetGrammarBytes()
	if _, err := tmpFile.Write(content); err != nil {
		_ = tmpFile.Close()
		cleanup()
		return "", nil, fmt.Errorf("failed to write grammar content: %w", err)
	}

	// Close immediately so external processes (llama-cli.exe) can read or remove it on Windows.
	if err := tmpFile.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to close temporary grammar file: %w", err)
	}

	return path, cleanup, nil
}

// WriteGrammarToFile writes the authoritative GBNF grammar to a specific destination path
// in raw UTF-8 without BOM.
func WriteGrammarToFile(destPath string) error {
	content := GetGrammarBytes()
	return os.WriteFile(destPath, content, 0644)
}
