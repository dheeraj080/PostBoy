package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	minWidth  = 60
	minHeight = 20
)

// View implements tea.Model. It must not mutate component sizes; see layout.
func (m Model) View() string {
	if m.termWidth == 0 || m.termHeight == 0 {
		return "Initializing..."
	}
	if m.termWidth < minWidth || m.termHeight < minHeight {
		return lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center,
			statusYellowStyle.Render(fmt.Sprintf("Terminal too small (%dx%d)\nResize to at least %dx%d",
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

	width := m.usableWidth()
	reqH, resH := m.panelHeights()

	sections := []string{
		m.renderTopBar(),
		m.renderInfoBar(width),
		"",
		m.renderRequestTabs(),
		m.renderRequestPanel(width, reqH),
		m.renderResponseHeader(),
		m.renderResponsePanel(width, resH),
		m.renderFooter(),
	}
	return strings.Join(sections, "\n")
}

func (m Model) renderTopBar() string {
	urlStyle := urlInputStyle
	if m.focus == focusURL {
		urlStyle = focusedURLInputStyle
	}
	send := sendButtonStyle.Render("Send")
	if m.loading {
		send = sendButtonLoadingStyle.Render("Loading...")
	}
	return lipgloss.JoinHorizontal(lipgloss.Center,
		methodStyle.Render(m.method),
		urlStyle.Render(m.urlInput.View()),
		send)
}

func (m Model) renderInfoBar(width int) string {
	badge := envBadgeStyle.Render(m.env().Name)
	title := lipgloss.NewStyle().Foreground(textPrimary).Bold(true).Render(truncate(m.requestTitle(), width/2)) + "  "
	style := statusGreenStyle
	if m.statusErr {
		style = statusRedStyle
	}
	maxStatus := max(width-lipgloss.Width(badge)-lipgloss.Width(title)-1, 10)
	status := style.Render(truncate(m.status, maxStatus))
	gap := max(width-lipgloss.Width(title)-lipgloss.Width(status)-lipgloss.Width(badge), 1)
	return title + status + strings.Repeat(" ", gap) + badge
}

func (m Model) renderRequestPanel(width, height int) string {
	focused := m.focus == focusReqContent
	style := panelStyle
	if focused {
		style = focusedPanelStyle
	}
	var content string
	switch m.reqTab {
	case reqTabParams:
		content = m.params.View(focused, width-2)
	case reqTabAuth:
		content = m.auth.View(focused, width-2)
	case reqTabHeaders:
		content = m.headers.View(focused, width-2)
	case reqTabBody:
		content = m.renderBodyTab(width-2, height)
	}
	return style.Width(width).Height(height).MaxHeight(height + 1).Render(content)
}

func (m Model) renderResponsePanel(width, height int) string {
	style := panelStyle
	if m.focus == focusResContent {
		style = focusedPanelStyle
	}
	var content string
	if m.resTab == resTabBody && m.respBody == "" && !m.loading && m.statusCode == 0 {
		empty := dimStyle.Render("Response will appear here\n\n") +
			helpStyle.Render("Press Enter in the URL bar or Alt+R to send")
		content = lipgloss.Place(width-2, height-1, lipgloss.Center, lipgloss.Center, empty)
	} else {
		content = m.viewport.View()
	}
	return style.Width(width).Height(height).Render(content)
}

// renderResponseHeader joins the response tabs, metadata and toolbar,
// truncating the toolbar to the remaining width.
func (m Model) renderResponseHeader() string {
	left := lipgloss.JoinHorizontal(lipgloss.Top, m.renderResponseTabs(), "  ", m.renderResponseMeta(), "  ")
	room := m.termWidth - lipgloss.Width(left) - 1
	if room <= 0 {
		return left
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, left, ansi.Truncate(m.renderResponseToolbar(), room, "…"))
}

func (m Model) renderRequestTabs() string {
	tabs := []struct {
		tab  requestTab
		name string
	}{
		{reqTabParams, "Params"},
		{reqTabAuth, "Authorization"},
		{reqTabHeaders, "Headers"},
		{reqTabBody, "Body"},
	}
	var out []string
	for _, t := range tabs {
		if t.tab == reqTabBody && !m.supportsBody() {
			out = append(out, inactiveTabStyle.Foreground(textMuted).Render(t.name))
			continue
		}
		out = append(out, m.tabStyle(m.reqTab == t.tab, m.focus == focusReqTabs).Render(t.name))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, out...)
}

func (m Model) renderResponseTabs() string {
	return lipgloss.JoinHorizontal(lipgloss.Left,
		m.tabStyle(m.resTab == resTabBody, m.focus == focusResTabs).Render("Body"),
		m.tabStyle(m.resTab == resTabHeaders, m.focus == focusResTabs).Render("Headers"))
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

func (m Model) renderResponseMeta() string {
	if m.statusCode == 0 {
		return ""
	}
	parts := []string{statusStyle(m.statusCode).Render(fmt.Sprintf("%d", m.statusCode))}
	if m.responseTime > 0 {
		parts = append(parts, responseMetaStyle.Render(m.responseTime.Round(time.Millisecond).String()))
	}
	parts = append(parts, responseMetaStyle.Render(formatSize(m.respSize)))
	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func (m Model) renderFooter() string {
	var parts []string
	for _, kb := range m.keys.footerBindings() {
		h := kb.Help()
		parts = append(parts, h.Key+": "+shortDesc(h.Desc))
	}
	return helpStyle.Render(truncate(strings.Join(parts, " | "), m.termWidth))
}

// shortDesc trims parenthetical notes from help text for the footer.
func shortDesc(s string) string {
	if i := strings.Index(s, " ("); i > 0 {
		return s[:i]
	}
	return s
}
