package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
)

type colRow struct {
	ci, ri int
}

type colInputMode int

const (
	colInputNone colInputMode = iota
	colInputNewCollection
	colInputRename
)

type colConfirm int

const (
	confirmNone colConfirm = iota
	confirmOpen
	confirmDelete
)

func newColInput() textinput.Model {
	in := textinput.New()
	in.CharLimit = 128
	return in
}

func (m *Model) colRows() []colRow {
	var rows []colRow
	for i, c := range m.collections {
		rows = append(rows, colRow{ci: i, ri: -1})
		if m.colExpanded[c.ID] {
			for j := range c.Requests {
				rows = append(rows, colRow{ci: i, ri: j})
			}
		}
	}
	return rows
}

func (m *Model) openCollections() {
	m.modal = modalCollections
	m.colConfirm = confirmNone
	m.colInputMode = colInputNone
	if ci, ri := m.originCollection(); ri >= 0 {
		m.colExpanded[m.collections[ci].ID] = true
		for i, r := range m.colRows() {
			if r.ci == ci && r.ri == ri {
				m.colCursor = i
			}
		}
	}
	m.clampColCursor()
}

func (m *Model) clampColCursor() {
	n := len(m.colRows())
	if m.colCursor >= n {
		m.colCursor = n - 1
	}
	if m.colCursor < 0 {
		m.colCursor = 0
	}
}

func (m *Model) startColInput(mode colInputMode, value, placeholder string) tea.Cmd {
	m.colInputMode = mode
	m.colInput.SetValue(value)
	m.colInput.Placeholder = placeholder
	m.colInput.CursorEnd()
	return m.colInput.Focus()
}

func (m *Model) stopColInput() {
	m.colInputMode = colInputNone
	m.colInput.Blur()
	m.colInput.SetValue("")
}

func (m Model) updateCollections(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.colInputMode != colInputNone {
		switch msg.String() {
		case "esc":
			m.stopColInput()
			return m, nil
		case "enter":
			m.commitColInput()
			return m, nil
		}
		var cmd tea.Cmd
		m.colInput, cmd = m.colInput.Update(msg)
		return m, cmd
	}

	rows := m.colRows()
	confirm := m.colConfirm
	m.colConfirm = confirmNone

	switch msg.String() {
	case "esc", "alt+o":
		m.closeModal()
	case "up", "k":
		if m.colCursor > 0 {
			m.colCursor--
		}
	case "down", "j":
		if m.colCursor < len(rows)-1 {
			m.colCursor++
		}
	case "n":
		return m, m.startColInput(colInputNewCollection, "", "Collection name")
	case "e":
		if len(rows) == 0 {
			return m, nil
		}
		path, err := m.exportCollection(rows[m.colCursor].ci)
		if err != nil {
			m.setStatus("Export failed: "+err.Error(), true)
		} else {
			m.setStatus("✓ Exported to "+path, false)
		}
	case "i":
		return m, m.openImport()
	case "enter", " ", "right", "l", "left", "h":
		if len(rows) == 0 {
			return m, nil
		}
		row := rows[m.colCursor]
		if row.ri < 0 {
			id := m.collections[row.ci].ID
			switch msg.String() {
			case "left", "h":
				m.colExpanded[id] = false
			case "right", "l":
				m.colExpanded[id] = true
			default:
				m.colExpanded[id] = !m.colExpanded[id]
			}
			m.clampColCursor()
			return m, nil
		}
		if msg.String() != "enter" {
			return m, nil
		}
		if m.isDirty() && confirm != confirmOpen {
			m.colConfirm = confirmOpen
			return m, nil
		}
		c := m.collections[row.ci]
		r := c.Requests[row.ri]
		m.loadRequest(r, &config.Origin{CollectionID: c.ID, RequestID: r.ID}, r)
		m.persistOrWarn()
		m.closeModal()
		m.setStatus(fmt.Sprintf("Opened '%s'", r.Name), false)
		return m, m.setFocus(focusURL)
	case "r":
		if len(rows) == 0 {
			return m, nil
		}
		row := rows[m.colCursor]
		name := m.collections[row.ci].Name
		if row.ri >= 0 {
			name = m.collections[row.ci].Requests[row.ri].Name
		}
		return m, m.startColInput(colInputRename, name, "New name")
	case "d":
		if len(rows) == 0 {
			return m, nil
		}
		if confirm != confirmDelete {
			m.colConfirm = confirmDelete
			return m, nil
		}
		m.deleteColRow(rows[m.colCursor])
	}
	return m, nil
}

