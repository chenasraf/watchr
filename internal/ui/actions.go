package ui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) actionReload() (tea.Model, tea.Cmd) {
	m.refreshGeneration++
	cmd := m.startStreaming()
	return m, tea.Batch(cmd, m.spinnerTickCmd())
}

func (m *model) actionReloadClear() (tea.Model, tea.Cmd) {
	m.lines = nil
	m.updateFiltered()
	return m.actionReload()
}

func (m *model) actionDeleteLine() (tea.Model, tea.Cmd) {
	if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
		idx := m.filtered[m.cursor]
		if idx < len(m.lines) {
			m.lines = append(m.lines[:idx], m.lines[idx+1:]...)
			m.updateFiltered()
		}
	}
	return m, nil
}

func (m *model) actionClearAllLines() (tea.Model, tea.Cmd) {
	m.confirmMode = true
	m.confirmMessage = "Clear all lines? (y/N)"
	m.confirmAction = func(m *model) (tea.Model, tea.Cmd) {
		m.lines = nil
		m.updateFiltered()
		m.statusMsg = "All lines cleared"
		return m, m.statusTimeoutCmd()
	}
	return m, nil
}

func (m *model) actionStopCommand() (tea.Model, tea.Cmd) {
	if m.streaming {
		m.cancel()
		m.statusMsg = "Command stopped"
		return m, m.statusTimeoutCmd()
	}
	return m, nil
}

func (m *model) actionTogglePreview() (tea.Model, tea.Cmd) {
	m.showPreview = !m.showPreview
	m.adjustOffset()
	return m, nil
}

func (m *model) actionIncreasePreview() (tea.Model, tea.Cmd) {
	if m.showPreview {
		m.config.PreviewSize += previewSizeStep(m.config.PreviewSizeIsPercent)
		m.adjustOffset()
	}
	return m, nil
}

func (m *model) actionDecreasePreview() (tea.Model, tea.Cmd) {
	if m.showPreview {
		step := previewSizeStep(m.config.PreviewSizeIsPercent)
		if m.config.PreviewSize > step {
			m.config.PreviewSize -= step
			m.adjustOffset()
		}
	}
	return m, nil
}

func (m *model) actionGoToFirst() (tea.Model, tea.Cmd) {
	m.userScrolled = true
	m.previewOffset = 0
	m.cursor = 0
	m.offset = 0
	return m, nil
}

func (m *model) actionGoToLast() (tea.Model, tea.Cmd) {
	m.userScrolled = false
	m.previewOffset = 0
	if len(m.filtered) > 0 {
		m.cursor = len(m.filtered) - 1
		m.adjustOffset()
	}
	return m, nil
}

func (m *model) actionEnterFilter() (tea.Model, tea.Cmd) {
	m.filterMode = true
	m.filterInput.Cursor = len(m.filterInput.Text)
	return m, nil
}

func (m *model) actionToggleRegexFilter() (tea.Model, tea.Cmd) {
	m.filterMode = true
	m.filterRegex = !m.filterRegex
	m.filterRegexErr = nil
	m.filterInput.Cursor = len(m.filterInput.Text)
	m.updateFiltered()
	return m, nil
}

func (m *model) actionCopyLine(plain bool) (tea.Model, tea.Cmd) {
	if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
		idx := m.filtered[m.cursor]
		if idx < len(m.lines) {
			content := m.lines[idx].Content
			if plain {
				content = stripANSI(content)
			}
			if err := copyToClipboard(content); err != nil {
				m.statusMsg = "Failed to copy"
			} else if plain {
				m.statusMsg = "Copied to clipboard (plain)"
			} else {
				m.statusMsg = "Copied to clipboard"
			}
			return m, m.statusTimeoutCmd()
		}
	}
	return m, nil
}

func (m *model) actionShowHelp() (tea.Model, tea.Cmd) {
	m.showHelp = true
	return m, nil
}

func (m *model) actionQuit() (tea.Model, tea.Cmd) {
	m.cancel()
	return m, tea.Quit
}

func (m *model) actionOpenPalette() (tea.Model, tea.Cmd) {
	m.cmdPaletteMode = true
	m.cmdPaletteInput.clear()
	m.cmdPaletteSelected = 0
	return m, nil
}

// statusTimeoutCmd returns a command that clears the status message after 2 seconds.
func (m model) statusTimeoutCmd() tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
}
