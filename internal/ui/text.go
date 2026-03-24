package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

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

// wordBoundaryLeft returns the cursor position after jumping left to the previous word boundary.
func wordBoundaryLeft(s string, pos int) int {
	for pos > 0 && s[pos-1] == ' ' {
		pos--
	}
	for pos > 0 && s[pos-1] != ' ' {
		pos--
	}
	return pos
}

// wordBoundaryRight returns the cursor position after jumping right to the next word boundary.
func wordBoundaryRight(s string, pos int) int {
	for pos < len(s) && s[pos] != ' ' {
		pos++
	}
	for pos < len(s) && s[pos] == ' ' {
		pos++
	}
	return pos
}

// textInsert inserts a string at the cursor position, returning new text and cursor.
func textInsert(text, insert string, cursor int) (string, int) {
	return text[:cursor] + insert + text[cursor:], cursor + len(insert)
}

// textBackspace deletes one character before the cursor, returning new text and cursor.
func textBackspace(text string, cursor int) (string, int) {
	if cursor <= 0 {
		return text, cursor
	}
	return text[:cursor-1] + text[cursor:], cursor - 1
}

// textBackspaceWord deletes the word before the cursor, returning new text and cursor.
func textBackspaceWord(text string, cursor int) (string, int) {
	if cursor <= 0 {
		return text, cursor
	}
	newPos := wordBoundaryLeft(text, cursor)
	return text[:newPos] + text[cursor:], newPos
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
