package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap is the single source of truth for global keybindings; the help
// modal is generated from it.
type keyMap struct {
	Send          key.Binding
	Cancel        key.Binding
	Save          key.Binding
	SaveAs        key.Binding
	NewRequest    key.Binding
	Collections   key.Binding
	Import        key.Binding
	CopyCurl      key.Binding
	NextFocus     key.Binding
	PrevFocus     key.Binding
	TabLeft       key.Binding
	TabRight      key.Binding
	CycleMethod   key.Binding
	CycleBodyMode key.Binding
	Headers       key.Binding
	History       key.Binding
	CycleEnv      key.Binding
	EnvVars       key.Binding
	Secrets       key.Binding
	Help          key.Binding
	Close         key.Binding
	Quit          key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Send:          key.NewBinding(key.WithKeys("alt+r"), key.WithHelp("Alt+R", "Send")),
		Cancel:        key.NewBinding(key.WithKeys("alt+x"), key.WithHelp("Alt+X", "Cancel request")),
		Save:          key.NewBinding(key.WithKeys("ctrl+s"), key.WithHelp("Ctrl+S", "Save")),
		SaveAs:        key.NewBinding(key.WithKeys("alt+s"), key.WithHelp("Alt+S", "Save request as (copy)")),
		NewRequest:    key.NewBinding(key.WithKeys("alt+n"), key.WithHelp("Alt+N", "New")),
		Collections:   key.NewBinding(key.WithKeys("alt+o"), key.WithHelp("Alt+O", "Collections")),
		Import:        key.NewBinding(key.WithKeys("alt+i"), key.WithHelp("Alt+I", "Import")),
		CopyCurl:      key.NewBinding(key.WithKeys("alt+c"), key.WithHelp("Alt+C", "Copy curl")),
		NextFocus:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("Tab", "Next panel")),
		PrevFocus:     key.NewBinding(key.WithKeys("shift+tab"), key.WithHelp("Shift+Tab", "Previous panel")),
		TabLeft:       key.NewBinding(key.WithKeys("left", "h"), key.WithHelp("←/h", "Previous tab (tab bar focused)")),
		TabRight:      key.NewBinding(key.WithKeys("right", "l"), key.WithHelp("→/l", "Next tab (tab bar focused)")),
		CycleMethod:   key.NewBinding(key.WithKeys("alt+m"), key.WithHelp("Alt+M", "Cycle HTTP method")),
		CycleBodyMode: key.NewBinding(key.WithKeys("alt+t"), key.WithHelp("Alt+T", "Cycle request body type")),
		Headers:       key.NewBinding(key.WithKeys("alt+l"), key.WithHelp("Alt+L", "Jump to headers")),
		History:       key.NewBinding(key.WithKeys("alt+h"), key.WithHelp("Alt+H", "History")),
		CycleEnv:      key.NewBinding(key.WithKeys("alt+e"), key.WithHelp("Alt+E", "Cycle environment")),
		EnvVars:       key.NewBinding(key.WithKeys("alt+v"), key.WithHelp("Alt+V", "Env")),
		Secrets:       key.NewBinding(key.WithKeys("alt+k"), key.WithHelp("Alt+K", "Secrets manager")),
		Help:          key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "Help (when not typing)")),
		Close:         key.NewBinding(key.WithKeys("esc"), key.WithHelp("Esc", "Close dialog / cancel edit")),
		Quit:          key.NewBinding(key.WithKeys("ctrl+c"), key.WithHelp("Ctrl+C", "Quit")),
	}
}

// helpBindings lists bindings in the order shown in the help modal.
func (k keyMap) helpBindings() []key.Binding {
	return []key.Binding{
		k.Send, k.Cancel, k.Save, k.SaveAs, k.NewRequest, k.Collections, k.Import, k.CopyCurl,
		k.NextFocus, k.PrevFocus, k.TabLeft, k.TabRight,
		k.CycleMethod, k.CycleBodyMode, k.Headers, k.History, k.CycleEnv, k.EnvVars, k.Secrets,
		k.Help, k.Close, k.Quit,
	}
}

// footerBindings lists the bindings shown in the footer.
func (k keyMap) footerBindings() []key.Binding {
	return []key.Binding{k.Send, k.Save, k.Collections, k.NewRequest, k.Import, k.CopyCurl, k.History, k.EnvVars, k.Help, k.Quit}
}

// listKeyMap holds keybindings for key/value lists (headers, params).
type listKeyMap struct {
	Up, Down, New, Edit, Toggle, Delete key.Binding
}

func defaultListKeyMap() listKeyMap {
	return listKeyMap{
		Up:     key.NewBinding(key.WithKeys("up", "k"), key.WithHelp("↑/k", "Up")),
		Down:   key.NewBinding(key.WithKeys("down", "j"), key.WithHelp("↓/j", "Down")),
		New:    key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "New")),
		Edit:   key.NewBinding(key.WithKeys("enter"), key.WithHelp("Enter", "Edit")),
		Toggle: key.NewBinding(key.WithKeys("t", " "), key.WithHelp("t", "Toggle")),
		Delete: key.NewBinding(key.WithKeys("d", "delete"), key.WithHelp("d", "Delete")),
	}
}
