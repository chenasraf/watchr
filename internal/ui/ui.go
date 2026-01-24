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
	RefreshInterval      time.Duration
	Interactive          bool
}

// model represents the application state
type model struct {
	config        Config
	lines         []runner.Line
	filtered      []int // indices into lines that match filter
	cursor        int   // cursor position in filtered list
	offset        int   // scroll offset for visible window
	filter        string
	filterMode    bool
	showPreview   bool
	showHelp      bool // help overlay visible
	width         int
	height        int
	runner        *runner.Runner
	ctx           context.Context
	cancel        context.CancelFunc
	loading       bool
	streaming     bool                    // true while command is running (streaming output)
	streamResult  *runner.StreamingResult // current streaming result
	lastLineCount int                     // track line count for updates
	userScrolled  bool                    // true if user manually scrolled during streaming
	spinnerFrame  int                     // current spinner animation frame
	errorMsg      string
	statusMsg     string // temporary status message (e.g., "Yanked!")
	exitCode      int    // last command exit code
}

// messages
type resultMsg struct {
	lines    []runner.Line
	exitCode int
}
type errMsg struct{ err error }
type tickMsg time.Time
type clearStatusMsg struct{}
type spinnerTickMsg time.Time
type streamTickMsg time.Time // periodic check for streaming updates
type startStreamMsg struct{} // trigger to start streaming

// Spinner frames for the loading animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

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

	var r *runner.Runner
	if cfg.Interactive {
		r = runner.NewInteractiveRunner(cfg.Shell, cfg.Command)
	} else {
		r = runner.NewRunner(cfg.Shell, cfg.Command)
	}

	return model{
		config:      cfg,
		lines:       []runner.Line{},
		filtered:    []int{},
		cursor:      0,
		offset:      0,
		filter:      "",
		filterMode:  false,
		showPreview: false,
		runner:      r,
		ctx:         ctx,
		cancel:      cancel,
		loading:     true,
	}
}

func (m *model) Init() tea.Cmd {
	// Send a message to start streaming (handled in Update with pointer receiver)
	return func() tea.Msg {
		return startStreamMsg{}
	}
}

func (m model) spinnerTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return spinnerTickMsg(t)
	})
}

func (m model) streamTickCmd() tea.Cmd {
	return tea.Tick(50*time.Millisecond, func(t time.Time) tea.Msg {
		return streamTickMsg(t)
	})
}

func (m *model) startStreaming() tea.Cmd {
	// Cancel any existing context and create a new one
	if m.cancel != nil {
		m.cancel()
	}
	m.ctx, m.cancel = context.WithCancel(context.Background())

	// Pass previous lines for in-place updates
	m.streamResult = m.runner.RunStreaming(m.ctx, m.lines)
	m.streaming = true
	m.loading = true
	m.lastLineCount = len(m.lines)
	m.exitCode = -1
	m.errorMsg = ""
	m.userScrolled = false

	return m.streamTickCmd()
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyPress(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case startStreamMsg:
		cmd := m.startStreaming()
		return m, tea.Batch(cmd, m.spinnerTickCmd())

	case resultMsg:
		m.lines = msg.lines
		m.exitCode = msg.exitCode
		m.loading = false
		m.streaming = false
		m.updateFiltered()
		return m, nil

	case streamTickMsg:
		if m.streamResult == nil {
			return m, nil
		}

		// Check for new lines
		newLines := m.streamResult.GetLines()
		newCount := len(newLines)

		if newCount != m.lastLineCount {
			m.lines = newLines
			m.lastLineCount = newCount
			m.updateFiltered()

			// Auto-scroll to bottom if user hasn't manually scrolled
			if !m.userScrolled {
				visible := m.visibleLines()
				if visible > 0 {
					m.cursor = max(len(m.filtered)-1, 0)
					m.offset = max(len(m.filtered)-visible, 0)
				}
			}
		}

		// Check if command completed
		if m.streamResult.IsDone() {
			m.streaming = false
			m.loading = false
			m.exitCode = m.streamResult.ExitCode
			if m.streamResult.Error != nil {
				m.errorMsg = m.streamResult.Error.Error()
			}

			// Trim excess lines from previous run
			currentCount := m.streamResult.GetCurrentLineCount()
			if currentCount < len(m.lines) {
				m.lines = m.lines[:currentCount]
				m.updateFiltered()
			}

			// If auto-refresh is enabled, schedule the next run
			if m.config.RefreshInterval > 0 {
				return m, m.tickCmd()
			}
			return m, nil
		}

		// Continue streaming
		return m, m.streamTickCmd()

	case tickMsg:
		if m.config.RefreshInterval > 0 && !m.streaming {
			// Restart streaming for refresh
			cmd := m.startStreaming()
			return m, tea.Batch(cmd, m.spinnerTickCmd())
		}
		return m, nil

	case errMsg:
		m.errorMsg = msg.Error()
		m.loading = false
		m.streaming = false
		return m, nil

	case clearStatusMsg:
		m.statusMsg = ""
		return m, nil

	case spinnerTickMsg:
		if m.loading || m.streaming {
			m.spinnerFrame = (m.spinnerFrame + 1) % len(spinnerFrames)
			return m, m.spinnerTickCmd()
		}
		return m, nil
	}

	return m, nil
}

