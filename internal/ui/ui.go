package ui

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
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
	RefreshFromStart     bool // If true, refresh timer starts when command starts; if false, when command ends (default)
	Interactive          bool
}

// model represents the application state
type model struct {
	config            Config
	lines             []runner.Line
	filtered          []int // indices into lines that match filter
	cursor            int   // cursor position in filtered list
	offset            int   // scroll offset for visible window
	filter            string
	filterCursor      int // cursor position within filter string
	filterMode        bool
	filterRegex       bool  // true when filter is in regex mode
	filterRegexErr    error // non-nil when regex pattern is invalid
	showPreview       bool
	previewOffset     int  // scroll offset for preview pane
	showHelp          bool // help overlay visible
	width             int
	height            int
	runner            *runner.Runner
	ctx               context.Context
	cancel            context.CancelFunc
	loading           bool
	streaming         bool                    // true while command is running (streaming output)
	streamResult      *runner.StreamingResult // current streaming result
	lastLineCount     int                     // track line count for updates
	userScrolled      bool                    // true if user manually scrolled during streaming
	refreshGeneration int                     // incremented on manual refresh to reset timer
	refreshStartTime  time.Time               // when the refresh timer was started
	spinnerFrame      int                     // current spinner animation frame
	errorMsg          string
	statusMsg         string // temporary status message (e.g., "Yanked!")
	exitCode          int    // last command exit code

	cmdPaletteMode     bool   // whether command palette is open
	cmdPaletteFilter   string // current filter text
	cmdPaletteCursor   int    // cursor position within filter string
	cmdPaletteSelected int    // selected item index in filtered list
}

// messages
type resultMsg struct {
	lines    []runner.Line
	exitCode int
}
type errMsg struct{ err error }
type tickMsg struct {
	generation int
}
type clearStatusMsg struct{}
type spinnerTickMsg time.Time
type streamTickMsg time.Time   // periodic check for streaming updates
type startStreamMsg struct{}   // trigger to start streaming
type countdownTickMsg struct { // periodic update for refresh countdown display
	generation int
}

// Spinner frames for the loading animation
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (e errMsg) Error() string { return e.err.Error() }

// command represents a command palette entry
type command struct {
	name     string // display name
	shortcut string // keybinding hint
	action   func(m *model) (tea.Model, tea.Cmd)
}

// commands returns the list of available command palette entries.
func commands() []command {
	return []command{
		{"Reload command", "r / Ctrl+r", func(m *model) (tea.Model, tea.Cmd) {
			m.refreshGeneration++
			cmd := m.startStreaming()
			return m, tea.Batch(cmd, m.spinnerTickCmd())
		}},
		{"Stop running command", "c", func(m *model) (tea.Model, tea.Cmd) {
			if m.streaming {
				m.cancel()
				m.statusMsg = "Command stopped"
				return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
			}
			return m, nil
		}},
		{"Toggle preview pane", "p", func(m *model) (tea.Model, tea.Cmd) {
			m.showPreview = !m.showPreview
			m.adjustOffset()
			return m, nil
		}},
		{"Increase preview size", "+", func(m *model) (tea.Model, tea.Cmd) {
			if m.showPreview {
				m.config.PreviewSize += previewSizeStep(m.config.PreviewSizeIsPercent)
				m.adjustOffset()
			}
			return m, nil
		}},
		{"Decrease preview size", "-", func(m *model) (tea.Model, tea.Cmd) {
			if m.showPreview {
				step := previewSizeStep(m.config.PreviewSizeIsPercent)
				if m.config.PreviewSize > step {
					m.config.PreviewSize -= step
					m.adjustOffset()
				}
			}
			return m, nil
		}},
		{"Go to first line", "g", func(m *model) (tea.Model, tea.Cmd) {
			m.userScrolled = true
			m.previewOffset = 0
			m.cursor = 0
			m.offset = 0
			return m, nil
		}},
		{"Go to last line", "G", func(m *model) (tea.Model, tea.Cmd) {
			m.userScrolled = false
			m.previewOffset = 0
			if len(m.filtered) > 0 {
				m.cursor = len(m.filtered) - 1
				m.adjustOffset()
			}
			return m, nil
		}},
		{"Enter filter mode", "/", func(m *model) (tea.Model, tea.Cmd) {
			m.filterMode = true
			m.filterCursor = len(m.filter)
			return m, nil
		}},
		{"Toggle regex filter", "//", func(m *model) (tea.Model, tea.Cmd) {
			m.filterMode = true
			m.filterRegex = !m.filterRegex
			m.filterRegexErr = nil
			m.filterCursor = len(m.filter)
			m.updateFiltered()
			return m, nil
		}},
		{"Copy line to clipboard", "y", func(m *model) (tea.Model, tea.Cmd) {
			if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
				idx := m.filtered[m.cursor]
				if idx < len(m.lines) {
					content := m.lines[idx].Content
					if err := copyToClipboard(content); err != nil {
						m.statusMsg = "Failed to copy"
					} else {
						m.statusMsg = "Copied to clipboard"
					}
					return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
				}
			}
			return m, nil
		}},
		{"Copy line (plain text)", "Y", func(m *model) (tea.Model, tea.Cmd) {
			if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
				idx := m.filtered[m.cursor]
				if idx < len(m.lines) {
					content := stripANSI(m.lines[idx].Content)
					if err := copyToClipboard(content); err != nil {
						m.statusMsg = "Failed to copy"
					} else {
						m.statusMsg = "Copied to clipboard (plain)"
					}
					return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg { return clearStatusMsg{} })
				}
			}
			return m, nil
		}},
		{"Show help", "?", func(m *model) (tea.Model, tea.Cmd) {
			m.showHelp = true
			return m, nil
		}},
		{"Quit", "q", func(m *model) (tea.Model, tea.Cmd) {
			m.cancel()
			return m, tea.Quit
		}},
	}
}

