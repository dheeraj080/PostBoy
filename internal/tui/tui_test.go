package tui

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/httpclient"
	"github.com/dheeraj080/PostBoy/internal/secrets"
	"github.com/dheeraj080/PostBoy/internal/store"
)

// newTestModel returns a sized, "ready" model backed by a temp dir and an
// in-memory keychain, without touching the real OS keychain.
func newTestModel(t *testing.T) Model {
	t.Helper()
	dir := t.TempDir()
	st, err := store.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })

	m := New(Options{Dir: dir})
	m = update(t, m, tea.WindowSizeMsg{Width: 120, Height: 40})
	m = update(t, m, servicesMsg{secrets: secrets.NewMemory(nil), store: st, cfg: config.Default(), dir: dir})
	return m
}

func update(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(Model)
}

func press(t *testing.T, m Model, keys ...string) Model {
	t.Helper()
	for _, k := range keys {
		var msg tea.KeyMsg
		switch k {
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "right":
			msg = tea.KeyMsg{Type: tea.KeyRight}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "left":
			msg = tea.KeyMsg{Type: tea.KeyLeft}
		case "ctrl+s":
			msg = tea.KeyMsg{Type: tea.KeyCtrlS}
		default:
			if strings.HasPrefix(k, "alt+") {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(strings.TrimPrefix(k, "alt+")), Alt: true}
			} else {
				msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
			}
		}
		m = update(t, m, msg)
	}
	return m
}

func typeText(t *testing.T, m Model, s string) Model {
	t.Helper()
	for _, r := range s {
		m = press(t, m, string(r))
	}
	return m
}

// Regression: 'h', 'l' and '?' used to be swallowed as shortcuts while typing.
func TestTypingShortcutCharsIntoURL(t *testing.T) {
	m := newTestModel(t)
	m = typeText(t, m, "https://hello.dev/path?l=1")
	if got := m.urlInput.Value(); got != "https://hello.dev/path?l=1" {
		t.Fatalf("url input = %q", got)
	}
	if m.modal != modalNone {
		t.Fatal("'?' opened a modal while typing in URL")
	}
}

func TestHelpOpensWhenNotTyping(t *testing.T) {
	m := newTestModel(t)
	m = press(t, m, "tab") // focus request tab bar
	m = press(t, m, "?")
	if m.modal != modalHelp {
		t.Fatal("help did not open")
	}
	m = press(t, m, "esc")
	if m.modal != modalNone {
		t.Fatal("help did not close")
	}
}

func TestLayoutPersistsViewportSize(t *testing.T) {
	m := newTestModel(t)
	if m.viewport.Height <= 1 || m.viewport.Width <= 1 {
		t.Fatalf("viewport not sized: %dx%d", m.viewport.Width, m.viewport.Height)
	}
}

func TestStaleResponseIgnored(t *testing.T) {
	m := newTestModel(t)
	m.reqID = 2
	m.loading = true
	m = update(t, m, responseMsg{id: 1, res: &httpclient.Response{StatusCode: 200, Body: []byte("old")}})
	if !m.loading || strings.Contains(m.respBody, "old") {
		t.Fatal("stale response was applied")
	}
}

func TestAddHeaderViaList(t *testing.T) {
	m := newTestModel(t)
	before := len(m.headers.items)
	m = press(t, m, "alt+l", "n")
	m = typeText(t, m, "X-Trace: abc")
	m = press(t, m, "enter")
	if len(m.headers.items) != before+1 {
		t.Fatalf("header not added: %+v", m.headers.items)
	}
	last := m.headers.items[len(m.headers.items)-1]
	if last.Key != "X-Trace" || last.Value != "abc" || !last.Enabled {
		t.Fatalf("bad header: %+v", last)
	}
	// Persisted to disk.
	c, err := config.Load(m.dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Draft.Headers) != before+1 {
		t.Fatal("header not persisted in draft")
	}
}

