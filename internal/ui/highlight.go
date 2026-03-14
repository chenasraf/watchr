package ui

import (
	"bytes"
	"encoding/json"
	"regexp"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// ansiEscPattern matches ANSI escape sequences (CSI and simple ESC sequences).
var ansiEscPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z]|\x1b[^\[]`)

// stripANSI removes all ANSI escape sequences from a string.
func stripANSI(s string) string {
	return ansiEscPattern.ReplaceAllString(s, "")
}

// extractJSON finds the first JSON object or array in a string,
// skipping any leading non-JSON content (e.g. ANSI codes, log prefixes).
// Returns the JSON substring and any prefix before it.
func extractJSON(s string) (prefix, jsonStr string, ok bool) {
	// Find first { or [
	idx := strings.IndexAny(s, "{[")
	if idx < 0 {
		return "", "", false
	}
	return s[:idx], s[idx:], true
}

// highlightJSON attempts to detect JSON content, pretty-print it, and apply
// syntax highlighting for terminal output. Returns the original string
// unchanged if the content is not valid JSON.
func highlightJSON(s string) string {
	trimmed := strings.TrimSpace(s)
	if len(trimmed) == 0 {
		return s
	}

	// Strip ANSI codes for JSON detection and parsing
	clean := stripANSI(trimmed)

	// Extract JSON from the string (may have a non-JSON prefix)
	prefix, jsonStr, ok := extractJSON(clean)
	if !ok {
		return s
	}

	// Try to pretty-print the JSON portion
	var buf bytes.Buffer
	if err := json.Indent(&buf, []byte(jsonStr), "", "  "); err != nil {
		return s
	}
	pretty := buf.String()

	// Highlight with chroma (only the JSON portion)
	lexer := lexers.Get("json")
	if lexer == nil {
		return pretty
	}
	lexer = chroma.Coalesce(lexer)

	style := styles.Get("monokai")
	if style == nil {
		style = styles.Fallback
	}

	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	iterator, err := lexer.Tokenise(nil, pretty)
	if err != nil {
		return pretty
	}

	var out bytes.Buffer
	if err := formatter.Format(&out, style, iterator); err != nil {
		return pretty
	}

	// Re-attach any non-JSON prefix (stripped of ANSI)
	result := out.String()
	if prefix != "" {
		prefix = strings.TrimSpace(prefix)
		if prefix != "" {
			result = prefix + "\n" + result
		}
	}

	return result
}
