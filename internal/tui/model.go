// Package tui implements PostBoy's Bubble Tea terminal UI.
package tui

import (
	"context"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/httpclient"
	"github.com/dheeraj080/PostBoy/internal/secrets"
	"github.com/dheeraj080/PostBoy/internal/store"
)

const historyLoadLimit = 100

type requestTab int

const (
	reqTabParams requestTab = iota
	reqTabAuth
	reqTabHeaders
	reqTabBody
)

type responseTab int

const (
	resTabBody responseTab = iota
	resTabHeaders
)

type focusArea int

const (
	focusURL focusArea = iota
	focusReqTabs
	focusReqContent
	focusResTabs
	focusResContent
)

var focusOrder = []focusArea{focusURL, focusReqTabs, focusReqContent, focusResTabs, focusResContent}

type modalKind int

const (
	modalNone modalKind = iota
	modalHistory
	modalSecrets
	modalEnv
	modalHelp
	modalCollections
	modalSave
	modalImport
)

type secretStage int

const (
	secretList secretStage = iota
	secretEnterName
	secretEnterValue
)

// Options configures the UI.
type Options struct {
	// Dir overrides the data directory (defaults to config.Dir()).
	Dir string
	// Version is shown in the help screen.
	Version string
	// ExportDir is where collection exports are written (default: cwd).
	ExportDir string
}

// Model is the root Bubble Tea model.
type Model struct {
	opts   Options
	keys   keyMap
	client *httpclient.Client

	// Request editor.
	method        string
	urlInput      textinput.Model
	bodyInput     textarea.Model
	headers       kvList
	params        kvList
	auth          authEditor
	bodyMode      config.BodyMode
	form          kvList
	bodyFileInput textinput.Model

	// Saved requests.
	collections   []collection.Collection
	origin        *config.Origin // saved request the editor is linked to
	requestName   string
	savedSnapshot config.Request // baseline for dirty tracking
	confirmNew    bool           // Alt+N pressed once with unsaved changes

	// Response view.
	resp         responseView
	viewport     viewport.Model
	respBody     string
	respHeaders  string
	respSize     int
	responseTime time.Duration
	statusCode   int
	lastTarget   string

	// Configuration and services.
	cfg      config.Config
	dir      string
	secrets  *secrets.Store
	store    *store.Store
	ready    bool
	timeout  time.Duration
	envIndex int

	// In-flight request.
	loading   bool
	reqID     int
	cancelReq context.CancelFunc

	// Layout and focus.
	reqTab     requestTab
	resTab     responseTab
	focus      focusArea
	termWidth  int
	termHeight int

	// Status bar.
	status    string
	statusErr bool

	// Modals.
	modal         modalKind
	history       []store.HistoryRecord
	historyCursor int
	secretStage   secretStage
	secretCursor  int
	secretName    string
	secretInput   textinput.Model
	secretIsSet   map[string]bool

	colCursor    int
	colExpanded  map[string]bool
	colInputMode colInputMode
	colInput     textinput.Model
	colConfirm   colConfirm
	saveStage    saveStage
	saveName     string
	saveCursor   int

	envView          int // environment shown in the editor
	envVars          kvList
	envInputMode     envInputMode
	envInput         textinput.Model
	envConfirmDelete bool

	importInput   textinput.Model
	importConfirm bool
}

