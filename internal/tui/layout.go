package tui

import "github.com/charmbracelet/lipgloss"

// fixBox renders s in a box of exactly w columns and h rows, truncating
// or padding as needed. It never wraps.
func fixBox(s string, w, h int) string {
	return lipgloss.NewStyle().
		MaxWidth(w).
		Height(h).
		MaxHeight(h).
		Render(s)
}

// fitRow joins left and right on one row, right-aligned. If right doesn't
// fit, it is dropped.
func fitRow(left, right string, width int) string {
	lw := lipgloss.Width(left)
	rw := lipgloss.Width(right)
	if rw > 0 && lw+rw+1 > width {
		return fixBox(left, width, 1)
	}
	if rw == 0 {
		return fixBox(left, width, 1)
	}
	gap := width - lw - rw
	if gap < 1 {
		gap = 1
	}
	return left + spaces(gap) + right
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
