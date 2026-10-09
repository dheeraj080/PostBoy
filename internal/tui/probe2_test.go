package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
)

// TestProbeText logs the plain-text (ANSI-stripped) rendering of the panes and
// the header/URL rows at 120x35 on the BODY tab. It asserts nothing.
func TestProbeText(t *testing.T) {
	m := newTestModel(t)
	m = update(t, m, tea.WindowSizeMsg{Width: 120, Height: 35})
	m.reqTab = reqTabBody

	logLines := func(name, out string, limit int) {
		for i, line := range strings.Split(out, "\n") {
			if i >= limit {
				break
			}
			t.Logf("%s[%d] |%s|", name, i, ansi.Strip(line))
		}
	}

	logLines("left", m.renderLeftPane(), 12)
	logLines("right", m.renderRightPane(), 12)
	logLines("header", m.renderHeaderLine(), 12)
	logLines("urlbar", m.renderURLBar(), 12)
}
