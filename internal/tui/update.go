package tui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/httpclient"
)

type responseMsg struct {
	id      int
	method  string
	rawURL  string
	rawBody string
	res     *httpclient.Response
	err     error
}

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth, m.termHeight = msg.Width, msg.Height
		m.layout()
		return m, nil

	case servicesMsg:
		m.secrets = msg.secrets
		m.store = msg.store
		m.history = msg.history
		m.dir = msg.dir
		m.ready = true
		m.applyConfig(msg.cfg, msg.collections)
		if msg.warn != "" {
			m.setStatus("Ready • "+msg.warn, true)
		} else {
			m.setStatus("Ready", false)
		}
		return m, nil

	case responseMsg:
		m.handleResponse(msg)
		return m, nil

	case editorFinishedMsg:
		if msg.err != nil {
			m.setStatus("Editor failed: "+msg.err.Error(), true)
			return m, nil
		}
		m.bodyInput.SetValue(msg.body)
		m.bodyInput.CursorEnd()
		m.persistOrWarn()
		m.setStatus("✓ Body updated from $EDITOR", false)
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	// Route everything else (cursor blink etc.) to the focused component.
	return m, m.updateFocused(msg)
}

func (m *Model) updateFocused(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	switch {
	case m.modal == modalSecrets && m.secretStage != secretList:
		m.secretInput, cmd = m.secretInput.Update(msg)
	case m.modal == modalCollections || m.modal == modalSave:
		m.colInput, cmd = m.colInput.Update(msg)
	case m.modal == modalImport:
		m.importInput, cmd = m.importInput.Update(msg)
	case m.modal == modalEnv && m.envInputMode != envInputNone:
		m.envInput, cmd = m.envInput.Update(msg)
	case m.modal == modalEnv && m.envVars.editing:
		m.envVars.input, cmd = m.envVars.input.Update(msg)
	case m.modal != modalNone:
	case m.focus == focusURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case m.focus == focusReqContent && m.reqTab == reqTabBody && m.supportsBody():
		switch m.bodyMode {
		case config.BodyURLEncoded, config.BodyMultipart:
			if m.form.editing {
				m.form.input, cmd = m.form.input.Update(msg)
			}
		case config.BodyFile:
			m.bodyFileInput, cmd = m.bodyFileInput.Update(msg)
		default:
			m.bodyInput, cmd = m.bodyInput.Update(msg)
		}
	case m.focus == focusReqContent && m.reqTab == reqTabHeaders && m.headers.editing:
		m.headers.input, cmd = m.headers.input.Update(msg)
	case m.focus == focusReqContent && m.reqTab == reqTabParams && m.params.editing:
		m.params.input, cmd = m.params.input.Update(msg)
	case m.focus == focusReqContent && m.reqTab == reqTabAuth && m.auth.editing:
		m.auth.input, cmd = m.auth.input.Update(msg)
	case m.focus == focusResContent && m.resp.inputMode != respInputNone:
		m.resp.input, cmd = m.resp.input.Update(msg)
	case m.focus == focusResContent:
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return cmd
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Keys that work everywhere, including inside modals.
	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Cancel):
		m.cancelInFlight()
		return m, nil
	case key.Matches(msg, m.keys.Send):
		if m.loading {
			m.cancelInFlight()
			return m, nil
		}
		return m, m.startRequest()
	}

	if m.modal != modalNone {
		return m.updateModal(msg)
	}

	// Alt+N needs two presses when there are unsaved changes; any other key
	// resets the confirmation.
	confirmedNew := m.confirmNew
	m.confirmNew = false

	// Global shortcuts (Alt/Ctrl chords are safe while typing).
	switch {
	case key.Matches(msg, m.keys.Save):
		m.cancelInlineEdits()
		return m, m.saveRequest()
	case key.Matches(msg, m.keys.SaveAs):
		m.cancelInlineEdits()
		return m, m.openSaveDialog()
	case key.Matches(msg, m.keys.NewRequest):
		m.cancelInlineEdits()
		if !m.newRequest(confirmedNew) {
			m.confirmNew = true
			return m, nil
		}
		return m, m.setFocus(focusURL)
	case key.Matches(msg, m.keys.Collections):
		m.cancelInlineEdits()
		m.openCollections()
		return m, nil
	case key.Matches(msg, m.keys.Import):
		m.cancelInlineEdits()
		return m, m.openImport()
	case key.Matches(msg, m.keys.CopyCurl):
		m.cancelInlineEdits()
		m.copyAsCurl()
		return m, nil
	case key.Matches(msg, m.keys.History):
		m.cancelInlineEdits()
		m.modal = modalHistory
		m.historyCursor = 0
		return m, nil
	case key.Matches(msg, m.keys.CycleEnv):
		m.envIndex = (m.envIndex + 1) % len(m.cfg.Environments)
		m.setStatus("Switched to "+m.env().Name, false)
		m.persistOrWarn()
		return m, nil
	case key.Matches(msg, m.keys.Secrets):
		m.cancelInlineEdits()
		m.openSecrets()
		return m, nil
	case key.Matches(msg, m.keys.EnvVars):
		m.cancelInlineEdits()
		m.openEnvEditor()
		return m, nil
	case key.Matches(msg, m.keys.Headers):
		m.cancelInlineEdits()
		m.reqTab = reqTabHeaders
		return m, m.setFocus(focusReqContent)
	case key.Matches(msg, m.keys.CycleMethod):
		m.cycleMethod()
		return m, nil
	case key.Matches(msg, m.keys.CycleBodyMode):
		m.cycleBodyMode()
		return m, nil
	case msg.String() == "ctrl+o" && m.focus == focusReqContent && m.reqTab == reqTabBody && m.supportsBody() && m.bodyMode == config.BodyRaw:
		return m, openBodyEditorCmd(m.bodyInput.Value())
	case key.Matches(msg, m.keys.Help) && !m.isTyping():
		m.modal = modalHelp
		return m, nil
	case key.Matches(msg, m.keys.NextFocus), key.Matches(msg, m.keys.PrevFocus):
		m.cancelInlineEdits()
		return m, m.cycleFocus(key.Matches(msg, m.keys.NextFocus))
	case (m.focus == focusReqTabs || m.focus == focusResTabs) &&
		(key.Matches(msg, m.keys.TabLeft) || key.Matches(msg, m.keys.TabRight)):
		m.moveTab(key.Matches(msg, m.keys.TabRight))
		return m, nil
	case m.focus == focusURL && msg.String() == "enter":
		if m.loading {
			return m, nil
		}
		return m, m.startRequest()
	}

	// Focused component.
	var cmd tea.Cmd
	switch m.focus {
	case focusURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
	case focusReqContent:
		switch m.reqTab {
		case reqTabBody:
			if !m.supportsBody() {
				break
			}
			switch m.bodyMode {
			case config.BodyURLEncoded, config.BodyMultipart:
				var changed bool
				if cmd, changed = m.form.Update(msg); changed {
					m.persistOrWarn()
				}
			case config.BodyFile:
				m.bodyFileInput, cmd = m.bodyFileInput.Update(msg)
			default:
				m.bodyInput, cmd = m.bodyInput.Update(msg)
			}
		case reqTabHeaders:
			var changed bool
			if cmd, changed = m.headers.Update(msg); changed {
				m.persistOrWarn()
			}
		case reqTabParams:
			var changed bool
			if cmd, changed = m.params.Update(msg); changed {
				m.persistOrWarn()
			}
		case reqTabAuth:
			var changed bool
			if cmd, changed = m.auth.Update(msg); changed {
				m.persistOrWarn()
			}
		}
	case focusResContent:
		var handled bool
		if cmd, handled = m.updateResponseKeys(msg); !handled {
			m.viewport, cmd = m.viewport.Update(msg)
		}
	}
	return m, cmd
}

