package ui

import (
	"testing"
)

func TestCommandsCount(t *testing.T) {
	cmds := commands()
	if len(cmds) != 16 {
		t.Errorf("expected 16 commands, got %d", len(cmds))
	}
}

func TestFilteredCommandsNoFilter(t *testing.T) {
	m := testModelWithLines()
	m.cmdPaletteInput.Text = ""
	filtered := m.filteredCommands()
	all := commands()
	if len(filtered) != len(all) {
		t.Errorf("expected %d commands with no filter, got %d", len(all), len(filtered))
	}
}

func TestFilteredCommandsWithFilter(t *testing.T) {
	m := testModelWithLines()
	m.cmdPaletteInput.Text = "reload"
	filtered := m.filteredCommands()
	// "Reload command" and "Reload & clear lines"
	if len(filtered) != 2 {
		t.Errorf("expected 2 commands matching 'reload', got %d", len(filtered))
	}
}

func TestFilteredCommandsCaseInsensitive(t *testing.T) {
	m := testModelWithLines()
	m.cmdPaletteInput.Text = "QUIT"
	filtered := m.filteredCommands()
	if len(filtered) != 1 {
		t.Errorf("expected 1 command matching 'QUIT', got %d", len(filtered))
	}
}

func TestCommandPaletteTogglePreview(t *testing.T) {
	m := testModelWithLines()
	m.showPreview = false

	// Find and execute "Toggle preview pane" command
	cmds := commands()
	for _, cmd := range cmds {
		if cmd.name == "Toggle preview pane" {
			cmd.action(m)
			break
		}
	}

	if !m.showPreview {
		t.Error("expected showPreview true after toggle command")
	}
}
