package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// TestProbeChips logs response-pane chips/header widths at 120x35 with a JSON
// response loaded. It asserts nothing.
func TestProbeChips(t *testing.T) {
	m := newTestModel(t)
	m = update(t, m, tea.WindowSizeMsg{Width: 120, Height: 35})
	m = withResponse(t, m, "application/json", []byte(`{"a":1}`))

	t.Logf("hasBody=%v binary=%v", m.resp.hasBody(), m.resp.binary)

	qa := m.renderResponseQuickActions()
	t.Logf("quickActions width=%d |%s|", lipgloss.Width(qa), ansi.Strip(qa))

	_, rightW := m.columnWidths()
	tabs := m.renderResponseTabs(rightW - 2)
	t.Logf("paneW passed to renderResponseHeader=%d tabsWidth=%d", rightW, lipgloss.Width(tabs))

	for i, line := range strings.Split(m.renderResponseHeader(rightW), "\n") {
		t.Logf("header[%d] width=%d |%s|", i, lipgloss.Width(line), ansi.Strip(line))
	}
}
