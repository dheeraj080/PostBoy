package tui

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/httpclient"
)

func TestViewFillsTerminal(t *testing.T) {
	sizes := [][2]int{{80, 20}, {80, 24}, {100, 30}, {120, 40}, {200, 60}}
	tabs := []requestTab{reqTabParams, reqTabAuth, reqTabHeaders, reqTabBody}
	modes := config.BodyModes

	for _, size := range sizes {
		for _, tab := range tabs {
			for _, mode := range modes {
				for _, loading := range []bool{false, true} {
					m := newTestModel(t)
					m.termWidth = size[0]
					m.termHeight = size[1]
					m.method = "POST"
					m.reqTab = tab
					m.bodyMode = mode
					m.loading = loading
					m.status = strings.Repeat("x", 200)
					m.statusErr = true

					// 50+ headers with long values, cursor on the last one.
					items := make([]config.KeyValue, 55)
					for i := range items {
						items[i] = config.KeyValue{
							Key:     fmt.Sprintf("X-Header-%02d", i),
							Value:   strings.Repeat("v", 300),
							Enabled: true,
						}
					}
					items[54].Key = "X-Key"
					m.headers.setItems(items)
					m.headers.cursor = len(items) - 1

					// Long response body.
					m.respBody = strings.Repeat("line of response body\n", 300)
					m.statusCode = 201
					m.responseTime = 142000000
					m.respSize = 1400
					m.resp.headers = http.Header{"Content-Type": {"application/json"}}

					// Long JSON request body.
					m.bodyInput.SetValue(`{"title":"` + strings.Repeat("t", 300) + `","labels":["bug","terminal"]}`)

					m.layout()
					m.syncViewport()

					view := m.View()
					h := lipgloss.Height(view)
					w := lipgloss.Width(view)
					if h != size[1] {
						t.Fatalf("size=%v tab=%d mode=%s loading=%v: height=%d want %d", size, tab, mode, loading, h, size[1])
					}
					if w > size[0] {
						t.Fatalf("size=%v tab=%d mode=%s loading=%v: width=%d > %d", size, tab, mode, loading, w, size[0])
					}
					if tab == reqTabHeaders && !strings.Contains(view, "X-Key") {
						t.Fatalf("size=%v mode=%s loading=%v: selected header key missing", size, mode, loading)
					}
				}
			}
		}
	}
}

func TestViewFillsTerminalWithResponse(t *testing.T) {
	m := newTestModel(t)
	m.termWidth = 120
	m.termHeight = 40
	m.method = "POST"
	m.reqTab = reqTabBody
	m.bodyMode = config.BodyRaw
	m.statusCode = 201
	m.responseTime = 142000000
	m.respSize = 1400
	m.respBody = `{"id":84019284,"title":"CLI headless test runner fails in CI","state":"open"}`
	m.resp.headers = http.Header{"Content-Type": {"application/json"}}
	m.bodyInput.SetValue(`{"title":"CLI headless test runner fails in CI","labels":["bug","terminal"]}`)
	m.layout()
	m.syncViewport()

	view := m.View()
	if h := lipgloss.Height(view); h != 40 {
		t.Fatalf("height=%d want 40", h)
	}
	if w := lipgloss.Width(view); w > 120 {
		t.Fatalf("width=%d > 120", w)
	}
	if !strings.Contains(view, "201 Created") {
		t.Fatal("status missing")
	}
	if !strings.Contains(view, "142ms") {
		t.Fatal("response time missing")
	}
	if !strings.Contains(view, "1.4 KB") {
		t.Fatal("response size missing")
	}
}

var _ = httpclient.MaxBodyBytes
