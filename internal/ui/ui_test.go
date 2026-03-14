package ui

import (
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chenasraf/watchr/internal/runner"
)

func TestConfig(t *testing.T) {
	cfg := Config{
		Command:              "echo test",
		Shell:                "sh",
		PreviewSize:          40,
		PreviewSizeIsPercent: true,
		PreviewPosition:      PreviewBottom,
		ShowLineNums:         true,
		LineNumWidth:         6,
		Prompt:               "watchr> ",
		RefreshInterval:      5 * time.Second,
	}

	if cfg.Command != "echo test" {
		t.Errorf("expected command 'echo test', got %q", cfg.Command)
	}

	if cfg.Shell != "sh" {
		t.Errorf("expected shell 'sh', got %q", cfg.Shell)
	}

	if cfg.PreviewSize != 40 {
		t.Errorf("expected preview size 40, got %d", cfg.PreviewSize)
	}

	if !cfg.PreviewSizeIsPercent {
		t.Error("expected PreviewSizeIsPercent to be true")
	}

	if cfg.PreviewPosition != PreviewBottom {
		t.Errorf("expected preview position 'bottom', got %q", cfg.PreviewPosition)
	}

	if !cfg.ShowLineNums {
		t.Error("expected ShowLineNums to be true")
	}

	if cfg.LineNumWidth != 6 {
		t.Errorf("expected line num width 6, got %d", cfg.LineNumWidth)
	}

	if cfg.Prompt != "watchr> " {
		t.Errorf("expected prompt 'watchr> ', got %q", cfg.Prompt)
	}

	if cfg.RefreshInterval != 5*time.Second {
		t.Errorf("expected refresh interval 5s, got %v", cfg.RefreshInterval)
	}
}

func TestPreviewPositionConstants(t *testing.T) {
	tests := []struct {
		pos  PreviewPosition
		want string
	}{
		{PreviewBottom, "bottom"},
		{PreviewTop, "top"},
		{PreviewLeft, "left"},
		{PreviewRight, "right"},
	}

	for _, tt := range tests {
		if string(tt.pos) != tt.want {
			t.Errorf("PreviewPosition %v != %q", tt.pos, tt.want)
		}
	}
}

func TestConfigDefaults(t *testing.T) {
	// Test with zero values
	cfg := Config{}

	if cfg.Command != "" {
		t.Errorf("expected empty command, got %q", cfg.Command)
	}

	if cfg.Shell != "" {
		t.Errorf("expected empty shell, got %q", cfg.Shell)
	}

	if cfg.PreviewSize != 0 {
		t.Errorf("expected preview size 0, got %d", cfg.PreviewSize)
	}

	if cfg.PreviewSizeIsPercent {
		t.Error("expected PreviewSizeIsPercent to be false")
	}

	if cfg.PreviewPosition != "" {
		t.Errorf("expected empty preview position, got %q", cfg.PreviewPosition)
	}

	if cfg.ShowLineNums {
		t.Error("expected ShowLineNums to be false")
	}

	if cfg.LineNumWidth != 0 {
		t.Errorf("expected line num width 0, got %d", cfg.LineNumWidth)
	}

	if cfg.Prompt != "" {
		t.Errorf("expected empty prompt, got %q", cfg.Prompt)
	}
}

func TestInitialModel(t *testing.T) {
	cfg := Config{
		Command:              "echo test",
		Shell:                "sh",
		PreviewSize:          40,
		PreviewSizeIsPercent: true,
		PreviewPosition:      PreviewBottom,
		ShowLineNums:         true,
		LineNumWidth:         6,
		Prompt:               "watchr> ",
	}

	m := initialModel(cfg)

	if m.config.Command != cfg.Command {
		t.Errorf("expected command %q, got %q", cfg.Command, m.config.Command)
	}

	if m.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", m.cursor)
	}

	if m.offset != 0 {
		t.Errorf("expected offset at 0, got %d", m.offset)
	}

	if m.filterMode {
		t.Error("expected filterMode to be false")
	}

	if m.showPreview {
		t.Error("expected showPreview to be false")
	}

	if !m.loading {
		t.Error("expected loading to be true initially")
	}
}

