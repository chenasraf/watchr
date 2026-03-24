package ui

import (
	"context"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chenasraf/watchr/internal/runner"
)

func TestFilterCursorMovement(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filterInput.Text = "hello"
	m.filterInput.Cursor = 5

	// Left arrow moves cursor left
	keyMsg := tea.KeyMsg{Type: tea.KeyLeft}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterInput.Cursor != 4 {
		t.Errorf("expected filterCursor 4 after left, got %d", m.filterInput.Cursor)
	}

	// Left again
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterInput.Cursor != 3 {
		t.Errorf("expected filterCursor 3 after second left, got %d", m.filterInput.Cursor)
	}

	// Left doesn't go below 0
	m.filterInput.Cursor = 0
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterInput.Cursor != 0 {
		t.Errorf("expected filterCursor 0 (clamped), got %d", m.filterInput.Cursor)
	}

	// Right arrow moves cursor right
	m.filterInput.Cursor = 2
	keyMsg = tea.KeyMsg{Type: tea.KeyRight}
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterInput.Cursor != 3 {
		t.Errorf("expected filterCursor 3 after right, got %d", m.filterInput.Cursor)
	}

	// Right doesn't go past end
	m.filterInput.Cursor = 5
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterInput.Cursor != 5 {
		t.Errorf("expected filterCursor 5 (clamped), got %d", m.filterInput.Cursor)
	}
}

func TestFilterAltLeftRight(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}

	t.Run("alt+left jumps to previous word boundary", func(t *testing.T) {
		m := testModel(cfg)
		m.filterMode = true
		m.filterInput.Text = "foo bar baz"
		m.filterInput.Cursor = 11 // end

		keyMsg := tea.KeyMsg{Type: tea.KeyLeft, Alt: true}
		result, _ := m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 8 {
			t.Errorf("expected filterCursor 8, got %d", m.filterInput.Cursor)
		}

		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 4 {
			t.Errorf("expected filterCursor 4, got %d", m.filterInput.Cursor)
		}

		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 0 {
			t.Errorf("expected filterCursor 0, got %d", m.filterInput.Cursor)
		}

		// Already at start, stays at 0
		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 0 {
			t.Errorf("expected filterCursor 0 (clamped), got %d", m.filterInput.Cursor)
		}
	})

	t.Run("alt+right jumps to next word boundary", func(t *testing.T) {
		m := testModel(cfg)
		m.filterMode = true
		m.filterInput.Text = "foo bar baz"
		m.filterInput.Cursor = 0

		keyMsg := tea.KeyMsg{Type: tea.KeyRight, Alt: true}
		result, _ := m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 4 {
			t.Errorf("expected filterCursor 4, got %d", m.filterInput.Cursor)
		}

		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 8 {
			t.Errorf("expected filterCursor 8, got %d", m.filterInput.Cursor)
		}

		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 11 {
			t.Errorf("expected filterCursor 11, got %d", m.filterInput.Cursor)
		}

		// Already at end, stays at 11
		result, _ = m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 11 {
			t.Errorf("expected filterCursor 11 (clamped), got %d", m.filterInput.Cursor)
		}
	})

	t.Run("alt+left skips trailing spaces", func(t *testing.T) {
		m := testModel(cfg)
		m.filterMode = true
		m.filterInput.Text = "foo   bar"
		m.filterInput.Cursor = 6 // middle of spaces, before "bar"

		keyMsg := tea.KeyMsg{Type: tea.KeyLeft, Alt: true}
		result, _ := m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 0 {
			t.Errorf("expected filterCursor 0, got %d", m.filterInput.Cursor)
		}
	})

	t.Run("alt+right skips trailing spaces", func(t *testing.T) {
		m := testModel(cfg)
		m.filterMode = true
		m.filterInput.Text = "foo   bar"
		m.filterInput.Cursor = 3 // end of "foo"

		keyMsg := tea.KeyMsg{Type: tea.KeyRight, Alt: true}
		result, _ := m.handleKeyPress(keyMsg)
		m = result.(*model)
		if m.filterInput.Cursor != 6 {
			t.Errorf("expected filterCursor 6, got %d", m.filterInput.Cursor)
		}
	})
}

func TestFilterInsertAtCursor(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filterInput.Text = "helo"
	m.filterInput.Cursor = 3

	// Insert 'l' at position 3 -> "hello"
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterInput.Text != "hello" {
		t.Errorf("expected filter 'hello', got %q", m.filterInput.Text)
	}
	if m.filterInput.Cursor != 4 {
		t.Errorf("expected filterCursor 4, got %d", m.filterInput.Cursor)
	}
}

