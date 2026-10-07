package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"

	"github.com/dheeraj080/PostBoy/internal/config"
)

func newFormList() kvList {
	l := newKVList("=", "form field")
	l.showFile = true
	return l
}

func newBodyFileInput() textinput.Model {
	in := textinput.New()
	in.Placeholder = "/path/to/file"
	in.CharLimit = 2048
	return in
}

// cycleBodyMode moves the request body editor through Raw, Form, Multipart
// and File modes.
func (m *Model) cycleBodyMode() {
	if !m.supportsBody() {
		m.setStatus("No body for "+m.method+" requests", false)
		return
	}
	order := config.BodyModes
	idx := 0
	for i, b := range order {
		if b == m.bodyMode {
			idx = i
			break
		}
	}
	m.bodyMode = order[(idx+1)%len(order)]
	m.form.cancelEdit()
	m.persistOrWarn()
	m.setStatus("Body type: "+m.bodyMode.Label(), false)
}

// renderBodyTab renders the body editor for the current body mode.
func (m Model) renderBodyTab(width, height int) string {
	if !m.supportsBody() {
		return dimStyle.Render("No body for " + m.method + " requests")
	}
	var parts []string
	parts = append(parts, m.renderBodyModeBar())
	switch m.bodyMode {
	case config.BodyURLEncoded, config.BodyMultipart:
		parts = append(parts, m.form.View(m.focus == focusReqContent, width))
		if m.bodyMode == config.BodyMultipart {
			parts = append(parts, dimStyle.Render("Press 'v' on a form field to toggle between text and file. File values are paths."))
		} else {
			parts = append(parts, dimStyle.Render("Form fields are sent with Content-Type: application/x-www-form-urlencoded"))
		}
	case config.BodyFile:
		parts = append(parts, helpStyle.Render("File path to send as the request body (Content-Type guessed from extension):"))
		m.bodyFileInput.Width = max(width-4, 10)
		parts = append(parts, m.bodyFileInput.View())
	default:
		parts = append(parts, m.bodyInput.View())
		parts = append(parts, helpStyle.Render("Ctrl+O: edit body in $EDITOR"))
	}
	return strings.Join(parts, "\n")
}

func (m Model) renderBodyModeBar() string {
	var out []string
	for _, b := range config.BodyModes {
		label := b.Label()
		if b == m.bodyMode {
			out = append(out, activeTabStyle.Render(label))
		} else {
			out = append(out, inactiveTabStyle.Render(label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, out...) + "\n" + dimStyle.Render("Alt+T cycles type")
}
