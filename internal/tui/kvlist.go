package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dheeraj080/PostBoy/internal/config"
)

// kvList is an editable list of toggleable key/value pairs. It backs both the
// Headers and Params tabs.
type kvList struct {
	items   []config.KeyValue
	cursor  int
	editing bool
	isNew   bool
	input   textinput.Model
	sep     string // separator between key and value when editing, e.g. ":" or "="
	noun    string // "header", "parameter"
	keys    listKeyMap

	// noToggle hides the ENABLED column and disables toggling (env vars).
	noToggle bool
	// showFile adds a TYPE column and 'v' toggles KeyValue.File (multipart).
	showFile bool
	// validate, if set, rejects a key for row idx; the edit stays open and
	// the error is shown.
	validate func(items []config.KeyValue, idx int, key string) error
	err      string
}

func newKVList(sep, noun string) kvList {
	in := textinput.New()
	in.CharLimit = 4096
	return kvList{sep: sep, noun: noun, input: in, keys: defaultListKeyMap()}
}

// setItems replaces the list contents (copying the slice).
func (l *kvList) setItems(items []config.KeyValue) {
	l.items = append([]config.KeyValue(nil), items...)
	l.cancelEdit()
	if l.cursor >= len(l.items) {
		l.cursor = max(len(l.items)-1, 0)
	}
}

// snapshot returns a copy of the items safe to hand to other goroutines.
func (l *kvList) snapshot() []config.KeyValue {
	return append([]config.KeyValue(nil), l.items...)
}

func (l *kvList) remove(idx int) {
	if idx < 0 || idx >= len(l.items) {
		return
	}
	l.items = append(l.items[:idx], l.items[idx+1:]...)
	if l.cursor >= len(l.items) && l.cursor > 0 {
		l.cursor--
	}
}

// cancelEdit aborts an in-progress edit, discarding an unsaved new row.
func (l *kvList) cancelEdit() {
	if !l.editing {
		return
	}
	if l.isNew {
		l.remove(l.cursor)
	}
	l.editing = false
	l.isNew = false
	l.err = ""
	l.input.Blur()
}

func (l *kvList) startEdit(isNew bool) tea.Cmd {
	l.editing = true
	l.isNew = isNew
	if isNew {
		l.input.SetValue("")
	} else {
		it := l.items[l.cursor]
		sep := l.sep
		if sep == ":" {
			sep = ": "
		}
		l.input.SetValue(it.Key + sep + it.Value)
	}
	l.input.CursorEnd()
	return l.input.Focus()
}

// commitEdit applies the edit. It returns false (keeping the edit open) when
// validation fails.
func (l *kvList) commitEdit() bool {
	parts := strings.SplitN(l.input.Value(), l.sep, 2)
	k := strings.TrimSpace(parts[0])
	var v string
	if len(parts) > 1 {
		v = strings.TrimSpace(parts[1])
	}
	if k != "" && l.validate != nil {
		if err := l.validate(l.items, l.cursor, k); err != nil {
			l.err = err.Error()
			return false
		}
	}
	l.err = ""
	switch {
	case k != "" && l.cursor < len(l.items):
		l.items[l.cursor].Key = k
		l.items[l.cursor].Value = v
	case k == "":
		// Empty key: drop a new row, keep an existing one unchanged.
		if l.isNew {
			l.remove(l.cursor)
		}
	}
	l.editing = false
	l.isNew = false
	l.input.Blur()
	return true
}

// Update handles a key press. changed reports whether items were modified
// and should be persisted.
func (l *kvList) Update(msg tea.KeyMsg) (cmd tea.Cmd, changed bool) {
	if l.editing {
		switch msg.String() {
		case "enter":
			return nil, l.commitEdit()
		case "esc":
			l.cancelEdit()
			return nil, false
		}
		l.input, cmd = l.input.Update(msg)
		return cmd, false
	}

	switch {
	case key.Matches(msg, l.keys.Up):
		if l.cursor > 0 {
			l.cursor--
		}
	case key.Matches(msg, l.keys.Down):
		if l.cursor < len(l.items)-1 {
			l.cursor++
		}
	case key.Matches(msg, l.keys.Edit):
		if len(l.items) > 0 {
			return l.startEdit(false), false
		}
	case key.Matches(msg, l.keys.New):
		l.items = append(l.items, config.KeyValue{Enabled: true})
		l.cursor = len(l.items) - 1
		return l.startEdit(true), false
	case key.Matches(msg, l.keys.Delete):
		if len(l.items) > 0 {
			l.remove(l.cursor)
			return nil, true
		}
	case key.Matches(msg, l.keys.Toggle):
		if len(l.items) > 0 && !l.noToggle {
			l.items[l.cursor].Enabled = !l.items[l.cursor].Enabled
			return nil, true
		}
	case l.showFile && msg.String() == "v":
		if len(l.items) > 0 {
			l.items[l.cursor].File = !l.items[l.cursor].File
			return nil, true
		}
	}
	return nil, false
}