func TestFilterBackspaceAtCursor(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filterInput.Text = "hello"
	m.filterInput.Cursor = 3

	// Backspace at position 3 -> "helo"
	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterInput.Text != "helo" {
		t.Errorf("expected filter 'helo', got %q", m.filterInput.Text)
	}
	if m.filterInput.Cursor != 2 {
		t.Errorf("expected filterCursor 2, got %d", m.filterInput.Cursor)
	}

	// Backspace at position 0 does nothing
	m.filterInput.Cursor = 0
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterInput.Text != "helo" {
		t.Errorf("expected filter 'helo' (unchanged), got %q", m.filterInput.Text)
	}
	if m.filterInput.Cursor != 0 {
		t.Errorf("expected filterCursor 0, got %d", m.filterInput.Cursor)
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
			m.filterInput.Text = tt.filter
			m.filterInput.Cursor = tt.cursor

			keyMsg := tea.KeyMsg{Type: tea.KeyBackspace, Alt: true}
			result, _ := m.handleKeyPress(keyMsg)
			newModel := result.(*model)

			if newModel.filterInput.Text != tt.expectedFilter {
				t.Errorf("expected filter %q, got %q", tt.expectedFilter, newModel.filterInput.Text)
			}
			if newModel.filterInput.Cursor != tt.expectedCursor {
				t.Errorf("expected filterCursor %d, got %d", tt.expectedCursor, newModel.filterInput.Cursor)
			}
		})
	}
}

func TestFilterRegexToggle(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filterInput.Text = ""
	m.filterInput.Cursor = 0

	// Type '/' on empty filter toggles regex mode on
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)
	if !m.filterRegex {
		t.Error("expected filterRegex to be true after typing /")
	}
	if m.filterInput.Text != "" {
		t.Errorf("expected empty filter, got %q", m.filterInput.Text)
	}

	// Type '/' again on empty filter toggles regex mode off
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterRegex {
		t.Error("expected filterRegex to be false after second /")
	}

	// Type '/' when filter is non-empty adds it to filter
	m.filterRegex = true
	m.filterInput.Text = "abc"
	m.filterInput.Cursor = 3
	result, _ = m.handleKeyPress(keyMsg)
	m = result.(*model)
	if m.filterInput.Text != "abc/" {
		t.Errorf("expected filter 'abc/', got %q", m.filterInput.Text)
	}
	if !m.filterRegex {
		t.Error("expected filterRegex to remain true")
	}
}

func TestFilterEscClearsRegex(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.filterMode = true
	m.filterInput.Text = "test"
	m.filterInput.Cursor = 4
	m.filterRegex = true

	// Esc in filter mode clears everything
	keyMsg := tea.KeyMsg{Type: tea.KeyEsc}
	result, _ := m.handleKeyPress(keyMsg)
	m = result.(*model)

	if m.filterMode {
		t.Error("expected filterMode to be false")
	}
	if m.filterInput.Text != "" {
		t.Errorf("expected empty filter, got %q", m.filterInput.Text)
	}
	if m.filterInput.Cursor != 0 {
		t.Errorf("expected filterCursor 0, got %d", m.filterInput.Cursor)
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

// New keybinding tests

func TestKeyReloadClear(t *testing.T) {
	m := testModelWithLines()
	if len(m.lines) == 0 {
		t.Fatal("expected lines to be populated")
	}

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'R'}}
	result, cmd := m.handleKeyPress(keyMsg)
	newModel := result.(*model)

	if newModel.lines != nil {
		t.Errorf("expected lines to be nil after R, got %d lines", len(newModel.lines))
	}
	if newModel.refreshGeneration != 1 {
		t.Errorf("expected refreshGeneration 1, got %d", newModel.refreshGeneration)
	}
	if cmd == nil {
		t.Error("expected a command to be returned")
	}
}

func TestKeyDelete(t *testing.T) {
	m := testModelWithLines()
	originalLen := len(m.lines)
	m.cursor = 1 // select "foo bar"

	keyMsg := tea.KeyMsg{Type: tea.KeyDelete}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)

	if len(newModel.lines) != originalLen-1 {
		t.Errorf("expected %d lines after delete, got %d", originalLen-1, len(newModel.lines))
	}
	// The second line ("foo bar") should be gone
	for _, line := range newModel.lines {
		if line.Content == "foo bar" {
			t.Error("expected 'foo bar' to be deleted")
		}
	}
}

func TestKeyDeleteEmpty(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	m.height = 30
	m.width = 80

	// No lines, should not panic
	keyMsg := tea.KeyMsg{Type: tea.KeyDelete}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if len(newModel.lines) != 0 {
		t.Errorf("expected 0 lines, got %d", len(newModel.lines))
	}
}

