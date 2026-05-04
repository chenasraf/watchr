package ui

import (
	"context"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/chenasraf/watchr/internal/runner"
)

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
	// lastLineCount tracks the streamResult's CurrentLineCount (lines produced
	// by the current run), which starts at 0 for a fresh streaming result.
	m.lastLineCount = 0
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

		isDone := m.streamResult.IsDone()
		currentCount := m.streamResult.GetCurrentLineCount()

		// Sync m.lines whenever the stream has produced new line writes since
		// the last tick (in-place edits and appends both bump CurrentLineCount),
		// or once on completion so the trim-to-currentCount path runs.
		if currentCount != m.lastLineCount || isDone {
			newLines := m.streamResult.GetLines()
			// On completion, drop any leftover slots that the new run never
			// wrote into — they still hold previous-run content.
			if isDone && currentCount < len(newLines) {
				newLines = newLines[:currentCount]
			}
			m.lines = newLines
			m.lastLineCount = currentCount
			m.updateFiltered()

			if !m.userScrolled {
				visible := m.visibleLines()
				if visible > 0 {
					m.cursor = max(len(m.filtered)-1, 0)
					m.offset = max(len(m.filtered)-visible, 0)
				}
			}
		}

		if isDone {
			m.streaming = false
			m.loading = false
			m.exitCode = m.streamResult.ExitCode
			if m.streamResult.Error != nil {
				m.errorMsg = m.streamResult.Error.Error()
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