func TestModelUpdateFiltered(t *testing.T) {
	cfg := Config{
		Command: "echo test",
		Shell:   "sh",
	}

	m := initialModel(cfg)

	// Add some test lines
	m.lines = []runner.Line{
		{Number: 1, Content: "hello world"},
		{Number: 2, Content: "foo bar"},
		{Number: 3, Content: "hello foo"},
		{Number: 4, Content: "baz qux"},
	}

	// Test with no filter
	m.filter = ""
	m.updateFiltered()

	if len(m.filtered) != 4 {
		t.Errorf("expected 4 filtered lines, got %d", len(m.filtered))
	}

	// Test with filter
	m.filter = "hello"
	m.updateFiltered()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered lines for 'hello', got %d", len(m.filtered))
	}

	// Test case insensitive
	m.filter = "HELLO"
	m.updateFiltered()

	if len(m.filtered) != 2 {
		t.Errorf("expected 2 filtered lines for 'HELLO' (case insensitive), got %d", len(m.filtered))
	}

	// Test no matches
	m.filter = "xyz"
	m.updateFiltered()

	if len(m.filtered) != 0 {
		t.Errorf("expected 0 filtered lines for 'xyz', got %d", len(m.filtered))
	}
}

func TestModelMoveCursor(t *testing.T) {
	cfg := Config{
		Command: "echo test",
		Shell:   "sh",
	}

	m := initialModel(cfg)
	m.filtered = []int{0, 1, 2, 3, 4}
	m.height = 100 // enough height for all lines

	// Move down
	m.moveCursor(1)
	if m.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", m.cursor)
	}

	// Move down more
	m.moveCursor(2)
	if m.cursor != 3 {
		t.Errorf("expected cursor at 3, got %d", m.cursor)
	}

	// Move past end
	m.moveCursor(10)
	if m.cursor != 4 {
		t.Errorf("expected cursor at 4 (clamped), got %d", m.cursor)
	}

	// Move up
	m.moveCursor(-2)
	if m.cursor != 2 {
		t.Errorf("expected cursor at 2, got %d", m.cursor)
	}

	// Move past beginning
	m.moveCursor(-10)
	if m.cursor != 0 {
		t.Errorf("expected cursor at 0 (clamped), got %d", m.cursor)
	}
}

func TestVisibleLines(t *testing.T) {
	cfg := Config{
		Command:              "echo test",
		Shell:                "sh",
		PreviewSize:          40,
		PreviewSizeIsPercent: true,
		PreviewPosition:      PreviewBottom,
	}

	m := initialModel(cfg)
	m.height = 100

	// Fixed lines: top border (1) + header (1) + separator (1) + bottom border (1) + prompt (1) = 5
	fixedLines := 5

	// Without preview
	m.showPreview = false
	visible := m.visibleLines()
	expected := 100 - fixedLines
	if visible != expected {
		t.Errorf("expected %d visible lines without preview, got %d", expected, visible)
	}

	// With preview at bottom (percentage)
	m.showPreview = true
	visible = m.visibleLines()
	previewHeight := 100 * 40 / 100 // 40%
	// Add 1 for the separator between content and preview
	expected = 100 - fixedLines - previewHeight - 1
	if visible != expected {
		t.Errorf("expected %d visible lines with preview, got %d", expected, visible)
	}

	// With preview using absolute size
	m.config.PreviewSizeIsPercent = false
	m.config.PreviewSize = 10
	visible = m.visibleLines()
	// Add 1 for the separator between content and preview
	expected = 100 - fixedLines - 10 - 1
	if visible != expected {
		t.Errorf("expected %d visible lines with absolute preview size, got %d", expected, visible)
	}
}

func TestUpdateFilteredPreservesOffset(t *testing.T) {
	cfg := Config{
		Command: "echo test",
		Shell:   "sh",
	}

	m := initialModel(cfg)
	m.height = 20 // Enough for visibleLines to return > 0

	// Add many test lines
	for i := 1; i <= 100; i++ {
		m.lines = append(m.lines, runner.Line{Number: i, Content: "line content"})
	}

	// Set initial state with offset
	m.filter = ""
	m.updateFiltered()
	m.offset = 50
	m.cursor = 55

	// Simulate streaming update - add more lines without changing filter
	m.lines = append(m.lines, runner.Line{Number: 101, Content: "new line"})
	m.updateFiltered()

	// Offset should be preserved (or clamped if necessary)
	if m.offset < 50 {
		t.Errorf("expected offset to be preserved (>= 50), got %d", m.offset)
	}

	// Cursor should be preserved
	if m.cursor != 55 {
		t.Errorf("expected cursor to be preserved at 55, got %d", m.cursor)
	}
}

