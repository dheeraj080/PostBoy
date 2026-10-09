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
	minHeight = 24
)

// View implements tea.Model. It must not mutate component sizes; see layout.
//
// Layout, top to bottom (all rows are exactly innerWidth cells wide):
//
//	3 rows  URL bar   brand | method chip | URL box | Send
//	1 row   status    title • status • time            env badge
//	H-8     panes     request tabs + rule + panel  │  response tabs + rule + status + viewport
//	1 row   footer rule
//	1 row   footer    key/desc pairs
func (m Model) View() string {
	if m.termWidth == 0 || m.termHeight == 0 {
		return "Initializing..."
	}
	if m.termWidth < minWidth || m.termHeight < minHeight {
		msg := lipgloss.NewStyle().Foreground(statusWarn).Background(darkBg).Render(
			fmt.Sprintf("Terminal too small (%dx%d)\nResize to at least %dx%d",
				m.termWidth, m.termHeight, minWidth, minHeight))
		placed := lipgloss.Place(m.termWidth, m.termHeight, lipgloss.Center, lipgloss.Center, msg)
		return fixBlock(placed, m.termWidth, m.termHeight)
	}

	switch m.modal {
	case modalHistory:
		return fixBlock(m.renderHistoryModal(), m.termWidth, m.termHeight)
	case modalSecrets:
		return fixBlock(m.renderSecretsModal(), m.termWidth, m.termHeight)
	case modalEnv:
		return fixBlock(m.renderEnvModal(), m.termWidth, m.termHeight)
	case modalHelp:
		return fixBlock(m.renderHelpModal(), m.termWidth, m.termHeight)
	case modalCollections:
		return fixBlock(m.renderCollectionsModal(), m.termWidth, m.termHeight)
	case modalSave:
		return fixBlock(m.renderSaveModal(), m.termWidth, m.termHeight)
	case modalImport:
		return fixBlock(m.renderImportModal(), m.termWidth, m.termHeight)
	}

	content := strings.Join([]string{
		m.renderURLBar(),
		m.renderHeaderLine(),
		m.renderMain(),
		m.renderFooterRule(),
		m.renderFooter(),
	}, "\n")
	return frameStyle.Width(m.termWidth - 2).Height(m.termHeight - 2).Render(content)
}

// renderMain joins the two panes, re-padding each side so a short row can
// never shift the divider.
func (m Model) renderMain() string {
	leftW, rightW := m.columnWidths()
	left := strings.Split(m.renderLeftPane(), "\n")
	right := strings.Split(m.renderRightPane(), "\n")
	h := max(len(left), len(right))
	for len(left) < h {
		left = append(left, "")
	}
	for len(right) < h {
		right = append(right, "")
	}
	rows := make([]string, h)
	for i := range rows {
		// The right pane carries the one-cell divider as its left border.
		rows[i] = fitLine(left[i], leftW) + fitLine(right[i], rightW+1)
	}
	return strings.Join(rows, "\n")
}

// renderLeftPane renders the request tab bar (labels + rule) and the panel
// below it: paneHeadRows + panelHeight rows in total.
func (m Model) renderLeftPane() string {
	leftW, _ := m.columnWidths()
	innerW := leftW - panelStyle.GetHorizontalPadding()
	head := panelStyle.Width(leftW).Height(paneHeadRows).Render(m.renderRequestTabs(innerW))
	panel := m.renderRequestPanel(leftW, m.panelHeight())
	return head + "\n" + panel
}

// renderRightPane renders the response tab bar, status line and viewport,
// with the pane divider as its left border.
func (m Model) renderRightPane() string {
	_, rightW := m.columnWidths()
	style := rightPaneStyle
	if m.focus == focusResContent {
		style = focusedRightPaneStyle
	}
	column := strings.Join([]string{
		m.renderResponseHeader(rightW),
		m.renderResponsePanel(),
	}, "\n")
	return style.Width(rightW).Height(m.panelHeight() + paneHeadRows).Render(column)
}

// urlBarParts holds the fixed-width siblings of the URL box.
type urlBarParts struct {
	brand  string
	method string
	send   string
}

func (m Model) renderURLBarParts() urlBarParts {
	return urlBarParts{
		brand:  brandStyle.Render("PostBoy"),
		method: m.renderMethodChip(),
		send:   m.sendLabel(),
	}
}