func TestSaveNewRequestToNewCollection(t *testing.T) {
	m := newTestModel(t)
	m.urlInput.SetValue("api.test/users")
	m = press(t, m, "ctrl+s")
	if m.modal != modalSave {
		t.Fatal("save dialog not opened for untitled request")
	}
	if got := m.colInput.Value(); got != "GET /users" {
		t.Fatalf("default name = %q", got)
	}
	m = press(t, m, "enter") // accept name; no collections -> new collection stage
	if m.saveStage != saveNewCollection {
		t.Fatalf("stage = %d", m.saveStage)
	}
	m = press(t, m, "enter") // accept "My Collection"
	if m.modal != modalNone || len(m.collections) != 1 || len(m.collections[0].Requests) != 1 {
		t.Fatalf("not saved: modal=%d cols=%+v", m.modal, m.collections)
	}
	if m.origin == nil || m.isDirty() {
		t.Fatal("editor not linked to saved request or still dirty")
	}
	if !strings.Contains(m.requestTitle(), "My Collection › GET /users") {
		t.Fatalf("title = %q", m.requestTitle())
	}

	// Edit and Ctrl+S again overwrites in place.
	m.urlInput.SetValue("api.test/users?page=2")
	if !m.isDirty() {
		t.Fatal("edit not detected as dirty")
	}
	m = press(t, m, "ctrl+s")
	if m.modal != modalNone || len(m.collections[0].Requests) != 1 || m.isDirty() {
		t.Fatal("ctrl+s did not overwrite in place")
	}
	if m.collections[0].Requests[0].URL != "api.test/users?page=2" {
		t.Fatalf("saved url = %q", m.collections[0].Requests[0].URL)
	}
}

func TestSaveAsAddsToExistingCollection(t *testing.T) {
	m := newTestModel(t)
	m.urlInput.SetValue("api.test/a")
	if err := m.saveAs("A", -1, "Col"); err != nil {
		t.Fatal(err)
	}
	m = press(t, m, "alt+s")
	if m.modal != modalSave || m.colInput.Value() != "A copy" {
		t.Fatalf("save-as default = %q", m.colInput.Value())
	}
	m = press(t, m, "enter") // name
	if m.saveStage != savePickCollection {
		t.Fatal("expected collection picker")
	}
	m = press(t, m, "enter") // pick "Col"
	if len(m.collections) != 1 || len(m.collections[0].Requests) != 2 {
		t.Fatalf("collections = %+v", m.collections)
	}
}

func TestOpenRequestFromCollectionsConfirmsDiscard(t *testing.T) {
	m := newTestModel(t)
	m.urlInput.SetValue("api.test/saved")
	m.auth.set(config.Auth{Type: config.AuthBearer, Token: "{{ secret.T }}"})
	if err := m.saveAs("Saved", -1, "Col"); err != nil {
		t.Fatal(err)
	}
	m = press(t, m, "alt+n")
	if m.origin != nil || m.urlInput.Value() != "" || m.auth.auth.Type != config.AuthNone {
		t.Fatal("alt+n did not reset to a blank request")
	}

	m.urlInput.SetValue("unsaved work")
	m = press(t, m, "alt+o")
	if m.modal != modalCollections {
		t.Fatal("collections not opened")
	}
	m = press(t, m, "down", "enter") // collection row is expanded; select request
	if m.colConfirm != confirmOpen || m.modal != modalCollections {
		t.Fatal("expected discard confirmation")
	}
	m = press(t, m, "enter")
	if m.modal != modalNone || m.urlInput.Value() != "api.test/saved" || m.auth.auth.Token != "{{ secret.T }}" {
		t.Fatalf("request not opened: url=%q auth=%+v", m.urlInput.Value(), m.auth.auth)
	}
	if m.isDirty() {
		t.Fatal("freshly opened request is dirty")
	}
}

func TestNewRequestRequiresConfirmWhenDirty(t *testing.T) {
	m := newTestModel(t)
	m.urlInput.SetValue("work in progress")
	m = press(t, m, "alt+n")
	if m.urlInput.Value() == "" {
		t.Fatal("discarded without confirmation")
	}
	m = press(t, m, "alt+n")
	if m.urlInput.Value() != "" {
		t.Fatal("second alt+n did not discard")
	}
}

func TestDeleteOpenRequestUnlinksEditor(t *testing.T) {
	m := newTestModel(t)
	m.urlInput.SetValue("api.test/x")
	if err := m.saveAs("X", -1, "Col"); err != nil {
		t.Fatal(err)
	}
	m = press(t, m, "alt+o") // cursor on the open request
	m = press(t, m, "d", "d")
	if len(m.collections[0].Requests) != 0 || m.origin != nil {
		t.Fatal("request not deleted / editor still linked")
	}
	if m.urlInput.Value() != "api.test/x" {
		t.Fatal("editor contents lost on delete")
	}
}

func TestDraftAndOriginSurviveRestart(t *testing.T) {
	m := newTestModel(t)
	m.urlInput.SetValue("api.test/x")
	if err := m.saveAs("X", -1, "Col"); err != nil {
		t.Fatal(err)
	}
	m.urlInput.SetValue("api.test/x?edited=1") // unsaved edit
	m.Close()

	cfg, err := config.Load(m.dir)
	if err != nil {
		t.Fatal(err)
	}
	cols, err := collection.Load(m.dir)
	if err != nil {
		t.Fatal(err)
	}
	m2 := New(Options{Dir: m.dir})
	m2 = update(t, m2, tea.WindowSizeMsg{Width: 120, Height: 40})
	m2 = update(t, m2, servicesMsg{secrets: secrets.NewMemory(nil), cfg: cfg, collections: cols, dir: m.dir})
	if m2.urlInput.Value() != "api.test/x?edited=1" {
		t.Fatalf("draft not restored: %q", m2.urlInput.Value())
	}
	if m2.origin == nil || !m2.isDirty() {
		t.Fatal("origin/dirty state not restored")
	}
}