func (m *Model) commitColInput() {
	name := strings.TrimSpace(m.colInput.Value())
	mode := m.colInputMode
	m.stopColInput()
	if name == "" {
		m.setStatus("Name cannot be empty", true)
		return
	}
	switch mode {
	case colInputNewCollection:
		c := collection.Collection{ID: collection.NewID(), Name: name}
		m.collections = append(m.collections, c)
		rows := m.colRows()
		m.colCursor = len(rows) - 1
	case colInputRename:
		rows := m.colRows()
		if m.colCursor >= len(rows) {
			return
		}
		row := rows[m.colCursor]
		if row.ri < 0 {
			m.collections[row.ci].Name = name
		} else {
			m.collections[row.ci].Requests[row.ri].Name = name
			if m.origin != nil && m.origin.RequestID == m.collections[row.ci].Requests[row.ri].ID {
				m.requestName = name
				m.savedSnapshot.Name = name
			}
		}
	}
	if err := m.persistCollections(); err != nil {
		m.setStatus("Could not save collections: "+err.Error(), true)
		return
	}
	m.setStatus("✓ Collections updated", false)
}

func (m *Model) deleteColRow(row colRow) {
	c := &m.collections[row.ci]
	var deleted string
	unlink := false
	if row.ri < 0 {
		deleted = c.Name
		unlink = m.origin != nil && m.origin.CollectionID == c.ID
		m.collections = append(m.collections[:row.ci], m.collections[row.ci+1:]...)
	} else {
		r := c.Requests[row.ri]
		deleted = r.Name
		unlink = m.origin != nil && m.origin.RequestID == r.ID
		c.Requests = append(c.Requests[:row.ri], c.Requests[row.ri+1:]...)
	}
	if unlink {
		m.origin = nil
		m.requestName = ""
		m.savedSnapshot = config.NewRequest()
		m.persistOrWarn()
	}
	m.clampColCursor()
	if err := m.persistCollections(); err != nil {
		m.setStatus("Could not save collections: "+err.Error(), true)
		return
	}
	m.setStatus(fmt.Sprintf("✓ Deleted '%s'", deleted), false)
}

func (m Model) renderCollectionsModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Collections") + "\n")
	b.WriteString(helpStyle.Render("j/k: Move  Enter: Open/Expand  n: New  r: Rename  d: Delete  e: Export (Postman)  i: Import  Esc: Close") + "\n\n")

	rows := m.colRows()
	if len(rows) == 0 {
		b.WriteString(dimStyle.Render("No collections yet. Press 'n' to create one, or Ctrl+S in the editor to save a request.") + "\n")
	}

	visible := max(m.termHeight-18, 3)
	start := 0
	if m.colCursor >= visible {
		start = m.colCursor - visible + 1
	}
	end := min(start+visible, len(rows))
	width := max(m.termWidth-30, 20)

	for i := start; i < end; i++ {
		row := rows[i]
		selected := i == m.colCursor
		prefix := "  "
		if selected {
			prefix = cursorStyle.Render("▶ ")
		}

		var line string
		if row.ri < 0 {
			c := m.collections[row.ci]
			arrow := "▸"
			if m.colExpanded[c.ID] {
				arrow = "▾"
			}
			tStyle := titleStyle
			if selected {
				tStyle = tStyle.Background(panelBg)
			}
			line = fmt.Sprintf("%s%s %s %s", prefix, arrow, tStyle.Render(c.Name),
				dimStyle.Render(fmt.Sprintf("(%d)", len(c.Requests))))
		} else {
			r := m.collections[row.ci].Requests[row.ri]
			current := ""
			if m.origin != nil && m.origin.RequestID == r.ID {
				current = cursorStyle.Render(" ●")
			}
			mStyle := lipgloss.NewStyle().Foreground(postmanOrange).Bold(true).Width(8)
			if selected {
				mStyle = mStyle.Background(panelBg)
			}
			line = fmt.Sprintf("%s    %s %s%s", prefix,
				mStyle.Render(r.Method),
				truncate(r.Name, width), current)
		}

		if selected {
			line = lipgloss.NewStyle().Background(panelBg).Render(line)
		}
		b.WriteString(line + "\n")
	}

	switch {
	case m.colInputMode != colInputNone:
		b.WriteString("\n")
		label := "New collection name:"
		if m.colInputMode == colInputRename {
			label = "Rename to:"
		}
		b.WriteString(helpStyle.Render(label) + "\n")
		b.WriteString(m.colInput.View() + "\n")
		b.WriteString(helpStyle.Render("Enter to save • Esc to cancel"))
	case m.colConfirm == confirmOpen:
		b.WriteString("\n" + statusYellowStyle.Render("Unsaved changes in the editor. Press Enter again to discard them and open."))
	case m.colConfirm == confirmDelete:
		b.WriteString("\n" + statusRedStyle.Render("Press d again to delete permanently."))
	}
	return m.placeModal("39", b.String())
}

