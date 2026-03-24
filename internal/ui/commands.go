package ui

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// command represents a command palette entry
type command struct {
	name     string // display name
	shortcut string // keybinding hint
	action   func(m *model) (tea.Model, tea.Cmd)
}

// commands returns the list of available command palette entries.
func commands() []command {
	return []command{
		{"Reload command", "r / Ctrl+r", (*model).actionReload},
		{"Reload & clear lines", "R", (*model).actionReloadClear},
		{"Delete selected line", "d / Del", (*model).actionDeleteLine},
		{"Clear all lines", "D", (*model).actionClearAllLines},
		{"Stop running command", "c", (*model).actionStopCommand},
		{"Toggle preview pane", "p", (*model).actionTogglePreview},
		{"Increase preview size", "+", (*model).actionIncreasePreview},
		{"Decrease preview size", "-", (*model).actionDecreasePreview},
		{"Go to first line", "g", (*model).actionGoToFirst},
		{"Go to last line", "G", (*model).actionGoToLast},
		{"Enter filter mode", "/", (*model).actionEnterFilter},
		{"Toggle regex filter", "//", (*model).actionToggleRegexFilter},
		{"Copy line to clipboard", "y", func(m *model) (tea.Model, tea.Cmd) { return m.actionCopyLine(false) }},
		{"Copy line (plain text)", "Y", func(m *model) (tea.Model, tea.Cmd) { return m.actionCopyLine(true) }},
		{"Show help", "?", (*model).actionShowHelp},
		{"Quit", "q", (*model).actionQuit},
	}
}

// filteredCommands returns commands matching the current palette filter.
func (m *model) filteredCommands() []command {
	all := commands()
	if m.cmdPaletteInput.Text == "" {
		return all
	}
	filter := strings.ToLower(m.cmdPaletteInput.Text)
	var result []command
	for _, c := range all {
		if strings.Contains(strings.ToLower(c.name), filter) {
			result = append(result, c)
		}
	}
	return result
}

// copyToClipboard copies text to the system clipboard using OS-specific commands
func copyToClipboard(text string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		// Try xclip first, fall back to xsel
		if _, err := exec.LookPath("xclip"); err == nil {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		} else {
			cmd = exec.Command("xsel", "--clipboard", "--input")
		}
	case "windows":
		cmd = exec.Command("clip")
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}

	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
