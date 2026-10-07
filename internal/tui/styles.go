package tui

import "github.com/charmbracelet/lipgloss"

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
			Foreground(postmanOrange).
			Bold(true).
			Padding(0, 1).
			MarginLeft(1)

	sendButtonLoadingStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#F48771")).
				Bold(true).
				Padding(0, 1).
				MarginLeft(1)

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(textSecondary).
				Padding(0, 2)

	activeTabStyle = lipgloss.NewStyle().
			Foreground(postmanOrange).
			Bold(true).
			Padding(0, 2)

	urlInputStyle = lipgloss.NewStyle().
			Background(panelBg).
			Padding(0, 1)

	focusedURLInputStyle = urlInputStyle.Foreground(postmanOrange)

	panelStyle = lipgloss.NewStyle().
			Background(panelBg).
			Border(lipgloss.NormalBorder(), false, true, true, true).
			BorderForeground(borderColor)

	focusedPanelStyle = panelStyle.BorderForeground(postmanOrange)

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
)

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
