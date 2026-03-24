package ui

import (
	"testing"

	"github.com/chenasraf/watchr/internal/runner"
)

func TestActionReload(t *testing.T) {
	m := testModelWithLines()
	_, cmd := m.actionReload()
	if m.refreshGeneration != 1 {
		t.Errorf("expected refreshGeneration 1, got %d", m.refreshGeneration)
	}
	if cmd == nil {
		t.Error("expected command to be returned")
	}
}

func TestActionReloadClear(t *testing.T) {
	m := testModelWithLines()
	originalLines := len(m.lines)
	if originalLines == 0 {
		t.Fatal("expected lines to be populated")
	}

	_, cmd := m.actionReloadClear()
	if m.lines != nil {
		t.Errorf("expected lines nil, got %d", len(m.lines))
	}
	if m.refreshGeneration != 1 {
		t.Errorf("expected refreshGeneration 1, got %d", m.refreshGeneration)
	}
	if cmd == nil {
		t.Error("expected command to be returned")
	}
}

func TestActionDeleteLine(t *testing.T) {
	m := testModelWithLines()
	m.cursor = 1 // "foo bar"
	m.actionDeleteLine()

	for _, line := range m.lines {
		if line.Content == "foo bar" {
			t.Error("expected 'foo bar' to be deleted")
		}
	}
	if len(m.lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(m.lines))
	}
}

func TestActionDeleteLineEmpty(t *testing.T) {
	cfg := Config{Command: "echo test", Shell: "sh"}
	m := testModel(cfg)
	// Should not panic
	m.actionDeleteLine()
}

func TestActionClearAllLines(t *testing.T) {
	m := testModelWithLines()
	m.actionClearAllLines()

	if !m.confirmMode {
		t.Error("expected confirmMode true")
	}
	if m.confirmMessage != "Clear all lines? (y/N)" {
		t.Errorf("expected confirm message, got %q", m.confirmMessage)
	}
}

func TestActionStopCommand(t *testing.T) {
	m := testModelWithCancel()
	m.streaming = true

	_, cmd := m.actionStopCommand()
	if m.statusMsg != "Command stopped" {
		t.Errorf("expected 'Command stopped', got %q", m.statusMsg)
	}
	if cmd == nil {
		t.Error("expected timeout command")
	}
}

func TestActionStopCommandNotStreaming(t *testing.T) {
	m := testModelWithCancel()
	m.streaming = false

	_, cmd := m.actionStopCommand()
	if m.statusMsg != "" {
		t.Errorf("expected empty status, got %q", m.statusMsg)
	}
	if cmd != nil {
		t.Error("expected nil command when not streaming")
	}
}

func TestActionTogglePreview(t *testing.T) {
	m := testModelWithLines()
	m.showPreview = false

	m.actionTogglePreview()
	if !m.showPreview {
		t.Error("expected showPreview true")
	}

	m.actionTogglePreview()
	if m.showPreview {
		t.Error("expected showPreview false")
	}
}

func TestActionPreviewResize(t *testing.T) {
	m := testModelWithLines()
	m.showPreview = true
	m.config.PreviewSize = 10
	m.config.PreviewSizeIsPercent = false

	m.actionIncreasePreview()
	if m.config.PreviewSize != 12 {
		t.Errorf("expected 12, got %d", m.config.PreviewSize)
	}

	m.actionDecreasePreview()
	if m.config.PreviewSize != 10 {
		t.Errorf("expected 10, got %d", m.config.PreviewSize)
	}
}

func TestActionGoToFirstLast(t *testing.T) {
	m := testModelWithLines()
	m.cursor = 2

	m.actionGoToFirst()
	if m.cursor != 0 {
		t.Errorf("expected cursor 0, got %d", m.cursor)
	}

	m.actionGoToLast()
	if m.cursor != len(m.filtered)-1 {
		t.Errorf("expected cursor %d, got %d", len(m.filtered)-1, m.cursor)
	}
}

func TestActionCopyLine(t *testing.T) {
	m := testModelWithLines()
	m.cursor = 0

	// Test copy (may succeed or fail depending on clipboard availability)
	_, cmd := m.actionCopyLine(false)
	if m.statusMsg == "" {
		t.Error("expected statusMsg to be set")
	}
	if cmd == nil {
		t.Error("expected timeout command")
	}
}

func TestActionCopyLinePlain(t *testing.T) {
	m := testModelWithLines()
	m.lines = []runner.Line{{Number: 1, Content: "\x1b[31mred\x1b[0m"}}
	m.updateFiltered()
	m.cursor = 0

	_, cmd := m.actionCopyLine(true)
	if m.statusMsg == "" {
		t.Error("expected statusMsg to be set")
	}
	if cmd == nil {
		t.Error("expected timeout command")
	}
}

func TestActionShowHelp(t *testing.T) {
	m := testModelWithLines()
	m.actionShowHelp()
	if !m.showHelp {
		t.Error("expected showHelp true")
	}
}

func TestActionOpenPalette(t *testing.T) {
	m := testModelWithLines()
	m.actionOpenPalette()
	if !m.cmdPaletteMode {
		t.Error("expected cmdPaletteMode true")
	}
}

func TestActionEnterFilter(t *testing.T) {
	m := testModelWithLines()
	m.actionEnterFilter()
	if !m.filterMode {
		t.Error("expected filterMode true")
	}
}
