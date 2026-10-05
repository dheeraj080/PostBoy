package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/zalando/go-keyring"
	_ "modernc.org/sqlite"
)

const (
	serviceName          = "postboy"
	maxResponseBodyBytes = 10 << 20 // 10 MB maximum payload buffer
	requestTimeout       = 15 * time.Second
)

// Focus Panel Identifier
type activePanel int

const (
	panelURL activePanel = iota
	panelBody
	panelResponse
)

// --- STYLES ---
var (
	titleStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	methodStyle      = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("86"))
	statusGreenStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("46"))
	statusRedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	helpStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	envBadgeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true)
	secBadgeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)

	focusedBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("205")).
			Padding(0, 1)

	unfocusedBoxStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("238")).
				Padding(0, 1)
)

// --- SECURITY & KEYRING SERVICE ---
type KeyringManager struct {
	fallbackStore map[string]string
	useFallback   bool
}

func NewKeyringManager() *KeyringManager {
	km := &KeyringManager{
		fallbackStore: make(map[string]string),
	}
	// Test keyring availability
	testErr := keyring.Set(serviceName, "__test_ping__", "pong")
	if testErr != nil {
		km.useFallback = true
	} else {
		_ = keyring.Delete(serviceName, "__test_ping__")
	}
	return km
}

func (km *KeyringManager) SetSecret(key, val string) error {
	if km.useFallback {
		km.fallbackStore[key] = val
		return nil
	}
	return keyring.Set(serviceName, key, val)
}

func (km *KeyringManager) GetSecret(key string) (string, error) {
	if km.useFallback {
		if val, ok := km.fallbackStore[key]; ok {
			return val, nil
		}
		return "", fmt.Errorf("secret not found in memory store")
	}
	return keyring.Get(serviceName, key)
}

// --- SENSITIVE HEADER REDACTOR ---
var sensitiveHeaderKeys = map[string]bool{
	"authorization":     true,
	"x-api-key":         true,
	"api-key":           true,
	"cookie":            true,
	"set-cookie":        true,
	"x-auth-token":      true,
	"sec-websocket-key": true,
}

func RedactHeaderValue(key, val string) string {
	lowerKey := strings.ToLower(key)
	if sensitiveHeaderKeys[lowerKey] {
		if strings.HasPrefix(strings.ToLower(val), "bearer ") {
			return "Bearer [REDACTED]"
		}
		if len(val) > 8 {
			return val[:4] + "...." + val[len(val)-4:] + " [REDACTED]"
		}
		return "[REDACTED]"
	}
	return val
}

func RedactHeaders(headers map[string]string) map[string]string {
	redacted := make(map[string]string)
	for k, v := range headers {
		redacted[k] = RedactHeaderValue(k, v)
	}
	return redacted
}

// --- MESSAGES ---
type responseMsg struct {
	statusCode   int
	headers      http.Header
	body         string
	interpolated string
	err          error
}

type initServicesMsg struct {
	km      *KeyringManager
	storage *Storage
}

// --- PERSISTENCE LAYER ---
type Storage struct {
	db *sql.DB
}

func initStorage() (*Storage, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		configDir = "."
	}
	appDir := filepath.Join(configDir, "postboy")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return nil, err
	}

	dbPath := filepath.Join(appDir, "postboy.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS history (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		method TEXT NOT NULL,
		url TEXT NOT NULL,
		headers_json TEXT,
		status_code INTEGER NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, err
	}

	return &Storage{db: db}, nil
}

func (s *Storage) SaveHistory(method, url string, headers map[string]string, statusCode int) error {
	if s == nil || s.db == nil {
		return nil
	}
	// Redact headers prior to database insertion
	cleanHeaders := RedactHeaders(headers)
	headersJSON, _ := json.Marshal(cleanHeaders)

	_, err := s.db.Exec("INSERT INTO history (method, url, headers_json, status_code) VALUES (?, ?, ?, ?)",
		method, url, string(headersJSON), statusCode)
	return err
}