func (m model) tickCmd() tea.Cmd {
	return tea.Tick(m.config.RefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *model) handleKeyPress(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// In help mode, any key closes it
	if m.showHelp {
		switch msg.String() {
		case "?", "esc", "q", "enter":
			m.showHelp = false
		}
		return m, nil
	}

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
	case "q", "ctrl+c":
		m.cancel()
		return m, tea.Quit
	case "esc":
		// Clear filter if active, otherwise quit
		if m.filter != "" {
			m.filter = ""
			m.updateFiltered()
			return m, nil
		}
		m.cancel()
		return m, tea.Quit

	case "j", "down", "ctrl+n":
		m.userScrolled = true
		m.moveCursor(1)
	case "k", "up", "ctrl+p":
		m.userScrolled = true
		m.moveCursor(-1)
	case "g", "home":
		m.userScrolled = true
		m.cursor = 0
		m.offset = 0
	case "G", "end":
		m.userScrolled = false // Resume following output
		if len(m.filtered) > 0 {
			m.cursor = len(m.filtered) - 1
			m.adjustOffset()
		}
	case "ctrl+d":
		m.userScrolled = true
		m.moveCursor(m.visibleLines() / 2)
	case "ctrl+u":
		m.userScrolled = true
		m.moveCursor(-m.visibleLines() / 2)
	case "pgdown", "ctrl+f":
		m.userScrolled = true
		m.moveCursor(m.visibleLines())
	case "pgup", "ctrl+b":
		m.userScrolled = true
		m.moveCursor(-m.visibleLines())
	case "p":
		m.showPreview = !m.showPreview
		m.adjustOffset() // Keep selected line visible after preview toggle
	case "r", "ctrl+r":
		// Restart streaming
		cmd := m.startStreaming()
		return m, tea.Batch(cmd, m.spinnerTickCmd())
	case "/":
		m.filterMode = true
		m.filter = ""
	case "?":
		m.showHelp = true
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
	idealOffset = max(idealOffset, 0)
	maxOffset := max(len(m.filtered)-visible, 0)
	idealOffset = min(idealOffset, maxOffset)

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
	// Fixed lines: top border (1) + header (1) + separator (1) + bottom border (1) + prompt (1) = 5
	fixedLines := 5
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
	var result strings.Builder
	currentWidth := 0
	for _, r := range s {
		runeWidth := lipgloss.Width(string(r))
		if currentWidth+runeWidth > targetWidth {
			break
		}
		result.WriteRune(r)
		currentWidth += runeWidth
	}
	return result.String() + ellipsis
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

	// Clamp offset to valid bounds instead of resetting to 0
	// This preserves scroll position during streaming updates
	visible := m.visibleLines()
	if visible > 0 {
		maxOffset := max(len(m.filtered)-visible, 0)
		if m.offset > maxOffset {
			m.offset = maxOffset
		}
	}
}

// renderHelpOverlay creates the help box content (without positioning)
func (m model) renderHelpOverlay() (box string, boxWidth, boxHeight int) {
	keyStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")) // green

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // light gray

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")) // blue

	// Define keybindings
	bindings := []struct {
		key  string
		desc string
	}{
		{"j / k", "Move down / up"},
		{"g / G", "Go to first / last line"},
		{"Ctrl+d / Ctrl+u", "Half page down / up"},
		{"PgDn / PgUp", "Full page down / up"},
		{"Ctrl+f / Ctrl+b", "Full page down / up"},
		{"", ""},
		{"p", "Toggle preview pane"},
		{"/", "Enter filter mode"},
		{"Esc", "Exit filter / clear"},
		{"", ""},
		{"r / Ctrl+r", "Reload command"},
		{"y", "Copy line to clipboard"},
		{"q / Esc", "Quit"},
		{"?", "Toggle this help"},
	}

	// Build content
	var content strings.Builder
	content.WriteString(titleStyle.Render("Keybindings"))
	content.WriteString("\n\n")

	for _, b := range bindings {
		if b.key == "" {
			content.WriteString("\n")
			continue
		}
		key := keyStyle.Render(fmt.Sprintf("%-18s", b.key))
		desc := descStyle.Render(b.desc)
		content.WriteString(fmt.Sprintf("  %s  %s\n", key, desc))
	}

	content.WriteString("\n")
	content.WriteString(descStyle.Render("Press any key to close"))

	// Create box style
	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("12")).
		Padding(1, 2)

	box = boxStyle.Render(content.String())
	boxWidth = lipgloss.Width(box)
	boxHeight = lipgloss.Height(box)

	return box, boxWidth, boxHeight
}

