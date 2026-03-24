package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

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
		{"R", "Reload & clear lines"},
		{"d / Del", "Delete selected line"},
		{"D", "Clear all lines"},
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
		fmt.Fprintf(&content, "  %s  %s\n", key, desc)
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

// renderConfirmOverlay creates a confirmation dialog overlay
func (m model) renderConfirmOverlay() (box string, boxWidth, boxHeight int) {
	msgStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252"))

	boxStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("11")).
		Padding(1, 2)

	content := msgStyle.Render(m.confirmMessage)
	box = boxStyle.Render(content)
	boxWidth = lipgloss.Width(box)
	boxHeight = lipgloss.Height(box)

	return box, boxWidth, boxHeight
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

	// Overlay confirmation dialog if active
	if m.confirmMode {
		box, boxWidth, boxHeight := m.renderConfirmOverlay()
		return overlayBox(mainView, box, boxWidth, boxHeight, m.width, m.height)
	}

	return mainView
}

// Box drawing characters (rounded)
const (
	boxTopLeft     = "╭"
	boxTopRight    = "╮"
	boxBottomLeft  = "╰"
	boxBottomRight = "╯"
	boxHorizontal  = "─"
	boxVertical    = "│"
	boxLeftT       = "├"
	boxRightT      = "┤"
	boxTopT        = "┬"
	boxBottomT     = "┴"
)

// viewContext holds shared rendering state for a single View() call.
type viewContext struct {
	innerWidth  int
	borderStyle lipgloss.Style
}

func (vc viewContext) hLine(left, right string, splitPos int, junction string) string {
	if splitPos > 0 && splitPos < vc.innerWidth {
		return vc.borderStyle.Render(left + strings.Repeat(boxHorizontal, splitPos) + junction + strings.Repeat(boxHorizontal, vc.innerWidth-splitPos-1) + right)
	}
	return vc.borderStyle.Render(left + strings.Repeat(boxHorizontal, vc.innerWidth) + right)
}

func (vc viewContext) padLine(content string) string {
	contentWidth := lipgloss.Width(content)
	if contentWidth < vc.innerWidth {
		content += strings.Repeat(" ", vc.innerWidth-contentWidth)
	} else if contentWidth > vc.innerWidth {
		content = lipgloss.NewStyle().MaxWidth(vc.innerWidth-1).Render(content) + ellipsis
	}
	return vc.borderStyle.Render(boxVertical) + content + vc.borderStyle.Render(boxVertical)
}

func (m model) renderMainView() string {
	borderColor := lipgloss.Color("240")
	vc := viewContext{
		innerWidth:  m.width - 2,
		borderStyle: lipgloss.NewStyle().Foreground(borderColor),
	}

	commandLine := m.renderHeaderLine(vc.innerWidth)
	promptLine := m.renderPromptLine()
	listHeight, listWidth := m.listDimensions(vc.innerWidth)
	listLines := m.renderListLines(listHeight, listWidth)

	// Preview content
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

	// Vertical split position for left/right preview
	var vSplitPos int
	if m.showPreview {
		switch m.config.PreviewPosition {
		case PreviewLeft:
			vSplitPos = m.previewSize()
		case PreviewRight:
			vSplitPos = vc.innerWidth - m.previewSize() - 1
		}
	}

	// Build the unified box
	var lines []string
	lines = append(lines, vc.hLine(boxTopLeft, boxTopRight, 0, boxTopT))
	lines = append(lines, vc.padLine(commandLine))
	lines = append(lines, vc.hLine(boxLeftT, boxRightT, vSplitPos, boxTopT))

	// Content area
	if !m.showPreview {
		lines = append(lines, m.renderContentNoPreview(vc, listLines, listHeight)...)
	} else {
		lines = append(lines, m.renderContentWithPreview(vc, listLines, listHeight, previewContent)...)
	}

	lines = append(lines, vc.hLine(boxBottomLeft, boxBottomRight, vSplitPos, boxBottomT))

	return strings.Join(lines, "\n") + "\n" + promptLine
}

func (m model) renderHeaderLine(innerWidth int) string {
	titleStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12")).Bold(true)
	prefix := titleStyle.Render("watchr") + " • "

	var commandLine string
	switch {
	case m.streaming:
		streamStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
		commandLine = prefix + streamStyle.Render("◉ "+m.config.Command)
	case m.loading:
		commandLine = prefix + m.config.Command
	case m.exitCode == 0:
		successStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		commandLine = prefix + successStyle.Render("✓ "+m.config.Command)
	default:
		failStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		commandLine = prefix + failStyle.Render(fmt.Sprintf("✗ [%d] %s", m.exitCode, m.config.Command))
	}

	if m.config.RefreshInterval > time.Second && !m.streaming && !m.refreshStartTime.IsZero() {
		elapsed := time.Since(m.refreshStartTime)
		remaining := m.config.RefreshInterval - elapsed
		if remaining > 0 {
			countdownStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
			countdown := countdownStyle.Render(fmt.Sprintf("(%ds)", int(remaining.Seconds())+1))
			cmdWidth := lipgloss.Width(commandLine)
			countdownWidth := lipgloss.Width(countdown)
			gap := innerWidth - cmdWidth - countdownWidth
			if gap > 0 {
				commandLine += strings.Repeat(" ", gap) + countdown
			}
		}
	}

	return commandLine
}

func (m model) renderPromptLine() string {
	promptStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("14"))
	filterStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	filterRegexStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("13"))
	filterErrStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))

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
		statusStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
		promptLine += " " + statusStyle.Render(m.statusMsg)
	}

	helpHint := lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("? for help")
	promptWidth := lipgloss.Width(promptLine)
	hintWidth := lipgloss.Width(helpHint)
	gap := m.width - promptWidth - hintWidth
	if gap > 0 {
		promptLine += strings.Repeat(" ", gap) + helpHint
	}

	return promptLine
}

