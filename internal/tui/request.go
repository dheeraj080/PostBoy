package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
)

// currentRequest captures the editor state as a Request.
func (m *Model) currentRequest() config.Request {
	r := config.Request{
		Name:     m.requestName,
		Method:   m.method,
		URL:      m.urlInput.Value(),
		Headers:  m.headers.snapshot(),
		Params:   m.params.snapshot(),
		Auth:     m.auth.auth,
		BodyMode: m.bodyMode,
		Form:     m.form.snapshot(),
		BodyFile: m.bodyFileInput.Value(),
	}
	if m.bodyMode == config.BodyRaw {
		r.Body = m.bodyInput.Value()
	}
	if m.origin != nil {
		r.ID = m.origin.RequestID
	}
	return r.Normalize()
}

// loadRequest puts r into the editor. origin links it to a saved request
// (nil for an unsaved draft); saved is the baseline for dirty tracking.
func (m *Model) loadRequest(r config.Request, origin *config.Origin, saved config.Request) {
	r = r.Normalize()
	m.method = r.Method
	m.urlInput.SetValue(r.URL)
	m.urlInput.CursorEnd()
	m.bodyInput.SetValue(r.Body)
	m.bodyMode = r.BodyMode
	m.form.setItems(r.Form)
	m.bodyFileInput.SetValue(r.BodyFile)
	m.headers.setItems(r.Headers)
	m.params.setItems(r.Params)
	m.auth.set(r.Auth)
	m.origin = origin
	m.requestName = saved.Name
	m.savedSnapshot = saved.Normalize()
	if origin == nil {
		m.requestName = ""
		m.savedSnapshot = config.NewRequest()
	}
	if !m.supportsBody() && m.reqTab == reqTabBody {
		m.reqTab = reqTabHeaders
	}
}

// isDirty reports whether the editor has unsaved changes.
func (m *Model) isDirty() bool {
	return !config.SameContent(m.currentRequest(), m.savedSnapshot)
}

// originCollection returns the collection the open request belongs to.
func (m *Model) originCollection() (ci, ri int) {
	if m.origin == nil {
		return -1, -1
	}
	return collection.Find(m.collections, m.origin.CollectionID, m.origin.RequestID)
}

// requestTitle is shown in the info bar.
func (m *Model) requestTitle() string {
	title := "Untitled request"
	if ci, ri := m.originCollection(); ri >= 0 {
		title = m.collections[ci].Name + " › " + m.collections[ci].Requests[ri].Name
	}
	if m.isDirty() {
		title += " ●"
	}
	return title
}

func (m *Model) persistCollections() error {
	if m.dir == "" {
		return nil
	}
	return collection.Save(m.dir, m.collections)
}

// saveRequest overwrites the linked saved request, or opens the "save as"
// dialog when the draft is not linked to one.
func (m *Model) saveRequest() tea.Cmd {
	ci, ri := m.originCollection()
	if ri < 0 {
		return m.openSaveDialog()
	}
	r := m.currentRequest()
	r.ID = m.collections[ci].Requests[ri].ID
	r.Name = m.collections[ci].Requests[ri].Name
	m.collections[ci].Requests[ri] = r
	if err := m.persistCollections(); err != nil {
		m.setStatus("Could not save collections: "+err.Error(), true)
		return nil
	}
	m.savedSnapshot = r
	m.persistOrWarn()
	m.setStatus(fmt.Sprintf("✓ Saved '%s'", r.Name), false)
	return nil
}

// saveAs stores the current request as a new entry named name in
// collection ci (or a new collection called newCollection when ci < 0).
func (m *Model) saveAs(name string, ci int, newCollection string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("request name cannot be empty")
	}
	if ci < 0 {
		newCollection = strings.TrimSpace(newCollection)
		if newCollection == "" {
			return fmt.Errorf("collection name cannot be empty")
		}
		m.collections = append(m.collections, collection.Collection{ID: collection.NewID(), Name: newCollection})
		ci = len(m.collections) - 1
	}
	r := m.currentRequest()
	r.ID = collection.NewID()
	r.Name = name
	m.collections[ci].Requests = append(m.collections[ci].Requests, r)
	if err := m.persistCollections(); err != nil {
		return err
	}
	m.origin = &config.Origin{CollectionID: m.collections[ci].ID, RequestID: r.ID}
	m.requestName = name
	m.savedSnapshot = r
	m.persistOrWarn()
	m.colExpanded[m.collections[ci].ID] = true
	m.setStatus(fmt.Sprintf("✓ Saved '%s' to %s", name, m.collections[ci].Name), false)
	return nil
}

// newRequest resets the editor to a blank draft. Returns false (and sets a
// warning) when there are unsaved changes that have not been confirmed.
func (m *Model) newRequest(confirmed bool) bool {
	if m.isDirty() && !confirmed {
		m.setStatus("Unsaved changes — press Alt+N again to discard", true)
		return false
	}
	m.loadRequest(config.NewRequest(), nil, config.Request{})
	m.persistOrWarn()
	m.setStatus("New request", false)
	return true
}

// defaultRequestName suggests a name from the method and URL path.
func defaultRequestName(method, rawURL string) string {
	u := strings.TrimSpace(rawURL)
	for _, p := range []string{"https://", "http://"} {
		u = strings.TrimPrefix(u, p)
	}
	if i := strings.IndexAny(u, "?#"); i >= 0 {
		u = u[:i]
	}
	if i := strings.Index(u, "/"); i >= 0 {
		u = u[i:]
	}
	if u == "" || u == "/" {
		return method + " request"
	}
	return method + " " + u
}
