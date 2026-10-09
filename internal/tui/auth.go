package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dheeraj080/PostBoy/internal/config"
)

type authField int

const (
	afType authField = iota
	afToken
	afUsername
	afPassword
	afKey
	afValue
	afIn
)

func (f authField) label() string {
	switch f {
	case afType:
		return "Type"
	case afToken:
		return "Token"
	case afUsername:
		return "Username"
	case afPassword:
		return "Password"
	case afKey:
		return "Key"
	case afValue:
		return "Value"
	case afIn:
		return "Add to"
	}
	return ""
}

type authEditor struct {
	auth    config.Auth
	cursor  int
	editing bool
	input   textinput.Model
}

func newAuthEditor() authEditor {
	in := textinput.New()
	in.CharLimit = 4096
	return authEditor{input: in}
}

func (a *authEditor) set(auth config.Auth) {
	a.auth = auth
	a.cursor = 0
	a.cancelEdit()
}

func (a *authEditor) fields() []authField {
	switch a.auth.Type {
	case config.AuthBearer:
		return []authField{afType, afToken}
	case config.AuthBasic:
		return []authField{afType, afUsername, afPassword}
	case config.AuthAPIKey:
		return []authField{afType, afKey, afValue, afIn}
	}
	return []authField{afType}
}

func (a *authEditor) value(f authField) *string {
	switch f {
	case afToken:
		return &a.auth.Token
	case afUsername:
		return &a.auth.Username
	case afPassword:
		return &a.auth.Password
	case afKey:
		return &a.auth.Key
	case afValue:
		return &a.auth.Value
	}
	return nil
}

func (a *authEditor) current() authField {
	fs := a.fields()
	if a.cursor >= len(fs) {
		a.cursor = max(len(fs)-1, 0)
	}
	return fs[a.cursor]
}

func (a *authEditor) cycleType(forward bool) {
	idx := 0
	for i, t := range config.AuthTypes {
		if t == a.auth.Type {
			idx = i
			break
		}
	}
	n := len(config.AuthTypes)
	if forward {
		idx = (idx + 1) % n
	} else {
		idx = (idx - 1 + n) % n
	}
	a.auth.Type = config.AuthTypes[idx]
	if a.cursor >= len(a.fields()) {
		a.cursor = max(len(a.fields())-1, 0)
	}
}

func (a *authEditor) toggleIn() {
	if a.auth.In == config.APIKeyInQuery {
		a.auth.In = config.APIKeyInHeader
	} else {
		a.auth.In = config.APIKeyInQuery
	}
}

func (a *authEditor) cancelEdit() {
	a.editing = false
	a.input.Blur()
	a.input.EchoMode = textinput.EchoNormal
}

func (a *authEditor) Update(msg tea.KeyMsg) (cmd tea.Cmd, changed bool) {
	if a.editing {
		switch msg.String() {
		case "enter":
			if p := a.value(a.current()); p != nil {
				*p = a.input.Value()
			}
			a.cancelEdit()
			return nil, true
		case "esc":
			a.cancelEdit()
			return nil, false
		}
		a.input, cmd = a.input.Update(msg)
		return cmd, false
	}

	f := a.current()
	switch msg.String() {
	case "up", "k":
		if a.cursor > 0 {
			a.cursor--
		}
	case "down", "j":
		if a.cursor < len(a.fields())-1 {
			a.cursor++
		}
	case "left", "h":
		switch f {
		case afType:
			a.cycleType(false)
			return nil, true
		case afIn:
			a.toggleIn()
			return nil, true
		}
	case "right", "l", " ", "enter":
		switch f {
		case afType:
			a.cycleType(true)
			return nil, true
		case afIn:
			a.toggleIn()
			return nil, true
		}
		if msg.String() == "enter" {
			if p := a.value(f); p != nil {
				a.editing = true
				a.input.SetValue(*p)
				a.input.CursorEnd()
				a.input.Placeholder = "value or {{ secret.NAME }}"
				if f == afPassword {
					a.input.EchoMode = textinput.EchoPassword
				}
				return a.input.Focus(), false
			}
		}
	}
	return nil, false
}

// View renders at most 7 rows with no trailing newline.
func (a *authEditor) View(focused bool, width int) string {
	var b strings.Builder
	fields := a.fields()
	if a.cursor >= len(fields) {
		a.cursor = max(len(fields)-1, 0)
	}

	labelStyle := lipgloss.NewStyle().Foreground(textSecondary).Background(darkBg).Width(12)
	valStyle := lipgloss.NewStyle().Foreground(textPrimary).Background(darkBg)

	for i, f := range fields {
		selected := focused && i == a.cursor
		prefix := "  "
		curLabel := labelStyle
		curVal := valStyle
		if selected && !a.editing {
			curLabel = curLabel.Background(borderColor)
			curVal = curVal.Background(borderColor)
			prefix = cursorStyle.Bold(true).Background(borderColor).Render("▶ ")
		}

		var val string
		switch f {
		case afType:
			val = cursorStyle.Render("◀ ") + curVal.Bold(true).Render(a.auth.Type.Label()) + cursorStyle.Render(" ▶")
		case afIn:
			in := "Header"
			if a.auth.In == config.APIKeyInQuery {
				in = "Query Params"
			}
			val = cursorStyle.Render("◀ ") + curVal.Render(in) + cursorStyle.Render(" ▶")
		default:
			if selected && a.editing {
				val = a.input.View()
			} else {
				v := *a.value(f)
				switch {
				case v == "":
					val = dimStyle.Render("(empty)")
				case f == afPassword && !isTemplate(v):
					val = curVal.Render(strings.Repeat("•", min(len([]rune(v)), 12)))
				default:
					val = curVal.Render(truncate(v, max(width-18, 10)))
				}
			}
		}
		b.WriteString(prefix + curLabel.Render(f.label()) + val + "\n")
	}

	b.WriteString("\n")
	switch {
	case a.auth.Type == config.AuthNone:
		b.WriteString(dimStyle.Render(truncate("This request does not use any authorization.", width)) + "\n")
	case a.hasPlaintextCredential():
		b.WriteString(statusYellowStyle.Render(truncate("⚠ Credential is saved in plain text. Prefer {{ secret.NAME }} (Alt+K).", width)) + "\n")
	default:
		b.WriteString(dimStyle.Render(truncate("Auth is applied at send time and never saved to history.", width)) + "\n")
	}

	if focused && !a.editing {
		b.WriteString(helpStyle.Render(truncate("[←/→] Change  [Enter] Edit  [↑/↓] Move", width)))
	} else if a.editing {
		b.WriteString(helpStyle.Render(truncate("Enter to save • Esc to cancel", width)))
	}
	return b.String()
}

func (a *authEditor) hasPlaintextCredential() bool {
	var vals []string
	switch a.auth.Type {
	case config.AuthBearer:
		vals = []string{a.auth.Token}
	case config.AuthBasic:
		vals = []string{a.auth.Password}
	case config.AuthAPIKey:
		vals = []string{a.auth.Value}
	}
	for _, v := range vals {
		if v != "" && !isTemplate(v) {
			return true
		}
	}
	return false
}

func isTemplate(s string) bool { return strings.Contains(s, "{{") }