func (m model) listDimensions(innerWidth int) (height, width int) {
	height = m.visibleLines()
	width = innerWidth - 1
	if m.showPreview && (m.config.PreviewPosition == PreviewLeft || m.config.PreviewPosition == PreviewRight) {
		width = innerWidth - m.previewSize() - 2
	}
	return height, width
}

func (m model) renderListLines(listHeight, listWidth int) []string {
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("15")).
		Foreground(lipgloss.Color("#000000")).
		Bold(true)
	lineNumStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("241"))

	var listLines []string
	for i := range listHeight {
		lineIdx := m.offset + i
		if lineIdx >= len(m.filtered) {
			listLines = append(listLines, "")
			continue
		}

		idx := m.filtered[lineIdx]
		if idx >= len(m.lines) {
			listLines = append(listLines, "")
			continue
		}
		line := m.lines[idx]
		isSelected := lineIdx == m.cursor
		fullWidth := listWidth + 1

		var lineText string
		if m.config.ShowLineNums {
			lineNumStr := fmt.Sprintf("%*d  ", m.config.LineNumWidth, line.Number)
			lineNumWidth := len(lineNumStr)
			contentWidth := listWidth - lineNumWidth
			content := truncateToWidth(line.Content, contentWidth)

			if isSelected {
				plainContent := stripANSI(content)
				selectedLineNumStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("15")).
					Foreground(lipgloss.Color("241"))
				selectedContentStyle := lipgloss.NewStyle().
					Background(lipgloss.Color("15")).
					Foreground(lipgloss.Color("#000000")).
					Bold(true)
				contentPadded := plainContent
				padding := fullWidth - lineNumWidth - len(plainContent)
				if padding > 0 {
					contentPadded = plainContent + strings.Repeat(" ", padding)
				}
				lineText = selectedLineNumStyle.Render(lineNumStr) + selectedContentStyle.Render(contentPadded)
			} else {
				lineText = lineNumStyle.Render(lineNumStr) + content
			}
		} else {
			lineText = truncateToWidth(line.Content, listWidth)
			if isSelected {
				lineText = stripANSI(lineText)
				padding := fullWidth - len(lineText)
				if padding > 0 {
					lineText += strings.Repeat(" ", padding)
				}
				lineText = selectedStyle.Render(lineText)
			}
		}

		listLines = append(listLines, lineText)
	}
	return listLines
}

func (m model) renderContentNoPreview(vc viewContext, listLines []string, listHeight int) []string {
	var lines []string
	for i := range listHeight {
		if i < len(listLines) {
			lines = append(lines, vc.padLine(listLines[i]))
		} else {
			lines = append(lines, vc.padLine(""))
		}
	}
	return lines
}

func (m model) renderContentWithPreview(vc viewContext, listLines []string, listHeight int, previewContent string) []string {
	switch m.config.PreviewPosition {
	case PreviewTop, PreviewBottom:
		return m.renderVerticalPreview(vc, listLines, listHeight, previewContent)
	case PreviewLeft, PreviewRight:
		return m.renderHorizontalPreview(vc, listLines, listHeight, previewContent)
	}
	return nil
}

func (m *model) renderVerticalPreview(vc viewContext, listLines []string, listHeight int, previewContent string) []string {
	previewH := m.previewSize()

	var previewLines []string
	if previewContent != "" {
		previewLines = wrapPreviewContent(previewContent, vc.innerWidth)
	}
	previewLines = m.applyPreviewOffset(previewLines, previewH)
	for len(previewLines) < previewH {
		previewLines = append(previewLines, "")
	}

	paddedList := m.renderContentNoPreview(vc, listLines, listHeight)
	var paddedPreview []string
	for _, line := range previewLines[:previewH] {
		paddedPreview = append(paddedPreview, vc.padLine(line))
	}

	separator := vc.hLine(boxLeftT, boxRightT, 0, boxTopT)

	if m.config.PreviewPosition == PreviewTop {
		result := paddedPreview
		result = append(result, separator)
		result = append(result, paddedList...)
		return result
	}
	// PreviewBottom
	result := paddedList
	result = append(result, separator)
	result = append(result, paddedPreview...)
	return result
}

func (m *model) renderHorizontalPreview(vc viewContext, listLines []string, listHeight int, previewContent string) []string {
	var leftW, rightW int
	if m.config.PreviewPosition == PreviewLeft {
		leftW = m.previewSize()
		rightW = vc.innerWidth - leftW - 1
	} else {
		rightW = m.previewSize()
		leftW = vc.innerWidth - rightW - 1
	}

	var previewLines []string
	if previewContent != "" {
		previewW := leftW
		if m.config.PreviewPosition == PreviewRight {
			previewW = rightW
		}
		previewLines = wrapPreviewContent(previewContent, previewW)
	}
	previewLines = m.applyPreviewOffset(previewLines, listHeight)
	for len(previewLines) < listHeight {
		previewLines = append(previewLines, "")
	}

	fitToWidth := func(s string, w int, isPreview bool) string {
		sw := lipgloss.Width(s)
		if sw > w {
			if isPreview {
				return s + strings.Repeat(" ", w-sw)
			}
			return lipgloss.NewStyle().MaxWidth(w-1).Render(s) + ellipsis
		}
		return s + strings.Repeat(" ", w-sw)
	}

	var lines []string
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

		line := vc.borderStyle.Render(boxVertical) + leftContent + vc.borderStyle.Render(boxVertical) + rightContent + vc.borderStyle.Render(boxVertical)
		lines = append(lines, line)
	}
	return lines
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