// --- TEMPLATING & SECRET RESOLVER ---
func interpolateTemplate(input string, env map[string]string, km *KeyringManager) string {
	// Fallback if keyring hasn't loaded yet
	if km == nil {
		km = &KeyringManager{fallbackStore: make(map[string]string), useFallback: true}
	}

	// 1. Resolve secrets: {{ secret.KEY_NAME }} or {{ secret "KEY_NAME" }}
	secretRegex := regexp.MustCompile(`\{\{\s*secret[\.\s"]+([a-zA-Z0-9_]+)"?\s*\}\}`)
	input = secretRegex.ReplaceAllStringFunc(input, func(match string) string {
		sub := secretRegex.FindStringSubmatch(match)
		if len(sub) > 1 {
			if secretVal, err := km.GetSecret(sub[1]); err == nil {
				return secretVal
			}
		}
		return "[MISSING_SECRET]"
	})

	// 2. Resolve environment variables: {{ .VAR }} or {{ VAR }}
	envRegex := regexp.MustCompile(`\{\{\s*\.?([a-zA-Z0-9_]+)\s*\}\}`)
	return envRegex.ReplaceAllStringFunc(input, func(match string) string {
		sub := envRegex.FindStringSubmatch(match)
		if len(sub) > 1 {
			varName := sub[1]
			if val, ok := env[varName]; ok {
				return val
			}
		}
		return match
	})
}

// --- MODEL ---
type model struct {
	urlInput   textinput.Model
	bodyInput  textinput.Model
	viewport   viewport.Model
	method     string
	headers    map[string]string
	envVars    map[string]string
	status     string
	lastTarget string
	loading    bool
	storage    *Storage
	keyring    *KeyringManager

	activePanel activePanel
	termWidth   int
	termHeight  int
}

func initialModel() model {
	ti := textinput.New()
	ti.Placeholder = "{{ .BASE_URL }}/user"
	ti.SetValue("{{ .BASE_URL }}/user")
	ti.Focus()
	ti.CharLimit = 512

	bi := textinput.New()
	bi.Placeholder = `{"key": "value"}`
	bi.CharLimit = 2048

	vp := viewport.New(80, 10)
	vp.SetContent("Press [Enter] to execute request.")

	return model{
		urlInput:    ti,
		bodyInput:   bi,
		viewport:    vp,
		method:      "GET",
		activePanel: panelURL,
		headers: map[string]string{
			"User-Agent":    "PostBoy/4.0",
			"Content-Type":  "application/json",
			"Authorization": "Bearer {{ secret.GITHUB_TOKEN }}",
		},
		envVars: map[string]string{
			"BASE_URL": "https://api.github.com",
		},
		status: "Loading services...",
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		textinput.Blink,
		loadServices(), // Run initialization in the background
	)
}

