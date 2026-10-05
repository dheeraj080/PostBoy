package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zalando/go-keyring"
	_ "modernc.org/sqlite"
)

const (
	serviceName          = "postboy"
	maxResponseBodyBytes = 10 << 20 // 10 MB
	defaultTimeout       = 15 * time.Second
	maxHistoryRows       = 500
	configFileName       = "config.json"
)

// --- POSTMAN-LIKE TABS & FOCUS ---
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

type activeFocus int

const (
	focusURL activeFocus = iota
	focusReqTabs
	focusReqContent
	focusResTabs
	focusResContent
)

// --- SHARED HTTP CLIENT ---
var httpClient = &http.Client{
	Transport: &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        10,
		IdleConnTimeout:     60 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	},
}

// --- PRECOMPILED REGEXES ---
var (
	templateRegex   = regexp.MustCompile(`\{\{\s*(?:secret[\.\s"]+([a-zA-Z0-9_]+)"?|\.?([a-zA-Z0-9_]+))\s*\}\}`)
	secretNameRegex = regexp.MustCompile(`^[A-Za-z0-9_]+$`)
)

// --- STYLES (Postman-inspired) ---
var (
	postmanOrange      = lipgloss.Color("#FF6C37")
	postmanOrangeDark  = lipgloss.Color("#E55A2B")
	darkBg             = lipgloss.Color("#1E1E1E")
	panelBg            = lipgloss.Color("#252526")
	borderColor        = lipgloss.Color("#3E3E42")
	focusedBorderColor = postmanOrange

	// Text colors
	textPrimary   = lipgloss.Color("#CCCCCC")
	textSecondary = lipgloss.Color("#858585")
	textMuted     = lipgloss.Color("#6E6E6E")

	// Status colors
	statusSuccess = lipgloss.Color("#4EC9B0")
	statusError   = lipgloss.Color("#F48771")
	statusWarn    = lipgloss.Color("#DCDCAA")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#569CD6"))

	methodStyle = lipgloss.NewStyle().
			Background(postmanOrange).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 2).
			MarginRight(1)

	sendButtonStyle = lipgloss.NewStyle().
			Background(postmanOrange).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 3).
			MarginLeft(1)

	sendButtonLoadingStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#C75450")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Padding(0, 3).
				MarginLeft(1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(textSecondary).
				Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(postmanOrange).
			Bold(true).
			Padding(0, 2).
			BorderBottom(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(postmanOrange)

	urlInputStyle = lipgloss.NewStyle().
			Background(panelBg).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	focusedURLInputStyle = urlInputStyle.BorderForeground(postmanOrange)

	panelStyle = lipgloss.NewStyle().
			Background(panelBg).
			Border(lipgloss.NormalBorder()).
			BorderForeground(borderColor)

	focusedPanelStyle = panelStyle.BorderForeground(postmanOrange)

	statusGreenStyle  = lipgloss.NewStyle().Foreground(statusSuccess)
	statusRedStyle    = lipgloss.NewStyle().Foreground(statusError)
	statusYellowStyle = lipgloss.NewStyle().Foreground(statusWarn)
	helpStyle         = lipgloss.NewStyle().Foreground(textMuted)
	dimStyle          = lipgloss.NewStyle().Foreground(textSecondary)

	envBadgeStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("#007ACC")).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 1).
			MarginRight(1)

	responseMetaStyle = lipgloss.NewStyle().
				Foreground(textSecondary).
				Padding(0, 1).
				MarginLeft(2)
)

// --- KEYRING ---
type KeyringManager struct {
	mu            sync.Mutex
	fallbackStore map[string]string
	useFallback   bool
}

func NewKeyringManager() *KeyringManager {
	km := &KeyringManager{fallbackStore: make(map[string]string)}
	if err := keyring.Set(serviceName, "__test_ping__", "pong"); err != nil {
		km.useFallback = true
	} else {
		_ = keyring.Delete(serviceName, "__test_ping__")
	}
	return km
}

func (km *KeyringManager) SetSecret(key, val string) error {
	if km.useFallback {
		km.mu.Lock()
		km.fallbackStore[key] = val
		km.mu.Unlock()
		return nil
	}
	return keyring.Set(serviceName, key, val)
}

func (km *KeyringManager) GetSecret(key string) (string, error) {
	if km.useFallback {
		km.mu.Lock()
		defer km.mu.Unlock()
		if val, ok := km.fallbackStore[key]; ok {
			return val, nil
		}
		return "", fmt.Errorf("secret not found")
	}
	return keyring.Get(serviceName, key)
}

func (km *KeyringManager) DeleteSecret(key string) error {
	if km.useFallback {
		km.mu.Lock()
		delete(km.fallbackStore, key)
		km.mu.Unlock()
		return nil
	}
	return keyring.Delete(serviceName, key)
}

// --- SENSITIVE HEADER REDACTOR ---
var sensitiveHeaderKeys = map[string]bool{
	"authorization": true, "x-api-key": true, "api-key": true,
	"cookie": true, "set-cookie": true, "x-auth-token": true,
}

const RedactedPlaceholder = "[REDACTED]"

func RedactHeaderValue(key, val string) string {
	if sensitiveHeaderKeys[strings.ToLower(key)] {
		return RedactedPlaceholder
	}
	return val
}

func RedactHeaders(headers map[string]string) map[string]string {
	redacted := make(map[string]string, len(headers))
	for k, v := range headers {
		redacted[k] = RedactHeaderValue(k, v)
	}
	return redacted
}

// --- CONFIG ---
type Environment struct {
	Name string            `json:"name"`
	Vars map[string]string `json:"vars"`
}

