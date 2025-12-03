package ui

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
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
	Command              string
	Shell                string
	PreviewSize          int
	PreviewSizeIsPercent bool
	PreviewPosition      PreviewPosition
	ShowLineNums         bool
	LineNumWidth         int
	Prompt               string
	RefreshSeconds       int
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
	statusMsg   string // temporary status message (e.g., "Yanked!")
}

// messages
type linesMsg []runner.Line
type errMsg struct{ err error }
type tickMsg time.Time
type clearStatusMsg struct{}

func (e errMsg) Error() string { return e.err.Error() }

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

	case clearStatusMsg:
		m.statusMsg = ""
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
	case "y":
		// Yank (copy) selected line to clipboard
		if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
			idx := m.filtered[m.cursor]
			if idx < len(m.lines) {
				content := m.lines[idx].Content
				if err := copyToClipboard(content); err != nil {
					m.statusMsg = "Failed to copy"
				} else {
					m.statusMsg = "Copied to clipboard"
				}
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
					return clearStatusMsg{}
				})
			}
		}
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

func (m model) previewSize() int {
	if m.config.PreviewSizeIsPercent {
		if m.config.PreviewPosition == PreviewLeft || m.config.PreviewPosition == PreviewRight {
			return m.width * m.config.PreviewSize / 100
		}
		return m.height * m.config.PreviewSize / 100
	}
	return m.config.PreviewSize
}

func (m model) visibleLines() int {
	// Fixed lines: top border (1) + header (2) + separator (1) + bottom border (1) + prompt (1) = 6
	fixedLines := 6
	if m.showPreview && (m.config.PreviewPosition == PreviewTop || m.config.PreviewPosition == PreviewBottom) {
		// Add preview height + separator between content and preview
		return m.height - fixedLines - m.previewSize() - 1
	}
	return m.height - fixedLines
}

const ellipsis = "…"

// truncateToWidth truncates a string to fit within the given visual width,
// adding an ellipsis if truncation occurs. Uses visual width, not byte count.
func truncateToWidth(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	sw := lipgloss.Width(s)
	if sw <= maxWidth {
		return s
	}
	// Need to truncate - leave room for ellipsis (1 char wide)
	targetWidth := maxWidth - 1
	if targetWidth <= 0 {
		return ellipsis
	}

	// Truncate rune by rune until we fit
	result := ""
	currentWidth := 0
	for _, r := range s {
		runeWidth := lipgloss.Width(string(r))
		if currentWidth+runeWidth > targetWidth {
			break
		}
		result += string(r)
		currentWidth += runeWidth
	}
	return result + ellipsis
}