// New returns the initial model. Services (config, keychain, database) are
// loaded asynchronously from Init.
func New(opts Options) Model {
	ti := textinput.New()
	ti.Placeholder = "https://api.example.com/users"
	ti.CharLimit = 4096
	ti.Prompt = ""
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(textSecondary)
	ti.Focus()

	bi := textarea.New()
	bi.Placeholder = "{\n  \"key\": \"value\"\n}"
	bi.ShowLineNumbers = false
	bi.CharLimit = 0
	bi.MaxHeight = 0
	bi.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(textSecondary)

	si := textinput.New()
	si.CharLimit = 4096

	m := Model{
		opts:          opts,
		keys:          defaultKeyMap(),
		client:        httpclient.New(),
		method:        "GET",
		urlInput:      ti,
		bodyInput:     bi,
		headers:       newKVList(":", "header"),
		params:        newKVList("=", "parameter"),
		auth:          newAuthEditor(),
		bodyMode:      config.BodyRaw,
		form:          newFormList(),
		bodyFileInput: newBodyFileInput(),
		viewport:      viewport.New(0, 0),
		resp:          responseView{input: newRespInput()},
		secretInput:   si,
		colInput:      newColInput(),
		colExpanded:   map[string]bool{},
		envVars:       newEnvVarList(),
		envInput:      newEnvInput(),
		importInput:   newImportInput(),
		focus:         focusURL,
		reqTab:        reqTabHeaders,
		resTab:        resTabBody,
		status:        "Initializing...",
	}
	m.applyConfig(config.Default(), nil)
	return m
}

// Close persists the draft and releases resources (cancels in-flight
// requests, closes the DB).
func (m Model) Close() {
	if m.cancelReq != nil {
		m.cancelReq()
	}
	if m.ready {
		_ = m.persistConfig()
	}
	_ = m.store.Close()
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, loadServices(m.opts.Dir))
}

type servicesMsg struct {
	secrets     *secrets.Store
	store       *store.Store
	history     []store.HistoryRecord
	cfg         config.Config
	collections []collection.Collection
	dir         string
	warn        string
}

func loadServices(dir string) tea.Cmd {
	return func() tea.Msg {
		if dir == "" {
			dir = config.Dir()
		}
		var warns []string

		cfg, err := config.Load(dir)
		if err != nil {
			warns = append(warns, "config: "+err.Error())
		}

		cols, err := collection.Load(dir)
		if err != nil {
			warns = append(warns, "collections: "+err.Error())
		}

		sec := secrets.New()
		if !sec.Persistent() {
			warns = append(warns, "OS keychain unavailable: secrets will NOT persist after exit")
		}

		st, err := store.Open(dir)
		if err != nil {
			warns = append(warns, "storage: "+err.Error())
		}
		var hist []store.HistoryRecord
		if st != nil {
			if hist, err = st.LoadHistory(historyLoadLimit); err != nil {
				warns = append(warns, "history: "+err.Error())
			}
		}
		return servicesMsg{secrets: sec, store: st, history: hist, cfg: cfg, collections: cols, dir: dir, warn: strings.Join(warns, "; ")}
	}
}

// applyConfig loads settings and the draft. The draft's origin is kept only
// if the saved request still exists in cols.
func (m *Model) applyConfig(c config.Config, cols []collection.Collection) {
	c = config.Normalize(c)
	m.cfg = c
	m.envIndex = c.ActiveEnv
	m.timeout = c.Timeout()
	m.collections = cols

	var origin *config.Origin
	saved := config.Request{}
	if c.Origin != nil {
		if ci, ri := collection.Find(cols, c.Origin.CollectionID, c.Origin.RequestID); ri >= 0 {
			o := *c.Origin
			origin = &o
			saved = cols[ci].Requests[ri]
		}
	}
	m.loadRequest(c.Draft, origin, saved)
}

func (m *Model) env() config.Environment { return m.cfg.Environments[m.envIndex] }

func (m *Model) persistConfig() error {
	m.cfg.ActiveEnv = m.envIndex
	m.cfg.Draft = m.currentRequest()
	m.cfg.Origin = m.origin
	if m.dir == "" {
		return nil
	}
	return config.Save(m.dir, m.cfg)
}

func (m *Model) persistOrWarn() {
	if err := m.persistConfig(); err != nil {
		m.setStatus("Could not save config: "+err.Error(), true)
	}
}

func (m *Model) setStatus(msg string, isErr bool) {
	m.status = msg
	m.statusErr = isErr
}

func (m *Model) supportsBody() bool { return httpclient.SupportsBody(m.method) }

