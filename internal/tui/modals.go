package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/interp"
)

func (m *Model) closeModal() {
	m.modal = modalNone
	m.resetSecretInput()
	m.stopColInput()
	m.colConfirm = confirmNone
	m.stopEnvInput()
	m.envVars.cancelEdit()
	m.envConfirmDelete = false
	m.importInput.Blur()
	m.importConfirm = false
}

func (m Model) updateModal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.modal {
	case modalImport:
		return m.updateImport(msg)
	case modalEnv:
		return m.updateEnvEditor(msg)
	case modalCollections:
		return m.updateCollections(msg)
	case modalSave:
		return m.updateSave(msg)
	case modalHistory:
		return m.updateHistory(msg)
	case modalSecrets:
		return m.updateSecrets(msg)
	case modalHelp:
		if key.Matches(msg, m.keys.Close) || key.Matches(msg, m.keys.Help) {
			m.closeModal()
		}
	}
	return m, nil
}

// ---- History ----

func (m Model) updateHistory(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "alt+h":
		m.closeModal()
	case "up", "k":
		if m.historyCursor > 0 {
			m.historyCursor--
		}
	case "down", "j":
		if m.historyCursor < len(m.history)-1 {
			m.historyCursor++
		}
	case "enter":
		if m.historyCursor >= len(m.history) {
			return m, nil
		}
		rec := m.history[m.historyCursor]
		// The history entry becomes an untitled draft so Ctrl+S never
		// silently overwrites the previously open saved request. Headers
		// are merged, never replaced: redacted ones (e.g. Authorization)
		// are not stored in history; auth settings are kept as-is.
		draft := m.currentRequest()
		draft.Method = rec.Method
		draft.URL = rec.URL
		draft.Body = rec.Body
		draft.Headers = mergeKV(m.headers.items, rec.Headers)
		m.loadRequest(draft, nil, config.Request{})
		m.persistOrWarn()
		m.closeModal()
		return m, m.setFocus(focusURL)
	case "c":
		if err := m.store.ClearHistory(); err != nil {
			m.setStatus("Could not clear history: "+err.Error(), true)
		} else {
			m.history = nil
			m.historyCursor = 0
			m.setStatus("History cleared", false)
		}
	}
	return m, nil
}