// sendLabel is the Send button's content for the current state.
func (m Model) sendLabel() string {
	if m.loading {
		return sendButtonLoadingStyle.Render("Loading...")
	}
	return sendButtonStyle.Render("Send")
}

// renderMethodChip renders the [POST] chip: an orange bold label inside a
// grey rounded border.
func (m Model) renderMethodChip() string {
	return methodChipStyle.Render("[" + strings.ToUpper(m.method) + "]")
}

// renderURLBar renders the three-row URL band: brand, method chip, the
// orange-bordered URL box and the Send button.
func (m Model) renderURLBar() string {
	innerW := m.termWidth - frameStyle.GetHorizontalFrameSize()

	parts := m.renderURLBarParts()
	brandW := lipgloss.Width(parts.brand)
	chip := rows3(parts.method, lipgloss.Width(parts.method))
	sendW := lipgloss.Width(parts.send)

	boxW := innerW - (brandW + 1 + lipgloss.Width(chip[1]) + 1 + 1 + sendW)
	if boxW < 24 {
		boxW = 24
	}
	box := rows3(m.renderURLBox(boxW), boxW)

	gap := fill(1)
	blankBrand, blankSend := fill(brandW), fill(sendW)
	rows := [3]string{
		fitLine(blankBrand+gap+chip[0]+gap+box[0]+gap+blankSend, innerW),
		fitLine(parts.brand+gap+chip[1]+gap+box[1]+gap+parts.send, innerW),
		fitLine(blankBrand+gap+chip[2]+gap+box[2]+gap+blankSend, innerW),
	}
	return strings.Join(rows[:], "\n")
}

// renderURLBox renders the orange-bordered URL box: three rows total, with
// the styled URL (scheme orange, rest white, block caret) inside.
func (m Model) renderURLBox(boxW int) string {
	borderW := urlBoxStyle.GetHorizontalBorderSize()
	interior := max(boxW-borderW-urlBoxStyle.GetHorizontalPadding(), 1)
	st := urlBoxStyle.Width(max(boxW-borderW, interior+urlBoxStyle.GetHorizontalPadding()))
	return st.Render(fitLine(m.renderURLValue(interior), interior))
}

// renderURLValue paints the URL text: the scheme in orange, the rest in
// white, and an orange block caret at the editing position. The value comes
// from the bubbles textinput model; only the styling is done here so the
// scheme can be coloured separately.
func (m Model) renderURLValue(w int) string {
	if w <= 0 {
		return ""
	}
	value := m.urlInput.Value()
	runes := []rune(value)
	if len(runes) == 0 {
		ph := truncate(m.urlInput.Placeholder, w)
		if !m.urlInput.Focused() {
			return fitLine(urlPlaceholderStyle.Render(ph), w)
		}
		phRunes := []rune(ph)
		if len(phRunes) == 0 {
			return fitLine(urlCursorStyle.Render(" "), w)
		}
		first, rest := string(phRunes[0]), string(phRunes[1:])
		return fitLine(urlCursorStyle.Render(first)+urlPlaceholderStyle.Render(rest), w)
	}

	schemeLen := 0
	for _, p := range []string{"https://", "http://", "{{BASE_URL}}"} {
		if strings.HasPrefix(value, p) {
			schemeLen = len([]rune(p))
			break
		}
	}

	pos := m.urlInput.Position()
	if pos < 0 {
		pos = 0
	}
	if pos > len(runes) {
		pos = len(runes)
	}
	caretAtEnd := m.urlInput.Focused() && pos == len(runes)

	// Window of runes that stays visible; the caret always has a cell.
	visible := w
	if caretAtEnd {
		visible = max(w-1, 1)
	}
	start := 0
	if pos >= visible {
		start = pos - visible + 1
	}
	end := min(len(runes), start+visible)

	var b strings.Builder
	for i := start; i < end; i++ {
		ch := string(runes[i])
		switch {
		case i == pos && m.urlInput.Focused():
			b.WriteString(urlCursorStyle.Render(ch))
		case i < schemeLen:
			b.WriteString(urlSchemeStyle.Render(ch))
		default:
			b.WriteString(urlTextStyle.Render(ch))
		}
	}
	if caretAtEnd {
		b.WriteString(urlCursorStyle.Render(" "))
	}
	return fitLine(b.String(), w)
}