// filteredCommands returns commands matching the current palette filter.
func (m *model) filteredCommands() []command {
	all := commands()
	if m.cmdPaletteFilter == "" {
		return all
	}
	filter := strings.ToLower(m.cmdPaletteFilter)
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

func (m model) countdownTickCmd() tea.Cmd {
	gen := m.refreshGeneration
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return countdownTickMsg{generation: gen}
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

	cmds := []tea.Cmd{m.streamTickCmd()}

	// Start refresh timer from command start if configured
	if m.config.RefreshFromStart && m.config.RefreshInterval > 0 {
		m.refreshStartTime = time.Now()
		cmds = append(cmds, m.tickCmd())
		if m.config.RefreshInterval > time.Second {
			cmds = append(cmds, m.countdownTickCmd())
		}
	}

	return tea.Batch(cmds...)
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

			// If auto-refresh is enabled and timer starts from end, schedule the next run
			if m.config.RefreshInterval > 0 && !m.config.RefreshFromStart {
				m.refreshStartTime = time.Now()
				cmds := []tea.Cmd{m.tickCmd()}
				// Start countdown display updates if interval > 1s
				if m.config.RefreshInterval > time.Second {
					cmds = append(cmds, m.countdownTickCmd())
				}
				return m, tea.Batch(cmds...)
			}
			return m, nil
		}

		// Continue streaming
		return m, m.streamTickCmd()

	case tickMsg:
		// Ignore ticks from before a manual refresh
		if msg.generation != m.refreshGeneration {
			return m, nil
		}
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

	case countdownTickMsg:
		// Ignore ticks from before a manual refresh
		if msg.generation != m.refreshGeneration {
			return m, nil
		}
		// Continue ticking if waiting for auto-refresh
		if m.config.RefreshInterval > time.Second && !m.streaming && !m.refreshStartTime.IsZero() {
			elapsed := time.Since(m.refreshStartTime)
			if elapsed < m.config.RefreshInterval {
				return m, m.countdownTickCmd()
			}
		}
		return m, nil
	}

	return m, nil
}

