package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/dheeraj080/PostBoy/internal/config"
)

type kvList struct {
	items    []config.KeyValue
	cursor   int
	editing  bool
	isNew    bool
	input    textinput.Model
	sep      string
	noun     string
	keys     listKeyMap
	noToggle bool
	showFile bool
	validate func(items []config.KeyValue, idx int, key string) error
	err      string
}

func newKVList(sep, noun string) kvList {
	in := textinput.New()
	in.CharLimit = 4096
	return kvList{sep: sep, noun: noun, input: in, keys: defaultListKeyMap()}
}

func (l *kvList) setItems(items []config.KeyValue) {
	l.items = append([]config.KeyValue(nil), items...)
	l.cancelEdit()
	if l.cursor >= len(l.items) {
		l.cursor = max(len(l.items)-1, 0)
	}
}

func (l *kvList) setWidth(inner int) {
	l.input.Width = max(inner-4, 10)
}

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
		if l.isNew {
			l.remove(l.cursor)
		}
	}
	l.editing = false
	l.isNew = false
	l.input.Blur()
	return true
}

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

// View renders at most height rows with no trailing newline.
func (l *kvList) View(focused bool, width, height int) string {
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

	avail := height - 1 // header row
	if avail < 0 {
		avail = 0
	}

	if len(l.items) == 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("No %ss configured", l.noun)) + "\n")
		b.WriteString(helpStyle.Render(fmt.Sprintf("Press 'n' to add a %s", l.noun)))
		return b.String()
	}

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

	// Window item rows around the cursor.
	start := 0
	if l.cursor >= avail {
		start = l.cursor - avail + 1
	}
	end := min(start+avail, len(l.items))
	for i := start; i < end; i++ {
		b.WriteString(l.renderRow(i, focused, keyCol, valCol, fileCol))
		b.WriteString("\n")
	}

	if l.editing {
		prompt := truncate(fmt.Sprintf("Edit %s (format: key%s value)", l.noun, l.sep), width)
		b.WriteString(helpStyle.Render(prompt) + "\n")
		b.WriteString(l.input.View() + "\n")
		if l.err != "" {
			b.WriteString(statusRedStyle.Render("✗ "+truncate(l.err, width)) + "\n")
		}
		b.WriteString(helpStyle.Render("Enter to save • Esc to cancel"))
	} else if focused {
		var hint string
		switch {
		case l.noToggle:
			hint = "[n] New  [Enter] Edit  [d] Delete"
		case l.showFile:
			hint = "[n] New  [Enter] Edit  [v] Text/File  [t] Toggle  [d] Delete"
		default:
			hint = "[n] New  [Enter] Edit  [t] Toggle  [d] Delete"
		}
		b.WriteString(helpStyle.Render(truncate(hint, width)))
	}
	return b.String()
}

func (l *kvList) renderRow(i int, focused bool, keyCol, valCol, fileCol int) string {
	it := l.items[i]
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

	cols := []string{
		prefix,
		keyStyle.Render(truncate(it.Key, keyCol-1)),
		valStyle.Render(truncate(it.Value, valCol-1)),
	}
	if l.showFile {
		kind := "text"
		if it.File {
			kind = "file"
		}
		fStyle := dimStyle.Width(fileCol)
		if selected {
			fStyle = fStyle.Background(borderColor)
		}
		cols = append(cols, fStyle.Render(kind))
	}
	if !l.noToggle {
		cols = append(cols, enStyle.Render(fmt.Sprintf("%v", it.Enabled)))
	}

	line := lipgloss.JoinHorizontal(lipgloss.Left, cols...)
	if selected && !l.editing {
		line = lipgloss.NewStyle().Background(borderColor).Render(line)
	}
	return line
}

func truncate(s string, n int) string {
	if n <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= n {
		return s
	}
	return ansi.Truncate(s, n, "…")
}

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
