package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	postmanOrange = lipgloss.Color("#FF6C37")
	darkBg        = lipgloss.Color("#1E1E1E")
	panelBg       = lipgloss.Color("#252526")
	borderColor   = lipgloss.Color("#3E3E42")
	keyColor      = lipgloss.Color("#9CDCFE")

	textPrimary   = lipgloss.Color("#CCCCCC")
	textSecondary = lipgloss.Color("#858585")
	textMuted     = lipgloss.Color("#6E6E6E")

	statusSuccess = lipgloss.Color("#4EC9B0")
	statusError   = lipgloss.Color("#F48771")
	statusWarn    = lipgloss.Color("#DCDCAA")

	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#569CD6"))

	methodStyle = lipgloss.NewStyle().
			Foreground(postmanOrange).
			Bold(true).
			Padding(0, 1).
			MarginRight(1)

	sendButtonStyle = lipgloss.NewStyle().
			Background(postmanOrange).
			Foreground(lipgloss.Color("#FFFFFF")).
			Bold(true).
			Padding(0, 2)

	sendButtonLoadingStyle = lipgloss.NewStyle().
				Background(lipgloss.Color("#F48771")).
				Foreground(lipgloss.Color("#FFFFFF")).
				Bold(true).
				Padding(1, 2).
				MarginLeft(1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(textSecondary).
				Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(postmanOrange).
			Bold(true).
			Padding(0, 1)

	urlInputStyle = lipgloss.NewStyle().
			Padding(0, 1)

	focusedURLInputStyle = urlInputStyle.Foreground(postmanOrange)

	urlInputTextStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#FFFFFF"))

	urlBoxStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	focusedURLBoxStyle = urlBoxStyle.BorderForeground(postmanOrange)

	mainHeaderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	// No backgrounds on panels, frame, or tab bar — inner styled segments
	// emit \x1b[0m which resets to terminal default. Solid fills only on
	// Send button, env badge, and selected-row highlight.
	panelStyle = lipgloss.NewStyle().
			Padding(0, 1)

	focusedPanelStyle = lipgloss.NewStyle().
			Padding(0, 1)

	rightPaneStyle = lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.NormalBorder()).
			BorderLeft(true).
			BorderForeground(borderColor)

	focusedRightPaneStyle = rightPaneStyle.BorderForeground(postmanOrange)

	frameStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(borderColor).
			Padding(0, 1)

	tabBarStyle = lipgloss.NewStyle().
			Padding(0, 1)

	footerKeyStyle = lipgloss.NewStyle().
			Foreground(postmanOrange).
			Bold(true)

	footerDescStyle = lipgloss.NewStyle().
			Foreground(textMuted)

	statusGreenStyle  = lipgloss.NewStyle().Foreground(statusSuccess)
	statusRedStyle    = lipgloss.NewStyle().Foreground(statusError)
	statusYellowStyle = lipgloss.NewStyle().Foreground(statusWarn)
	helpStyle         = lipgloss.NewStyle().Foreground(textMuted)
	dimStyle          = lipgloss.NewStyle().Foreground(textSecondary)
	cursorStyle       = lipgloss.NewStyle().Foreground(postmanOrange)
	selectedRowStyle  = lipgloss.NewStyle().Background(borderColor)

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

	actionPillStyle = lipgloss.NewStyle().
			Foreground(postmanOrange).
			Bold(true).
			Padding(0, 1).
			MarginLeft(1)

	quickActionStyle = lipgloss.NewStyle().
				Foreground(textPrimary).
				Padding(0, 1).
				MarginLeft(1)
)

// methodColor returns a method-specific color for the request-line badge.
func methodColor(method string) lipgloss.Color {
	switch strings.ToUpper(method) {
	case "POST":
		return postmanOrange
	case "GET":
		return lipgloss.Color("#00B894")
	case "PUT":
		return lipgloss.Color("#60A5FA")
	case "DELETE":
		return statusError
	case "PATCH":
		return lipgloss.Color("#A855F7")
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
		return lipgloss.NewStyle().Foreground(statusError)
	case code >= 300:
		return lipgloss.NewStyle().Foreground(statusWarn)
	default:
		return lipgloss.NewStyle().Foreground(statusSuccess)
	}
}