// renderHeaderLine shows the request title and response status on the left
// and the environment badge on the right.
func (m Model) renderHeaderLine() string {
	innerW := m.termWidth - frameStyle.GetHorizontalFrameSize()

	left := titleStyle.Render(m.requestTitle())
	switch {
	case m.statusErr:
		left += fill(2) + statusRedStyle.Render(m.status)
	case m.statusCode != 0:
		left += fill(2) + statusGreenBold.Render(
			fmt.Sprintf("%d %s", m.statusCode, http.StatusText(m.statusCode)))
		if m.responseTime > 0 {
			left += statusMetaStyle.Render(" • " + m.responseTime.Round(time.Millisecond).String())
		}
	case m.status != "" && m.status != "Ready":
		left += fill(2) + statusMetaStyle.Render(m.status)
	}

	badge := roundedChip("● "+m.env().Name, envBadgeEdgeStyle, envBadgeStyle)
	return fitRow(left, badge, innerW)
}

// renderRequestPanel renders the active request tab's content.
func (m Model) renderRequestPanel(width, height int) string {
	focused := m.focus == focusReqContent
	style := panelStyle
	if focused {
		style = focusedPanelStyle
	}
	inner := width - panelStyle.GetHorizontalPadding()
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
	return style.Width(width).Height(height).MaxHeight(height).Render(content)
}

// renderResponsePanel renders the viewport (or the empty state): one row
// less than the request panel, because the response header is three rows.
func (m Model) renderResponsePanel() string {
	height := max(m.panelHeight()-1, 1)
	_, rightW := m.columnWidths()
	inner := rightW - rightPaneStyle.GetHorizontalPadding()

	var content string
	if m.resTab == resTabBody && m.respBody == "" && m.statusCode == 0 {
		content = dimStyle.Render("Response will appear here")
	} else {
		content = m.viewport.View()
	}
	return fixBlock(content, inner, height)
}

// renderResponseHeader renders the response tab row (tabs + key hints), the
// rule below it and the status line: three rows of paneW-2 cells.
func (m Model) renderResponseHeader(paneW int) string {
	innerW := paneW - rightPaneStyle.GetHorizontalPadding()
	labels, tabRule := m.responseTabBlocks(innerW)
	// The hints share the tab labels' row; only as many as fit next to the
	// labels are shown.
	row1 := fitRow(labels, m.responseActionsFor(innerW-lipgloss.Width(labels)), innerW)
	row2 := fitLine(tabRule, innerW)
	row3 := fitLine(m.renderResponseStatusLine(innerW), innerW)
	return row1 + "\n" + row2 + "\n" + row3
}

// responseTabBlocks returns the response tab labels and the rule row that
// carries the accent underline below the active tab.
func (m Model) responseTabBlocks(innerW int) (labels, ruleRow string) {
	tabs := []struct {
		tab  responseTab
		name string
	}{
		{resTabBody, "BODY"},
		{resTabHeaders, "HEADERS"},
	}
	var labelCells, ruleCells []string
	for _, t := range tabs {
		st, ruleFn := inactiveTabStyle, rule
		if m.resTab == t.tab {
			st, ruleFn = activeTabStyle, activeRule
		}
		cell := st.Render(t.name)
		labelCells = append(labelCells, cell)
		ruleCells = append(ruleCells, ruleFn(lipgloss.Width(cell)))
	}
	ruleRow = strings.Join(ruleCells, "")
	if w := lipgloss.Width(ruleRow); w < innerW {
		ruleRow += rule(innerW - w)
	}
	return strings.Join(labelCells, ""), ruleRow
}

// renderResponseTabs returns the response tab labels only.
func (m Model) renderResponseTabs(paneW int) string {
	labels, _ := m.responseTabBlocks(max(paneW-rightPaneStyle.GetHorizontalPadding(), 1))
	return labels
}

// renderResponseStatusLine renders "201 Created  142ms  1.4 KB": the code in
// bold green, the timing and size in grey.
func (m Model) renderResponseStatusLine(innerW int) string {
	if m.resp.inputMode != respInputNone {
		return m.renderResponseToolbar()
	}
	if m.statusCode == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(statusGreenBold.Render(
		fmt.Sprintf("%d %s", m.statusCode, http.StatusText(m.statusCode))))
	if m.responseTime > 0 {
		b.WriteString(fill(2) + statusMetaStyle.Render(m.responseTime.Round(time.Millisecond).String()))
	}
	if m.respSize > 0 {
		b.WriteString(fill(2) + statusMetaStyle.Render(formatSize(m.respSize)))
	}
	return fitLine(b.String(), innerW)
}

