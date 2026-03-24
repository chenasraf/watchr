package ui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		return m.handleHelpMode(msg)
	}
	if m.confirmMode {
		return m.handleConfirmMode(msg)
	}
	if m.cmdPaletteMode {
		return m.handleCmdPaletteMode(msg)
	}
	if m.filterMode {
		return m.handleFilterMode(msg)
	}
	return m.handleNormalMode(msg)
}

func (m *model) handleHelpMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "?", "esc", "q", "enter":
		m.showHelp = false
	}
	return m, nil
}

func (m *model) handleConfirmMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		m.confirmMode = false
		if m.confirmAction != nil {
			return m.confirmAction(m)
		}
	default:
		m.confirmMode = false
	}
	return m, nil
}

func (m *model) handleCmdPaletteMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.cmdPaletteMode = false
		m.cmdPaletteInput.clear()
		m.cmdPaletteSelected = 0
		return m, nil
	case tea.KeyEnter:
		filtered := m.filteredCommands()
		if len(filtered) > 0 && m.cmdPaletteSelected < len(filtered) {
			m.cmdPaletteMode = false
			cmd := filtered[m.cmdPaletteSelected]
			m.cmdPaletteInput.clear()
			m.cmdPaletteSelected = 0
			return cmd.action(m)
		}
		return m, nil
	case tea.KeyUp:
		if m.cmdPaletteSelected > 0 {
			m.cmdPaletteSelected--
		}
		return m, nil
	case tea.KeyDown:
		filtered := m.filteredCommands()
		if m.cmdPaletteSelected < len(filtered)-1 {
			m.cmdPaletteSelected++
		}
		return m, nil
	default:
		if m.cmdPaletteInput.handleKey(msg) {
			m.cmdPaletteSelected = 0
		}
		return m, nil
	}
}

func (m *model) handleFilterMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.filterMode = false
		m.filterInput.clear()
		m.filterRegex = false
		m.filterRegexErr = nil
		m.updateFiltered()
		return m, nil
	case tea.KeyEnter:
		m.filterMode = false
		return m, nil
	default:
		// Special case: "/" on empty filter toggles regex mode
		if msg.Type == tea.KeyRunes && len(msg.Runes) > 0 && string(msg.Runes) == "/" && m.filterInput.Text == "" {
			m.filterRegex = !m.filterRegex
			m.filterRegexErr = nil
			m.updateFiltered()
			return m, nil
		}
		m.filterInput.handleKey(msg)
		m.updateFiltered()
		return m, nil
	}
}

func (m *model) handleNormalMode(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "ctrl+c":
		return m.actionQuit()
	case "esc":
		if m.filterInput.Text != "" || m.filterRegex {
			m.filterInput.clear()
			m.filterRegex = false
			m.filterRegexErr = nil
			m.updateFiltered()
			return m, nil
		}
		return m.actionQuit()

	case "j", "down", "ctrl+n":
		m.userScrolled = true
		m.moveCursor(1)
	case "k", "up", "ctrl+p":
		m.userScrolled = true
		m.moveCursor(-1)
	case "g", "home":
		return m.actionGoToFirst()
	case "G", "end":
		return m.actionGoToLast()
	case "ctrl+d":
		m.userScrolled = true
		m.moveCursor(m.visibleLines() / 2)
	case "ctrl+u":
		m.userScrolled = true
		m.moveCursor(-m.visibleLines() / 2)
	case "J":
		if m.showPreview {
			m.previewOffset++
			m.clampPreviewOffset()
		}
	case "K":
		if m.showPreview && m.previewOffset > 0 {
			m.previewOffset--
		}
	case "pgdown", "ctrl+f":
		m.userScrolled = true
		m.moveCursor(m.visibleLines())
	case "pgup", "ctrl+b":
		m.userScrolled = true
		m.moveCursor(-m.visibleLines())
	case "p":
		return m.actionTogglePreview()
	case "+", "=":
		return m.actionIncreasePreview()
	case "-":
		return m.actionDecreasePreview()
	case "r", "ctrl+r":
		return m.actionReload()
	case "R":
		return m.actionReloadClear()
	case "d", "delete":
		return m.actionDeleteLine()
	case "D":
		return m.actionClearAllLines()
	case "c":
		return m.actionStopCommand()
	case "/":
		return m.actionEnterFilter()
	case ":":
		return m.actionOpenPalette()
	case "?":
		return m.actionShowHelp()
	case "y":
		return m.actionCopyLine(false)
	case "Y":
		return m.actionCopyLine(true)
	}

	return m, nil
}