type HeaderEntry struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type Config struct {
	Environments   []Environment `json:"environments"`
	ActiveEnv      int           `json:"active_env"`
	Headers        []HeaderEntry `json:"headers"`
	SecretNames    []string      `json:"secret_names"`
	TimeoutSeconds int           `json:"timeout_seconds"`
}

func defaultConfig() Config {
	return Config{
		Environments: []Environment{
			{Name: "Development", Vars: map[string]string{"BASE_URL": "http://localhost:8080"}},
			{Name: "Staging", Vars: map[string]string{"BASE_URL": "https://staging.api.github.com"}},
			{Name: "Production", Vars: map[string]string{"BASE_URL": "https://api.github.com"}},
		},
		ActiveEnv: 2,
		Headers: []HeaderEntry{
			{Key: "User-Agent", Value: "PostBoy/4.0", Enabled: true},
			{Key: "Content-Type", Value: "application/json", Enabled: true},
		},
		TimeoutSeconds: int(defaultTimeout / time.Second),
	}
}

func normalizeConfig(c Config) Config {
	d := defaultConfig()
	if len(c.Environments) == 0 {
		c.Environments = d.Environments
		c.ActiveEnv = d.ActiveEnv
	}
	if c.ActiveEnv < 0 || c.ActiveEnv >= len(c.Environments) {
		c.ActiveEnv = 0
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = d.TimeoutSeconds
	}
	return c
}

func appDir() string {
	configDir, err := os.UserConfigDir()
	if err != nil || configDir == "" {
		configDir = "."
	}
	return filepath.Join(configDir, "postboy")
}

func loadConfig(dir string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(dir, configFileName))
	if errors.Is(err, os.ErrNotExist) {
		return defaultConfig(), nil
	}
	if err != nil {
		return defaultConfig(), err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return defaultConfig(), fmt.Errorf("parse %s: %w", configFileName, err)
	}
	return normalizeConfig(c), nil
}

func saveConfig(dir string, c Config) error {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	_ = os.Chmod(tmpName, 0o600)
	return os.Rename(tmpName, filepath.Join(dir, configFileName))
}

// --- MESSAGES & RECORDS ---
type responseMsg struct {
	statusCode   int
	headers      http.Header
	body         string
	truncated    bool
	interpolated string
	err          error
	method       string
	rawURL       string
	rawBody      string
	reqHeaders   map[string]string
	responseTime time.Duration
}

type initServicesMsg struct {
	km      *KeyringManager
	storage *Storage
	history []HistoryRecord
	cfg     Config
	dir     string
	warn    string
}

type HistoryRecord struct {
	ID, StatusCode               int
	Method, URL, Body, CreatedAt string
	Headers                      []HeaderEntry
}

// --- PERSISTENCE ---
type Storage struct {
	db *sql.DB
}