func TestUpdateFilteredClampsOffsetWhenNeeded(t *testing.T) {
	cfg := Config{
		Command: "echo test",
		Shell:   "sh",
	}

	m := initialModel(cfg)
	m.height = 20

	// Add test lines
	for i := 1; i <= 100; i++ {
		m.lines = append(m.lines, runner.Line{Number: i, Content: "line content"})
	}

	m.filter = ""
	m.updateFiltered()
	m.offset = 90
	m.cursor = 95

	// Now filter to fewer lines
	m.filter = "xyz" // No matches
	m.updateFiltered()

	// Offset should be clamped to valid range
	if m.offset != 0 {
		t.Errorf("expected offset to be clamped to 0, got %d", m.offset)
	}

	// Cursor should be clamped
	if m.cursor != 0 {
		t.Errorf("expected cursor to be clamped to 0, got %d", m.cursor)
	}
}

func TestConfigRefreshFromStart(t *testing.T) {
	// Test with RefreshFromStart false (default)
	cfg := Config{
		Command:          "echo test",
		Shell:            "sh",
		RefreshInterval:  5 * time.Second,
		RefreshFromStart: false,
	}

	if cfg.RefreshFromStart {
		t.Error("expected RefreshFromStart to be false by default")
	}

	// Test with RefreshFromStart true
	cfg.RefreshFromStart = true
	if !cfg.RefreshFromStart {
		t.Error("expected RefreshFromStart to be true after setting")
	}
}

func TestModelUserScrolled(t *testing.T) {
	cfg := Config{
		Command: "echo test",
		Shell:   "sh",
	}

	m := initialModel(cfg)

	// Initially should be false
	if m.userScrolled {
		t.Error("expected userScrolled to be false initially")
	}

	// After setting, should be true
	m.userScrolled = true
	if !m.userScrolled {
		t.Error("expected userScrolled to be true after setting")
	}
}

func TestModelRefreshGeneration(t *testing.T) {
	cfg := Config{
		Command: "echo test",
		Shell:   "sh",
	}

	m := initialModel(cfg)

	// Initially should be 0
	if m.refreshGeneration != 0 {
		t.Errorf("expected refreshGeneration to be 0 initially, got %d", m.refreshGeneration)
	}

	// After incrementing
	m.refreshGeneration++
	if m.refreshGeneration != 1 {
		t.Errorf("expected refreshGeneration to be 1 after increment, got %d", m.refreshGeneration)
	}
}

func testModel(cfg Config) *model {
	m := initialModel(cfg)
	return &m
}

func TestFilterCursorMovement(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filter = "hello"
	m.filterCursor = 5

	// Left arrow moves cursor left
	keyMsg := tea.KeyMsg{Type: tea.KeyLeft}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterCursor != 4 {
		t.Errorf("expected filterCursor 4 after left, got %d", m.filterCursor)
	}

	// Left again
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterCursor != 3 {
		t.Errorf("expected filterCursor 3 after second left, got %d", m.filterCursor)
	}

	// Left doesn't go below 0
	m.filterCursor = 0
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterCursor != 0 {
		t.Errorf("expected filterCursor 0 (clamped), got %d", m.filterCursor)
	}

	// Right arrow moves cursor right
	m.filterCursor = 2
	keyMsg = tea.KeyMsg{Type: tea.KeyRight}
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterCursor != 3 {
		t.Errorf("expected filterCursor 3 after right, got %d", m.filterCursor)
	}

	// Right doesn't go past end
	m.filterCursor = 5
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterCursor != 5 {
		t.Errorf("expected filterCursor 5 (clamped), got %d", m.filterCursor)
	}
}