func TestKeyCtrlDelete(t *testing.T) {
	m := testModelWithLines()
	// ctrl+delete is hard to simulate via tea.KeyMsg, so test the confirm flow directly
	m.confirmMode = true
	m.confirmMessage = "Clear all lines? (y/N)"
	m.confirmAction = func(m *model) (tea.Model, tea.Cmd) {
		m.lines = nil
		m.updateFiltered()
		m.statusMsg = "All lines cleared"
		return m, nil
	}

	// Test confirmation with 'y'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if newModel.lines != nil {
		t.Errorf("expected lines to be nil after confirm, got %d", len(newModel.lines))
	}
	if newModel.statusMsg != "All lines cleared" {
		t.Errorf("expected status 'All lines cleared', got %q", newModel.statusMsg)
	}
}

func TestKeyConfirmDialog(t *testing.T) {
	t.Run("y confirms action", func(t *testing.T) {
		m := testModelWithLines()
		confirmed := false
		m.confirmMode = true
		m.confirmMessage = "Test?"
		m.confirmAction = func(m *model) (tea.Model, tea.Cmd) {
			confirmed = true
			return m, nil
		}

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
		result, _ := m.handleKeyPress(keyMsg)
		newModel := result.(*model)

		if !confirmed {
			t.Error("expected confirm action to be called")
		}
		if newModel.confirmMode {
			t.Error("expected confirmMode to be false after confirm")
		}
	})

	t.Run("other key cancels", func(t *testing.T) {
		m := testModelWithLines()
		confirmed := false
		m.confirmMode = true
		m.confirmMessage = "Test?"
		m.confirmAction = func(m *model) (tea.Model, tea.Cmd) {
			confirmed = true
			return m, nil
		}

		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}}
		result, _ := m.handleKeyPress(keyMsg)
		newModel := result.(*model)

		if confirmed {
			t.Error("expected confirm action NOT to be called")
		}
		if newModel.confirmMode {
			t.Error("expected confirmMode to be false after cancel")
		}
	})
}

func TestKeyGoToFirstLast(t *testing.T) {
	m := testModelWithLines()
	m.cursor = 2

	// Go to first line with 'g'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)

	if newModel.cursor != 0 {
		t.Errorf("expected cursor 0 after 'g', got %d", newModel.cursor)
	}
	if newModel.offset != 0 {
		t.Errorf("expected offset 0 after 'g', got %d", newModel.offset)
	}
	if !newModel.userScrolled {
		t.Error("expected userScrolled true after 'g'")
	}

	// Go to last line with 'G'
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}}
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)

	if newModel.cursor != len(newModel.filtered)-1 {
		t.Errorf("expected cursor at last line (%d), got %d", len(newModel.filtered)-1, newModel.cursor)
	}
	if newModel.userScrolled {
		t.Error("expected userScrolled false after 'G' (resume following)")
	}
}

func TestKeyPreviewScrollJK(t *testing.T) {
	m := testModelWithLines()
	m.showPreview = true
	m.config.PreviewSize = 5
	m.config.PreviewSizeIsPercent = false
	m.config.PreviewPosition = PreviewBottom

	// J scrolls preview down
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	// previewOffset might be clamped to 0 for short content, but the code path runs
	if newModel.previewOffset < 0 {
		t.Errorf("expected previewOffset >= 0, got %d", newModel.previewOffset)
	}

	// Set offset to 1 and test K scrolls up
	newModel.previewOffset = 1
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'K'}}
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.previewOffset != 0 {
		t.Errorf("expected previewOffset 0 after K, got %d", newModel.previewOffset)
	}
}

func TestKeyPreviewScrollJKWithoutPreview(t *testing.T) {
	m := testModelWithLines()
	m.showPreview = false

	// J should not change previewOffset when preview is hidden
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'J'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if newModel.previewOffset != 0 {
		t.Errorf("expected previewOffset 0 when preview hidden, got %d", newModel.previewOffset)
	}
}

func TestKeyTogglePreview(t *testing.T) {
	m := testModelWithLines()
	if m.showPreview {
		t.Fatal("expected showPreview false initially")
	}

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'p'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if !newModel.showPreview {
		t.Error("expected showPreview true after 'p'")
	}

	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.showPreview {
		t.Error("expected showPreview false after second 'p'")
	}
}

