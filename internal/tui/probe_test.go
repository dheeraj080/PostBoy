package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// TestProbeWidths logs per-line widths for the left and right panes at a few
// terminal sizes on the HEADERS and BODY tabs. It asserts nothing.
func TestProbeWidths(t *testing.T) {
	sizes := [][2]int{{100, 30}, {120, 35}, {160, 40}}
	probeTabs := []struct {
		name string
		tab  requestTab
	}{
		{"HEADERS", reqTabHeaders},
		{"BODY", reqTabBody},
	}

	logPane := func(size [2]int, tabName, pane, out string) int {
		maxW := 0
		for i, line := range strings.Split(out, "\n") {
			w := lipgloss.Width(line)
			if w > maxW {
				maxW = w
			}
			t.Logf("size=%dx%d tab=%s %s line[%d] width=%d",
				size[0], size[1], tabName, pane, i, w)
		}
		return maxW
	}

	for _, size := range sizes {
		for _, tc := range probeTabs {
			m := newTestModel(t)
			m = update(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
			m.reqTab = tc.tab

			leftW, rightW := m.columnWidths()
			maxLeft := logPane(size, tc.name, "left", m.renderLeftPane())
			maxRight := logPane(size, tc.name, "right", m.renderRightPane())

			t.Logf("size=%dx%d tab=%s columnWidths=(left=%d, right=%d) maxLeftWidth=%d maxRightWidth=%d",
				size[0], size[1], tc.name, leftW, rightW, maxLeft, maxRight)
		}
	}
}
