package tui

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dheeraj080/PostBoy/internal/httpclient"
)

const usersJSON = `{"users":[{"name":"alice","age":30},{"name":"bob","age":25}],"total":2}`

// withResponse delivers a response and focuses the response panel.
func withResponse(t *testing.T, m Model, ct string, body []byte) Model {
	t.Helper()
	m.reqID++
	m = update(t, m, responseMsg{id: m.reqID, res: &httpclient.Response{
		StatusCode: 200,
		Headers:    http.Header{"Content-Type": {ct}, "X-Request-Id": {"abc123"}},
		Body:       body,
	}})
	m.opts.ExportDir = t.TempDir()
	m.setFocus(focusResContent)
	return m
}

func stubClipboard(t *testing.T) *string {
	t.Helper()
	var copied string
	old := copyToClipboard
	copyToClipboard = func(s string) error { copied = s; return nil }
	t.Cleanup(func() { copyToClipboard = old })
	return &copied
}

func TestResponseSearch(t *testing.T) {
	m := withResponse(t, newTestModel(t), "application/json", []byte(usersJSON))
	m = press(t, m, "/")
	if !m.isTyping() {
		t.Fatal("search input should count as typing")
	}
	m = typeText(t, m, "NAME") // case-insensitive
	m = press(t, m, "enter")
	if len(m.resp.matches) != 2 || m.resp.current != 0 {
		t.Fatalf("matches = %+v", m.resp.matches)
	}
	if !strings.Contains(m.status, "2 matches") {
		t.Fatalf("status = %q", m.status)
	}
	m = press(t, m, "n")
	if m.resp.current != 1 {
		t.Fatal("n did not advance")
	}
	m = press(t, m, "n")
	if m.resp.current != 0 {
		t.Fatal("n did not wrap")
	}
	m = press(t, m, "N")
	if m.resp.current != 1 {
		t.Fatal("N did not go back")
	}
	if !strings.Contains(m.renderResponseToolbar(), "2/2") {
		t.Fatalf("toolbar = %q", m.renderResponseToolbar())
	}
	m = press(t, m, "esc")
	if m.resp.search != "" || len(m.resp.matches) != 0 {
		t.Fatal("esc did not clear search")
	}
}

func TestResponseSearchNoMatchAndHeaders(t *testing.T) {
	m := withResponse(t, newTestModel(t), "application/json", []byte(usersJSON))
	m = press(t, m, "/")
	m = typeText(t, m, "zzz")
	m = press(t, m, "enter")
	if !m.statusErr || len(m.resp.matches) != 0 {
		t.Fatalf("expected no-match error, status=%q", m.status)
	}
	// Search applies to the headers tab too.
	m.resTab = resTabHeaders
	m.resp.search = "abc123"
	m.refreshResponse(false)
	if len(m.resp.matches) != 1 {
		t.Fatalf("header matches = %+v", m.resp.matches)
	}
}

func TestResponseFilter(t *testing.T) {
	copied := stubClipboard(t)
	m := withResponse(t, newTestModel(t), "application/json", []byte(usersJSON))
	m = press(t, m, "f")
	m = typeText(t, m, "users.#.name")
	m = press(t, m, "enter")
	if m.resp.filterErr != "" {
		t.Fatal(m.resp.filterErr)
	}
	if !strings.Contains(m.respBody, "alice") || strings.Contains(m.respBody, "age") {
		t.Fatalf("filtered body = %q", m.respBody)
	}
	m = press(t, m, "y")
	if !strings.Contains(*copied, `"alice"`) || strings.Contains(*copied, "age") {
		t.Fatalf("copied = %q", *copied)
	}

	// Query syntax.
	m = press(t, m, "f")
	m.resp.input.SetValue(`users.#(age>26).name`)
	m = press(t, m, "enter")
	if m.resp.filterPlain != `"alice"` {
		t.Fatalf("query result = %q", m.resp.filterPlain)
	}

	// Missing path.
	m = press(t, m, "f")
	m.resp.input.SetValue("nope")
	m = press(t, m, "enter")
	if m.resp.filterErr == "" || !m.statusErr || !strings.Contains(m.respBody, "no match") {
		t.Fatalf("expected filter error, body=%q", m.respBody)
	}

	// Clearing the filter restores the body.
	m = press(t, m, "esc")
	if m.resp.filter != "" || !strings.Contains(m.respBody, "age") {
		t.Fatal("esc did not clear filter")
	}
}

