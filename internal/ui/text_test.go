package ui

import (
	"strings"
	"testing"
)

func TestTruncateToWidth(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxWidth int
		want     string
	}{
		{"empty maxWidth", "hello", 0, ""},
		{"no truncation needed", "hello", 10, "hello"},
		{"exact fit", "hello", 5, "hello"},
		{"truncate with ellipsis", "hello world", 8, "hello w…"},
		{"maxWidth 1", "hello", 1, "…"},
		{"empty string", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateToWidth(tt.input, tt.maxWidth)
			if got != tt.want {
				t.Errorf("truncateToWidth(%q, %d) = %q, want %q", tt.input, tt.maxWidth, got, tt.want)
			}
		})
	}
}

func TestSplitAtVisualWidth(t *testing.T) {
	t.Run("plain text", func(t *testing.T) {
		left, right := splitAtVisualWidth("hello world", 5)
		if left != "hello" {
			t.Errorf("expected left 'hello', got %q", left)
		}
		if right != " world" {
			t.Errorf("expected right ' world', got %q", right)
		}
	})

	t.Run("pads if short", func(t *testing.T) {
		left, right := splitAtVisualWidth("hi", 5)
		if left != "hi   " {
			t.Errorf("expected left 'hi   ', got %q", left)
		}
		if right != "" {
			t.Errorf("expected empty right, got %q", right)
		}
	})

	t.Run("with ANSI codes", func(t *testing.T) {
		input := "\x1b[31mhello\x1b[0m world"
		left, _ := splitAtVisualWidth(input, 5)
		// Left should contain the ANSI code and "hello"
		if !strings.Contains(left, "\x1b[31m") {
			t.Error("expected left to contain ANSI color code")
		}
		if !strings.Contains(left, "hello") {
			t.Error("expected left to contain 'hello'")
		}
	})
}

func TestSkipVisualWidth(t *testing.T) {
	t.Run("plain text", func(t *testing.T) {
		result := skipVisualWidth("hello world", 6)
		if result != "world" {
			t.Errorf("expected 'world', got %q", result)
		}
	})

	t.Run("preserves ANSI state", func(t *testing.T) {
		input := "\x1b[31mhello world\x1b[0m"
		result := skipVisualWidth(input, 6)
		// Should preserve the ANSI code encountered during skip
		if !strings.Contains(result, "\x1b[31m") {
			t.Error("expected ANSI state to be preserved")
		}
	})
}

func TestIsAnsiTerminator(t *testing.T) {
	// Letters are terminators
	if !isAnsiTerminator('m') {
		t.Error("expected 'm' to be terminator")
	}
	if !isAnsiTerminator('A') {
		t.Error("expected 'A' to be terminator")
	}
	if !isAnsiTerminator('z') {
		t.Error("expected 'z' to be terminator")
	}

	// Digits are not terminators
	if isAnsiTerminator('0') {
		t.Error("expected '0' not to be terminator")
	}
	if isAnsiTerminator(';') {
		t.Error("expected ';' not to be terminator")
	}
}

func TestWordBoundaryLeft(t *testing.T) {
	tests := []struct {
		name string
		s    string
		pos  int
		want int
	}{
		{"end of string", "foo bar baz", 11, 8},
		{"middle of word", "foo bar baz", 9, 8},
		{"at word boundary", "foo bar baz", 8, 4},
		{"at start", "foo bar", 0, 0},
		{"with multiple spaces", "foo   bar", 9, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordBoundaryLeft(tt.s, tt.pos)
			if got != tt.want {
				t.Errorf("wordBoundaryLeft(%q, %d) = %d, want %d", tt.s, tt.pos, got, tt.want)
			}
		})
	}
}

func TestWordBoundaryRight(t *testing.T) {
	tests := []struct {
		name string
		s    string
		pos  int
		want int
	}{
		{"start of string", "foo bar baz", 0, 4},
		{"middle of word", "foo bar baz", 1, 4},
		{"at word boundary", "foo bar baz", 4, 8},
		{"at end", "foo bar", 7, 7},
		{"with multiple spaces", "foo   bar", 3, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wordBoundaryRight(tt.s, tt.pos)
			if got != tt.want {
				t.Errorf("wordBoundaryRight(%q, %d) = %d, want %d", tt.s, tt.pos, got, tt.want)
			}
		})
	}
}

func TestTextInsert(t *testing.T) {
	text, cursor := textInsert("helo", "l", 3)
	if text != "hello" || cursor != 4 {
		t.Errorf("got %q cursor %d, want 'hello' cursor 4", text, cursor)
	}
}

func TestTextBackspace(t *testing.T) {
	text, cursor := textBackspace("hello", 3)
	if text != "helo" || cursor != 2 {
		t.Errorf("got %q cursor %d, want 'helo' cursor 2", text, cursor)
	}

	// At start, no change
	text, cursor = textBackspace("hello", 0)
	if text != "hello" || cursor != 0 {
		t.Errorf("got %q cursor %d, want 'hello' cursor 0", text, cursor)
	}
}

func TestTextBackspaceWord(t *testing.T) {
	text, cursor := textBackspaceWord("hello world", 11)
	if text != "hello " || cursor != 6 {
		t.Errorf("got %q cursor %d, want 'hello ' cursor 6", text, cursor)
	}

	// At start, no change
	text, cursor = textBackspaceWord("hello", 0)
	if text != "hello" || cursor != 0 {
		t.Errorf("got %q cursor %d, want 'hello' cursor 0", text, cursor)
	}
}

func TestOverlayBox(t *testing.T) {
	base := "aaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa\naaaaaaaaaa"
	box := "XX\nXX"

	result := overlayBox(base, box, 2, 2, 10, 5)
	lines := strings.Split(result, "\n")

	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}

	// The overlay should appear in the middle rows
	// Center: startY = (5-2)/2 = 1, startX = (10-2)/2 = 4
	// Lines 1 and 2 should contain the overlay
	if !strings.Contains(lines[1], "XX") {
		t.Errorf("expected overlay in line 1, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "XX") {
		t.Errorf("expected overlay in line 2, got %q", lines[2])
	}
}
