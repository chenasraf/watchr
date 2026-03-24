package ui

import (
	"strings"
	"testing"
)

func TestRenderHelpOverlay(t *testing.T) {
	m := testModelWithLines()
	box, boxWidth, boxHeight := m.renderHelpOverlay()

	if boxWidth == 0 || boxHeight == 0 {
		t.Error("expected non-zero overlay dimensions")
	}

	// Should contain some keybinding text
	if !strings.Contains(box, "Keybindings") {
		t.Error("expected help overlay to contain 'Keybindings'")
	}
	if !strings.Contains(box, "Reload command") {
		t.Error("expected help overlay to contain 'Reload command'")
	}
}

func TestRenderConfirmOverlay(t *testing.T) {
	m := testModelWithLines()
	m.confirmMessage = "Delete everything?"

	box, boxWidth, boxHeight := m.renderConfirmOverlay()

	if boxWidth == 0 || boxHeight == 0 {
		t.Error("expected non-zero overlay dimensions")
	}

	if !strings.Contains(box, "Delete everything?") {
		t.Error("expected confirm overlay to contain the message")
	}
}

func TestRenderCmdPaletteOverlay(t *testing.T) {
	m := testModelWithLines()
	m.cmdPaletteMode = true
	m.cmdPaletteInput.Text = ""
	m.cmdPaletteInput.Cursor = 0
	m.cmdPaletteSelected = 0

	box, boxWidth, boxHeight := m.renderCmdPaletteOverlay()

	if boxWidth == 0 || boxHeight == 0 {
		t.Error("expected non-zero overlay dimensions")
	}

	// Should contain command names
	if !strings.Contains(box, "Reload command") {
		t.Error("expected palette to contain 'Reload command'")
	}
	if !strings.Contains(box, "Quit") {
		t.Error("expected palette to contain 'Quit'")
	}
}

func TestViewInitialLoading(t *testing.T) {
	m := testModelWithLines()
	m.width = 0
	m.height = 0

	view := m.View()
	if !strings.Contains(view, "Running command") {
		t.Errorf("expected loading view, got %q", view)
	}
}

func TestViewWithHelpOverlay(t *testing.T) {
	m := testModelWithLines()
	m.showHelp = true

	view := m.View()
	if !strings.Contains(view, "Keybindings") {
		t.Error("expected help overlay in view")
	}
}

func TestViewWithConfirmOverlay(t *testing.T) {
	m := testModelWithLines()
	m.confirmMode = true
	m.confirmMessage = "Are you sure?"

	view := m.View()
	if !strings.Contains(view, "Are you sure?") {
		t.Error("expected confirm overlay in view")
	}
}

func TestViewWithCmdPalette(t *testing.T) {
	m := testModelWithLines()
	m.cmdPaletteMode = true
	m.cmdPaletteInput.Text = ""
	m.cmdPaletteInput.Cursor = 0

	view := m.View()
	if !strings.Contains(view, "Reload command") {
		t.Error("expected command palette in view")
	}
}