type saveStage int

const (
	saveEnterName saveStage = iota
	savePickCollection
	saveNewCollection
)

func (m *Model) openSaveDialog() tea.Cmd {
	m.modal = modalSave
	m.saveStage = saveEnterName
	m.saveName = ""
	m.saveCursor = 0
	if ci, _ := m.originCollection(); ci >= 0 {
		m.saveCursor = ci
	}
	name := m.requestName
	if name == "" {
		name = defaultRequestName(m.method, m.urlInput.Value())
	} else {
		name += " copy"
	}
	return m.startColInput(colInputNone, name, "Request name")
}

func (m Model) updateSave(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.saveStage {
	case saveEnterName, saveNewCollection:
		switch msg.String() {
		case "esc":
			if m.saveStage == saveNewCollection {
				m.saveStage = savePickCollection
				m.colInput.Blur()
				return m, nil
			}
			m.colInput.Blur()
			m.closeModal()
			return m, nil
		case "enter":
			val := strings.TrimSpace(m.colInput.Value())
			if val == "" {
				m.setStatus("Name cannot be empty", true)
				return m, nil
			}
			if m.saveStage == saveEnterName {
				m.saveName = val
				m.colInput.Blur()
				if len(m.collections) == 0 {
					m.saveStage = saveNewCollection
					return m, m.startColInput(colInputNone, "My Collection", "Collection name")
				}
				m.saveStage = savePickCollection
				return m, nil
			}
			if err := m.saveAs(m.saveName, -1, val); err != nil {
				m.setStatus("Save failed: "+err.Error(), true)
				return m, nil
			}
			m.colInput.Blur()
			m.closeModal()
			return m, nil
		}
		var cmd tea.Cmd
		m.colInput, cmd = m.colInput.Update(msg)
		return m, cmd

	case savePickCollection:
		n := len(m.collections) + 1
		switch msg.String() {
		case "esc":
			m.saveStage = saveEnterName
			return m, m.startColInput(colInputNone, m.saveName, "Request name")
		case "up", "k":
			if m.saveCursor > 0 {
				m.saveCursor--
			}
		case "down", "j":
			if m.saveCursor < n-1 {
				m.saveCursor++
			}
		case "enter":
			if m.saveCursor == len(m.collections) {
				m.saveStage = saveNewCollection
				return m, m.startColInput(colInputNone, "", "Collection name")
			}
			if err := m.saveAs(m.saveName, m.saveCursor, ""); err != nil {
				m.setStatus("Save failed: "+err.Error(), true)
				return m, nil
			}
			m.closeModal()
		}
	}
	return m, nil
}

func (m Model) renderSaveModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Save Request") + "\n\n")
	switch m.saveStage {
	case saveEnterName:
		b.WriteString(helpStyle.Render("Request name:") + "\n")
		b.WriteString(m.colInput.View() + "\n\n")
		b.WriteString(helpStyle.Render("Enter: Next • Esc: Cancel"))
	case savePickCollection:
		b.WriteString(helpStyle.Render(fmt.Sprintf("Save '%s' to collection:", m.saveName)) + "\n\n")
		for i := 0; i <= len(m.collections); i++ {
			label := ""
			if i < len(m.collections) {
				label = m.collections[i].Name
			} else {
				label = statusGreenStyle.Render("+ New collection")
			}
			prefix := "  "
			if i == m.saveCursor {
				prefix = cursorStyle.Render("▶ ")
				label = lipgloss.NewStyle().Background(panelBg).Render(label)
			}
			b.WriteString(prefix + label + "\n")
		}
		b.WriteString("\n" + helpStyle.Render("j/k: Move • Enter: Save • Esc: Back"))
	case saveNewCollection:
		b.WriteString(helpStyle.Render("New collection name:") + "\n")
		b.WriteString(m.colInput.View() + "\n\n")
		b.WriteString(helpStyle.Render("Enter: Save • Esc: Back"))
	}
	return m.placeModal("39", b.String())
}