func TestResponseFilterNonJSON(t *testing.T) {
	m := withResponse(t, newTestModel(t), "text/plain", []byte("hello"))
	m = press(t, m, "f")
	m.resp.input.SetValue("a")
	m = press(t, m, "enter")
	if !strings.Contains(m.resp.filterErr, "not valid JSON") {
		t.Fatalf("filterErr = %q", m.resp.filterErr)
	}
}

func TestResponseRawToggle(t *testing.T) {
	m := withResponse(t, newTestModel(t), "application/json", []byte(usersJSON))
	if !strings.Contains(m.resp.bodyPlain(), "\n  ") {
		t.Fatal("expected pretty body by default")
	}
	m = press(t, m, "r")
	if !m.resp.rawMode || m.respBody != usersJSON {
		t.Fatalf("raw body = %q", m.respBody)
	}
	m = press(t, m, "r")
	if m.resp.rawMode {
		t.Fatal("r did not toggle back")
	}
}

func TestResponseSaveAndOverwriteConfirm(t *testing.T) {
	m := withResponse(t, newTestModel(t), "application/json", []byte(usersJSON))
	m = press(t, m, "s")
	if m.resp.input.Value() != "response.json" {
		t.Fatalf("default name = %q", m.resp.input.Value())
	}
	m = press(t, m, "enter")
	path := filepath.Join(m.opts.ExportDir, "response.json")
	data, err := os.ReadFile(path)
	if err != nil || string(data) != usersJSON {
		t.Fatalf("saved %q, %v", data, err)
	}

	// Saving again requires confirmation.
	m = press(t, m, "f")
	m.resp.input.SetValue("total")
	m = press(t, m, "enter")
	m = press(t, m, "s", "enter")
	if m.resp.inputMode != respInputSave || m.resp.saveConfirm == "" {
		t.Fatal("expected overwrite confirmation")
	}
	m = press(t, m, "enter")
	data, _ = os.ReadFile(path)
	if string(data) != "2" {
		t.Fatalf("filtered save = %q", data)
	}
	if m.resp.inputMode != respInputNone {
		t.Fatal("save input not closed")
	}
}

func TestBinaryResponse(t *testing.T) {
	copied := stubClipboard(t)
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR\xff\xfe")
	m := withResponse(t, newTestModel(t), "image/png", png)
	if !strings.Contains(m.respBody, "Binary response") {
		t.Fatalf("body = %q", m.respBody)
	}
	m = press(t, m, "y")
	if !m.statusErr || *copied != "" {
		t.Fatal("binary body should not be copied")
	}
	m = press(t, m, "s")
	if m.resp.input.Value() != "response.png" {
		t.Fatalf("default name = %q", m.resp.input.Value())
	}
	m = press(t, m, "enter")
	data, _ := os.ReadFile(filepath.Join(m.opts.ExportDir, "response.png"))
	if string(data) != string(png) {
		t.Fatal("binary body not saved verbatim")
	}
}

func TestResponseToolsResetOnNewRequest(t *testing.T) {
	m := withResponse(t, newTestModel(t), "application/json", []byte(usersJSON))
	m.resp.search = "x"
	m.resp.filter = "users"
	m = withResponse(t, m, "application/json", []byte(`{"a":1}`))
	if m.resp.search != "" || m.resp.filter != "" || m.resp.rawMode {
		t.Fatal("tool state not reset for new response")
	}
}

func TestResponseKeysIgnoredWithoutResponse(t *testing.T) {
	m := newTestModel(t)
	m.setFocus(focusResContent)
	m = press(t, m, "/")
	if m.resp.inputMode != respInputNone {
		t.Fatal("search opened with no response")
	}
}

func TestFindAndRenderMatches(t *testing.T) {
	text := "Foo bar foo\nnone\nFOO"
	ms := findMatches(text, "foo")
	if len(ms) != 3 || ms[1] != (match{line: 0, start: 8, end: 11}) || ms[2].line != 2 {
		t.Fatalf("matches = %+v", ms)
	}
	out := renderMatches(text, ms, 1)
	if strings.Count(out, "\n") != 2 || !strings.Contains(out, "none") {
		t.Fatalf("render changed line structure: %q", out)
	}
	if findMatches(text, "") != nil {
		t.Fatal("empty query should match nothing")
	}
}