// View renders the list at the given width.
func (l *kvList) View(focused bool, width int) string {
	var b strings.Builder

	enabledCol := 9
	if l.noToggle {
		enabledCol = 0
	}
	fileCol := 0
	if l.showFile {
		fileCol = 7
	}
	keyCol := max((width-2-enabledCol-fileCol)*2/5, 8)
	valCol := max(width-2-enabledCol-fileCol-keyCol, 8)
	l.input.Width = max(width-4, 10)

	if len(l.items) == 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("No %ss configured", l.noun)))
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("Press 'n' to add a %s", l.noun)))
	} else {
		head := lipgloss.NewStyle().Bold(true).Foreground(textSecondary)
		b.WriteString("  ")
		b.WriteString(head.Width(keyCol).Render("KEY"))
		b.WriteString(head.Width(valCol).Render("VALUE"))
		if l.showFile {
			b.WriteString(head.Width(fileCol).Render("TYPE"))
		}
		if !l.noToggle {
			b.WriteString(head.Render("ENABLED"))
		}
		b.WriteString("\n")

		for i, it := range l.items {
			selected := focused && i == l.cursor
			prefix := "  "
			if selected {
				prefix = cursorStyle.Render("▶ ")
			}
			keyStyle := lipgloss.NewStyle().Foreground(keyColor).Width(keyCol)
			valStyle := lipgloss.NewStyle().Foreground(textPrimary).Width(valCol)
			enStyle := statusGreenStyle
			if !it.Enabled {
				keyStyle = keyStyle.Foreground(textSecondary)
				valStyle = valStyle.Foreground(textSecondary)
				enStyle = dimStyle
			}
			cols := []string{prefix,
				keyStyle.Render(truncate(it.Key, keyCol-1)),
				valStyle.Render(truncate(it.Value, valCol-1))}
			if l.showFile {
				kind := "text"
				if it.File {
					kind = "file"
				}
				cols = append(cols, dimStyle.Width(fileCol).Render(kind))
			}
			if !l.noToggle {
				cols = append(cols, enStyle.Render(fmt.Sprintf("%v", it.Enabled)))
			}
			line := lipgloss.JoinHorizontal(lipgloss.Left, cols...)
			if selected && !l.editing {
				line = selectedRowStyle.Render(line)
			}
			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	if l.editing {
		b.WriteString("\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("Edit %s (format: key%s value)", l.noun, l.sep)))
		b.WriteString("\n")
		b.WriteString(l.input.View())
		b.WriteString("\n")
		if l.err != "" {
			b.WriteString(statusRedStyle.Render("✗ " + l.err))
			b.WriteString("\n")
		}
		b.WriteString(helpStyle.Render("Enter to save • Esc to cancel"))
	} else if focused {
		b.WriteString("\n")
		if l.noToggle {
			b.WriteString(helpStyle.Render("[n] New  [Enter] Edit  [d] Delete"))
		} else if l.showFile {
			b.WriteString(helpStyle.Render("[n] New  [Enter] Edit  [v] Text/File  [t] Toggle  [d] Delete"))
		} else {
			b.WriteString(helpStyle.Render("[n] New  [Enter] Edit  [t] Toggle  [d] Delete"))
		}
	}
	return b.String()
}

// truncate shortens s to at most n display runes, adding an ellipsis.
func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n == 1 {
		return "…"
	}
	return string(r[:n-1]) + "…"
}

// mergeKV overlays incoming entries onto existing ones (case-insensitive
// keys) without removing any existing entries.
func mergeKV(existing, incoming []config.KeyValue) []config.KeyValue {
	out := append([]config.KeyValue(nil), existing...)
	for _, in := range incoming {
		found := false
		for i := range out {
			if strings.EqualFold(out[i].Key, in.Key) {
				out[i].Value = in.Value
				out[i].Enabled = true
				found = true
				break
			}
		}
		if !found {
			out = append(out, in)
		}
	}
	return out
}