// splitAtVisualWidth splits a string at a visual width position, handling ANSI codes
// Returns (left part, right part) where left has exactly targetWidth visual width
func splitAtVisualWidth(s string, targetWidth int) (string, string) {
	var left, right strings.Builder
	visualWidth := 0
	inEscape := false
	runes := []rune(s)

	i := 0
	// Build left part up to targetWidth
	for i < len(runes) && visualWidth < targetWidth {
		r := runes[i]

		if r == '\x1b' {
			// Start of ANSI escape sequence - include it in left part
			left.WriteRune(r)
			i++
			for i < len(runes) && !isAnsiTerminator(runes[i]) {
				left.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				left.WriteRune(runes[i]) // terminator
				i++
			}
			continue
		}

		runeWidth := lipgloss.Width(string(r))
		if visualWidth+runeWidth <= targetWidth {
			left.WriteRune(r)
			visualWidth += runeWidth
			i++
		} else {
			break
		}
	}

	// Pad left if needed
	for visualWidth < targetWidth {
		left.WriteRune(' ')
		visualWidth++
	}

	// Skip runes in the "overlay zone" - we don't need them for right part calculation
	// The caller will handle inserting the overlay content

	// Build right part from remaining
	for ; i < len(runes); i++ {
		r := runes[i]
		if r == '\x1b' {
			right.WriteRune(r)
			i++
			for i < len(runes) && !isAnsiTerminator(runes[i]) {
				right.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				right.WriteRune(runes[i])
			}
			continue
		}
		right.WriteRune(r)
	}

	_ = inEscape // unused but kept for clarity
	return left.String(), right.String()
}

// skipVisualWidth skips a number of visual width units in a string, handling ANSI codes
// It preserves and returns ANSI sequences encountered during skipping so styling can be restored
func skipVisualWidth(s string, skipWidth int) string {
	var result strings.Builder
	var ansiState strings.Builder // collect ANSI codes while skipping
	visualWidth := 0
	runes := []rune(s)

	i := 0
	// Skip until we've passed skipWidth, but collect ANSI codes
	for i < len(runes) && visualWidth < skipWidth {
		r := runes[i]

		if r == '\x1b' {
			// ANSI escape - collect it (don't count visual width)
			ansiState.WriteRune(r)
			i++
			for i < len(runes) && !isAnsiTerminator(runes[i]) {
				ansiState.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				ansiState.WriteRune(runes[i]) // terminator
				i++
			}
			continue
		}

		runeWidth := lipgloss.Width(string(r))
		visualWidth += runeWidth
		i++
	}

	// Prepend collected ANSI state to restore styling
	result.WriteString(ansiState.String())

	// Output the rest
	for ; i < len(runes); i++ {
		result.WriteRune(runes[i])
	}

	return result.String()
}

