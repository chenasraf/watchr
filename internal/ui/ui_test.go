package ui

import (
	"testing"
	"time"

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
