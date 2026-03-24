package ui

import (
	"regexp"
	"strings"
)

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