func (m *Model) cancelInFlight() {
	if m.loading && m.cancelReq != nil {
		m.cancelReq()
		m.setStatus("Cancelling...", true)
		m.loading = false
	}
}

// startRequest resets response state and launches the HTTP request.
func (m *Model) startRequest() tea.Cmd {
	if !m.ready {
		m.setStatus("Services still loading...", true)
		return nil
	}
	if m.cancelReq != nil {
		m.cancelReq()
	}
	m.reqID++
	m.loading = true
	m.setStatus("Sending...", false)
	m.respHeaders, m.respBody = "", ""
	m.respSize, m.statusCode = 0, 0
	m.stopRespInput()
	m.resp.clear()
	m.syncViewport()

	ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
	m.cancelReq = cancel

	// Copy everything the goroutine touches so it never reads model state
	// that Update may mutate concurrently.
	env := m.activeVars()
	req := httpclient.FromConfig(m.currentRequest(), env, m.secrets)
	if req.Timeout > 0 {
		ctx, cancel = context.WithTimeout(context.Background(), req.Timeout)
		m.cancelReq = cancel
	}
	m.lastTimeout = m.timeout
	if req.Timeout > 0 {
		m.lastTimeout = req.Timeout
	}
	// Persist the draft so in-progress work survives a crash.
	m.persistOrWarn()
	id, client := m.reqID, m.client
	return func() tea.Msg {
		res, err := client.Do(ctx, req)
		return responseMsg{id: id, method: req.Method, rawURL: req.URL, rawBody: req.Body, res: res, err: err}
	}
}