func TestFilterAltLeftRight(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}

	t.Run("alt+left jumps to previous word boundary", func(t *testing.T) {
		m := testModel(cfg)
		m.filterMode = true
		m.filter = "foo bar baz"
		m.filterCursor = 11 // end

		keyMsg := tea.KeyMsg{Type: tea.KeyLeft, Alt: true}
		result, _ := m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 8 {
			t.Errorf("expected filterCursor 8, got %d", m.filterCursor)
		}

		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 4 {
			t.Errorf("expected filterCursor 4, got %d", m.filterCursor)
		}

		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 0 {
			t.Errorf("expected filterCursor 0, got %d", m.filterCursor)
		}

		// Already at start, stays at 0
		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 0 {
			t.Errorf("expected filterCursor 0 (clamped), got %d", m.filterCursor)
		}
	})

	t.Run("alt+right jumps to next word boundary", func(t *testing.T) {
		m := testModel(cfg)
		m.filterMode = true
		m.filter = "foo bar baz"
		m.filterCursor = 0

		keyMsg := tea.KeyMsg{Type: tea.KeyRight, Alt: true}
		result, _ := m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 4 {
			t.Errorf("expected filterCursor 4, got %d", m.filterCursor)
		}

		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 8 {
			t.Errorf("expected filterCursor 8, got %d", m.filterCursor)
		}

		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 11 {
			t.Errorf("expected filterCursor 11, got %d", m.filterCursor)
		}

		// Already at end, stays at 11
		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 11 {
			t.Errorf("expected filterCursor 11 (clamped), got %d", m.filterCursor)
		}
	})

	t.Run("alt+left skips trailing spaces", func(t *testing.T) {
		m := testModel(cfg)
		m.filterMode = true
		m.filter = "foo   bar"
		m.filterCursor = 6 // middle of spaces, before "bar"

		keyMsg := tea.KeyMsg{Type: tea.KeyLeft, Alt: true}
		result, _ := m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 0 {
			t.Errorf("expected filterCursor 0, got %d", m.filterCursor)
		}
	})

	t.Run("alt+right skips trailing spaces", func(t *testing.T) {
		m := testModel(cfg)
		m.filterMode = true
		m.filter = "foo   bar"
		m.filterCursor = 3 // end of "foo"

		keyMsg := tea.KeyMsg{Type: tea.KeyRight, Alt: true}
		result, _ := m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterCursor != 6 {
			t.Errorf("expected filterCursor 6, got %d", m.filterCursor)
		}
	})
}

func TestFilterInsertAtCursor(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filter = "helo"
	m.filterCursor = 3

	// Insert 'l' at position 3 -> "hello"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filter != "hello" {
		t.Errorf("expected filter 'hello', got %q", m.filter)
	}
	if m.filterCursor != 4 {
		t.Errorf("expected filterCursor 4, got %d", m.filterCursor)
	}
}

func TestFilterBackspaceAtCursor(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filter = "hello"
	m.filterCursor = 3

	// Backspace at position 3 -> "helo"
	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filter != "helo" {
		t.Errorf("expected filter 'helo', got %q", m.filter)
	}
	if m.filterCursor != 2 {
		t.Errorf("expected filterCursor 2, got %d", m.filterCursor)
	}

	// Backspace at position 0 does nothing
	m.filterCursor = 0
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filter != "helo" {
		t.Errorf("expected filter 'helo' (unchanged), got %q", m.filter)
	}
	if m.filterCursor != 0 {
		t.Errorf("expected filterCursor 0, got %d", m.filterCursor)
	}
}

func TestFilterAltBackspace(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}

	tests := []struct {
		name           string
		filter         string
		cursor         int
		expectedFilter string
		expectedCursor int
	}{
		{"delete last word", "hello world", 11, "hello ", 6},
		{"delete middle word", "foo bar baz", 7, "foo  baz", 4},
		{"delete first word", "hello world", 5, " world", 0},
		{"delete with trailing spaces", "hello   ", 8, "", 0},
		{"cursor at start", "hello", 0, "hello", 0},
		{"single word", "hello", 5, "", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := testModel(cfg)
			m.filterMode = true
			m.filter = tt.filter
			m.filterCursor = tt.cursor

			keyMsg := tea.KeyMsg{Type: tea.KeyBackspace, Alt: true}
			result, _ := m.handleKeyPress(keyMsg)
			newModel := result.(*model)

			if newModel.filter != tt.expectedFilter {
				t.Errorf("expected filter %q, got %q", tt.expectedFilter, newModel.filter)
			}
			if newModel.filterCursor != tt.expectedCursor {
				t.Errorf("expected filterCursor %d, got %d", tt.expectedCursor, newModel.filterCursor)
			}
		})
	}
}

func TestFilterRegexToggle(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filter = ""
	m.filterCursor = 0

	// Type '/' on empty filter toggles regex mode on
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)
	if !m.filterRegex {
		t.Error("expected filterRegex to be true after typing /")
	}
	if m.filter != "" {
		t.Errorf("expected empty filter, got %q", m.filter)
	}

	// Type '/' again on empty filter toggles regex mode off
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterRegex {
		t.Error("expected filterRegex to be false after second /")
	}

	// Type '/' when filter is non-empty adds it to filter
	m.filterRegex = true
	m.filter = "abc"
	m.filterCursor = 3
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filter != "abc/" {
		t.Errorf("expected filter 'abc/', got %q", m.filter)
	}
	if !m.filterRegex {
		t.Error("expected filterRegex to remain true")
	}
}