func TestAuthTabEditing(t *testing.T) {
	m := newTestModel(t)
	m.reqTab = reqTabAuth
	m.setFocus(focusReqContent)
	m = press(t, m, "right") // No Auth -> Bearer
	if m.auth.auth.Type != config.AuthBearer {
		t.Fatalf("type = %q", m.auth.auth.Type)
	}
	m = press(t, m, "down", "enter")
	if !m.isTyping() {
		t.Fatal("auth field edit should count as typing")
	}
	m = typeText(t, m, "{{ secret.TOKEN }}")
	m = press(t, m, "enter")
	if m.auth.auth.Token != "{{ secret.TOKEN }}" {
		t.Fatalf("token = %q", m.auth.auth.Token)
	}
	cfg, _ := config.Load(m.dir)
	if cfg.Draft.Auth.Token != "{{ secret.TOKEN }}" {
		t.Fatal("auth not persisted to draft")
	}
}

func TestAuthAppliedEndToEnd(t *testing.T) {
	var gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
	}))
	defer srv.Close()

	m := newTestModel(t)
	m.client = httpclient.NewWithHTTPClient(srv.Client())
	_ = m.secrets.Set("TOKEN", "abc")
	m.auth.set(config.Auth{Type: config.AuthBearer, Token: "{{ secret.TOKEN }}"})
	m.urlInput.SetValue(srv.URL)
	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = update(t, next.(Model), cmd())
	if gotAuth != "Bearer abc" {
		t.Fatalf("Authorization = %q", gotAuth)
	}
	if len(m.history) != 1 {
		t.Fatal("history not recorded")
	}
	for _, h := range m.history[0].Headers {
		if strings.EqualFold(h.Key, "Authorization") {
			t.Fatal("auth leaked into history")
		}
	}
}

func TestHistoryLoadUnlinksSavedRequest(t *testing.T) {
	m := newTestModel(t)
	m.urlInput.SetValue("api.test/x")
	if err := m.saveAs("X", -1, "Col"); err != nil {
		t.Fatal(err)
	}
	m.history = []store.HistoryRecord{{Method: "GET", URL: "other"}}
	m = press(t, m, "alt+h", "enter")
	if m.origin != nil {
		t.Fatal("history load should produce an untitled draft")
	}
	if m.collections[0].Requests[0].URL != "api.test/x" {
		t.Fatal("saved request modified")
	}
}

// Smoke test: every tab, modal and auth type renders at small and large sizes
// without panicking.
func TestRenderAllScreens(t *testing.T) {
	m := newTestModel(t)
	m.urlInput.SetValue("api.test/x")
	if err := m.saveAs("X", -1, "Col"); err != nil {
		t.Fatal(err)
	}
	m.history = []store.HistoryRecord{{Method: "GET", URL: "u", StatusCode: 200}}
	for _, size := range [][2]int{{60, 20}, {200, 60}} {
		m = update(t, m, tea.WindowSizeMsg{Width: size[0], Height: size[1]})
		for _, tab := range []requestTab{reqTabParams, reqTabAuth, reqTabHeaders, reqTabBody} {
			for _, at := range config.AuthTypes {
				m.reqTab = tab
				m.auth.set(config.Auth{Type: at, Token: "plain", Password: "pw", Value: "v"})
				m.focus = focusReqContent
				_ = m.View()
			}
		}
		for _, md := range []modalKind{modalHistory, modalSecrets, modalEnv, modalHelp, modalCollections, modalSave, modalImport} {
			m.modal = md
			if md == modalEnv {
				m.viewEnv(0)
			}
			if out := m.View(); out == "" {
				t.Fatalf("modal %d rendered empty", md)
			}
		}
		m.modal = modalNone
	}
}