// isTyping reports whether the focused element is a free-text input, in
// which case single-character shortcuts must not be intercepted.
func (m *Model) isTyping() bool {
	switch m.focus {
	case focusURL:
		return true
	case focusResContent:
		return m.resp.inputMode != respInputNone
	case focusReqContent:
		switch m.reqTab {
		case reqTabBody:
			if !m.supportsBody() {
				return false
			}
			switch m.bodyMode {
			case config.BodyRaw, config.BodyFile:
				return true
			default:
				return m.form.editing
			}
		case reqTabHeaders:
			return m.headers.editing
		case reqTabParams:
			return m.params.editing
		case reqTabAuth:
			return m.auth.editing
		}
	}
	return false
}

func (m *Model) cancelInlineEdits() {
	m.headers.cancelEdit()
	m.params.cancelEdit()
	m.auth.cancelEdit()
	m.form.cancelEdit()
	m.bodyFileInput.Blur()
	m.stopRespInput()
}

func (m *Model) setFocus(f focusArea) tea.Cmd {
	m.focus = f
	m.urlInput.Blur()
	m.bodyInput.Blur()
	m.form.input.Blur()
	m.bodyFileInput.Blur()
	switch {
	case f == focusURL:
		return m.urlInput.Focus()
	case f == focusReqContent && m.reqTab == reqTabBody && m.supportsBody():
		switch m.bodyMode {
		case config.BodyFile:
			return m.bodyFileInput.Focus()
		case config.BodyURLEncoded, config.BodyMultipart:
			return nil
		default:
			return m.bodyInput.Focus()
		}
	}
	return nil
}

func (m *Model) cycleFocus(forward bool) tea.Cmd {
	idx := 0
	for i, f := range focusOrder {
		if f == m.focus {
			idx = i
			break
		}
	}
	if forward {
		idx = (idx + 1) % len(focusOrder)
	} else {
		idx = (idx - 1 + len(focusOrder)) % len(focusOrder)
	}
	return m.setFocus(focusOrder[idx])
}

func (m *Model) moveTab(forward bool) {
	switch m.focus {
	case focusReqTabs:
		maxTab := reqTabBody
		if !m.supportsBody() {
			maxTab = reqTabHeaders
		}
		if forward && m.reqTab < maxTab {
			m.reqTab++
		} else if !forward && m.reqTab > 0 {
			m.reqTab--
		}
	case focusResTabs:
		prev := m.resTab
		if forward && m.resTab < resTabHeaders {
			m.resTab++
		} else if !forward && m.resTab > 0 {
			m.resTab--
		}
		if prev != m.resTab {
			m.syncViewport()
		}
	}
}

func (m *Model) cycleMethod() {
	for i, meth := range httpclient.Methods {
		if meth == m.method {
			m.method = httpclient.Methods[(i+1)%len(httpclient.Methods)]
			break
		}
	}
	if !m.supportsBody() && m.reqTab == reqTabBody {
		m.reqTab = reqTabHeaders
		m.setFocus(m.focus)
	}
}

// panelHeights returns the content heights of the request and response panels.
func (m *Model) panelHeights() (int, int) {
	// top bar (3) + info bar (1) + blank (1) + req tabs (2) + req panel border (1)
	// + res header (2) + res panel border (1) + footer (1) + slack (1)
	const fixedVerticalSpace = 13
	available := max(m.termHeight-fixedVerticalSpace, 10)
	req := available / 2
	return req, available - req
}

func (m *Model) usableWidth() int { return max(m.termWidth-4, 20) }

// layout sizes stateful components. It runs in Update (never View) so sizes
// persist on the model and scrolling works.
func (m *Model) layout() {
	w := m.usableWidth()
	reqH, resH := m.panelHeights()
	bodyH := max(reqH-1, 1)
	if m.reqTab == reqTabBody && m.supportsBody() {
		bodyH = max(reqH-4, 1)
	}
	m.bodyInput.SetWidth(w - 2)
	m.bodyInput.SetHeight(bodyH)
	m.viewport.Width = w - 2
	m.viewport.Height = max(resH-1, 1)
	m.urlInput.Width = max(w-30, 10)
}

// syncViewport loads the active response tab's content into the viewport
// and scrolls to the top.
func (m *Model) syncViewport() { m.refreshResponse(false) }