// renderResponseQuickActions renders the response key hints that fit in avail
// cells: rounded chips for filter and pretty, plain grey for copy/save. The
// plain hints are dropped first when the pane is narrow, then the chips.
func (m Model) renderResponseQuickActions() string {
	_, rightW := m.columnWidths()
	return m.responseActionsFor(max(rightW-rightPaneStyle.GetHorizontalPadding(), 1))
}

func (m Model) responseActionsFor(avail int) string {
	if !(m.resp.hasBody() || m.resp.binary) || avail <= 0 {
		return ""
	}
	plain := dimStyle.Render("y copy • s save")
	chipFilter := roundedChip("f: filter", hintChipEdgeStyle, hintChipStyle)
	chipPretty := roundedChip("r: pretty", hintChipEdgeStyle, hintChipStyle)
	chips := chipFilter + fill(1) + chipPretty
	switch {
	case lipgloss.Width(chips)+2+lipgloss.Width(plain) <= avail:
		return chips + fill(2) + plain
	case lipgloss.Width(chipPretty)+2+lipgloss.Width(chipFilter) <= avail:
		return chips
	case lipgloss.Width(chipFilter) <= avail:
		return chipFilter
	}
	return ""
}

// renderRequestTabs renders the request tab labels and the rule row below
// them: the active tab gets an accent underline sitting on the rule.
func (m Model) renderRequestTabs(innerW int) string {
	tabs := []struct {
		tab     requestTab
		name    string
		enabled bool
	}{
		{reqTabParams, "PARAMS", true},
		{reqTabAuth, "AUTH", true},
		{reqTabHeaders, "HEADERS", true},
		{reqTabBody, "BODY", m.supportsBody()},
	}
	var labelCells, ruleCells []string
	for _, t := range tabs {
		st, ruleFn := inactiveTabStyle, rule
		switch {
		case !t.enabled:
			st, ruleFn = disabledTabStyle, rule
		case m.reqTab == t.tab:
			st, ruleFn = activeTabStyle, activeRule
		}
		cell := st.Render(t.name)
		labelCells = append(labelCells, cell)
		ruleCells = append(ruleCells, ruleFn(lipgloss.Width(cell)))
	}
	labels := strings.Join(labelCells, "")
	ruleRow := strings.Join(ruleCells, "")
	if w := lipgloss.Width(ruleRow); w < innerW {
		ruleRow += rule(innerW - w)
	}
	return fitLine(labels, innerW) + "\n" + fitLine(ruleRow, innerW)
}

// renderFooterRule is the rule above the footer.
func (m Model) renderFooterRule() string {
	return rule(m.termWidth - frameStyle.GetHorizontalFrameSize())
}

// renderFooter renders the global key hints: orange key, grey label,
// separated by two background-filled spaces. Items are dropped from the
// middle when the terminal is too narrow, keeping Send and Quit.
func (m Model) renderFooter() string {
	innerW := m.termWidth - frameStyle.GetHorizontalFrameSize()
	var parts []string
	width := 0
	for _, kb := range m.keys.footerBindings() {
		h := kb.Help()
		p := footerKeyStyle.Render(shortKey(h.Key)) + fill(1) + footerDescStyle.Render(shortDesc(h.Desc))
		parts = append(parts, p)
		width += lipgloss.Width(p)
	}
	for w := width + 2*max(len(parts)-1, 0); w > innerW && len(parts) > 2; {
		mid := len(parts) / 2
		width -= lipgloss.Width(parts[mid])
		parts = append(parts[:mid], parts[mid+1:]...)
		w = width + 2*max(len(parts)-1, 0)
	}
	return fitLine(strings.Join(parts, fill(2)), innerW)
}

// rows3 splits a rendered block into exactly three rows of w cells.
func rows3(s string, w int) [3]string {
	lines := strings.Split(s, "\n")
	var out [3]string
	for i := range out {
		if i < len(lines) {
			out[i] = fitLine(lines[i], w)
		} else {
			out[i] = fill(w)
		}
	}
	return out
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