// Background command to initialize Keyring and SQLite
func loadServices() tea.Cmd {
	return func() tea.Msg {
		km := NewKeyringManager()
		// Store mock token in keyring/fallback store for testing
		_ = km.SetSecret("GITHUB_TOKEN", "ghp_1234567890SecretTokenValue")

		storage, _ := initStorage()

		return initServicesMsg{
			km:      km,
			storage: storage,
		}
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.termWidth = msg.Width
		m.termHeight = msg.Height
		m.resizeLayout()

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			if m.storage != nil && m.storage.db != nil {
				m.storage.db.Close()
			}
			return m, tea.Quit

		case "alt+m": // VS Code-safe method cycling
			methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
			for i, meth := range methods {
				if meth == m.method {
					m.method = methods[(i+1)%len(methods)]
					break
				}
			}
			if !m.supportsBody() && m.activePanel == panelBody {
				m.setFocus(panelURL)
			}
			m.resizeLayout()

		case "tab", "]": // VS Code-safe focus forward
			m.cycleFocus(true)

		case "shift+tab", "[": // VS Code-safe focus backward
			m.cycleFocus(false)

		case "enter":
			if m.activePanel == panelURL || m.activePanel == panelBody {
				if !m.loading {
					m.loading = true
					m.status = "Sending..."
					m.viewport.SetContent("Resolving keyring secrets & sending request...")
					return m, m.sendRequest()
				}
			}
		}

	case initServicesMsg: // Handle the loaded services
		m.keyring = msg.km
		m.storage = msg.storage
		m.status = "Ready"

	case responseMsg:
		m.loading = false
		m.lastTarget = msg.interpolated
		if msg.err != nil {
			m.status = "Error"
			m.viewport.SetContent(statusRedStyle.Render(fmt.Sprintf("Request failed: %v", msg.err)))
		} else {
			m.status = fmt.Sprintf("Status: %d", msg.statusCode)

			// Persist sanitized query into SQLite database
			if m.storage != nil {
				_ = m.storage.SaveHistory(m.method, msg.interpolated, m.headers, msg.statusCode)
			}

			// Pretty-print response body if valid JSON
			var prettyJSON bytes.Buffer
			if err := json.Indent(&prettyJSON, []byte(msg.body), "", "  "); err == nil {
				m.viewport.SetContent(prettyJSON.String())
			} else {
				m.viewport.SetContent(msg.body)
			}
		}
	}

	switch m.activePanel {
	case panelURL:
		m.urlInput, cmd = m.urlInput.Update(msg)
		cmds = append(cmds, cmd)
	case panelBody:
		m.bodyInput, cmd = m.bodyInput.Update(msg)
		cmds = append(cmds, cmd)
	case panelResponse:
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *model) supportsBody() bool {
	return m.method == "POST" || m.method == "PUT" || m.method == "PATCH"
}

func (m *model) cycleFocus(forward bool) {
	panels := []activePanel{panelURL}
	if m.supportsBody() {
		panels = append(panels, panelBody)
	}
	panels = append(panels, panelResponse)

	currentIndex := 0
	for i, p := range panels {
		if p == m.activePanel {
			currentIndex = i
			break
		}
	}

	if forward {
		currentIndex = (currentIndex + 1) % len(panels)
	} else {
		currentIndex = (currentIndex - 1 + len(panels)) % len(panels)
	}

	m.setFocus(panels[currentIndex])
}

func (m *model) setFocus(panel activePanel) {
	m.activePanel = panel
	m.urlInput.Blur()
	m.bodyInput.Blur()

	switch panel {
	case panelURL:
		m.urlInput.Focus()
	case panelBody:
		m.bodyInput.Focus()
	}
}

func (m *model) resizeLayout() {
	if m.termWidth <= 0 || m.termHeight <= 0 {
		return
	}

	contentWidth := m.termWidth - 4
	if contentWidth < 20 {
		contentWidth = 20
	}

	m.urlInput.Width = contentWidth - 12
	m.bodyInput.Width = contentWidth - 8

	headerHeight := 3
	urlPanelHeight := 3
	bodyPanelHeight := 0
	if m.supportsBody() {
		bodyPanelHeight = 3
	}
	statusLineHeight := 2
	footerHeight := 2

	usedHeight := headerHeight + urlPanelHeight + bodyPanelHeight + statusLineHeight + footerHeight
	viewportHeight := m.termHeight - usedHeight - 2

	if viewportHeight < 5 {
		viewportHeight = 5
	}

	m.viewport.Width = contentWidth
	m.viewport.Height = viewportHeight
}

