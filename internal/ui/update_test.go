package ui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestUpdateWindowSize(t *testing.T) {
	m := testModelWithLines()
	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	result, _ := m.Update(msg)
	newModel := result.(*model)

	if newModel.width != 120 {
		t.Errorf("expected width 120, got %d", newModel.width)
	}
	if newModel.height != 40 {
		t.Errorf("expected height 40, got %d", newModel.height)
	}
}

func TestUpdateClearStatusMsg(t *testing.T) {
	m := testModelWithLines()
	m.statusMsg = "some status"

	result, _ := m.Update(clearStatusMsg{})
	newModel := result.(*model)

	if newModel.statusMsg != "" {
		t.Errorf("expected empty statusMsg, got %q", newModel.statusMsg)
	}
}

func TestUpdateSpinnerTick(t *testing.T) {
	m := testModelWithLines()
	m.loading = true
	m.spinnerFrame = 0

	result, cmd := m.Update(spinnerTickMsg{})
	newModel := result.(*model)

	if newModel.spinnerFrame != 1 {
		t.Errorf("expected spinnerFrame 1, got %d", newModel.spinnerFrame)
	}
	if cmd == nil {
		t.Error("expected a command for next spinner tick")
	}
}

func TestUpdateSpinnerTickNotLoading(t *testing.T) {
	m := testModelWithLines()
	m.loading = false
	m.streaming = false
	m.spinnerFrame = 3

	result, cmd := m.Update(spinnerTickMsg{})
	newModel := result.(*model)

	// Frame should not advance
	if newModel.spinnerFrame != 3 {
		t.Errorf("expected spinnerFrame 3 (unchanged), got %d", newModel.spinnerFrame)
	}
	if cmd != nil {
		t.Error("expected no command when not loading/streaming")
	}
}

func TestUpdateErrMsg(t *testing.T) {
	m := testModelWithLines()
	m.loading = true
	m.streaming = true

	result, _ := m.Update(errMsg{err: fmt.Errorf("test error")})
	newModel := result.(*model)

	if newModel.errorMsg != "test error" {
		t.Errorf("expected errorMsg 'test error', got %q", newModel.errorMsg)
	}
	if newModel.loading {
		t.Error("expected loading false after error")
	}
	if newModel.streaming {
		t.Error("expected streaming false after error")
	}
}