func TestFilterRegexMatching(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := initialModel(cfg)
	m.lines = []runner.Line{
		{Number: 1, Content: "hello world"},
		{Number: 2, Content: "foo bar"},
		{Number: 3, Content: "hello foo"},
		{Number: 4, Content: "baz 123 qux"},
	}

	// Regex filter matching
	m.filterRegex = true
	m.filter = "hello.*foo"
	m.updateFiltered()
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 match for regex 'hello.*foo', got %d", len(m.filtered))
	}
	if len(m.filtered) > 0 && m.filtered[0] != 2 {
		t.Errorf("expected match at index 2, got %d", m.filtered[0])
	}

	// Regex with character class
	m.filter = "\\d+"
	m.updateFiltered()
	if len(m.filtered) != 1 {
		t.Errorf("expected 1 match for regex '\\d+', got %d", len(m.filtered))
	}

	// Regex is case insensitive
	m.filter = "HELLO"
	m.updateFiltered()
	if len(m.filtered) != 2 {
		t.Errorf("expected 2 matches for case-insensitive regex 'HELLO', got %d", len(m.filtered))
	}
}

func TestFilterRegexInvalid(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := initialModel(cfg)
	m.lines = []runner.Line{
		{Number: 1, Content: "hello world"},
		{Number: 2, Content: "foo bar"},
	}

	m.filterRegex = true
	m.filter = "[invalid"
	m.updateFiltered()

	// Should have an error
	if m.filterRegexErr == nil {
		t.Error("expected filterRegexErr to be non-nil for invalid regex")
	}

	// Should show all lines when regex is invalid
	if len(m.filtered) != 2 {
		t.Errorf("expected all 2 lines shown for invalid regex, got %d", len(m.filtered))
	}

	// Valid regex clears the error
	m.filter = "hello"
	m.updateFiltered()
	if m.filterRegexErr != nil {
		t.Errorf("expected filterRegexErr to be nil for valid regex, got %v", m.filterRegexErr)
	}
}

func TestFilterEscClearsRegex(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filter = "test"
	m.filterCursor = 4
	m.filterRegex = true

	// Esc in filter mode clears everything
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)

	if m.filterMode {
		t.Error("expected filterMode to be false")
	}
	if m.filter != "" {
		t.Errorf("expected empty filter, got %q", m.filter)
	}
	if m.filterCursor != 0 {
		t.Errorf("expected filterCursor 0, got %d", m.filterCursor)
	}
	if m.filterRegex {
		t.Error("expected filterRegex to be false")
	}
}

func TestStopCommandKeybinding(t *testing.T) {
	cfg := Config{
		Command: "echo test",
		Shell:   "sh",
	}

	t.Run("stops running command when streaming", func(t *testing.T) {
		m := initialModel(cfg)
		// Set up a cancellable context to track if cancel was called
		ctx, cancel := context.WithCancel(context.Background())
		m.ctx = ctx
		m.cancel = cancel
		m.streaming = true
		m.statusMsg = ""

		// Simulate pressing 'c'
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		result, cmd := m.handleKeyPress(keyMsg)
		newModel := result.(*model)

		// Should set status message
		if newModel.statusMsg != "Command stopped" {
			t.Errorf("expected statusMsg 'Command stopped', got %q", newModel.statusMsg)
		}

		// Should return a command (the tick for clearing status)
		if cmd == nil {
			t.Error("expected a command to be returned for status message timeout")
		}

		// Context should be cancelled
		select {
		case <-ctx.Done():
			// Good, context was cancelled
		default:
			t.Error("expected context to be cancelled")
		}
	})

	t.Run("does nothing when not streaming", func(t *testing.T) {
		m := initialModel(cfg)
		ctx, cancel := context.WithCancel(context.Background())
		m.ctx = ctx
		m.cancel = cancel
		m.streaming = false
		m.statusMsg = ""

		// Simulate pressing 'c'
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}}
		result, cmd := m.handleKeyPress(keyMsg)
		newModel := result.(*model)

		// Should not set status message
		if newModel.statusMsg != "" {
			t.Errorf("expected empty statusMsg, got %q", newModel.statusMsg)
		}

		// Should not return a command
		if cmd != nil {
			t.Error("expected no command to be returned when not streaming")
		}

		// Context should NOT be cancelled
		select {
		case <-ctx.Done():
			t.Error("expected context to NOT be cancelled when not streaming")
		default:
			// Good, context is still active
		}
	})
}
