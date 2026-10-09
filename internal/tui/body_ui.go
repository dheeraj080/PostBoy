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

// renderBodyTab renders the body editor: the sub-tab row, the content
// (JSON is syntax highlighted with two-space indent) and the hint row with
// the external-editor hint and the interpolation note.
func (m Model) renderBodyTab(width, height int) string {
	if !m.supportsBody() {
		return dimStyle.Render("No body for " + m.method + " requests")
	}
	modeBar := m.renderBodyModeBar(width)
	innerH := height - bodyChromeRows
	if innerH < 1 {
		innerH = 1
	}

	var content string
	switch m.bodyMode {
	case config.BodyURLEncoded, config.BodyMultipart:
		content = m.form.View(m.focus == focusReqContent, width, innerH)
		if m.bodyMode == config.BodyMultipart {
			hint := "Press 'v' on a form field to toggle between text and file. File values are paths."
			content += "\n" + dimStyle.Render(truncate(hint, width))
		} else {
			hint := "Form fields are sent with Content-Type: application/x-www-form-urlencoded"
			content += "\n" + dimStyle.Render(truncate(hint, width))
		}
	case config.BodyFile:
		hint := "File path to send as the request body (Content-Type guessed from extension):"
		content = helpStyle.Render(truncate(hint, width)) + "\n" + m.bodyFileInput.View()
	default:
		if m.focus == focusReqContent {
			content = m.bodyInput.View()
		} else {
			trim := strings.TrimSpace(m.bodyInput.Value())
			if strings.HasPrefix(trim, "{") || strings.HasPrefix(trim, "[") {
				content = highlightJSON(prettyJSON(m.bodyInput.Value()))
			} else {
				content = m.bodyInput.View()
			}
		}
	}

	content = fixBox(content, width, innerH)
	ruleRow := rule(width)
	footer := fitRow(helpStyle.Render("Ctrl+D: edit body in $EDITOR"),
		lipgloss.NewStyle().Foreground(infoBlue).Background(darkBg).Render("Interpolated at send time"),
		width)
	return modeBar + "\n" + content + "\n" + ruleRow + "\n" + footer
}

func modeLabel(b config.BodyMode) string {
	switch b {
	case config.BodyRaw:
		return "Raw (JSON)"
	case config.BodyURLEncoded:
		return "Form URL-Encoded"
	case config.BodyMultipart:
		return "Multipart"
	case config.BodyFile:
		return "Binary"
	default:
		return b.Label()
	}
}

func shortModeLabel(b config.BodyMode) string {
	switch b {
	case config.BodyRaw:
		return "JSON"
	case config.BodyURLEncoded:
		return "FORM"
	case config.BodyMultipart:
		return "MULTI"
	case config.BodyFile:
		return "BINARY"
	default:
		return b.Label()
	}
}

// renderBodyModeBar renders the body sub-tab row: full labels, the active
// one orange and underlined, with the "Alt+T cycles type" hint right-aligned
// (dropped when the pane is too narrow for it; short labels are the last
// resort).
func (m Model) renderBodyModeBar(width int) string {
	bar := func(short bool) string {
		var out []string
		for _, b := range config.BodyModes {
			label := modeLabel(b)
			if short {
				label = shortModeLabel(b)
			}
			if b == m.bodyMode {
				out = append(out, activeSubTabStyle.Render(label))
			} else {
				out = append(out, inactiveSubTabStyle.Render(label))
			}
		}
		return strings.Join(out, fill(1))
	}
	hint := dimStyle.Render("Alt+T cycles type")
	full := bar(false)
	switch {
	case lipgloss.Width(full)+1+lipgloss.Width(hint) <= width:
		return fitRow(full, hint, width)
	case lipgloss.Width(full) <= width:
		return fitLine(full, width)
	}
	short := bar(true)
	if lipgloss.Width(short)+1+lipgloss.Width(hint) <= width {
		return fitRow(short, hint, width)
	}
	return fitLine(short, width)
}
