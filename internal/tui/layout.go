package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// fillStyle paints background-only cells. Used for every gap, pad and
// trailing fill so no cell falls back to the terminal default background.
var fillStyle = lipgloss.NewStyle().Background(darkBg)

// fill returns n space cells painted with the application background.
func fill(n int) string {
	if n <= 0 {
		return ""
	}
	return fillStyle.Render(strings.Repeat(" ", n))
}

// fitLine truncates s to exactly w cells (ANSI aware) and pads the remainder
// with background-filled spaces. Rows of the UI are built with it so that a
// row is always exactly as wide as the space it occupies.
func fitLine(s string, w int) string {
	if w <= 0 {
		return ""
	}
	if lipgloss.Width(s) > w {
		s = ansi.Truncate(s, w, "")
	}
	if gap := w - lipgloss.Width(s); gap > 0 {
		s += fill(gap)
	}
	return s
}

// padLines widens every line of s to w cells. Viewport content is padded
// before it is handed to the viewport so that the fill after an inner reset
// still carries the background.
func padLines(s string, w int) string {
	if w <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i := range lines {
		lines[i] = fitLine(lines[i], w)
	}
	return strings.Join(lines, "\n")
}

// rule renders a w-cell horizontal rule in the border colour.
func rule(w int) string {
	if w <= 0 {
		return ""
	}
	return ruleStyle.Render(strings.Repeat("─", w))
}

// activeRule renders a w-cell rule in the accent colour (the underline that
// sits below the active tab).
func activeRule(w int) string {
	if w <= 0 {
		return ""
	}
	return activeRuleStyle.Render(strings.Repeat("─", w))
}

// fixBox renders s in a box of exactly w columns and h rows, truncating
// or padding as needed. It never wraps.
func fixBox(s string, w, h int) string {
	return lipgloss.NewStyle().
		MaxWidth(w).
		Height(h).
		MaxHeight(h).
		Render(s)
}

// fixBlock normalises a free-form block to exactly w columns and h rows:
// every line is truncated/padded to w and the block to h lines. Padding cells
// carry the application background.
func fixBlock(s string, w, h int) string {
	if w <= 0 || h <= 0 {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	for i := range lines {
		lines[i] = fitLine(lines[i], w)
	}
	return strings.Join(lines, "\n")
}

// fitRow joins left and right on one row, right-aligned, with a
// background-filled gap. When they do not fit, the left block is truncated.
func fitRow(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if rw == 0 {
		return fitLine(left, width)
	}
	if rw >= width {
		return fitLine(right, width)
	}
	if lw+rw > width {
		left = ansi.Truncate(left, max(width-rw-1, 0), "…")
		lw = lipgloss.Width(left)
	}
	return fitLine(left+fill(width-lw-rw)+right, width)
}

func spaces(n int) string {
	if n <= 0 {
		return ""
	}
	b := make([]byte, n)
	for i := range b {
		b[i] = ' '
	}
	return string(b)
}
