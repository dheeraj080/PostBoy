package tui

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

const (
	minWidth  = 80
	minHeight = 20
)

// View implements tea.Model. It must not mutate component sizes; see layout.
func (m Model) View() string {
	if m.termWidth == 0 || m.termHeight == 0 {
		return "Initializing..."
	}
	if m.termWidth < minWidth || m.termHeight < minHeight {
		return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center,
			lipgloss.NewStyle().Foreground(lipgloss.Color("#DCDCAA")).Render(
				fmt.Sprintf("Terminal too small (%dx%d)\nResize to at least %dx%d",
					m.termWidth, m.termHeight, minWidth, minHeight)))
	}

	switch m.modal {
	case modalHistory:
		return m.renderHistoryModal()
	case modalSecrets:
		return m.renderSecretsModal()
	case modalEnv:
		return m.renderEnvModal()
	case modalHelp:
		return m.renderHelpModal()
	case modalCollections:
		return m.renderCollectionsModal()
	case modalSave:
		return m.renderSaveModal()
	case modalImport:
		return m.renderImportModal()
	}

	left := m.renderLeftPane()
	right := m.renderRightPane()

	out := strings.Join([]string{
		m.renderURLBar(),
		m.renderHeaderLine(),
		"",
		lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#858585")).Render(strings.Repeat("─", m.termWidth)),
		m.renderFooter(),
	}, "\n")
	return fixBox(out, m.termWidth, m.termHeight)
}

func (m Model) renderLeftPane() string {
	leftW, _ := m.columnWidths()
	head := fixBox(m.renderRequestTabs(), leftW+2, paneHeadRows)
	return lipgloss.JoinVertical(lipgloss.Left, head, m.renderRequestPanel(leftW, m.panelHeight()))
}

func (m Model) renderRightPane() string {
	_, rightW := m.columnWidths()
	head := fixBox(m.renderResponseHeader(), rightW+2, paneHeadRows)
	return lipgloss.JoinVertical(lipgloss.Left, head, m.renderResponsePanel(rightW, m.panelHeight()))
}

// renderURLBar renders brand, method tag, standalone URL box, and Send button.
func (m Model) renderURLBar() string {
	brand := lipgloss.NewStyle().Foreground(lipgloss.Color("#858585")).Render("PostBoy")
	method := lipgloss.NewStyle().
		Foreground(methodColor(m.method)).
		Bold(true).
		Width(methodW).
		Align(lipgloss.Center).
		Render("[" + strings.ToUpper(m.method) + "]")

	urlBoxStyle := urlBoxStyle
	if m.focus == focusURL {
		urlBoxStyle = focusedURLBoxStyle
	}
	urlBox := urlBoxStyle.Render(m.urlInput.View())

	send := sendButtonStyle.Render(" Send ")
	if m.loading {
		send = sendButtonLoadingStyle.Render(" Loading... ")
	}
	send = lipgloss.NewStyle().Width(sendW).Align(lipgloss.Center).Render(send)

	return lipgloss.JoinHorizontal(lipgloss.Center, brand, " ", method, " ", urlBox, " ", send)
}

// renderHeaderLine shows request title + status on the left, env pill on the right.
func (m Model) renderHeaderLine() string {
	title := lipgloss.NewStyle().Foreground(lipgloss.Color("#CCCCCC")).Bold(true).Render(m.requestTitle())

	var meta string
	switch {
	case m.statusErr:
		meta = lipgloss.NewStyle().Foreground(lipgloss.Color("#F48771")).Render(m.status)
	case m.statusCode != 0:
		parts := []string{fmt.Sprintf("%d %s", m.statusCode, http.StatusText(m.statusCode))}
		if m.responseTime > 0 {
			parts = append(parts, m.responseTime.Round(time.Millisecond).String())
		}
		meta = statusStyle(m.statusCode).Render(strings.Join(parts, " • "))
	case m.status != "" && m.status != "Ready":
		meta = lipgloss.NewStyle().Foreground(lipgloss.Color("#4EC9B0")).Render(m.status)
	}
	left := title
	if meta != "" {
		left += "  " + meta
	}
	left = truncate(left, m.termWidth/2)

	env := lipgloss.NewStyle().
		Background(lipgloss.Color("#007ACC")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1).
		Render("• " + m.env().Name)
	return fitRow(left, env, m.termWidth)
}

