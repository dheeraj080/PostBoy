package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// TestProbeModeBar logs widths and plain text for the body mode bar, the body
// tab, and the header/URL/footer rows at 120x35. It asserts nothing.
func TestProbeModeBar(t *testing.T) {
	m := newTestModel(t)
	m = update(t, m, tea.WindowSizeMsg{Width: 120, Height: 35})
	m.method = "POST"
	m.reqTab = reqTabBody

	bar := m.renderBodyModeBar(55)
	t.Logf("modeBar width=%d |%s|", lipgloss.Width(bar), ansi.Strip(bar))

	bodyTab := m.renderBodyTab(55, m.panelHeight())
	for i, line := range strings.Split(bodyTab, "\n") {
		if i >= 4 {
			break
		}
		t.Logf("bodyTab[%d] width=%d |%s|", i, lipgloss.Width(line), ansi.Strip(line))
	}

	for i, line := range strings.Split(m.renderURLBar(), "\n") {
		t.Logf("urlBar[%d] width=%d |%s|", i, lipgloss.Width(line), ansi.Strip(line))
	}
	for i, line := range strings.Split(m.renderHeaderLine(), "\n") {
		t.Logf("headerLine[%d] width=%d |%s|", i, lipgloss.Width(line), ansi.Strip(line))
	}
	for i, line := range strings.Split(m.renderFooter(), "\n") {
		t.Logf("footer[%d] width=%d |%s|", i, lipgloss.Width(line), ansi.Strip(line))
	}
}