func (m model) tickCmd() tea.Cmd {
	gen := m.refreshGeneration
	return tea.Tick(m.config.RefreshInterval, func(t time.Time) tea.Msg {
		return tickMsg{generation: gen}
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

	// In command palette mode
	if m.cmdPaletteMode {
		switch msg.Type {
		case tea.KeyEsc:
			m.cmdPaletteMode = false
			m.cmdPaletteFilter = ""
			m.cmdPaletteCursor = 0
			m.cmdPaletteSelected = 0
			return m, nil
		case tea.KeyEnter:
			filtered := m.filteredCommands()
			if len(filtered) > 0 && m.cmdPaletteSelected < len(filtered) {
				m.cmdPaletteMode = false
				cmd := filtered[m.cmdPaletteSelected]
				m.cmdPaletteFilter = ""
				m.cmdPaletteCursor = 0
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
		case tea.KeyLeft:
			if msg.Alt {
				pos := m.cmdPaletteCursor
				for pos > 0 && m.cmdPaletteFilter[pos-1] == ' ' {
					pos--
				}
				for pos > 0 && m.cmdPaletteFilter[pos-1] != ' ' {
					pos--
				}
				m.cmdPaletteCursor = pos
			} else if m.cmdPaletteCursor > 0 {
				m.cmdPaletteCursor--
			}
			return m, nil
		case tea.KeyRight:
			if msg.Alt {
				pos := m.cmdPaletteCursor
				for pos < len(m.cmdPaletteFilter) && m.cmdPaletteFilter[pos] != ' ' {
					pos++
				}
				for pos < len(m.cmdPaletteFilter) && m.cmdPaletteFilter[pos] == ' ' {
					pos++
				}
				m.cmdPaletteCursor = pos
			} else if m.cmdPaletteCursor < len(m.cmdPaletteFilter) {
				m.cmdPaletteCursor++
			}
			return m, nil
		case tea.KeyBackspace:
			if msg.Alt {
				if m.cmdPaletteCursor > 0 {
					pos := m.cmdPaletteCursor
					for pos > 0 && m.cmdPaletteFilter[pos-1] == ' ' {
						pos--
					}
					for pos > 0 && m.cmdPaletteFilter[pos-1] != ' ' {
						pos--
					}
					m.cmdPaletteFilter = m.cmdPaletteFilter[:pos] + m.cmdPaletteFilter[m.cmdPaletteCursor:]
					m.cmdPaletteCursor = pos
				}
			} else if m.cmdPaletteCursor > 0 {
				m.cmdPaletteFilter = m.cmdPaletteFilter[:m.cmdPaletteCursor-1] + m.cmdPaletteFilter[m.cmdPaletteCursor:]
				m.cmdPaletteCursor--
			}
			m.cmdPaletteSelected = 0
			return m, nil
		default:
			if len(msg.Runes) > 0 {
				s := string(msg.Runes)
				m.cmdPaletteFilter = m.cmdPaletteFilter[:m.cmdPaletteCursor] + s + m.cmdPaletteFilter[m.cmdPaletteCursor:]
				m.cmdPaletteCursor += len(s)
				m.cmdPaletteSelected = 0
			}
			return m, nil
		}
	}

	// In filter mode, handle text input
	if m.filterMode {
		switch msg.Type {
		case tea.KeyEsc:
			m.filterMode = false
			m.filter = ""
			m.filterCursor = 0
			m.filterRegex = false
			m.filterRegexErr = nil
			m.updateFiltered()
			return m, nil
		case tea.KeyEnter:
			m.filterMode = false
			return m, nil
		case tea.KeyLeft:
			if msg.Alt {
				// Alt+Left: move to previous word boundary
				pos := m.filterCursor
				for pos > 0 && m.filter[pos-1] == ' ' {
					pos--
				}
				for pos > 0 && m.filter[pos-1] != ' ' {
					pos--
				}
				m.filterCursor = pos
			} else if m.filterCursor > 0 {
				m.filterCursor--
			}
			return m, nil
		case tea.KeyRight:
			if msg.Alt {
				// Alt+Right: move to next word boundary
				pos := m.filterCursor
				for pos < len(m.filter) && m.filter[pos] != ' ' {
					pos++
				}
				for pos < len(m.filter) && m.filter[pos] == ' ' {
					pos++
				}
				m.filterCursor = pos
			} else if m.filterCursor < len(m.filter) {
				m.filterCursor++
			}
			return m, nil
		case tea.KeyBackspace:
			if msg.Alt {
				// Alt+Backspace: delete word behind cursor
				if m.filterCursor > 0 {
					pos := m.filterCursor
					// Skip trailing spaces
					for pos > 0 && m.filter[pos-1] == ' ' {
						pos--
					}
					// Skip word characters
					for pos > 0 && m.filter[pos-1] != ' ' {
						pos--
					}
					m.filter = m.filter[:pos] + m.filter[m.filterCursor:]
					m.filterCursor = pos
					m.updateFiltered()
				}
			} else if m.filterCursor > 0 {
				m.filter = m.filter[:m.filterCursor-1] + m.filter[m.filterCursor:]
				m.filterCursor--
				m.updateFiltered()
			}
			return m, nil
		default:
			if len(msg.Runes) > 0 {
				s := string(msg.Runes)
				if s == "/" && m.filter == "" {
					m.filterRegex = !m.filterRegex
					m.filterRegexErr = nil
					m.updateFiltered()
				} else {
					m.filter = m.filter[:m.filterCursor] + s + m.filter[m.filterCursor:]
					m.filterCursor += len(s)
					m.updateFiltered()
				}
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
		if m.filter != "" || m.filterRegex {
			m.filter = ""
			m.filterCursor = 0
			m.filterRegex = false
			m.filterRegexErr = nil
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
		m.previewOffset = 0
		m.cursor = 0
		m.offset = 0
	case "G", "end":
		m.userScrolled = false // Resume following output
		m.previewOffset = 0
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
		m.showPreview = !m.showPreview
		m.adjustOffset() // Keep selected line visible after preview toggle
	case "+", "=":
		if m.showPreview {
			m.config.PreviewSize += previewSizeStep(m.config.PreviewSizeIsPercent)
			m.adjustOffset()
		}
	case "-":
		if m.showPreview {
			step := previewSizeStep(m.config.PreviewSizeIsPercent)
			if m.config.PreviewSize > step {
				m.config.PreviewSize -= step
				m.adjustOffset()
			}
		}
	case "r", "ctrl+r":
		// Restart streaming and reset auto-refresh timer
		m.refreshGeneration++
		cmd := m.startStreaming()
		return m, tea.Batch(cmd, m.spinnerTickCmd())
	case "c":
		// Stop the running command if one is running
		if m.streaming {
			m.cancel()
			m.statusMsg = "Command stopped"
			return m, tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
				return clearStatusMsg{}
			})
		}
	case "/":
		m.filterMode = true
		m.filterCursor = len(m.filter)
	case ":":
		m.cmdPaletteMode = true
		m.cmdPaletteFilter = ""
		m.cmdPaletteCursor = 0
		m.cmdPaletteSelected = 0
	case "?":
		m.showHelp = true
	case "y":
		// Yank (copy) selected line to clipboard (with ANSI codes)
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
	case "Y":
		// Yank (copy) selected line to clipboard, stripping ANSI codes
		if len(m.filtered) > 0 && m.cursor >= 0 && m.cursor < len(m.filtered) {
			idx := m.filtered[m.cursor]
			if idx < len(m.lines) {
				content := stripANSI(m.lines[idx].Content)
				if err := copyToClipboard(content); err != nil {
					m.statusMsg = "Failed to copy"
				} else {
					m.statusMsg = "Copied to clipboard (plain)"
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
	m.previewOffset = 0
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

func previewSizeStep(isPercent bool) int {
	if isPercent {
		return 5
	}
	return 2
}

// clampPreviewOffset computes the actual preview content size and clamps
// previewOffset so it can't exceed the scrollable range.
func (m *model) clampPreviewOffset() {
	if !m.showPreview || m.cursor < 0 || m.cursor >= len(m.filtered) {
		m.previewOffset = 0
		return
	}
	idx := m.filtered[m.cursor]
	if idx >= len(m.lines) {
		m.previewOffset = 0
		return
	}

	content := highlightJSON(m.lines[idx].Content)
	innerWidth := m.width - 2

	var previewW, visibleH int
	switch m.config.PreviewPosition {
	case PreviewTop, PreviewBottom:
		previewW = innerWidth
		visibleH = m.previewSize()
	case PreviewLeft:
		previewW = m.previewSize()
		visibleH = m.visibleLines()
	case PreviewRight:
		previewW = m.previewSize()
		visibleH = m.visibleLines()
	}

	previewLines := wrapPreviewContent(content, previewW)
	maxOffset := max(len(previewLines)-visibleH, 0)
	if m.previewOffset > maxOffset {
		m.previewOffset = maxOffset
	}
}

// applyPreviewOffset slices previewLines based on the current preview scroll
// offset, clamping the offset so it doesn't scroll past the content.
func (m *model) applyPreviewOffset(previewLines []string, visibleH int) []string {
	maxOffset := max(len(previewLines)-visibleH, 0)
	if m.previewOffset > maxOffset {
		m.previewOffset = maxOffset
	}
	if m.previewOffset > 0 {
		previewLines = previewLines[m.previewOffset:]
	}
	return previewLines
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
// It is ANSI-aware: escape sequences are preserved intact and don't count
// toward the visible width. When a line wraps, any active ANSI state is
// carried over so colours continue on the next line.
func wrapText(s string, width int) []string {
	if width <= 0 {
		return nil
	}
	if s == "" {
		return []string{""}
	}

	var lines []string
	var currentLine strings.Builder
	currentWidth := 0
	// Track the last seen ANSI escape so we can re-apply it after a wrap
	var activeANSI string

	i := 0
	runes := []rune(s)
	for i < len(runes) {
		// Check for ANSI escape sequence: ESC [ ... final_byte
		if runes[i] == '\033' && i+1 < len(runes) && runes[i+1] == '[' {
			// Consume entire escape sequence
			var seq strings.Builder
			seq.WriteRune(runes[i]) // ESC
			i++
			seq.WriteRune(runes[i]) // [
			i++
			for i < len(runes) {
				seq.WriteRune(runes[i])
				// Final byte of CSI sequence is in range 0x40-0x7E
				if runes[i] >= 0x40 && runes[i] <= 0x7E {
					i++
					break
				}
				i++
			}
			seqStr := seq.String()
			currentLine.WriteString(seqStr)
			// Track reset vs color sequences
			if seqStr == "\033[0m" || seqStr == "\033[m" {
				activeANSI = ""
			} else {
				activeANSI = seqStr
			}
			continue
		}

		r := runes[i]
		runeWidth := lipgloss.Width(string(r))
		if currentWidth+runeWidth > width {
			// Close any active ANSI on this line before wrapping
			if activeANSI != "" {
				currentLine.WriteString("\033[0m")
			}
			lines = append(lines, currentLine.String())
			currentLine.Reset()
			currentWidth = 0
			// Re-apply active ANSI on the new line
			if activeANSI != "" {
				currentLine.WriteString(activeANSI)
			}
		}
		currentLine.WriteRune(r)
		currentWidth += runeWidth
		i++
	}
	// Don't forget the last line
	if currentLine.Len() > 0 {
		lines = append(lines, currentLine.String())
	}

	return lines
}

// wrapPreviewContent splits multi-line content (e.g. pretty-printed JSON) by
// newlines first, then wraps each line to fit within the given width.
func wrapPreviewContent(s string, width int) []string {
	var result []string
	for line := range strings.SplitSeq(s, "\n") {
		if line == "" {
			result = append(result, "")
			continue
		}
		wrapped := wrapText(line, width)
		result = append(result, wrapped...)
	}
	return result
}

func (m *model) updateFiltered() {
	m.filtered = []int{}
	m.filterRegexErr = nil

	if m.filterRegex && m.filter != "" {
		re, err := regexp.Compile("(?i)" + m.filter)
		if err != nil {
			m.filterRegexErr = err
			// Show all lines when regex is invalid
			for i := range m.lines {
				m.filtered = append(m.filtered, i)
			}
		} else {
			for i, line := range m.lines {
				if re.MatchString(line.Content) {
					m.filtered = append(m.filtered, i)
				}
			}
		}
	} else {
		filter := strings.ToLower(m.filter)
		for i, line := range m.lines {
			if m.filter == "" || strings.Contains(strings.ToLower(line.Content), filter) {
				m.filtered = append(m.filtered, i)
			}
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

// renderCmdPaletteOverlay creates the command palette overlay box
func (m model) renderCmdPaletteOverlay() (box string, boxWidth, boxHeight int) {
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("241")) // dim

	nameStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	selectedNameStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("15")).
		Foreground(lipgloss.Color("#000000")).
		Bold(true)

	selectedKeyStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("15")).
		Foreground(lipgloss.Color("241"))

	filterStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("11"))

	borderColor := lipgloss.Color("12")

	allCommands := commands()
	filtered := m.filteredCommands()

	// Compute column width
	const paletteWidth = 40
	totalSlots := len(allCommands) // fixed height so box doesn't move

	var content strings.Builder

	// Filter input with bottom border
	before := m.cmdPaletteFilter[:m.cmdPaletteCursor]
	after := m.cmdPaletteFilter[m.cmdPaletteCursor:]
	filterLine := filterStyle.Render(":"+before) + "█" + filterStyle.Render(after)
	// Pad filter line to full width
	filterVisual := lipgloss.Width(filterLine)
	if filterVisual < paletteWidth {
		filterLine += strings.Repeat(" ", paletteWidth-filterVisual)
	}
	content.WriteString(filterLine + "\n")
	content.WriteString(lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("─", paletteWidth)) + "\n")

	// Command list (fixed number of rows)
	for i := range totalSlots {
		if i < len(filtered) {
			cmd := filtered[i]
			gap := max(paletteWidth-lipgloss.Width(cmd.name)-lipgloss.Width(cmd.shortcut), 2)
			if i == m.cmdPaletteSelected {
				line := selectedNameStyle.Render(cmd.name+strings.Repeat(" ", gap)) + selectedKeyStyle.Render(cmd.shortcut)
				content.WriteString(line + "\n")
			} else {
				content.WriteString(nameStyle.Render(cmd.name) + strings.Repeat(" ", gap) + keyStyle.Render(cmd.shortcut) + "\n")
			}
		} else {
			content.WriteString(strings.Repeat(" ", paletteWidth) + "\n")
		}
	}

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(borderColor)

	box = boxStyle.Render(content.String())
	boxWidth = lipgloss.Width(box)
	boxHeight = lipgloss.Height(box)

	return box, boxWidth, boxHeight
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
		{"+/-", "Resize preview pane"},
		{"J / K", "Scroll preview down / up"},
		{"/", "Enter filter mode"},
		{"//", "Toggle regex filter mode"},
		{"Esc", "Exit filter / clear"},
		{"", ""},
		{"r / Ctrl+r", "Reload command"},
		{"c", "Stop running command"},
		{"y", "Copy line to clipboard"},
		{"Y", "Copy line (plain text)"},
		{":", "Open command palette"},
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

	// Overlay command palette if active
	if m.cmdPaletteMode {
		box, boxWidth, boxHeight := m.renderCmdPaletteOverlay()
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

	filterRegexStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("13"))

	filterErrStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("9"))

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

	// Add refresh countdown on the right if auto-refresh is enabled and > 1s
	if m.config.RefreshInterval > time.Second && !m.streaming && !m.refreshStartTime.IsZero() {
		elapsed := time.Since(m.refreshStartTime)
		remaining := m.config.RefreshInterval - elapsed
		if remaining > 0 {
			countdownStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241")) // dim gray
			countdown := countdownStyle.Render(fmt.Sprintf("(%ds)", int(remaining.Seconds())+1))
			cmdWidth := lipgloss.Width(commandLine)
			countdownWidth := lipgloss.Width(countdown)
			gap := innerWidth - cmdWidth - countdownWidth
			if gap > 0 {
				commandLine += strings.Repeat(" ", gap) + countdown
			}
		}
	}

	// Build prompt line (will go at bottom)
	var promptLine string
	switch {
	case m.filterMode && m.filterRegex:
		label := filterRegexStyle.Render("regex/")
		before := m.filter[:m.filterCursor]
		after := m.filter[m.filterCursor:]
		input := filterStyle.Render(before) + "█" + filterStyle.Render(after)
		promptLine = label + input
		if m.filterRegexErr != nil {
			promptLine += " " + filterErrStyle.Render("(invalid regex)")
		}
	case m.filterMode:
		before := m.filter[:m.filterCursor]
		after := m.filter[m.filterCursor:]
		promptLine = filterStyle.Render("/"+before) + "█" + filterStyle.Render(after)
	case m.filter != "" && m.filterRegex:
		promptLine = promptStyle.Render(fmt.Sprintf("%s (regex: %s)", m.config.Prompt, m.filter))
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
				// For selected line: strip ANSI so highlight colors aren't mixed with content colors
				plainContent := stripANSI(content)

				// For selected line: gray line number + black content, both on white background
				selectedLineNumStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("15")).
					Foreground(lipgloss.Color("241"))
				selectedContentStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("15")).
					Foreground(lipgloss.Color("#000000")).
					Bold(true)

				// Pad content to fill remaining width
				contentPadded := plainContent
				padding := fullWidth - lineNumWidth - len(plainContent)
				if padding > 0 {
					contentPadded = plainContent + strings.Repeat(" ", padding)
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
				// Strip ANSI so highlight colors aren't mixed with content colors
				lineText = stripANSI(lineText)
				// Pad to full width for selection highlight
				padding := fullWidth - len(lineText)
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
			previewContent = highlightJSON(m.lines[idx].Content)
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
				previewLines = wrapPreviewContent(previewContent, innerWidth)
			}
			// Apply preview scroll offset
			previewLines = m.applyPreviewOffset(previewLines, previewH)
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
				previewW := leftW
				if m.config.PreviewPosition == PreviewRight {
					previewW = rightW
				}
				previewLines = wrapPreviewContent(previewContent, previewW)
			}
			// Apply preview scroll offset
			previewLines = m.applyPreviewOffset(previewLines, listHeight)
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