func TestDefaultRequestName(t *testing.T) {
	for in, want := range map[string]string{
		"https://api.test/users?x=1": "GET /users",
		"api.test":                   "GET api.test",
		"":                           "GET request",
		"{{BASE_URL}}/items/1":       "GET /items/1",
	} {
		if got := defaultRequestName("GET", in); got != want {
			t.Errorf("defaultRequestName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNewHeaderDiscardedOnTab(t *testing.T) {
	m := newTestModel(t)
	before := len(m.headers.items)
	m = press(t, m, "alt+l", "n", "tab")
	if len(m.headers.items) != before {
		t.Fatalf("empty placeholder row left behind: %+v", m.headers.items)
	}
}

func TestHistoryLoadMergesHeaders(t *testing.T) {
	m := newTestModel(t)
	m.headers.setItems([]config.KeyValue{{Key: "Authorization", Value: "Bearer x", Enabled: true}})
	m.history = []store.HistoryRecord{{Method: "POST", URL: "example.com", Headers: []config.KeyValue{{Key: "Accept", Value: "json", Enabled: true}}}}
	m = press(t, m, "alt+h", "enter")
	if m.method != "POST" || m.urlInput.Value() != "example.com" {
		t.Fatalf("history not loaded: %s %s", m.method, m.urlInput.Value())
	}
	if len(m.headers.items) != 2 || m.headers.items[0].Key != "Authorization" {
		t.Fatalf("headers not merged: %+v", m.headers.items)
	}
}

func TestEndToEndRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"hello":"world"}`))
	}))
	defer srv.Close()

	m := newTestModel(t)
	m.client = httpclient.NewWithHTTPClient(srv.Client())
	m.urlInput.SetValue(srv.URL)

	next, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	m = next.(Model)
	if !m.loading || cmd == nil {
		t.Fatal("request not started")
	}
	m = update(t, m, cmd())
	if m.loading || m.statusCode != 200 || !strings.Contains(m.respBody, "hello") {
		t.Fatalf("bad result: loading=%v status=%d body=%q", m.loading, m.statusCode, m.respBody)
	}
	if len(m.history) != 1 {
		t.Fatalf("history not recorded: %d", len(m.history))
	}
}

func TestCancelledRequestReportsCancelled(t *testing.T) {
	m := newTestModel(t)
	m.reqID = 1
	m.loading = true
	m = update(t, m, responseMsg{id: 1, err: context.Canceled})
	if m.status != "Cancelled" || m.loading {
		t.Fatalf("status = %q loading=%v", m.status, m.loading)
	}
}

func TestSecretsModalTypingQuestionMark(t *testing.T) {
	m := newTestModel(t)
	m = press(t, m, "alt+k", "n")
	m = typeText(t, m, "TOKEN")
	m = press(t, m, "enter")
	m = typeText(t, m, "a?b")
	m = press(t, m, "enter")
	if m.modal != modalSecrets {
		t.Fatal("secrets modal closed while typing")
	}
	if v, err := m.secrets.Get("TOKEN"); err != nil || v != "a?b" {
		t.Fatalf("secret = %q, %v", v, err)
	}
	if !m.secretIsSet["TOKEN"] {
		t.Fatal("secret state cache not refreshed")
	}
}

func TestCycleMethodLeavesBodyTab(t *testing.T) {
	m := newTestModel(t)
	m.method = "DELETE"
	m.reqTab = reqTabBody
	m = press(t, m, "alt+m") // -> HEAD
	if m.method != "HEAD" || m.reqTab == reqTabBody {
		t.Fatalf("method=%s tab=%d", m.method, m.reqTab)
	}
}

func TestViewRenders(t *testing.T) {
	m := newTestModel(t)
	if out := m.View(); !strings.Contains(out, "Send") {
		t.Fatal("main view missing Send button")
	}
	m = update(t, m, tea.WindowSizeMsg{Width: 30, Height: 10})
	if out := m.View(); !strings.Contains(out, "too small") {
		t.Fatal("expected too-small message")
	}
}

func TestMergeKV(t *testing.T) {
	existing := []config.KeyValue{{Key: "Authorization", Value: "Bearer x", Enabled: true}, {Key: "Accept", Value: "text/plain"}}
	got := mergeKV(existing, []config.KeyValue{{Key: "accept", Value: "application/json", Enabled: true}, {Key: "X-New", Value: "1", Enabled: true}})
	if len(got) != 3 || got[0].Value != "Bearer x" || got[1].Value != "application/json" || !got[1].Enabled {
		t.Fatalf("bad merge: %+v", got)
	}
	if existing[1].Value != "text/plain" {
		t.Fatal("mergeKV mutated its input")
	}
}

func TestTruncate(t *testing.T) {
	for _, tt := range []struct {
		in   string
		n    int
		want string
	}{{"hello", 10, "hello"}, {"hello", 3, "he…"}, {"héllo", 2, "h…"}, {"x", 0, ""}} {
		if got := truncate(tt.in, tt.n); got != tt.want {
			t.Errorf("truncate(%q,%d) = %q, want %q", tt.in, tt.n, got, tt.want)
		}
	}
}
