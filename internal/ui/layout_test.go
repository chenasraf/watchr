package ui

import (
	"testing"

	"github.com/chenasraf/watchr/internal/runner"
)

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

func TestPreviewSizeStep(t *testing.T) {
	if previewSizeStep(true) != 5 {
		t.Errorf("expected 5 for percent mode, got %d", previewSizeStep(true))
	}
	if previewSizeStep(false) != 2 {
		t.Errorf("expected 2 for absolute mode, got %d", previewSizeStep(false))
	}
}

func TestClampPreviewOffset(t *testing.T) {
	m := testModelWithLines()
	m.showPreview = true
	m.config.PreviewPosition = PreviewBottom
	m.config.PreviewSize = 5
	m.config.PreviewSizeIsPercent = false

	// Set an excessively high offset
	m.previewOffset = 100
	m.clampPreviewOffset()

	// Should be clamped to a valid range (0 for short content)
	if m.previewOffset > 0 {
		t.Errorf("expected previewOffset clamped to 0 for short content, got %d", m.previewOffset)
	}
}

func TestClampPreviewOffsetNoPreview(t *testing.T) {
	m := testModelWithLines()
	m.showPreview = false
	m.previewOffset = 5

	m.clampPreviewOffset()
	if m.previewOffset != 0 {
		t.Errorf("expected previewOffset reset to 0 when preview hidden, got %d", m.previewOffset)
	}
}

func TestApplyPreviewOffset(t *testing.T) {
	m := testModelWithLines()
	lines := []string{"a", "b", "c", "d", "e"}

	// No offset
	m.previewOffset = 0
	result := m.applyPreviewOffset(lines, 3)
	if len(result) != 5 {
		t.Errorf("expected 5 lines with no offset, got %d", len(result))
	}

	// With offset
	m.previewOffset = 2
	result = m.applyPreviewOffset(lines, 3)
	if len(result) != 3 || result[0] != "c" {
		t.Errorf("expected lines starting at 'c', got %v", result)
	}

	// Offset clamped if too high
	m.previewOffset = 10
	_ = m.applyPreviewOffset(lines, 3)
	if m.previewOffset != 2 {
		t.Errorf("expected previewOffset clamped to 2, got %d", m.previewOffset)
	}
}

func TestAdjustOffset(t *testing.T) {
	m := testModelWithLines()
	m.height = 15 // visibleLines = 15 - 5 = 10

	// Cursor near start - offset should be 0
	m.cursor = 0
	m.adjustOffset()
	if m.offset != 0 {
		t.Errorf("expected offset 0 for cursor at start, got %d", m.offset)
	}

	// Cursor in middle of many lines
	for i := range 50 {
		m.filtered = append(m.filtered, i)
	}
	m.cursor = 25
	m.adjustOffset()
	// Should center the cursor
	expected := 25 - m.visibleLines()/2
	if m.offset != expected {
		t.Errorf("expected offset %d for centered cursor, got %d", expected, m.offset)
	}
}
