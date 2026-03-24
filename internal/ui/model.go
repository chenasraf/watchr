package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

	confirmMode    bool   // whether a confirmation dialog is visible
	confirmMessage string // message to display in confirmation dialog
	confirmAction  func(m *model) (tea.Model, tea.Cmd)
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
