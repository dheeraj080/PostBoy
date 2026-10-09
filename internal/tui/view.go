package tui

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

const (
	minWidth  = 80
	minHeight = 24
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

	content := strings.Join([]string{
		m.renderURLBar(),
		m.renderHeaderLine(),
		lipgloss.JoinHorizontal(lipgloss.Top, left, right),
		lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("─", m.termWidth-frameStyle.GetHorizontalFrameSize())),
		m.renderFooter(),
	}, "\n")
	return frameStyle.Width(m.termWidth - 2).Height(m.termHeight - 2).Render(content)
}

func (m Model) renderLeftPane() string {
	leftW, _ := m.columnWidths()
	head := fixBox(tabBarStyle.Render(m.renderRequestTabs(leftW)), leftW, paneHeadRows)
	return lipgloss.JoinVertical(lipgloss.Left, head, m.renderRequestPanel(leftW, m.panelHeight()))
}

func (m Model) renderRightPane() string {
	_, rightW := m.columnWidths()
	style := rightPaneStyle
	if m.focus == focusResContent {
		style = focusedRightPaneStyle
	}
	head := m.renderResponseHeader(rightW)
	panel := m.renderResponsePanel()
	column := lipgloss.JoinVertical(lipgloss.Left, head, panel)
	return style.Width(rightW).Height(m.panelHeight()+paneHeadRows).Render(column)
}

// urlBarParts holds the rendered parts of the URL bar so that both
// urlChromeWidth and renderURLBar use the exact same strings.
type urlBarParts struct {
	brand  string
	method string
	send   string
}

func (m Model) renderURLBarParts() urlBarParts {
	brand := lipgloss.NewStyle().Foreground(textSecondary).Render("PostBoy")
	method := lipgloss.NewStyle().
		Foreground(methodColor(m.method)).
		Bold(true).
		Width(methodW).
		Align(lipgloss.Center).
		Render("[" + strings.ToUpper(m.method) + "]")
	w := lipgloss.Width("Send") + 4
	top := lipgloss.NewStyle().Foreground(postmanOrange).Render(strings.Repeat("▄", w))
	middle := sendButtonStyle.Render("Send")
	bottom := lipgloss.NewStyle().Foreground(postmanOrange).Render(strings.Repeat("▀", w))
	send := lipgloss.NewStyle().MarginLeft(1).Render(
		lipgloss.JoinVertical(lipgloss.Left, top, middle, bottom))
	return urlBarParts{brand: brand, method: method, send: send}
}