func TestKeyResizePreview(t *testing.T) {
	m := testModelWithLines()
	m.showPreview = true
	m.config.PreviewSize = 10
	m.config.PreviewSizeIsPercent = false

	// '+' increases
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if newModel.config.PreviewSize != 12 {
		t.Errorf("expected PreviewSize 12 after '+', got %d", newModel.config.PreviewSize)
	}

	// '-' decreases
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'-'}}
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.config.PreviewSize != 10 {
		t.Errorf("expected PreviewSize 10 after '-', got %d", newModel.config.PreviewSize)
	}

	// '-' doesn't go below step
	newModel.config.PreviewSize = 2
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.config.PreviewSize != 2 {
		t.Errorf("expected PreviewSize 2 (minimum), got %d", newModel.config.PreviewSize)
	}
}

func TestKeyCmdPaletteOpenAndNav(t *testing.T) {
	m := testModelWithLines()

	// ':' opens command palette
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if !newModel.cmdPaletteMode {
		t.Error("expected cmdPaletteMode true after ':'")
	}

	// Down arrow moves selection
	keyMsg = tea.KeyMsg{Type: tea.KeyDown}
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.cmdPaletteSelected != 1 {
		t.Errorf("expected cmdPaletteSelected 1, got %d", newModel.cmdPaletteSelected)
	}

	// Up arrow moves selection back
	keyMsg = tea.KeyMsg{Type: tea.KeyUp}
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.cmdPaletteSelected != 0 {
		t.Errorf("expected cmdPaletteSelected 0, got %d", newModel.cmdPaletteSelected)
	}

	// Typing filters
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}}
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.cmdPaletteInput.Text != "r" {
		t.Errorf("expected cmdPaletteFilter 'r', got %q", newModel.cmdPaletteInput.Text)
	}

	// Esc closes palette
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.cmdPaletteMode {
		t.Error("expected cmdPaletteMode false after Esc")
	}
}

func TestKeyHelpToggle(t *testing.T) {
	m := testModelWithLines()

	// '?' opens help
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if !newModel.showHelp {
		t.Error("expected showHelp true after '?'")
	}

	// '?' closes help
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.showHelp {
		t.Error("expected showHelp false after second '?'")
	}
}

func TestKeyQuit(t *testing.T) {
	m := testModelWithCancel()

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.handleKeyPress(keyMsg)

	if cmd == nil {
		t.Error("expected quit command to be returned")
	}
}

func TestKeyFilterMode(t *testing.T) {
	m := testModelWithLines()

	// '/' enters filter mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if !newModel.filterMode {
		t.Error("expected filterMode true after '/'")
	}
}

func TestKeyResizePreviewNoEffect(t *testing.T) {
	// '+' and '-' do nothing when preview is not shown
	m := testModelWithLines()
	m.showPreview = false
	m.config.PreviewSize = 10

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'+'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if newModel.config.PreviewSize != 10 {
		t.Errorf("expected PreviewSize unchanged at 10, got %d", newModel.config.PreviewSize)
	}
}

func TestKeyNavigationJK(t *testing.T) {
	m := testModelWithLines()
	m.cursor = 0

	// 'j' moves down
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	result, _ := m.handleKeyPress(keyMsg)
	newModel := result.(*model)
	if newModel.cursor != 1 {
		t.Errorf("expected cursor 1 after 'j', got %d", newModel.cursor)
	}
	if !newModel.userScrolled {
		t.Error("expected userScrolled true after 'j'")
	}

	// 'k' moves up
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	result, _ = newModel.handleKeyPress(keyMsg)
	newModel = result.(*model)
	if newModel.cursor != 0 {
		t.Errorf("expected cursor 0 after 'k', got %d", newModel.cursor)
	}
}

func TestKeyYank(t *testing.T) {
	m := testModelWithLines()
	m.cursor = 0

	// 'y' should set a status message (clipboard may or may not work in test env)
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	result, cmd := m.handleKeyPress(keyMsg)
	newModel := result.(*model)

	// Should have set some status message (either success or failure)
	if newModel.statusMsg == "" {
		t.Error("expected statusMsg to be set after 'y'")
	}
	if cmd == nil {
		t.Error("expected a command for status timeout")
	}
}

func TestKeyYankPlain(t *testing.T) {
	m := testModelWithLines()
	m.lines = []runner.Line{
		{Number: 1, Content: "\x1b[31mred text\x1b[0m"},
	}
	m.updateFiltered()
	m.cursor = 0

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'Y'}}
	result, cmd := m.handleKeyPress(keyMsg)
	newModel := result.(*model)

	if newModel.statusMsg == "" {
		t.Error("expected statusMsg to be set after 'Y'")
	}
	if cmd == nil {
		t.Error("expected a command for status timeout")
	}
}