// wrapText wraps text to fit within the given width, returning multiple lines.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return nil
	}
	if s == "" {
		return []string{""}
	}

	var lines []string
	currentLine := ""
	currentWidth := 0

	for _, r := range s {
		runeWidth := lipgloss.Width(string(r))
		if currentWidth+runeWidth > width {
			// Start new line
			lines = append(lines, currentLine)
			currentLine = string(r)
			currentWidth = runeWidth
		} else {
			currentLine += string(r)
			currentWidth += runeWidth
		}
	}
	// Don't forget the last line
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
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

	// Box drawing characters (rounded)
	const (
		topLeft     = "╭"
		topRight    = "╮"
		bottomLeft  = "╰"
		bottomRight = "╯"
		horizontal  = "─"
		vertical    = "│"
		leftT       = "├"
		rightT      = "┤"
		topT        = "┬"
		bottomT     = "┴"
	)

	borderColor := lipgloss.Color("240")
	borderStyle := lipgloss.NewStyle().Foreground(borderColor)

	// Styles
	headerTextStyle := lipgloss.NewStyle().
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

	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11"))

	// Inner width (excluding border characters)
	innerWidth := m.width - 2

	// Helper to create a horizontal line (optionally with a T-junction for vertical split)
	hLine := func(left, right string, splitPos int) string {
		if splitPos > 0 && splitPos < innerWidth {
			return borderStyle.Render(left + strings.Repeat(horizontal, splitPos) + topT + strings.Repeat(horizontal, innerWidth-splitPos-1) + right)
		}
		return borderStyle.Render(left + strings.Repeat(horizontal, innerWidth) + right)
	}

	// Helper for header separator line with T junction pointing down (for vertical split below)
	hLineMid := func(left, right string, splitPos int) string {
		if splitPos > 0 && splitPos < innerWidth {
			return borderStyle.Render(left + strings.Repeat(horizontal, splitPos) + topT + strings.Repeat(horizontal, innerWidth-splitPos-1) + right)
		}
		return borderStyle.Render(left + strings.Repeat(horizontal, innerWidth) + right)
	}

	// Helper for bottom line with vertical split
	hLineBottom := func(left, right string, splitPos int) string {
		if splitPos > 0 && splitPos < innerWidth {
			return borderStyle.Render(left + strings.Repeat(horizontal, splitPos) + bottomT + strings.Repeat(horizontal, innerWidth-splitPos-1) + right)
		}
		return borderStyle.Render(left + strings.Repeat(horizontal, innerWidth) + right)
	}

	// Helper to pad content to inner width
	padLine := func(content string) string {
		contentWidth := lipgloss.Width(content)
		if contentWidth < innerWidth {
			content += strings.Repeat(" ", innerWidth-contentWidth)
		} else if contentWidth > innerWidth {
			// Use lipgloss style with MaxWidth for ANSI-safe truncation
			content = lipgloss.NewStyle().MaxWidth(innerWidth-1).Render(content) + ellipsis
		}
		return borderStyle.Render(vertical) + content + borderStyle.Render(vertical)
	}

	// Build header content
	header := headerTextStyle.Render("r reload • q quit • j/k move • g/G first/last • ^d/u/f/b scroll • p preview • / filter • y yank")
	commandLine := fmt.Sprintf("Command: %s", m.config.Command)

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
	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
		promptLine += " " + statusStyle.Render(m.statusMsg)
	}

	// Calculate layout
	listHeight := m.visibleLines()
	// listWidth is content area minus 1 for padding before border
	listWidth := innerWidth - 1

	if m.showPreview && (m.config.PreviewPosition == PreviewLeft || m.config.PreviewPosition == PreviewRight) {
		// For horizontal split: innerWidth = leftW + 1 (middle border) + rightW
		// List gets the non-preview side width, minus 1 for padding
		listWidth = innerWidth - m.previewSize() - 2
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

		var lineText string
		if m.config.ShowLineNums {
			// Calculate widths without ANSI codes first
			lineNumStr := fmt.Sprintf("%*d  ", m.config.LineNumWidth, line.Number)
			lineNumWidth := len(lineNumStr) // Plain ASCII, so len() == visual width
			contentWidth := listWidth - lineNumWidth

			// Truncate content (no ANSI codes yet)
			content := truncateToWidth(line.Content, contentWidth)

			// Now apply styling
			lineText = lineNumStyle.Render(lineNumStr) + content
		} else {
			// No line numbers, just truncate content
			lineText = truncateToWidth(line.Content, listWidth)
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

	// Error message
	if m.errorMsg != "" {
		listLines = append(listLines, lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render("Error: "+m.errorMsg))
	}

	// Calculate vertical split position for left/right preview
	// This must match where the middle vertical bar falls in content lines
	var vSplitPos int
	if m.showPreview {
		switch m.config.PreviewPosition {
		case PreviewLeft:
			vSplitPos = m.previewSize()
		case PreviewRight:
			// leftW = innerWidth - previewSize - 1, so split is at leftW
			vSplitPos = innerWidth - m.previewSize() - 1
		}
	}

	// Build the unified box
	var lines []string

	// Top border (no junction - vertical split starts at header separator)
	lines = append(lines, hLine(topLeft, topRight, 0))

	// Header lines
	lines = append(lines, padLine(header))
	lines = append(lines, padLine(commandLine))

	// Separator between header and content (T junction if vertical split)
	lines = append(lines, hLineMid(leftT, rightT, vSplitPos))

	// Content area (with optional preview)
	if !m.showPreview {
		// Just content lines, padded to fill height
		for i := 0; i < listHeight; i++ {
			if i < len(listLines) {
				lines = append(lines, padLine(listLines[i]))
			} else {
				lines = append(lines, padLine(""))
			}
		}
	} else {
		previewH := m.previewSize()

		switch m.config.PreviewPosition {
		case PreviewTop, PreviewBottom:
			// Vertical split - preview above or below content
			var previewLines []string
			// Wrap preview content to fit width
			if previewContent != "" {
				previewLines = wrapText(previewContent, innerWidth)
			}
			// Pad preview to height
			for len(previewLines) < previewH {
				previewLines = append(previewLines, "")
			}

			if m.config.PreviewPosition == PreviewTop {
				// Preview first
				for _, line := range previewLines[:previewH] {
					lines = append(lines, padLine(line))
				}
				// Separator (no vertical split for top/bottom preview)
				lines = append(lines, hLine(leftT, rightT, 0))
				// Then content, padded to fill height
				for i := 0; i < listHeight; i++ {
					if i < len(listLines) {
						lines = append(lines, padLine(listLines[i]))
					} else {
						lines = append(lines, padLine(""))
					}
				}
			} else {
				// Content first, padded to fill height
				for i := 0; i < listHeight; i++ {
					if i < len(listLines) {
						lines = append(lines, padLine(listLines[i]))
					} else {
						lines = append(lines, padLine(""))
					}
				}
				// Separator (no vertical split for top/bottom preview)
				lines = append(lines, hLine(leftT, rightT, 0))
				// Then preview
				for _, line := range previewLines[:previewH] {
					lines = append(lines, padLine(line))
				}
			}

		case PreviewLeft, PreviewRight:
			// Horizontal split: |leftContent|rightContent|
			// innerWidth = leftW + 1 (middle border) + rightW
			var leftW, rightW int
			if m.config.PreviewPosition == PreviewLeft {
				leftW = m.previewSize()
				rightW = innerWidth - leftW - 1
			} else {
				rightW = m.previewSize()
				leftW = innerWidth - rightW - 1
			}

			// Prepare preview lines (wrap text instead of truncating)
			var previewLines []string
			if previewContent != "" {
				// Determine preview width for wrapping
				previewW := leftW
				if m.config.PreviewPosition == PreviewRight {
					previewW = rightW
				}
				previewLines = wrapText(previewContent, previewW)
			}
			for len(previewLines) < listHeight {
				previewLines = append(previewLines, "")
			}

			// Helper to truncate/pad to width
			fitToWidth := func(s string, w int, isPreview bool) string {
				sw := lipgloss.Width(s)
				if sw > w {
					if isPreview {
						// Preview is already wrapped, just pad
						return s + strings.Repeat(" ", w-sw)
					}
					// List content may have ANSI codes, use lipgloss for safe truncation
					return lipgloss.NewStyle().MaxWidth(w-1).Render(s) + ellipsis
				}
				return s + strings.Repeat(" ", w-sw)
			}

			// Build combined lines
			for i := 0; i < listHeight; i++ {
				var leftContent, rightContent string
				var leftIsPreview, rightIsPreview bool

				if m.config.PreviewPosition == PreviewLeft {
					leftContent = previewLines[i]
					leftIsPreview = true
					if i < len(listLines) {
						rightContent = listLines[i]
					}
				} else {
					if i < len(listLines) {
						leftContent = listLines[i]
					}
					rightContent = previewLines[i]
					rightIsPreview = true
				}

				leftContent = fitToWidth(leftContent, leftW, leftIsPreview)
				rightContent = fitToWidth(rightContent, rightW, rightIsPreview)

				line := borderStyle.Render(vertical) + leftContent + borderStyle.Render(vertical) + rightContent + borderStyle.Render(vertical)
				lines = append(lines, line)
			}
		}
	}

	// Bottom border
	lines = append(lines, hLineBottom(bottomLeft, bottomRight, vSplitPos))

	// Combine box with prompt
	fullView := strings.Join(lines, "\n") + "\n" + promptLine

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
