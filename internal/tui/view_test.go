package tui

import (
	"net/http"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

// TestViewFitsAtAllSizes renders View() at multiple terminal sizes and
// asserts every line fits within the width and the line count equals the height.
func TestViewFitsAtAllSizes(t *testing.T) {
	sizes := [][2]int{{80, 24}, {100, 30}, {140, 50}}

	// Force TrueColor profile for deterministic width assertions.
	oldProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(oldProfile) })

	for _, size := range sizes {
		m := newTestModel(t)
		m = update(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})

		// Set a response so JSON highlighting is exercised.
		m.statusCode = 201
		m.responseTime = 142000000
		m.respSize = 1400
		m.respBody = `{"id":84019284,"title":"CLI headless test runner fails in CI","state":"open","labels":["bug","terminal"]}`
		m.resp.headers = http.Header{"Content-Type": {"application/json"}}
		m.bodyInput.SetValue(`{"title":"CLI headless test runner fails in CI","labels":["bug","terminal"]}`)
		m.method = "POST"
		m.reqTab = reqTabBody
		m.syncViewport()

		view := m.View()
		lines := strings.Split(view, "\n")

		if len(lines) != size[1] {
			t.Fatalf("size=%v: got %d lines, want %d", size, len(lines), size[1])
		}
		for i, line := range lines {
			w := lipgloss.Width(line)
			if w != size[0] {
				t.Fatalf("size=%v line %d: width=%d, want %d", size, i, w, size[0])
			}
		}
	}
}