// renderURLBar renders brand, method tag, standalone URL box, and Send button.
func (m Model) renderURLBar() string {
	parts := m.renderURLBarParts()

	innerW := m.termWidth - frameStyle.GetHorizontalFrameSize()
	urlBoxWidth := innerW - lipgloss.Width(parts.brand) - lipgloss.Width(parts.method) - 2 - lipgloss.Width(parts.send)

	urlBoxStyle := urlBoxStyle
	if m.focus == focusURL {
		urlBoxStyle = focusedURLBoxStyle
	}
	urlBoxStyle = urlBoxStyle.Width(urlBoxWidth - urlBoxStyle.GetHorizontalBorderSize())
	m.urlInput.Width = urlBoxWidth - urlBoxStyle.GetHorizontalBorderSize() - urlBoxStyle.GetHorizontalPadding() - 1
	urlBox := urlBoxStyle.Render(m.urlInput.View())

	bar := lipgloss.JoinHorizontal(lipgloss.Center, urlBox, parts.send)
	return lipgloss.JoinHorizontal(lipgloss.Center, parts.brand, " ", parts.method, " ", bar)
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
	left = ansi.Truncate(left, m.termWidth/2, "…")

	env := lipgloss.NewStyle().
		Background(lipgloss.Color("#007ACC")).
		Foreground(lipgloss.Color("#FFFFFF")).
		Bold(true).
		Padding(0, 1).
		Render("• " + m.env().Name)
	left = strings.ReplaceAll(left, "\n", " ")
	env = strings.ReplaceAll(env, "\n", " ")
	gap := m.termWidth - frameStyle.GetHorizontalFrameSize() - lipgloss.Width(left) - lipgloss.Width(env)
	if gap < 1 {
		gap = 1
	}
	return left + spaces(gap) + env
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

func (m Model) renderResponsePanel() string {
	var content string
	if m.resTab == resTabBody && m.respBody == "" && !m.loading && m.statusCode == 0 {
		content = "\n" + lipgloss.NewStyle().Foreground(lipgloss.Color("#858585")).Render("Response will appear here") + "\n\n" +
			helpStyle.Render("Press Enter in the URL bar or Alt+R to send")
	} else {
		content = m.viewport.View()
	}
	return content
}

// renderResponseHeader renders response tabs + actions + rule + status line.
func (m Model) renderResponseHeader(paneW int) string {
	innerW := paneW - 2
	rule := lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("─", innerW))
	tabs := m.renderResponseTabs(innerW)
	actions := m.renderResponseQuickActions()
	actions = helpStyle.Render(actions)
	row1 := fitRow(tabs, actions, innerW)

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
	return row1 + "\n" + rule + "\n" + row2
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

func (m Model) renderRequestTabs(paneW int) string {
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
	var filler []string
	for _, t := range tabs {
		if t.tab == reqTabBody && !m.supportsBody() {
			name := inactiveTabStyle.Foreground(lipgloss.Color("#6E6E6E")).Render(t.name)
			out = append(out, name)
			filler = append(filler, lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("─", lipgloss.Width(name))))
			continue
		}
		active := m.reqTab == t.tab
		name := m.tabStyle(active, m.focus == focusReqTabs).Render(t.name)
		out = append(out, name)
		fg := borderColor
		if active {
			fg = postmanOrange
		}
		filler = append(filler, lipgloss.NewStyle().Foreground(fg).Render(strings.Repeat("─", lipgloss.Width(name))))
	}
	tabsRow := lipgloss.JoinHorizontal(lipgloss.Left, out...)
	fillerRow := lipgloss.JoinHorizontal(lipgloss.Left, filler...)
	fillerWidth := lipgloss.Width(fillerRow)
	if fillerWidth < paneW {
		fillerRow += lipgloss.NewStyle().Foreground(borderColor).Render(strings.Repeat("─", paneW-fillerWidth))
	}
	return tabsRow + "\n" + fillerRow
}

func (m Model) renderResponseTabs(paneW int) string {
	return lipgloss.JoinHorizontal(lipgloss.Left,
		m.tabStyle(m.resTab == resTabBody, m.focus == focusResTabs).Render("BODY"),
		m.tabStyle(m.resTab == resTabHeaders, m.focus == focusResTabs).Render("HEADERS"))
}

func (m Model) tabStyle(active, barFocused bool) lipgloss.Style {
	if !active {
		return inactiveTabStyle
	}
	if barFocused {
		return activeTabStyle.Foreground(postmanOrange).Bold(true)
	}
	return activeTabStyle
}

func (m Model) renderFooter() string {
	all := m.keys.footerBindings()
	var parts []string
	for _, kb := range all {
		h := kb.Help()
		parts = append(parts, footerKeyStyle.Render(shortKey(h.Key))+" "+footerDescStyle.Render(shortDesc(h.Desc)))
	}
	// Drop from the middle, keeping first (Send) and last (Quit).
	innerW := frameStyle.GetHorizontalFrameSize()
	for lipgloss.Width(strings.Join(parts, "   ")) > m.termWidth-innerW && len(parts) > 2 {
		mid := len(parts) / 2
		parts = append(parts[:mid], parts[mid+1:]...)
	}
	return fixBox(strings.Join(parts, "   "), m.termWidth-innerW, 1)
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