func (m Model) renderHistoryModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Request History"))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("j/k: Navigate  Enter: Load  c: Clear  Esc: Close"))
	b.WriteString("\n\n")

	if len(m.history) == 0 {
		b.WriteString(dimStyle.Render("No history yet"))
	} else {
		// Window the list around the cursor so long histories fit.
		rows := max(m.termHeight-16, 3)
		start := 0
		if m.historyCursor >= rows {
			start = m.historyCursor - rows + 1
		}
		end := min(start+rows, len(m.history))
		width := max(m.termWidth-30, 20)
		for i := start; i < end; i++ {
			rec := m.history[i]
			prefix := "  "
			if i == m.historyCursor {
				prefix = cursorStyle.Render("▶ ")
			}
			line := fmt.Sprintf("%s%s %s %s", prefix,
				methodStyle.Render(rec.Method),
				truncate(rec.URL, width),
				statusStyle(rec.StatusCode).Render(fmt.Sprintf("%d", rec.StatusCode)))
			if i == m.historyCursor {
				line = lipgloss.NewStyle().Background(panelBg).Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
		if len(m.history) > rows {
			b.WriteString(dimStyle.Render(fmt.Sprintf("\n%d/%d", m.historyCursor+1, len(m.history))))
		}
	}
	return m.placeModal("205", b.String())
}

// ---- Secrets ----

func (m *Model) openSecrets() {
	m.modal = modalSecrets
	m.secretCursor = 0
	m.secretStage = secretList
	m.refreshSecretState()
}

// refreshSecretState caches which secrets exist so View never hits the OS
// keychain on every frame.
func (m *Model) refreshSecretState() {
	m.secretIsSet = make(map[string]bool, len(m.cfg.SecretNames))
	if m.secrets == nil {
		return
	}
	for _, name := range m.cfg.SecretNames {
		_, err := m.secrets.Get(name)
		m.secretIsSet[name] = err == nil
	}
}

func (m *Model) resetSecretInput() {
	m.secretStage = secretList
	m.secretInput.SetValue("")
	m.secretInput.EchoMode = textinput.EchoNormal
	m.secretInput.Blur()
}

func (m *Model) addSecretName(name string) {
	for _, n := range m.cfg.SecretNames {
		if n == name {
			return
		}
	}
	names := append(append([]string{}, m.cfg.SecretNames...), name)
	sort.Strings(names)
	m.cfg.SecretNames = names
}

func (m Model) updateSecrets(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.secretStage != secretList {
		switch msg.String() {
		case "esc":
			m.resetSecretInput()
			return m, nil
		case "enter":
			if m.secretStage == secretEnterName {
				name := strings.TrimSpace(m.secretInput.Value())
				if !interp.ValidSecretName(name) {
					m.setStatus("Invalid secret name (letters, numbers, underscores)", true)
					return m, nil
				}
				m.secretName = name
				m.secretStage = secretEnterValue
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
			if m.secrets == nil {
				m.setStatus("Keychain not ready", true)
				return m, nil
			}
			if err := m.secrets.Set(m.secretName, val); err != nil {
				m.setStatus("Could not store secret: "+err.Error(), true)
				return m, nil
			}
			m.addSecretName(m.secretName)
			if err := m.persistConfig(); err != nil {
				m.setStatus("Secret saved, config error: "+err.Error(), true)
			} else {
				m.setStatus(fmt.Sprintf("✓ Secret '%s' saved", m.secretName), false)
			}
			m.resetSecretInput()
			m.refreshSecretState()
			return m, nil
		}
		var cmd tea.Cmd
		m.secretInput, cmd = m.secretInput.Update(msg)
		return m, cmd
	}

	names := m.cfg.SecretNames
	switch msg.String() {
	case "esc", "alt+k":
		m.closeModal()
	case "up", "k":
		if m.secretCursor > 0 {
			m.secretCursor--
		}
	case "down", "j":
		if m.secretCursor < len(names)-1 {
			m.secretCursor++
		}
	case "n":
		m.secretStage = secretEnterName
		m.secretInput.SetValue("")
		m.secretInput.Placeholder = "SECRET_NAME"
		m.secretInput.EchoMode = textinput.EchoNormal
		return m, m.secretInput.Focus()
	case "enter":
		if len(names) > 0 {
			m.secretName = names[m.secretCursor]
			m.secretStage = secretEnterValue
			m.secretInput.SetValue("")
			m.secretInput.Placeholder = "new value (hidden)"
			m.secretInput.EchoMode = textinput.EchoPassword
			return m, m.secretInput.Focus()
		}
	case "d":
		if len(names) > 0 {
			name := names[m.secretCursor]
			if m.secrets != nil {
				if err := m.secrets.Delete(name); err != nil {
					m.setStatus("Could not delete secret: "+err.Error(), true)
					return m, nil
				}
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
			m.refreshSecretState()
		}
	}
	return m, nil
}

func (m Model) renderSecretsModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Secrets Manager"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("j/k: Navigate  n: New  Enter: Edit  d: Delete  Esc: Close"))
	b.WriteString("\n\n")

	if m.secrets != nil && !m.secrets.Persistent() {
		b.WriteString(statusYellowStyle.Render("⚠ OS keychain unavailable - secrets are kept in memory only"))
		b.WriteString("\n\n")
	}
	if len(m.cfg.SecretNames) == 0 && m.secretStage == secretList {
		b.WriteString(dimStyle.Render("No secrets yet. Press 'n' to create one."))
		b.WriteString("\n")
	}
	for i, name := range m.cfg.SecretNames {
		selected := i == m.secretCursor && m.secretStage == secretList
		prefix := "  "
		if selected {
			prefix = cursorStyle.Render("▶ ")
		}
		state := statusRedStyle.Render("● missing")
		if m.secretIsSet[name] {
			state = statusGreenStyle.Render("● set")
		}
		line := fmt.Sprintf("%s%s  %s", prefix, name, state)
		if selected {
			line = lipgloss.NewStyle().Background(panelBg).Render(line)
		}
		b.WriteString(line)
		b.WriteString("\n")
	}

	switch m.secretStage {
	case secretEnterName:
		b.WriteString("\n")
		b.WriteString(helpStyle.Render("Secret name (letters, numbers, underscores only):"))
		b.WriteString("\n")
		b.WriteString(m.secretInput.View())
		b.WriteString("\n")
	case secretEnterValue:
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("Value for '%s' (hidden):", m.secretName)))
		b.WriteString("\n")
		b.WriteString(m.secretInput.View())
		b.WriteString("\n")
	}
	b.WriteString("\n")
	b.WriteString(dimStyle.Render("Use in requests as: {{ secret.NAME }}"))
	return m.placeModal("39", b.String())
}

// ---- Help ----

func (m Model) renderHelpModal() string {
	var b strings.Builder
	title := "Keybindings"
	if m.opts.Version != "" {
		title += dimStyle.Render("  PostBoy " + m.opts.Version)
	}
	b.WriteString(titleStyle.Render(title))
	b.WriteString("\n\n")

	keyStyle := lipgloss.NewStyle().Foreground(postmanOrange).Bold(true).Width(18)
	descStyle := lipgloss.NewStyle().Foreground(textPrimary)
	for _, kb := range m.keys.helpBindings() {
		h := kb.Help()
		fmt.Fprintf(&b, "  %s %s\n", keyStyle.Render(h.Key), descStyle.Render(h.Desc))
	}
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Headers / Params lists"))
	b.WriteString("\n\n")
	lk := defaultListKeyMap()
	for _, kb := range []key.Binding{lk.Up, lk.Down, lk.New, lk.Edit, lk.Toggle, lk.Delete} {
		h := kb.Help()
		fmt.Fprintf(&b, "  %s %s\n", keyStyle.Render(h.Key), descStyle.Render(h.Desc))
	}
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Response panel (when focused)"))
	b.WriteString("\n\n")
	for _, kv := range [][2]string{
		{"/", "Search (n / N: next / previous match)"},
		{"f", "Filter JSON with a path, e.g. data.#.id or items.#(price>10)"},
		{"r", "Toggle raw / formatted body"},
		{"y", "Copy body (or filter result) to clipboard"},
		{"s", "Save body (or filter result) to a file"},
		{"Esc", "Clear search and filter"},
	} {
		fmt.Fprintf(&b, "  %s %s\n", keyStyle.Render(kv[0]), descStyle.Render(kv[1]))
	}
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("Press Esc or ? to close"))
	return m.placeModal("205", b.String())
}

func (m Model) placeModal(color, content string) string {
	style := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(color)).
		Padding(1, 3).
		Width(max(m.termWidth-10, 20)).
		MaxHeight(max(m.termHeight-2, 5)).
		Background(darkBg)
	return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, style.Render(content))
}