func (m model) View() string {
	if m.termWidth == 0 {
		return "Initializing terminal security engine...\n(Press Ctrl+C if stuck)"
	}

	// 1. Header & Active Environment / Keyring Status Bar
	header := titleStyle.Render("🚀 PostBoy - Keyring & Security Engine")

	keyringStatus := "OS Keychain (Active)"
	if m.keyring != nil && m.keyring.useFallback {
		keyringStatus = "In-Memory Fallback"
	} else if m.keyring == nil {
		keyringStatus = "Loading..."
	}

	secSummary := secBadgeStyle.Render("Security: ") + fmt.Sprintf("[Keyring: %s] | ", keyringStatus) +
		envBadgeStyle.Render("Auth Header: ") + RedactHeaderValue("Authorization", m.headers["Authorization"])

	// 2. Request / URL Box
	methodView := methodStyle.Render("[" + m.method + "]")
	urlContent := lipgloss.JoinHorizontal(lipgloss.Center, methodView, " ", m.urlInput.View())
	urlStyle := unfocusedBoxStyle
	if m.activePanel == panelURL {
		urlStyle = focusedBoxStyle
	}
	urlPanel := urlStyle.Render(urlContent)

	// 3. Body Panel (Conditional)
	var bodyPanel string
	if m.supportsBody() {
		bodyStyle := unfocusedBoxStyle
		if m.activePanel == panelBody {
			bodyStyle = focusedBoxStyle
		}
		bodyPanel = bodyStyle.Render("Body: " + m.bodyInput.View())
	}

	// 4. Response Box
	respStyle := unfocusedBoxStyle
	if m.activePanel == panelResponse {
		respStyle = focusedBoxStyle
	}
	respPanel := respStyle.Render(m.viewport.View())

	// 5. Status View & Rendered URL preview
	statusRendered := statusGreenStyle.Render(m.status)
	if strings.Contains(m.status, "Error") {
		statusRendered = statusRedStyle.Render(m.status)
	}
	if m.lastTarget != "" {
		statusRendered += helpStyle.Render(fmt.Sprintf("  (Target: %s)", m.lastTarget))
	}

	// 6. Help / Footer
	hint := helpStyle.Render(" [Tab or ]/[ ] Focus | [Alt+M] Method | [Enter] Send | [q/Ctrl+C] Quit ")

	// Layout Composition
	items := []string{
		header,
		secSummary,
		urlPanel,
	}
	if bodyPanel != "" {
		items = append(items, bodyPanel)
	}
	items = append(items, statusRendered, respPanel, hint)

	return lipgloss.JoinVertical(lipgloss.Left, items...)
}

// --- COMMANDS ---
func (m model) sendRequest() tea.Cmd {
	return func() tea.Msg {
		rawURL := m.urlInput.Value()
		if rawURL == "" {
			return responseMsg{err: fmt.Errorf("URL cannot be empty")}
		}

		// Interpolate environment variables and OS keyring secrets
		interpolatedURL := interpolateTemplate(rawURL, m.envVars, m.keyring)
		if !strings.HasPrefix(interpolatedURL, "http://") && !strings.HasPrefix(interpolatedURL, "https://") {
			interpolatedURL = "https://" + interpolatedURL
		}

		rawBody := m.bodyInput.Value()
		interpolatedBody := interpolateTemplate(rawBody, m.envVars, m.keyring)

		ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
		defer cancel()

		var bodyReader io.Reader
		if m.supportsBody() && interpolatedBody != "" {
			bodyReader = strings.NewReader(interpolatedBody)
		}

		req, err := http.NewRequestWithContext(ctx, m.method, interpolatedURL, bodyReader)
		if err != nil {
			return responseMsg{err: err, interpolated: interpolatedURL}
		}

		// Resolve secrets in headers before sending
		for k, v := range m.headers {
			resolvedVal := interpolateTemplate(v, m.envVars, m.keyring)
			req.Header.Set(k, resolvedVal)
		}

		client := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyFromEnvironment,
			},
		}

		resp, err := client.Do(req)
		if err != nil {
			return responseMsg{err: err, interpolated: interpolatedURL}
		}
		defer resp.Body.Close()

		limitedReader := io.LimitReader(resp.Body, maxResponseBodyBytes)
		bodyBytes, err := io.ReadAll(limitedReader)
		if err != nil {
			return responseMsg{err: err, interpolated: interpolatedURL}
		}

		return responseMsg{
			statusCode:   resp.StatusCode,
			headers:      resp.Header,
			body:         string(bodyBytes),
			interpolated: interpolatedURL,
		}
	}
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Printf("Fatal application error: %v\n", err)
		os.Exit(1)
	}
}