func isAnsiTerminator(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// overlayBox composites an overlay box on top of a base view
func overlayBox(base string, box string, boxWidth, boxHeight, screenWidth, screenHeight int) string {
	// ANSI reset sequence to stop any styling from bleeding into overlay
	const ansiReset = "\x1b[0m"

	// Split base into lines
	baseLines := strings.Split(base, "\n")

	// Ensure we have enough lines
	for len(baseLines) < screenHeight {
		baseLines = append(baseLines, "")
	}

	// Split box into lines
	boxLines := strings.Split(box, "\n")

	// Calculate center position
	startX := (screenWidth - boxWidth) / 2
	startY := (screenHeight - boxHeight) / 2

	if startX < 0 {
		startX = 0
	}
	if startY < 0 {
		startY = 0
	}

	// Overlay box onto base
	for i, boxLine := range boxLines {
		y := startY + i
		if y >= len(baseLines) {
			break
		}

		baseLine := baseLines[y]
		baseVisualWidth := lipgloss.Width(baseLine)

		// Get left part (before overlay)
		leftPart, _ := splitAtVisualWidth(baseLine, startX)

		// Get right part (after overlay)
		endX := startX + boxWidth
		var rightPart string
		if endX < baseVisualWidth {
			rightPart = skipVisualWidth(baseLine, endX)
		}

		// Combine: left + reset + box + right
		// Reset before overlay to stop highlight bleeding into overlay
		baseLines[y] = leftPart + ansiReset + boxLine + rightPart
	}

	return strings.Join(baseLines, "\n")
}

func (m *model) View() string {
	if m.width == 0 || m.height == 0 {
		return spinnerFrames[m.spinnerFrame] + " Running command…"
	}

	// Render the main UI
	mainView := m.renderMainView()

	// Overlay help if active
	if m.showHelp {
		box, boxWidth, boxHeight := m.renderHelpOverlay()
		return overlayBox(mainView, box, boxWidth, boxHeight, m.width, m.height)
	}

	return mainView
}

func (m model) renderMainView() string {
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
	promptStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("14"))

	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("15")).
		Foreground(lipgloss.Color("#000000")).
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

	// Build header content with status indicator
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true) // blue
	prefix := titleStyle.Render("watchr") + " • "

	var commandLine string
	switch {
	case m.streaming:
		// Streaming - show streaming indicator
		streamStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14")) // cyan
		commandLine = prefix + streamStyle.Render("◉ "+m.config.Command)
	case m.loading:
		// Still loading - no status yet
		commandLine = prefix + m.config.Command
	case m.exitCode == 0:
		// Success - green checkmark and green command
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
		commandLine = prefix + successStyle.Render("✓ "+m.config.Command)
	default:
		// Failure - red cross with exit code and red command
		failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9")) // red
		commandLine = prefix + failStyle.Render(fmt.Sprintf("✗ [%d] %s", m.exitCode, m.config.Command))
	}

	// Build prompt line (will go at bottom)
	var promptLine string
	switch {
	case m.filterMode:
		promptLine = filterStyle.Render(fmt.Sprintf("/%s█", m.filter))
	case m.filter != "":
		promptLine = promptStyle.Render(fmt.Sprintf("%s (filter: %s)", m.config.Prompt, m.filter))
	default:
		promptLine = promptStyle.Render(m.config.Prompt)
	}
	if m.streaming {
		promptLine += " " + spinnerFrames[m.spinnerFrame] + " Streaming…"
	} else if m.loading {
		promptLine += " " + spinnerFrames[m.spinnerFrame] + " Running command…"
	}
	if m.statusMsg != "" {
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10")) // green
		promptLine += " " + statusStyle.Render(m.statusMsg)
	}

	// Add help hint on the right
	helpHint := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("? for help")
	promptWidth := lipgloss.Width(promptLine)
	hintWidth := lipgloss.Width(helpHint)
	gap := m.width - promptWidth - hintWidth
	if gap > 0 {
		promptLine += strings.Repeat(" ", gap) + helpHint
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
	for i := range listHeight {
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
		isSelected := lineIdx == m.cursor

		// Full width including the padding space before border
		fullWidth := listWidth + 1

		if m.config.ShowLineNums {
			// Calculate widths without ANSI codes first
			lineNumStr := fmt.Sprintf("%*d  ", m.config.LineNumWidth, line.Number)
			lineNumWidth := len(lineNumStr) // Plain ASCII, so len() == visual width
			contentWidth := listWidth - lineNumWidth

			// Truncate content (no ANSI codes yet)
			content := truncateToWidth(line.Content, contentWidth)

			if isSelected {
				// For selected line: gray line number + black content, both on white background
				selectedLineNumStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("15")).
					Foreground(lipgloss.Color("241"))
				selectedContentStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("15")).
					Foreground(lipgloss.Color("#000000")).
					Bold(true)

				// Pad content to fill remaining width
				contentPadded := content
				padding := fullWidth - lineNumWidth - lipgloss.Width(content)
				if padding > 0 {
					contentPadded = content + strings.Repeat(" ", padding)
				}
				lineText = selectedLineNumStyle.Render(lineNumStr) + selectedContentStyle.Render(contentPadded)
			} else {
				// Normal line - style line numbers differently
				lineText = lineNumStyle.Render(lineNumStr) + content
			}
		} else {
			// No line numbers, just truncate content
			lineText = truncateToWidth(line.Content, listWidth)

			if isSelected {
				// Pad to full width for selection highlight
				padding := fullWidth - lipgloss.Width(lineText)
				if padding > 0 {
					lineText += strings.Repeat(" ", padding)
				}
				lineText = selectedStyle.Render(lineText)
			}
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

	// Header line (command only)
	lines = append(lines, padLine(commandLine))

	// Separator between header and content (T junction if vertical split)
	lines = append(lines, hLineMid(leftT, rightT, vSplitPos))

	// Content area (with optional preview)
	if !m.showPreview {
		// Just content lines, padded to fill height
		for i := range listHeight {
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
				for i := range listHeight {
					if i < len(listLines) {
						lines = append(lines, padLine(listLines[i]))
					} else {
						lines = append(lines, padLine(""))
					}
				}
			} else {
				// Content first, padded to fill height
				for i := range listHeight {
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
			for i := range listHeight {
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
	p := tea.NewProgram(&m, tea.WithAltScreen())

	_, err := p.Run()
	return err
}