func (m Model) renderRequestPanel(width, height int) string {
	focused := m.focus == focusReqContent
	style := panelStyle
	if focused {
		style = focusedPanelStyle
	}
	inner := width - 2
	var content string
	switch m.reqTab {
	case reqTabParams:
		content = m.params.View(focused, inner, height)
	case reqTabAuth:
		content = m.auth.View(focused, inner)
	case reqTabHeaders:
		content = m.headers.View(focused, inner, height)
	case reqTabBody:
		content = m.renderBodyTab(inner, height)
	}
	return style.Width(width).Height(height).MaxHeight(height + 2).Render(content)
}

func (m Model) renderResponsePanel(width, height int) string {
	style := panelStyle
	if m.focus == focusResContent {
		style = focusedPanelStyle
	}
	var content string
	if m.resTab == resTabBody && m.respBody == "" && !m.loading && m.statusCode == 0 {
		content = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#858585")).Render("Response will appear here") + "\n\n" +
			helpStyle.Render("Press Enter in the URL bar or Alt+R to send")
	} else {
		content = m.viewport.View()
	}
	return style.Width(width).Height(height).MaxHeight(height + 2).Render(content)
}

// renderResponseHeader renders response tabs + quick actions, then meta/input.
func (m Model) renderResponseHeader() string {
	tabs := m.renderResponseTabs()
	actions := m.renderResponseQuickActions()
	row1 := fitRow(tabs, actions, m.termWidth)

	var row2 string
	if m.resp.inputMode != respInputNone {
		row2 = m.renderResponseToolbar()
	} else {
		parts := []string{}
		if m.statusCode != 0 {
			parts = append(parts, fmt.Sprintf("%d %s", m.statusCode, http.StatusText(m.statusCode)))
		}
		if m.responseTime > 0 {
			parts = append(parts, m.responseTime.Round(time.Millisecond).String())
		}
		if m.respSize > 0 {
			parts = append(parts, formatSize(m.respSize))
		}
		if len(parts) == 0 {
			row2 = helpStyle.Render("Response metadata will appear here")
		} else {
			row2 = statusStyle(m.statusCode).Render(strings.Join(parts, "  "))
		}
	}
	return row1 + "\n" + row2
}

func (m Model) renderResponseQuickActions() string {
	if !(m.resp.hasBody() || m.resp.binary) {
		return ""
	}
	pills := []string{"f: filter", "r: pretty", "y copy", "• s save"}
	var kept []string
	for i := len(pills) - 1; i >= 0; i-- {
		candidate := pills[i]
		if len(kept) > 0 {
			candidate = candidate + " " + strings.Join(kept, " ")
		}
		if lipgloss.Width(candidate) <= m.termWidth {
			kept = append([]string{pills[i]}, kept...)
		}
	}
	if len(kept) == 0 {
		return ""
	}
	return strings.Join(kept, "  ")
}

func (m Model) renderRequestTabs() string {
	tabs := []struct {
		tab  requestTab
		name string
	}{
		{reqTabParams, "PARAMS"},
		{reqTabAuth, "AUTH"},
		{reqTabHeaders, "HEADERS"},
		{reqTabBody, "BODY"},
	}
	var out []string
	for _, t := range tabs {
		if t.tab == reqTabBody && !m.supportsBody() {
			out = append(out, inactiveTabStyle.Foreground(lipgloss.Color("#6E6E6E")).Render(t.name))
			continue
		}
		out = append(out, m.tabStyle(m.reqTab == t.tab, m.focus == focusReqTabs).Render(t.name))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, out...)
}

func (m Model) renderResponseTabs() string {
	return lipgloss.JoinHorizontal(lipgloss.Left,
		m.tabStyle(m.resTab == resTabBody, m.focus == focusResTabs).Render("BODY"),
		m.tabStyle(m.resTab == resTabHeaders, m.focus == focusResTabs).Render("HEADERS"))
}

func (m Model) tabStyle(active, barFocused bool) lipgloss.Style {
	if !active {
		return inactiveTabStyle
	}
	if barFocused {
		return activeTabStyle.Background(darkBg)
	}
	return activeTabStyle
}

func (m Model) renderFooter() string {
	all := m.keys.footerBindings()
	var parts []string
	for _, kb := range all {
		h := kb.Help()
		parts = append(parts, shortKey(h.Key)+" "+shortDesc(h.Desc))
	}
	// Drop optional items from the left until everything fits.
	for lipgloss.Width(strings.Join(parts, "   ")) > m.termWidth && len(parts) > 2 {
		parts = parts[1:]
	}
	return fixBox(strings.Join(parts, "   "), m.termWidth, 1)
}

func shortKey(k string) string {
	if i := strings.Index(k, " /"); i > 0 {
		return k[:i]
	}
	return k
}

func shortDesc(s string) string {
	if i := strings.Index(s, " ("); i > 0 {
		return s[:i]
	}
	return s
}
