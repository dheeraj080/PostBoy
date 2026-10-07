package tui

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// editorFinishedMsg is emitted after the external editor exits.
type editorFinishedMsg struct {
	body string
	err  error
}

// openBodyEditorCmd suspends the TUI and opens the raw request body in the
// user's $VISUAL/$EDITOR. If neither is set, Notepad (Windows) or vi is used.
var openBodyEditorCmd = func(text string) tea.Cmd {
	return func() tea.Msg {
		f, err := os.CreateTemp("", "postboy-body-*.txt")
		if err != nil {
			return editorFinishedMsg{err: err}
		}
		path := f.Name()
		if _, err := f.WriteString(text); err != nil {
			f.Close()
			os.Remove(path)
			return editorFinishedMsg{err: err}
		}
		f.Close()

		editor := strings.TrimSpace(os.Getenv("VISUAL"))
		if editor == "" {
			editor = strings.TrimSpace(os.Getenv("EDITOR"))
		}
		if editor == "" {
			if runtime.GOOS == "windows" {
				editor = "notepad"
			} else {
				editor = "vi"
			}
		}
		parts := strings.Fields(editor)
		if len(parts) == 0 {
			os.Remove(path)
			return editorFinishedMsg{err: os.ErrInvalid}
		}
		cmd := exec.Command(parts[0], append(parts[1:], path)...)
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			data, rerr := os.ReadFile(path)
			os.Remove(path)
			if err != nil {
				return editorFinishedMsg{err: err}
			}
			if rerr != nil {
				return editorFinishedMsg{err: rerr}
			}
			return editorFinishedMsg{body: string(data)}
		})()
	}
}
