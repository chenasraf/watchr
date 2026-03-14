package ui

import (
	"strings"
	"testing"
)

func TestWrapTextANSI(t *testing.T) {
	t.Run("ANSI sequences are not split", func(t *testing.T) {
		// Red "ab" then reset: \033[31mab\033[0m
		input := "\033[31mabcdef\033[0m"
		lines := wrapText(input, 3)
		// Should wrap into "abc" and "def", each with proper ANSI
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d: %q", len(lines), lines)
		}
		// First line should have the color and a reset
		if !strings.Contains(lines[0], "\033[31m") {
			t.Errorf("first line missing color code: %q", lines[0])
		}
		if !strings.Contains(lines[0], "abc") {
			t.Errorf("first line missing 'abc': %q", lines[0])
		}
		// Second line should re-apply the color
		if !strings.Contains(lines[1], "\033[31m") {
			t.Errorf("second line missing re-applied color code: %q", lines[1])
		}
		if !strings.Contains(lines[1], "def") {
			t.Errorf("second line missing 'def': %q", lines[1])
		}
	})

	t.Run("no ANSI still works", func(t *testing.T) {
		lines := wrapText("abcdef", 3)
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d", len(lines))
		}
		if lines[0] != "abc" {
			t.Errorf("expected 'abc', got %q", lines[0])
		}
		if lines[1] != "def" {
			t.Errorf("expected 'def', got %q", lines[1])
		}
	})

	t.Run("ANSI reset clears active state", func(t *testing.T) {
		// Color "ab", reset, then "cd"
		input := "\033[31mab\033[0mcd"
		lines := wrapText(input, 2)
		// "ab" on first line (with color), "cd" on second (no color re-applied)
		if len(lines) != 2 {
			t.Fatalf("expected 2 lines, got %d: %q", len(lines), lines)
		}
		// Second line should NOT have the red color re-applied
		if strings.Contains(lines[1], "\033[31m") {
			t.Errorf("second line should not have color after reset: %q", lines[1])
		}
	})
}

func TestWrapPreviewContentANSI(t *testing.T) {
	// Multi-line input with ANSI codes should survive split + wrap
	input := "\033[31mhello\033[0m\n\033[32mworld\033[0m"
	lines := wrapPreviewContent(input, 80)
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "\033[31m") || !strings.Contains(lines[0], "hello") {
		t.Errorf("first line missing color or content: %q", lines[0])
	}
	if !strings.Contains(lines[1], "\033[32m") || !strings.Contains(lines[1], "world") {
		t.Errorf("second line missing color or content: %q", lines[1])
	}
}

func TestHighlightJSON(t *testing.T) {
	t.Run("valid JSON object is pretty-printed and highlighted", func(t *testing.T) {
		input := `{"name":"test","count":42}`
		result := highlightJSON(input)

		// Should be pretty-printed (multi-line)
		if !strings.Contains(result, "\n") {
			t.Error("expected multi-line pretty-printed output")
		}
		// Should contain the key and value
		if !strings.Contains(result, "name") {
			t.Error("expected output to contain 'name'")
		}
		if !strings.Contains(result, "test") {
			t.Error("expected output to contain 'test'")
		}
		if !strings.Contains(result, "42") {
			t.Error("expected output to contain '42'")
		}
		// Should contain ANSI escape codes (syntax highlighting)
		if !strings.Contains(result, "\033[") {
			t.Error("expected ANSI color codes in output")
		}
	})

	t.Run("valid JSON array", func(t *testing.T) {
		input := `[1, 2, 3]`
		result := highlightJSON(input)
		if !strings.Contains(result, "\033[") {
			t.Error("expected ANSI color codes for JSON array")
		}
	})

	t.Run("non-JSON returns unchanged", func(t *testing.T) {
		input := "hello world"
		result := highlightJSON(input)
		if result != input {
			t.Errorf("expected unchanged output for non-JSON, got %q", result)
		}
	})

	t.Run("invalid JSON returns unchanged", func(t *testing.T) {
		input := `{"broken": }`
		result := highlightJSON(input)
		if result != input {
			t.Errorf("expected unchanged output for invalid JSON, got %q", result)
		}
	})

	t.Run("empty string returns unchanged", func(t *testing.T) {
		result := highlightJSON("")
		if result != "" {
			t.Errorf("expected empty string, got %q", result)
		}
	})

	t.Run("whitespace-only returns unchanged", func(t *testing.T) {
		input := "   "
		result := highlightJSON(input)
		if result != input {
			t.Errorf("expected unchanged output, got %q", result)
		}
	})

	t.Run("JSON with leading ANSI escape sequences", func(t *testing.T) {
		input := "\x1b[34h\x1b[?25h{\"key\":\"value\"}"
		result := highlightJSON(input)
		if !strings.Contains(result, "key") {
			t.Error("expected output to contain 'key'")
		}
		if !strings.Contains(result, "value") {
			t.Error("expected output to contain 'value'")
		}
		if !strings.Contains(result, "\033[") {
			t.Error("expected ANSI color codes in output")
		}
	})

	t.Run("JSON with non-JSON prefix text", func(t *testing.T) {
		input := `some prefix {"key":"value"}`
		result := highlightJSON(input)
		if !strings.Contains(result, "key") {
			t.Error("expected output to contain 'key'")
		}
		if !strings.Contains(result, "some prefix") {
			t.Error("expected output to preserve prefix")
		}
	})
}
