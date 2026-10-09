package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette from the design spec.
//
// Every style carries Background(darkBg) on purpose: inner styles end with a
// reset sequence, so an inherited background would be lost the moment a styled
// run is rendered inside another one. Painting the background on each style
// (and on every padding/gap string produced by fill) guarantees that no cell
// falls back to the terminal default.
var (
	postmanOrange = lipgloss.Color("#ff6b35")
	darkBg        = lipgloss.Color("#1c1c1c")
	panelBg       = lipgloss.Color("#2a2a2a")
	borderColor   = lipgloss.Color("#3a3a3a")
	keyColor      = lipgloss.Color("#9cdcfe")

	textPrimary   = lipgloss.Color("#d4d4d4")
	textSecondary = lipgloss.Color("#8a8a8a")
	textMuted     = lipgloss.Color("#8a8a8a")
	textWhite     = lipgloss.Color("#ffffff")

	statusSuccess = lipgloss.Color("#4ec9a0")
	statusError   = lipgloss.Color("#f48771")
	statusWarn    = lipgloss.Color("#dcdcaa")

	infoBlue  = lipgloss.Color("#4fc1ff")
	badgeBlue = lipgloss.Color("#0a74c9")
)

var (
	// base is the default text style for unlabelled content.
	base = lipgloss.NewStyle().Foreground(textPrimary).Background(darkBg)

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(textWhite).
			Background(darkBg)

	methodStyle = lipgloss.NewStyle().
			Foreground(postmanOrange).
			Bold(true).
			Background(darkBg).
			Padding(0, 1)

	sendButtonStyle = lipgloss.NewStyle().
			Background(postmanOrange).
			Foreground(darkBg).
			Bold(true).
			Padding(0, 2)

	sendButtonLoadingStyle = lipgloss.NewStyle().
				Background(postmanOrange).
				Foreground(darkBg).
				Bold(true).
				Padding(0, 2)

	// Request/response tab bars: inactive labels are grey, the active one
	// sits on a slightly lighter fill with an orange underline drawn on the
	// rule row right below the tab bar.
	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(textSecondary).
				Background(darkBg).
				Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(postmanOrange).
			Background(panelBg).
			Bold(true).
			Padding(0, 2)

	disabledTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#5a5a5a")).
				Background(darkBg).
				Padding(0, 2)

	// Body sub-tabs are plain labels separated by a single space; the active
	// one is orange and underlined (the panel has no spare row for a rule).
	activeSubTabStyle   = lipgloss.NewStyle().Foreground(postmanOrange).Background(darkBg).Underline(true)
	inactiveSubTabStyle = lipgloss.NewStyle().Foreground(textSecondary).Background(darkBg)

	// URL bar.
	brandStyle = lipgloss.NewStyle().Foreground(textSecondary).Background(darkBg)

	methodChipStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(textSecondary).
			BorderBackground(darkBg).
			Foreground(postmanOrange).
			Bold(true).
			Background(darkBg).
			Padding(0, 1)

	urlBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(postmanOrange).
			BorderBackground(darkBg).
			Foreground(textWhite).
			Background(darkBg).
			Padding(0, 1)

	urlInputTextStyle   = lipgloss.NewStyle().Foreground(textWhite).Background(darkBg)
	urlSchemeStyle      = lipgloss.NewStyle().Foreground(postmanOrange).Background(darkBg)
	urlTextStyle        = lipgloss.NewStyle().Foreground(textWhite).Background(darkBg)
	urlCursorStyle      = lipgloss.NewStyle().Foreground(darkBg).Background(postmanOrange)
	urlPlaceholderStyle = lipgloss.NewStyle().Foreground(textSecondary).Background(darkBg)

	// Outer frame.
	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			BorderBackground(darkBg).
			Foreground(textPrimary).
			Background(darkBg).
			Padding(0, 1)

	panelStyle        = lipgloss.NewStyle().Foreground(textPrimary).Background(darkBg).Padding(0, 1)
	focusedPanelStyle = panelStyle

	rightPaneStyle = lipgloss.NewStyle().
			Foreground(textPrimary).
			Background(darkBg).
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeft(true).
			BorderForeground(borderColor).
			BorderBackground(darkBg)

	focusedRightPaneStyle = rightPaneStyle.BorderForeground(postmanOrange)

	footerKeyStyle  = lipgloss.NewStyle().Foreground(postmanOrange).Bold(true).Background(darkBg)
	footerDescStyle = lipgloss.NewStyle().Foreground(textMuted).Background(darkBg)

	ruleStyle       = lipgloss.NewStyle().Foreground(borderColor).Background(darkBg)
	activeRuleStyle = lipgloss.NewStyle().Foreground(postmanOrange).Background(darkBg)

	statusGreenStyle  = lipgloss.NewStyle().Foreground(statusSuccess).Background(darkBg)
	statusGreenBold   = lipgloss.NewStyle().Foreground(statusSuccess).Bold(true).Background(darkBg)
	statusRedStyle    = lipgloss.NewStyle().Foreground(statusError).Background(darkBg)
	statusYellowStyle = lipgloss.NewStyle().Foreground(statusWarn).Background(darkBg)
	statusMetaStyle   = lipgloss.NewStyle().Foreground(textSecondary).Background(darkBg)
	helpStyle         = lipgloss.NewStyle().Foreground(textMuted).Background(darkBg)
	dimStyle          = lipgloss.NewStyle().Foreground(textSecondary).Background(darkBg)
	cursorStyle       = lipgloss.NewStyle().Foreground(postmanOrange).Background(darkBg)
	selectedRowStyle  = lipgloss.NewStyle().Foreground(textPrimary).Background(panelBg)

	envBadgeStyle = lipgloss.NewStyle().
			Background(badgeBlue).
			Foreground(textWhite).
			Bold(true).
			Padding(0, 1)
	envBadgeEdgeStyle = lipgloss.NewStyle().Foreground(badgeBlue).Background(darkBg)

	responseMetaStyle = lipgloss.NewStyle().Foreground(textSecondary).Background(darkBg)

	hintChipStyle = lipgloss.NewStyle().
			Foreground(textSecondary).
			Background(darkBg).
			Padding(0, 1)
	hintChipEdgeStyle = lipgloss.NewStyle().
				Foreground(textSecondary).
				Background(darkBg)
)

// roundedChip renders text on a single row with rounded ends. A real
// RoundedBorder box needs three rows, and both the response key hints and the
// environment badge share a one-row band, so the corners are drawn as side
// glyphs instead.
func roundedChip(text string, edge, inner lipgloss.Style) string {
	return edge.Render("╭") + inner.Render(text) + edge.Render("╮")
}

// methodColor returns a method-specific colour for method labels.
func methodColor(method string) lipgloss.Color {
	switch strings.ToUpper(method) {
	case "POST":
		return postmanOrange
	case "GET":
		return lipgloss.Color("#00b894")
	case "PUT":
		return lipgloss.Color("#60a5fa")
	case "DELETE":
		return statusError
	case "PATCH":
		return lipgloss.Color("#a855f7")
	case "HEAD", "OPTIONS":
		return textSecondary
	default:
		return textPrimary
	}
}

// statusStyle picks a colour for an HTTP status code.
func statusStyle(code int) lipgloss.Style {
	switch {
	case code >= 400 || code == 0:
		return statusRedStyle
	case code >= 300:
		return statusYellowStyle
	default:
		return statusGreenStyle
	}
}
