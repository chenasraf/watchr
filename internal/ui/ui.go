package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/chenasraf/watchr/internal/runner"
)

// PreviewPosition defines where the preview panel is displayed
type PreviewPosition string

const (
	PreviewBottom PreviewPosition = "bottom"
	PreviewTop    PreviewPosition = "top"
	PreviewLeft   PreviewPosition = "left"
	PreviewRight  PreviewPosition = "right"
)

// Config holds the UI configuration
type Config struct {
	Command         string
	Shell           string
	PreviewHeight   int
	PreviewPosition PreviewPosition
	ShowLineNums    bool
	LineNumWidth    int
	Prompt          string
	RefreshSeconds  int
}

// model represents the application state
type model struct {
	config      Config
	lines       []runner.Line
	filtered    []int // indices into lines that match filter
	cursor      int   // cursor position in filtered list
	offset      int   // scroll offset for visible window
	filter      string
	filterMode  bool
	showPreview bool
	width       int
	height      int
	runner      *runner.Runner
	ctx         context.Context
	cancel      context.CancelFunc
	loading     bool
	errorMsg    string
}

// messages
type linesMsg []runner.Line
type errMsg struct{ err error }
type tickMsg time.Time

func (e errMsg) Error() string { return e.err.Error() }

func initialModel(cfg Config) model {
	ctx, cancel := context.WithCancel(context.Background())
	return model{
		config:      cfg,
		lines:       []runner.Line{},
		filtered:    []int{},
		cursor:      0,
		offset:      0,
		filter:      "",
		filterMode:  false,
		showPreview: false,
		runner:      runner.NewRunner(cfg.Shell, cfg.Command),
		ctx:         ctx,
		cancel:      cancel,
		loading:     true,
	}
}

func (m model) Init() tea.Cmd {
	return m.runCommand()
}

func (m model) runCommand() tea.Cmd {
	r := m.runner
	ctx := m.ctx
	return func() tea.Msg {
		lines, err := r.Run(ctx)
		if err != nil {
			return errMsg{err}
		}
		return linesMsg(lines)
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case linesMsg:
		m.lines = []runner.Line(msg)
		m.loading = false
		m.updateFiltered()
		return m, nil

	case tickMsg:
		if m.config.RefreshSeconds > 0 {
			return m, tea.Batch(
				m.runCommand(),
				m.tickCmd(),
			)
		}
		return m, nil

	case errMsg:
		m.errorMsg = msg.Error()
		m.loading = false
		return m, nil
	}

	return m, nil
}

func (m model) tickCmd() tea.Cmd {
	return tea.Tick(time.Duration(m.config.RefreshSeconds)*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// In filter mode, handle text input
	if m.filterMode {
		switch msg.Type {
		case tea.KeyEsc:
			m.filterMode = false
			m.filter = ""
			m.updateFiltered()
			return m, nil
		case tea.KeyEnter:
			m.filterMode = false
			return m, nil
		case tea.KeyBackspace:
			if len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.updateFiltered()
			}
			return m, nil
		default:
			if msg.Type == tea.KeyRunes {
				m.filter += string(msg.Runes)
				m.updateFiltered()
			}
			return m, nil
		}
	}

	// Normal mode keybindings
	switch msg.String() {
	case "q", "esc", "ctrl+c":
		m.cancel()
		return m, tea.Quit

	case "j", "down", "ctrl+n":
		m.moveCursor(1)
	case "k", "up", "ctrl+p":
		m.moveCursor(-1)
	case "g", "home":
		m.cursor = 0
		m.offset = 0
	case "G", "end":
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
			m.adjustOffset()
		}
	case "ctrl+d":
		m.moveCursor(m.visibleLines() / 2)
	case "ctrl+u":
		m.moveCursor(-m.visibleLines() / 2)
	case "pgdown", "ctrl+f":
		m.moveCursor(m.visibleLines())
	case "pgup", "ctrl+b":
		m.moveCursor(-m.visibleLines())
	case "p":
		m.showPreview = !m.showPreview
		m.adjustOffset() // Keep selected line visible after preview toggle
	case "r", "ctrl+r":
		m.loading = true
		return m, m.runCommand()
	case "/":
		m.filterMode = true
		m.filter = ""
	}

	return m, nil
}

func (m *model) moveCursor(delta int) {
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.adjustOffset()
}

func (m *model) adjustOffset() {
	visible := m.visibleLines()
	if visible <= 0 {
		return
	}

	// Try to center the cursor
	idealOffset := m.cursor - visible/2

	// Clamp to valid range
	if idealOffset < 0 {
		idealOffset = 0
	}
	maxOffset := len(m.filtered) - visible
	if maxOffset < 0 {
		maxOffset = 0
	}
	if idealOffset > maxOffset {
		idealOffset = maxOffset
	}

	m.offset = idealOffset
}