func (m *Model) handleResponse(msg responseMsg) {
	if msg.id != m.reqID {
		return // stale response from a cancelled/superseded request
	}
	m.loading = false
	if m.cancelReq != nil {
		m.cancelReq()
		m.cancelReq = nil
	}
	res := msg.res
	if res == nil {
		res = &httpclient.Response{}
	}
	m.lastTarget = res.URL
	m.responseTime = res.Duration
	m.statusCode = res.StatusCode
	m.respSize = len(res.Body)

	if msg.err != nil {
		m.statusErr = true
		m.resp.clear()
		m.respHeaders = ""
		m.respSize = 0
		m.statusCode = 0
		switch {
		case errors.Is(msg.err, context.Canceled):
			m.status = "Cancelled"
			m.respBody = statusRedStyle.Render("✗ Request cancelled by user")
		case errors.Is(msg.err, context.DeadlineExceeded):
			m.status = "Timeout"
			m.respBody = statusRedStyle.Render(fmt.Sprintf("✗ Request timed out after %s", m.lastTimeout))
		default:
			m.status = "Error"
			m.respBody = statusRedStyle.Render(fmt.Sprintf("✗ %v", msg.err))
		}
		m.syncViewport()
		return
	}

	m.setStatus(fmt.Sprintf("%d %s • %s", res.StatusCode, http.StatusText(res.StatusCode),
		res.Duration.Round(time.Millisecond)), res.StatusCode >= 400)
	m.resp.set(res)

	if m.store != nil {
		if err := m.store.SaveHistory(msg.method, msg.rawURL, msg.rawBody, res.RawHeaders, res.StatusCode); err != nil {
			m.status += " • history save failed"
		} else if h, err := m.store.LoadHistory(historyLoadLimit); err == nil {
			m.history = h
		}
	}
	m.syncViewport()
}

func highlight(src, lexer string) string {
	var buf bytes.Buffer
	if err := quick.Highlight(&buf, src, lexer, "terminal256", "dracula"); err != nil {
		return src
	}
	return buf.String()
}

func formatResponseHeaders(h http.Header) string {
	if len(h) == 0 {
		return ""
	}
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	keyStyle := lipgloss.NewStyle().Foreground(keyColor)
	valStyle := lipgloss.NewStyle().Foreground(textSecondary)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s: %s\n", keyStyle.Render(k), valStyle.Render(strings.Join(h[k], ", ")))
	}
	return b.String()
}

func formatSize(n int) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
