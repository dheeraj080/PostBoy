package tui

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/interp"
)

type envInputMode int

const (
	envInputNone envInputMode = iota
	envInputNew
	envInputRename
)

func newEnvVarList() kvList {
	l := newKVList("=", "variable")
	l.noToggle = true
	l.validate = func(items []config.KeyValue, idx int, key string) error {
		if !interp.ValidVarName(key) {
			return fmt.Errorf("invalid name %q (letters, digits, _; not starting with a digit)", key)
		}
		for i, it := range items {
			if i != idx && it.Key == key {
				return fmt.Errorf("variable %q already exists", key)
			}
		}
		return nil
	}
	return l
}

func newEnvInput() textinput.Model {
	in := textinput.New()
	in.CharLimit = 64
	return in
}

func (m *Model) openEnvEditor() {
	m.modal = modalEnv
	m.envConfirmDelete = false
	m.envInputMode = envInputNone
	m.viewEnv(m.envIndex)
}

// viewEnv shows environment i in the editor.
func (m *Model) viewEnv(i int) {
	n := len(m.cfg.Environments)
	m.envView = ((i % n) + n) % n
	vars := m.cfg.Environments[m.envView].Vars
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	items := make([]config.KeyValue, 0, len(keys))
	for _, k := range keys {
		items = append(items, config.KeyValue{Key: k, Value: vars[k], Enabled: true})
	}
	m.envVars.setItems(items)
	m.envVars.cursor = 0
}

// storeEnvVars writes the edited list back to the viewed environment.
func (m *Model) storeEnvVars() {
	vars := make(map[string]string, len(m.envVars.items))
	for _, it := range m.envVars.items {
		if it.Key != "" {
			vars[it.Key] = it.Value
		}
	}
	m.cfg.Environments[m.envView].Vars = vars
	m.persistOrWarn()
}

func (m *Model) stopEnvInput() {
	m.envInputMode = envInputNone
	m.envInput.Blur()
	m.envInput.SetValue("")
}

func (m *Model) envNameTaken(name string, except int) bool {
	for i, e := range m.cfg.Environments {
		if i != except && strings.EqualFold(e.Name, name) {
			return true
		}
	}
	return false
}

func (m Model) updateEnvEditor(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.envInputMode != envInputNone {
		switch msg.String() {
		case "esc":
			m.stopEnvInput()
		case "enter":
			name := strings.TrimSpace(m.envInput.Value())
			except := -1
			if m.envInputMode == envInputRename {
				except = m.envView
			}
			switch {
			case name == "":
				m.setStatus("Environment name cannot be empty", true)
				return m, nil
			case m.envNameTaken(name, except):
				m.setStatus(fmt.Sprintf("Environment %q already exists", name), true)
				return m, nil
			}
			if m.envInputMode == envInputNew {
				m.cfg.Environments = append(m.cfg.Environments, config.Environment{Name: name, Vars: map[string]string{}})
				m.viewEnv(len(m.cfg.Environments) - 1)
				m.setStatus(fmt.Sprintf("✓ Created environment %q", name), false)
			} else {
				m.cfg.Environments[m.envView].Name = name
				m.setStatus(fmt.Sprintf("✓ Renamed to %q", name), false)
			}
			m.stopEnvInput()
			m.persistOrWarn()
		default:
			var cmd tea.Cmd
			m.envInput, cmd = m.envInput.Update(msg)
			return m, cmd
		}
		return m, nil
	}

	if m.envVars.editing {
		cmd, changed := m.envVars.Update(msg)
		if changed {
			m.storeEnvVars()
		}
		return m, cmd
	}

	confirmDelete := m.envConfirmDelete
	m.envConfirmDelete = false

	switch msg.String() {
	case "esc", "alt+v":
		m.closeModal()
	case "left", "h", "shift+tab":
		m.viewEnv(m.envView - 1)
	case "right", "l", "tab":
		m.viewEnv(m.envView + 1)
	case "a":
		m.envIndex = m.envView
		m.persistOrWarn()
		m.setStatus("Switched to "+m.env().Name, false)
	case "alt+e":
		m.envIndex = (m.envIndex + 1) % len(m.cfg.Environments)
		m.persistOrWarn()
		m.setStatus("Switched to "+m.env().Name, false)
	case "N":
		m.envInputMode = envInputNew
		m.envInput.SetValue("")
		m.envInput.Placeholder = "Environment name"
		return m, m.envInput.Focus()
	case "R":
		m.envInputMode = envInputRename
		m.envInput.SetValue(m.cfg.Environments[m.envView].Name)
		m.envInput.CursorEnd()
		return m, m.envInput.Focus()
	case "D":
		if len(m.cfg.Environments) <= 1 {
			m.setStatus("Cannot delete the only environment", true)
			return m, nil
		}
		if !confirmDelete {
			m.envConfirmDelete = true
			return m, nil
		}
		name := m.cfg.Environments[m.envView].Name
		m.cfg.Environments = append(m.cfg.Environments[:m.envView], m.cfg.Environments[m.envView+1:]...)
		switch {
		case m.envIndex == m.envView:
			m.envIndex = 0
		case m.envIndex > m.envView:
			m.envIndex--
		}
		m.viewEnv(min(m.envView, len(m.cfg.Environments)-1))
		m.persistOrWarn()
		m.setStatus(fmt.Sprintf("✓ Deleted environment %q", name), false)
	default:
		cmd, changed := m.envVars.Update(msg)
		if changed {
			m.storeEnvVars()
		}
		return m, cmd
	}
	return m, nil
}

func (m Model) renderEnvModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Environments"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("←/→: Switch  a: Use this env  N: New env  R: Rename  D: Delete  Esc: Close"))
	b.WriteString("\n\n")

	// Environment tabs.
	var tabs []string
	for i, e := range m.cfg.Environments {
		name := e.Name
		if i == m.envIndex {
			name = "● " + name
		}
		if i == m.envView {
			tabs = append(tabs, activeTabStyle.Render(name))
		} else {
			tabs = append(tabs, inactiveTabStyle.Render(name))
		}
	}
	b.WriteString(lipgloss.JoinHorizontal(lipgloss.Bottom, tabs...))
	b.WriteString("\n\n")

	width := max(m.termWidth-20, 30)
	b.WriteString(m.envVars.View(m.envInputMode == envInputNone, width))
	b.WriteString("\n")

	switch {
	case m.envInputMode != envInputNone:
		label := "New environment name:"
		if m.envInputMode == envInputRename {
			label = "Rename environment to:"
		}
		b.WriteString("\n" + helpStyle.Render(label) + "\n" + m.envInput.View() + "\n")
		b.WriteString(helpStyle.Render("Enter to save • Esc to cancel"))
	case m.envConfirmDelete:
		b.WriteString("\n" + statusRedStyle.Render(fmt.Sprintf("Press D again to delete %q.", m.cfg.Environments[m.envView].Name)))
	default:
		b.WriteString("\n" + dimStyle.Render("Use variables as {{NAME}}; values may reference other variables or {{ secret.NAME }}."))
		b.WriteString("\n" + dimStyle.Render("Built-ins: "))
		var names []string
		for _, d := range interp.DynamicVars {
			names = append(names, "{{"+d.Name+"}}")
		}
		b.WriteString(dimStyle.Render(strings.Join(names, " ")))
	}
	return m.placeModal("39", b.String())
}