func (m model) visibleLines() int {
	// header (1) + command (1) + prompt at bottom (1) = 3 fixed lines
	fixedLines := 3
	if m.showPreview && (m.config.PreviewPosition == PreviewTop || m.config.PreviewPosition == PreviewBottom) {
		previewHeight := m.height * m.config.PreviewHeight / 100
		return m.height - fixedLines - previewHeight
	}
	return m.height - fixedLines
}

func (m *model) updateFiltered() {
	m.filtered = []int{}

	filter := strings.ToLower(m.filter)
	for i, line := range m.lines {
		if m.filter == "" || strings.Contains(strings.ToLower(line.Content), filter) {
			m.filtered = append(m.filtered, i)
		}
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.offset = 0
}

func (m model) View() string {
	if m.width == 0 || m.height == 0 {
		return "Loading..."
	}

	// Styles
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12"))

	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("236")).
		Foreground(lipgloss.Color("15")).
		Bold(true)

	lineNumStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241"))

	previewStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)

	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11"))

	// Build header (2 lines at top)
	var headerLines []string
	header := "r reload • q quit • j/k move • g/G first/last • ^d/u/f/b scroll • p preview • / filter"
	headerLines = append(headerLines, headerStyle.Render(header))
	headerLines = append(headerLines, fmt.Sprintf("Command: %s", m.config.Command))

	// Build prompt line (will go at bottom)
	var promptLine string
	if m.filterMode {
		promptLine = filterStyle.Render(fmt.Sprintf("/%s█", m.filter))
	} else if m.filter != "" {
		promptLine = promptStyle.Render(fmt.Sprintf("%s (filter: %s)", m.config.Prompt, m.filter))
	} else {
		promptLine = promptStyle.Render(m.config.Prompt)
	}
	if m.loading {
		promptLine += " [loading...]"
	}

	// Calculate layout
	listHeight := m.visibleLines()
	listWidth := m.width

	if m.showPreview && (m.config.PreviewPosition == PreviewLeft || m.config.PreviewPosition == PreviewRight) {
		listWidth = m.width / 2
	}

	// Build lines view
	var listLines []string
	for i := 0; i < listHeight; i++ {
		lineIdx := m.offset + i
		if lineIdx >= len(m.filtered) {
			// Empty line to fill space
			listLines = append(listLines, "")
			continue
		}

		idx := m.filtered[lineIdx]
		if idx >= len(m.lines) {
			listLines = append(listLines, "")
			continue
		}
		line := m.lines[idx]

		lineText := line.Content
		if m.config.ShowLineNums {
			lineNum := lineNumStyle.Render(fmt.Sprintf("%*d  ", m.config.LineNumWidth, line.Number))
			lineText = lineNum + line.Content
		}

		// Truncate if too long
		if len(lineText) > listWidth-2 && listWidth > 5 {
			lineText = lineText[:listWidth-5] + "..."
		}

		if lineIdx == m.cursor {
			// Pad to full width for selection highlight
			padding := listWidth - lipgloss.Width(lineText)
			if padding > 0 {
				lineText = lineText + strings.Repeat(" ", padding)
			}
			lineText = selectedStyle.Render(lineText)
		}

		listLines = append(listLines, lineText)
	}

	// Build preview content
	var previewContent string
	if m.showPreview && len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
		idx := m.filtered[m.cursor]
		if idx < len(m.lines) {
			previewContent = m.lines[idx].Content
		}
	}

	// Compose final view
	var contentSection string

	if !m.showPreview {
		contentSection = strings.Join(listLines, "\n")
	} else {
		previewH := m.height * m.config.PreviewHeight / 100
		previewW := m.width - 4

		if m.config.PreviewPosition == PreviewLeft || m.config.PreviewPosition == PreviewRight {
			previewW = m.width/2 - 4
		}

		styledPreview := previewStyle.
			Width(previewW).
			Height(previewH - 2).
			Render(previewContent)

		listContent := strings.Join(listLines, "\n")

		switch m.config.PreviewPosition {
		case PreviewTop:
			contentSection = styledPreview + "\n" + listContent
		case PreviewBottom:
			contentSection = listContent + "\n" + styledPreview
		case PreviewLeft:
			contentSection = lipgloss.JoinHorizontal(lipgloss.Top, styledPreview, " ", listContent)
		case PreviewRight:
			contentSection = lipgloss.JoinHorizontal(lipgloss.Top, listContent, " ", styledPreview)
		}
	}

	// Error message
	if m.errorMsg != "" {
		contentSection += "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: "+m.errorMsg)
	}

	// Combine: header at top, content in middle, prompt at bottom
	fullView := strings.Join(headerLines, "\n") + "\n" + contentSection + "\n" + promptLine

	return fullView
}

// Run starts the UI
func Run(cfg Config) error {
	if cfg.PreviewPosition == "" {
		cfg.PreviewPosition = PreviewBottom
	}

	m := initialModel(cfg)
	p := tea.NewProgram(m, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