func openStorage(dir string) (*Storage, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create config dir: %w", err)
	}
	path := filepath.Join(dir, "postboy.db")

	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create db file: %w", err)
	}
	f.Close()
	_ = os.Chmod(path, 0o600)

	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT, method TEXT, url TEXT,
		headers_json TEXT, status_code INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`); err != nil {
		db.Close()
		return nil, fmt.Errorf("create history table: %w", err)
	}
	if _, err := db.Exec(`ALTER TABLE history ADD COLUMN body TEXT`); err != nil &&
		!strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		db.Close()
		return nil, fmt.Errorf("migrate history table: %w", err)
	}
	return &Storage{db: db}, nil
}

func (s *Storage) Close() {
	if s != nil && s.db != nil {
		s.db.Close()
	}
}

func (s *Storage) SaveHistory(method, url, body string, headers map[string]string, statusCode int) error {
	if s == nil || s.db == nil {
		return nil
	}
	headersJSON, _ := json.Marshal(RedactHeaders(headers))
	if _, err := s.db.Exec(
		"INSERT INTO history (method, url, body, headers_json, status_code) VALUES (?, ?, ?, ?, ?)",
		method, url, body, string(headersJSON), statusCode); err != nil {
		return err
	}
	_, err := s.db.Exec(
		"DELETE FROM history WHERE id NOT IN (SELECT id FROM history ORDER BY id DESC LIMIT ?)", maxHistoryRows)
	return err
}

func (s *Storage) LoadHistory() ([]HistoryRecord, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	rows, err := s.db.Query(
		"SELECT id, method, url, COALESCE(body, ''), COALESCE(headers_json, '{}'), status_code, created_at FROM history ORDER BY id DESC LIMIT 50")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []HistoryRecord
	for rows.Next() {
		var r HistoryRecord
		var headersJSON string
		if err := rows.Scan(&r.ID, &r.Method, &r.URL, &r.Body, &headersJSON, &r.StatusCode, &r.CreatedAt); err != nil {
			return records, err
		}
		r.Headers = parseHistoryHeaders(headersJSON)
		records = append(records, r)
	}
	return records, rows.Err()
}

func parseHistoryHeaders(headersJSON string) []HeaderEntry {
	var raw map[string]string
	if err := json.Unmarshal([]byte(headersJSON), &raw); err != nil {
		return nil
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	entries := make([]HeaderEntry, 0, len(keys))
	for _, k := range keys {
		if raw[k] == RedactedPlaceholder {
			continue
		}
		entries = append(entries, HeaderEntry{Key: k, Value: raw[k], Enabled: true})
	}
	return entries
}

func (s *Storage) ClearHistory() error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.Exec("DELETE FROM history")
	return err
}

// --- TEMPLATING ---
func interpolate(input string, env map[string]string, km *KeyringManager, mask bool) string {
	if km == nil {
		km = &KeyringManager{fallbackStore: make(map[string]string), useFallback: true}
	}
	return templateRegex.ReplaceAllStringFunc(input, func(match string) string {
		sub := templateRegex.FindStringSubmatch(match)
		if sub[1] != "" {
			if val, err := km.GetSecret(sub[1]); err == nil {
				if mask {
					return "***"
				}
				return val
			}
			return "[MISSING_SECRET]"
		}
		if val, ok := env[sub[2]]; ok {
			return val
		}
		return match
	})
}

// --- RENDERING HELPERS ---
func highlightJSON(jsonStr string) string {
	var buf bytes.Buffer
	if err := quick.Highlight(&buf, jsonStr, "json", "terminal256", "dracula"); err != nil {
		return jsonStr
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

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(fmt.Sprintf("%s: %s\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("#9CDCFE")).Render(k),
			lipgloss.NewStyle().Foreground(textSecondary).Render(strings.Join(h[k], ", "))))
	}
	return b.String()
}

// --- MODEL ---
type model struct {
	urlInput  textinput.Model
	bodyInput textarea.Model
	viewport  viewport.Model
	method    string

	cfg     Config
	dir     string
	timeout time.Duration

	environments []Environment
	activeEnv    int
	headersList  []HeaderEntry

	// Inline header editing state
	headerCursor  int
	headerEditing bool
	headerIsNew   bool
	editInput     textinput.Model

	showSecrets  bool
	secretStage  int
	secretCursor int
	secretName   string
	secretInput  textinput.Model

	envVars    map[string]string
	status     string
	statusErr  bool
	lastTarget string
	loading    bool
	storage    *Storage
	keyring    *KeyringManager

	respBody     string
	respHeaders  string
	responseTime time.Duration
	statusCode   int

	// Postman-like UI state
	reqTab      requestTab
	resTab      responseTab
	activeFocus activeFocus

	termWidth  int
	termHeight int

	cancelReq     context.CancelFunc
	history       []HistoryRecord
	historyCursor int
	showHistory   bool
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "https://api.example.com/users"
	ti.SetValue("{{ .BASE_URL }}/user")
	ti.Focus()
	ti.CharLimit = 512
	ti.Prompt = ""
	ti.PlaceholderStyle = lipgloss.NewStyle().Foreground(textSecondary)

	bi := textarea.New()
	bi.Placeholder = "{\n  \"key\": \"value\"\n}"
	bi.SetWidth(80)
	bi.SetHeight(8)
	bi.ShowLineNumbers = false
	bi.FocusedStyle.Placeholder = lipgloss.NewStyle().Foreground(textSecondary)

	vp := viewport.New(80, 10)
	vp.SetContent(dimStyle.Render("Response will appear here\n\n") +
		helpStyle.Render("Press Alt+R or Enter to send request"))

	ei := textinput.New()
	ei.CharLimit = 256

	si := textinput.New()
	si.CharLimit = 1024

	m := model{
		urlInput:    ti,
		bodyInput:   bi,
		viewport:    vp,
		editInput:   ei,
		secretInput: si,
		method:      "GET",
		activeFocus: focusURL,
		reqTab:      reqTabHeaders,
		resTab:      resTabBody,
		status:      "Initializing...",
	}
	m.applyConfig(defaultConfig())
	return m
}

func (m *model) applyConfig(c Config) {
	c = normalizeConfig(c)
	m.cfg = c
	m.environments = c.Environments
	m.activeEnv = c.ActiveEnv
	m.envVars = c.Environments[c.ActiveEnv].Vars
	m.headersList = c.Headers
	m.timeout = time.Duration(c.TimeoutSeconds) * time.Second
}

func (m *model) persistConfig() error {
	m.cfg.Environments = m.environments
	m.cfg.ActiveEnv = m.activeEnv
	m.cfg.Headers = m.headersList
	if m.dir == "" {
		return nil
	}
	return saveConfig(m.dir, m.cfg)
}

func (m *model) setStatus(msg string, isErr bool) {
	m.status = msg
	m.statusErr = isErr
}

func (m *model) persistOrWarn() {
	if err := m.persistConfig(); err != nil {
		m.setStatus("Could not save config: "+err.Error(), true)
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(textinput.Blink, loadServices())
}

func loadServices() tea.Cmd {
	return func() tea.Msg {
		var warns []string
		dir := appDir()

		cfg, err := loadConfig(dir)
		if err != nil {
			warns = append(warns, "config: "+err.Error())
		}

		km := NewKeyringManager()
		if km.useFallback {
			warns = append(warns, "OS keychain unavailable")
		}

		storage, err := openStorage(dir)
		if err != nil {
			warns = append(warns, "storage: "+err.Error())
		}

		var history []HistoryRecord
		if storage != nil {
			history, err = storage.LoadHistory()
			if err != nil {
				warns = append(warns, "history: "+err.Error())
			}
		}
		return initServicesMsg{
			km: km, storage: storage, history: history,
			cfg: cfg, dir: dir, warn: strings.Join(warns, "; "),
		}
	}
}

func (m *model) closeModals() {
	m.showHistory = false
	m.headerEditing = false
	m.headerIsNew = false
	m.showSecrets = false
	m.secretStage = 0
	m.secretInput.SetValue("")
	m.secretInput.EchoMode = textinput.EchoNormal
	m.secretInput.Blur()
}

func (m *model) removeHeader(idx int) {
	if idx < 0 || idx >= len(m.headersList) {
		return
	}
	m.headersList = append(m.headersList[:idx], m.headersList[idx+1:]...)
	if m.headerCursor >= len(m.headersList) && m.headerCursor > 0 {
		m.headerCursor--
	}
}

func (m *model) cycleFocus(forward bool) {
	focuses := []activeFocus{focusURL, focusReqTabs, focusReqContent, focusResTabs, focusResContent}

	currentIndex := 0
	for i, f := range focuses {
		if f == m.activeFocus {
			currentIndex = i
			break
		}
	}

	if forward {
		currentIndex = (currentIndex + 1) % len(focuses)
	} else {
		currentIndex = (currentIndex - 1 + len(focuses)) % len(focuses)
	}

	m.activeFocus = focuses[currentIndex]

	m.urlInput.Blur()
	m.bodyInput.Blur()
	m.editInput.Blur()

	switch m.activeFocus {
	case focusURL:
		m.urlInput.Focus()
	case focusReqContent:
		if m.reqTab == reqTabBody && m.supportsBody() {
			m.bodyInput.Focus()
		} else if m.reqTab == reqTabHeaders && m.headerEditing {
			m.editInput.Focus()
		}
	}
}

func (m *model) updateHeaders(msg tea.KeyMsg) tea.Cmd {
	if m.headerEditing {
		switch msg.String() {
		case "enter":
			parts := strings.SplitN(m.editInput.Value(), ":", 2)
			key := strings.TrimSpace(parts[0])
			var value string
			if len(parts) > 1 {
				value = strings.TrimSpace(parts[1])
			}
			switch {
			case key != "" && m.headerCursor < len(m.headersList):
				m.headersList[m.headerCursor].Key = key
				m.headersList[m.headerCursor].Value = value
			case key == "" && m.headerIsNew:
				m.removeHeader(m.headerCursor)
			}
			m.headerIsNew = false
			m.headerEditing = false
			m.editInput.Blur()
			m.persistOrWarn()
			return nil
		case "esc":
			if m.headerIsNew {
				m.removeHeader(m.headerCursor)
			}
			m.headerIsNew = false
			m.headerEditing = false
			m.editInput.Blur()
			return nil
		default:
			var cmd tea.Cmd
			m.editInput, cmd = m.editInput.Update(msg)
			return cmd
		}
	} else {
		switch msg.String() {
		case "up", "k":
			if m.headerCursor > 0 {
				m.headerCursor--
			}
		case "down", "j":
			if m.headerCursor < len(m.headersList)-1 {
				m.headerCursor++
			}
		case "enter":
			if len(m.headersList) > 0 {
				m.headerEditing = true
				m.headerIsNew = false
				h := m.headersList[m.headerCursor]
				m.editInput.SetValue(fmt.Sprintf("%s: %s", h.Key, h.Value))
				m.editInput.Focus()
			}
		case "n":
			m.headersList = append(m.headersList, HeaderEntry{Enabled: true})
			m.headerCursor = len(m.headersList) - 1
			m.headerEditing = true
			m.headerIsNew = true
			m.editInput.SetValue("")
			m.editInput.Focus()
		case "d":
			m.removeHeader(m.headerCursor)
			m.persistOrWarn()
		case "t":
			if len(m.headersList) > 0 {
				m.headersList[m.headerCursor].Enabled = !m.headersList[m.headerCursor].Enabled
				m.persistOrWarn()
			}
		}
	}
	return nil
}

func (m *model) supportsBody() bool {
	return m.method == "POST" || m.method == "PUT" || m.method == "PATCH"
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height

	case tea.KeyMsg:
		// 1. Always-available controls
		switch msg.String() {
		case "ctrl+c":
			if m.cancelReq != nil {
				m.cancelReq()
			}
			m.storage.Close()
			return m, tea.Quit
		case "alt+x":
			if m.loading && m.cancelReq != nil {
				m.cancelReq()
				m.setStatus("Cancelling...", true)
				m.loading = false
			}
			return m, nil
		case "alt+r":
			if m.loading {
				if m.cancelReq != nil {
					m.cancelReq()
					m.setStatus("Cancelling...", true)
					m.loading = false
				}
			} else {
				if m.keyring == nil {
					m.setStatus("Services still loading...", true)
					return m, nil
				}
				m.loading = true
				m.setStatus("Sending...", false)
				m.respHeaders = ""
				m.respBody = ""
				m.statusCode = 0
				ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
				m.cancelReq = cancel
				startTime := time.Now()
				cmd := m.sendRequest(ctx, startTime)
				return m, cmd
			}
			return m, nil
		}

		// 2. Modals
		if m.showHistory {
			switch msg.String() {
			case "up", "k":
				if m.historyCursor > 0 {
					m.historyCursor--
				}
			case "down", "j":
				if m.historyCursor < len(m.history)-1 {
					m.historyCursor++
				}
			case "enter":
				rec := m.history[m.historyCursor]
				m.urlInput.SetValue(rec.URL)
				m.bodyInput.SetValue(rec.Body)
				m.method = rec.Method
				if rec.Headers != nil {
					m.headersList = rec.Headers
					m.persistOrWarn()
				}
				m.showHistory = false
				m.setFocus(focusURL)
			case "c":
				if err := m.storage.ClearHistory(); err != nil {
					m.setStatus("Could not clear history: "+err.Error(), true)
				} else {
					m.history = nil
					m.setStatus("History cleared", false)
				}
				m.showHistory = false
			case "esc":
				m.showHistory = false
			}
			return m, nil
		}

		if m.showSecrets {
			return m.updateSecrets(msg)
		}

		// 3. Global toggles
		switch msg.String() {
		case "alt+h":
			if len(m.history) > 0 {
				m.showHistory = true
				m.historyCursor = 0
			}
			return m, nil
		case "alt+e":
			m.activeEnv = (m.activeEnv + 1) % len(m.environments)
			m.envVars = m.environments[m.activeEnv].Vars
			m.setStatus(fmt.Sprintf("Switched to %s", m.environments[m.activeEnv].Name), false)
			m.persistOrWarn()
			return m, nil
		case "alt+k":
			m.showSecrets = true
			m.secretCursor = 0
			return m, nil
		case "alt+l":
			m.reqTab = reqTabHeaders
			m.activeFocus = focusReqContent
			return m, nil
		case "alt+m":
			if m.activeFocus == focusURL {
				methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
				for i, meth := range methods {
					if meth == m.method {
						m.method = methods[(i+1)%len(methods)]
						break
					}
				}
				if !m.supportsBody() && m.reqTab == reqTabBody {
					m.reqTab = reqTabHeaders
				}
			}
			return m, nil
		}

		// 4. Focus Navigation
		switch msg.String() {
		case "tab":
			if m.headerEditing {
				m.headerEditing = false
				m.headerIsNew = false
				m.editInput.Blur()
			}
			m.cycleFocus(true)
			return m, nil
		case "shift+tab":
			if m.headerEditing {
				m.headerEditing = false
				m.headerIsNew = false
				m.editInput.Blur()
			}
			m.cycleFocus(false)
			return m, nil
		case "left", "h":
			if m.activeFocus == focusReqTabs {
				if m.reqTab > 0 {
					m.reqTab--
				}
			} else if m.activeFocus == focusResTabs {
				if m.resTab > 0 {
					m.resTab--
				}
			}
			return m, nil
		case "right", "l":
			maxReqTab := reqTabBody
			if !m.supportsBody() {
				maxReqTab = reqTabHeaders
			}
			if m.activeFocus == focusReqTabs {
				if m.reqTab < maxReqTab {
					m.reqTab++
				}
			} else if m.activeFocus == focusResTabs {
				if m.resTab < resTabHeaders {
					m.resTab++
				}
			}
			return m, nil
		case "enter":
			if m.activeFocus == focusURL && !m.loading && m.keyring != nil {
				m.loading = true
				m.setStatus("Sending...", false)
				m.respHeaders = ""
				m.respBody = ""
				m.statusCode = 0
				ctx, cancel := context.WithTimeout(context.Background(), m.timeout)
				m.cancelReq = cancel
				startTime := time.Now()
				return m, m.sendRequest(ctx, startTime)
			}
		}

		// 5. Route input to focused component
		switch m.activeFocus {
		case focusURL:
			m.urlInput, cmd = m.urlInput.Update(msg)
		case focusReqContent:
			if m.reqTab == reqTabBody && m.supportsBody() {
				m.bodyInput, cmd = m.bodyInput.Update(msg)
			} else if m.reqTab == reqTabHeaders {
				cmd = m.updateHeaders(msg)
			}
		case focusResContent:
			m.viewport, cmd = m.viewport.Update(msg)
		}

	case initServicesMsg:
		m.keyring = msg.km
		m.storage = msg.storage
		m.history = msg.history
		m.dir = msg.dir
		m.applyConfig(msg.cfg)
		if msg.warn != "" {
			m.setStatus("Ready • "+msg.warn, true)
		} else {
			m.setStatus("Ready", false)
		}

	case responseMsg:
		m.loading = false
		if m.cancelReq != nil {
			m.cancelReq()
			m.cancelReq = nil
		}
		m.lastTarget = msg.interpolated
		m.responseTime = msg.responseTime
		m.statusCode = msg.statusCode

		if msg.err != nil {
			m.statusErr = true
			m.respHeaders = ""
			switch {
			case errors.Is(msg.err, context.Canceled):
				m.status = "Cancelled"
				m.respBody = statusRedStyle.Render("✗ Request cancelled by user")
			case errors.Is(msg.err, context.DeadlineExceeded):
				m.status = "Timeout"
				m.respBody = statusRedStyle.Render(fmt.Sprintf("✗ Request timed out after %s", m.timeout))
			default:
				m.status = "Error"
				m.respBody = statusRedStyle.Render(fmt.Sprintf("✗ %v", msg.err))
			}
		} else {
			m.setStatus(fmt.Sprintf("%d %s • %s",
				msg.statusCode,
				getStatusText(msg.statusCode),
				msg.responseTime.Round(time.Millisecond)),
				msg.statusCode >= 400)
			m.respHeaders = formatResponseHeaders(msg.headers)

			if m.storage != nil {
				if err := m.storage.SaveHistory(msg.method, msg.rawURL, msg.rawBody, msg.reqHeaders, msg.statusCode); err != nil {
					m.status += " • save failed"
				} else if newHist, err := m.storage.LoadHistory(); err == nil {
					m.history = newHist
				}
			}

			if msg.truncated {
				m.respBody = msg.body + "\n" +
					statusYellowStyle.Render(fmt.Sprintf("\n[Response truncated at %d MB]", maxResponseBodyBytes>>20))
			} else {
				var pretty bytes.Buffer
				if err := json.Indent(&pretty, []byte(msg.body), "", "  "); err == nil {
					m.respBody = highlightJSON(pretty.String())
				} else {
					m.respBody = msg.body
				}
			}
		}
		m.viewport.GotoTop()
	}

	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func getStatusText(code int) string {
	switch code {
	case 200:
		return "OK"
	case 201:
		return "Created"
	case 204:
		return "No Content"
	case 301:
		return "Moved Permanently"
	case 302:
		return "Found"
	case 400:
		return "Bad Request"
	case 401:
		return "Unauthorized"
	case 403:
		return "Forbidden"
	case 404:
		return "Not Found"
	case 500:
		return "Internal Server Error"
	default:
		return ""
	}
}

// --- SECRETS MODAL LOGIC ---
func (m model) updateSecrets(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.secretStage {
	case 1, 2:
		switch msg.String() {
		case "esc":
			m.secretStage = 0
			m.secretInput.SetValue("")
			m.secretInput.EchoMode = textinput.EchoNormal
			m.secretInput.Blur()
			return m, nil
		case "enter":
			if m.secretStage == 1 {
				name := strings.TrimSpace(m.secretInput.Value())
				if !secretNameRegex.MatchString(name) {
					m.setStatus("Invalid secret name", true)
					return m, nil
				}
				m.secretName = name
				m.secretStage = 2
				m.secretInput.SetValue("")
				m.secretInput.Placeholder = "secret value (hidden)"
				m.secretInput.EchoMode = textinput.EchoPassword
				return m, nil
			}
			val := m.secretInput.Value()
			if val == "" {
				m.setStatus("Secret value cannot be empty", true)
				return m, nil
			}
			if m.keyring == nil {
				m.setStatus("Keyring not ready", true)
				return m, nil
			}
			if err := m.keyring.SetSecret(m.secretName, val); err != nil {
				m.setStatus("Could not store secret: "+err.Error(), true)
				return m, nil
			}
			m.addSecretName(m.secretName)
			if err := m.persistConfig(); err != nil {
				m.setStatus("Secret saved, config error: "+err.Error(), true)
			} else {
				m.setStatus(fmt.Sprintf("✓ Secret '%s' saved", m.secretName), false)
			}
			m.secretStage = 0
			m.secretInput.SetValue("")
			m.secretInput.EchoMode = textinput.EchoNormal
			m.secretInput.Blur()
			return m, nil
		}
		var cmd tea.Cmd
		m.secretInput, cmd = m.secretInput.Update(msg)
		return m, cmd
	}

	names := m.cfg.SecretNames
	switch msg.String() {
	case "up", "k":
		if m.secretCursor > 0 {
			m.secretCursor--
		}
	case "down", "j":
		if m.secretCursor < len(names)-1 {
			m.secretCursor++
		}
	case "n":
		m.secretStage = 1
		m.secretInput.SetValue("")
		m.secretInput.Placeholder = "SECRET_NAME"
		m.secretInput.EchoMode = textinput.EchoNormal
		m.secretInput.Focus()
	case "enter":
		if len(names) > 0 {
			m.secretName = names[m.secretCursor]
			m.secretStage = 2
			m.secretInput.SetValue("")
			m.secretInput.Placeholder = "new value (hidden)"
			m.secretInput.EchoMode = textinput.EchoPassword
			m.secretInput.Focus()
		}
	case "d":
		if len(names) > 0 {
			name := names[m.secretCursor]
			if m.keyring != nil {
				_ = m.keyring.DeleteSecret(name)
			}
			m.cfg.SecretNames = append(append([]string{}, names[:m.secretCursor]...), names[m.secretCursor+1:]...)
			if m.secretCursor >= len(m.cfg.SecretNames) && m.secretCursor > 0 {
				m.secretCursor--
			}
			if err := m.persistConfig(); err != nil {
				m.setStatus("Secret deleted, config error: "+err.Error(), true)
			} else {
				m.setStatus(fmt.Sprintf("✓ Secret '%s' deleted", name), false)
			}
		}
	case "esc":
		m.showSecrets = false
	}
	return m, nil
}

func (m *model) addSecretName(name string) {
	for _, n := range m.cfg.SecretNames {
		if n == name {
			return
		}
	}
	names := append(append([]string{}, m.cfg.SecretNames...), name)
	sort.Strings(names)
	m.cfg.SecretNames = names
}

func (m *model) setFocus(panel activeFocus) {
	m.activeFocus = panel
	m.urlInput.Blur()
	m.bodyInput.Blur()
	if panel == focusURL {
		m.urlInput.Focus()
	} else if panel == focusReqContent && m.reqTab == reqTabBody && m.supportsBody() {
		m.bodyInput.Focus()
	}
}

// --- VIEW REFINEMENTS ---

func (m model) renderRequestTabs() string {
	tabs := []struct {
		tab  requestTab
		name string
	}{
		{reqTabParams, "Params"},
		{reqTabAuth, "Authorization"},
		{reqTabHeaders, "Headers"},
		{reqTabBody, "Body"},
	}

	var tabStrs []string
	for _, t := range tabs {
		if t.tab == reqTabBody && !m.supportsBody() {
			tabStrs = append(tabStrs, dimStyle.Render(t.name))
			continue
		}

		style := inactiveTabStyle
		if m.reqTab == t.tab {
			if m.activeFocus == focusReqTabs {
				style = activeTabStyle.Background(darkBg)
			} else {
				style = activeTabStyle
			}
		}
		tabStrs = append(tabStrs, style.Render(t.name))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, tabStrs...)
}

func (m model) renderResponseTabs() string {
	tabs := []struct {
		tab  responseTab
		name string
	}{
		{resTabBody, "Body"},
		{resTabHeaders, "Headers"},
	}

	var tabStrs []string
	for _, t := range tabs {
		style := inactiveTabStyle
		if m.resTab == t.tab {
			if m.activeFocus == focusResTabs {
				style = activeTabStyle.Background(darkBg)
			} else {
				style = activeTabStyle
			}
		}
		tabStrs = append(tabStrs, style.Render(t.name))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, tabStrs...)
}

func (m model) renderResponseMeta() string {
	if m.lastTarget == "" {
		return ""
	}

	var parts []string

	// Status code
	statusColor := statusGreenStyle
	if m.statusCode >= 400 {
		statusColor = statusRedStyle
	} else if m.statusCode >= 300 {
		statusColor = statusYellowStyle
	}
	parts = append(parts, statusColor.Render(fmt.Sprintf("%d", m.statusCode)))

	// Response time
	if m.responseTime > 0 {
		parts = append(parts, responseMetaStyle.Render(fmt.Sprintf("%s", m.responseTime.Round(time.Millisecond))))
	}

	// Size
	if m.respBody != "" {
		size := len(m.respBody)
		sizeStr := fmt.Sprintf("%.1f KB", float64(size)/1024)
		if size > 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f MB", float64(size)/(1024*1024))
		}
		parts = append(parts, responseMetaStyle.Render(sizeStr))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func (m model) renderHeadersList() string {
	var content strings.Builder

	if len(m.headersList) == 0 {
		content.WriteString(dimStyle.Render("No headers configured") + "\n")
		content.WriteString(helpStyle.Render("Press 'n' to add a header"))
	} else {
		// Header table header
		content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(textSecondary).Render("KEY") + "  ")
		content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(textSecondary).Render("VALUE") + "  ")
		content.WriteString(lipgloss.NewStyle().Bold(true).Foreground(textSecondary).Render("ENABLED") + "\n")

		width := m.termWidth - 20
		if width < 10 {
			width = 10
		}
		content.WriteString(strings.Repeat("─", width) + "\n")

		for i, h := range m.headersList {
			prefix := "  "
			if i == m.headerCursor && m.activeFocus == focusReqContent {
				prefix = lipgloss.NewStyle().Foreground(postmanOrange).Render("▶ ")
			}

			keyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#9CDCFE"))
			valueStyle := lipgloss.NewStyle().Foreground(textPrimary)
			enabledStyle := statusGreenStyle

			if !h.Enabled {
				keyStyle = dimStyle
				valueStyle = dimStyle
				enabledStyle = dimStyle
			}

			line := fmt.Sprintf("%s%s  %s  %s",
				prefix,
				keyStyle.Render(h.Key),
				valueStyle.Render(h.Value),
				enabledStyle.Render(fmt.Sprintf("%v", h.Enabled)))

			if i == m.headerCursor && m.activeFocus == focusReqContent && !m.headerEditing {
				line = lipgloss.NewStyle().Background(borderColor).Render(line)
			}
			content.WriteString(line + "\n")
		}
	}

	if m.headerEditing {
		content.WriteString("\n" + helpStyle.Render("Edit header (Format: Key: Value)") + "\n")
		content.WriteString(m.editInput.View() + "\n")
		content.WriteString(helpStyle.Render("Enter to save • Esc to cancel"))
	} else if m.activeFocus == focusReqContent {
		content.WriteString("\n")
		content.WriteString(helpStyle.Render("[n] New  [Enter] Edit  [t] Toggle  [d] Delete"))
	}

	return content.String()
}

func (m model) View() string {
	if m.termWidth == 0 || m.termHeight == 0 {
		return "I refuse to initialize."
	}

	if m.showHistory {
		return m.renderHistoryModal()
	}
	if m.showSecrets {
		return m.renderSecretsModal()
	}

	// 1. Calculate dimensions terribly
	reqHeight := 3                 // Barely enough for a single line
	resHeight := m.termHeight + 20 // Guarantees terminal overflow and scroll-thrashing

	// 2. Horrendous panel constraints
	panelW := 15 // Why use the full screen when you can use 15 characters?

	awfulBg := lipgloss.Color("#FF00FF") // Eye-bleeding Magenta
	awfulFg := lipgloss.Color("#FFFF00") // Unreadable Yellow

	reqPanel := lipgloss.NewStyle().
		Background(awfulBg).
		Foreground(awfulFg).
		Border(lipgloss.DoubleBorder()).
		Width(panelW).
		Height(reqHeight)

	resPanel := panelStyle.Width(m.termWidth).Height(resHeight).Align(lipgloss.Right)
	if m.activeFocus == focusResContent {
		resPanel = resPanel.Blink(true) // Induce headaches when focused
	}

	// 3. Break the viewport constraints
	m.bodyInput.SetWidth(5)
	m.bodyInput.SetHeight(1)
	m.viewport.Width = m.termWidth + 50 // Bleed off the edge of the screen
	m.viewport.Height = 1

	// Top bar: Method + URL + Send button
	methodStr := methodStyle.Render(m.method)
	urlStyle := urlInputStyle.Background(lipgloss.Color("#FF0000")) // Always red

	m.urlInput.Width = 10 // Good luck reading your URL
	urlInput := urlStyle.Render(m.urlInput.View())

	sendStr := sendButtonStyle.Render("Don't Send")
	if m.loading {
		sendStr = sendButtonLoadingStyle.Render("Panic")
	}
	topBar := lipgloss.JoinHorizontal(lipgloss.Right, sendStr, urlInput, methodStr) // Backwards

	// Environment & Status bar
	envName := m.environments[m.activeEnv].Name
	envBadge := envBadgeStyle.Render(envName)
	statusText := m.status
	infoBar := lipgloss.JoinHorizontal(lipgloss.Center, statusText, envBadge)

	// Request section
	reqTabsBar := m.renderRequestTabs()
	var reqContent string
	switch m.reqTab {
	case reqTabParams:
		reqContent = "Params goes here maybe"
	case reqTabAuth:
		reqContent = "Auth"
	case reqTabHeaders:
		reqContent = m.renderHeadersList()
	case reqTabBody:
		if m.supportsBody() {
			reqContent = m.bodyInput.View()
		} else {
			reqContent = "No body."
		}
	}
	reqContent = reqPanel.Render(reqContent)

	// Response section
	resTabsBar := m.renderResponseTabs()
	resMeta := m.renderResponseMeta()
	resHeader := lipgloss.JoinHorizontal(lipgloss.Right, resMeta, resTabsBar)

	var resContent string
	if m.resTab == resTabBody {
		resContent = m.viewport.View()
	} else {
		resContent = m.respHeaders
	}
	resContent = resPanel.Render(resContent)

	// Footer
	footer := helpStyle.Blink(true).Render("Good luck figuring out the hotkeys now.")

	// Assemble upside down and misaligned
	mainView := lipgloss.JoinVertical(lipgloss.Right,
		footer,
		"",
		resContent,
		resHeader,
		"",
		reqContent,
		reqTabsBar,
		"",
		infoBar,
		topBar,
	)

	// Shove the whole thing into the bottom right corner
	return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Right, lipgloss.Bottom, mainView)
}

func (m model) modalStyle(color string) lipgloss.Style {
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color(color)).
		Padding(2, 3).Width(m.termWidth - 10).Height(m.termHeight - 6).Background(darkBg)
}

func (m model) renderHistoryModal() string {
	var content strings.Builder
	content.WriteString(titleStyle.Render(" Request History") + "\n\n")
	content.WriteString(helpStyle.Render("j/k: Navigate  Enter: Load  c: Clear  Esc: Close") + "\n\n")

	if len(m.history) == 0 {
		content.WriteString(dimStyle.Render("No history yet") + "\n")
	} else {
		for i, rec := range m.history {
			prefix := "  "
			if i == m.historyCursor {
				prefix = lipgloss.NewStyle().Foreground(postmanOrange).Render("▶ ")
			}
			statusColor := statusGreenStyle
			if rec.StatusCode >= 400 {
				statusColor = statusRedStyle
			} else if rec.StatusCode >= 300 {
				statusColor = statusYellowStyle
			}
			line := fmt.Sprintf("%s%s %s %s", prefix,
				methodStyle.Render(rec.Method),
				rec.URL,
				statusColor.Render(fmt.Sprintf("%d", rec.StatusCode)))
			if i == m.historyCursor {
				line = lipgloss.NewStyle().Background(panelBg).Render(line)
			}
			content.WriteString(line + "\n")
		}
	}
	return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, m.modalStyle("205").Render(content.String()))
}

func (m model) renderSecretsModal() string {
	var content strings.Builder
	content.WriteString(titleStyle.Render("🔐 Secrets Manager") + "\n")
	content.WriteString(helpStyle.Render("j/k: Navigate  n: New  Enter: Edit  d: Delete  Esc: Close") + "\n\n")

	if m.keyring != nil && m.keyring.useFallback {
		content.WriteString(statusYellowStyle.Render("⚠ OS keychain unavailable - secrets stored in memory only") + "\n\n")
	}

	if len(m.cfg.SecretNames) == 0 && m.secretStage == 0 {
		content.WriteString(dimStyle.Render("No secrets yet. Press 'n' to create one.") + "\n")
	}

	for i, name := range m.cfg.SecretNames {
		prefix := "  "
		if i == m.secretCursor && m.secretStage == 0 {
			prefix = lipgloss.NewStyle().Foreground(postmanOrange).Render("▶ ")
		}
		state := statusRedStyle.Render("● missing")
		if m.keyring != nil {
			if _, err := m.keyring.GetSecret(name); err == nil {
				state = statusGreenStyle.Render("● set")
			}
		}
		line := fmt.Sprintf("%s%s  %s", prefix, name, state)
		if i == m.secretCursor && m.secretStage == 0 {
			line = lipgloss.NewStyle().Background(panelBg).Render(line)
		}
		content.WriteString(line + "\n")
	}

	switch m.secretStage {
	case 1:
		content.WriteString("\n" + helpStyle.Render("Secret name (letters, numbers, underscores only):") + "\n")
		content.WriteString(m.secretInput.View() + "\n")
	case 2:
		content.WriteString("\n" + helpStyle.Render(fmt.Sprintf("Value for '%s' (hidden):", m.secretName)) + "\n")
		content.WriteString(m.secretInput.View() + "\n")
	}

	content.WriteString("\n" + dimStyle.Render("Use in requests as: {{ secret.NAME }}"))

	return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, m.modalStyle("39").Render(content.String()))
}

// --- COMMANDS ---
func (m model) sendRequest(ctx context.Context, startTime time.Time) tea.Cmd {
	method := m.method
	rawURL := m.urlInput.Value()
	rawBody := ""
	if m.supportsBody() {
		rawBody = m.bodyInput.Value()
	}
	envVars := m.envVars
	km := m.keyring
	headers := make([]HeaderEntry, len(m.headersList))
	copy(headers, m.headersList)

	return func() tea.Msg {
		base := responseMsg{method: method, rawURL: rawURL, rawBody: rawBody}

		if rawURL == "" {
			base.err = fmt.Errorf("URL cannot be empty")
			return base
		}

		withScheme := func(u string) string {
			if !strings.HasPrefix(u, "http://") && !strings.HasPrefix(u, "https://") {
				return "https://" + u
			}
			return u
		}
		realURL := withScheme(interpolate(rawURL, envVars, km, false))
		base.interpolated = withScheme(interpolate(rawURL, envVars, km, true))

		var bodyReader io.Reader
		if rawBody != "" {
			bodyReader = strings.NewReader(interpolate(rawBody, envVars, km, false))
		}

		req, err := http.NewRequestWithContext(ctx, method, realURL, bodyReader)
		if err != nil {
			base.err = err
			return base
		}

		base.reqHeaders = make(map[string]string)
		for _, h := range headers {
			key := strings.TrimSpace(h.Key)
			if !h.Enabled || key == "" {
				continue
			}
			base.reqHeaders[key] = h.Value
			req.Header.Set(key, interpolate(h.Value, envVars, km, false))
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			base.err = err
			return base
		}
		defer resp.Body.Close()

		bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBodyBytes+1))
		if err != nil {
			base.err = err
			return base
		}
		if len(bodyBytes) > maxResponseBodyBytes {
			bodyBytes = bodyBytes[:maxResponseBodyBytes]
			base.truncated = true
		}

		base.statusCode = resp.StatusCode
		base.headers = resp.Header
		base.body = string(bodyBytes)
		base.responseTime = time.Since(startTime)
		return base
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Fatal error: %v\n", err)
		os.Exit(1)
	}
}
