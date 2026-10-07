package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/aymanbagabas/go-osc52/v2"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/importer"
)

// copyToClipboard writes s to the system clipboard, falling back to the
// OSC 52 terminal escape (works over SSH in most modern terminals).
// Overridable in tests.
var copyToClipboard = func(s string) error {
	if err := clipboard.WriteAll(s); err == nil {
		return nil
	}
	_, err := osc52.New(s).WriteTo(os.Stderr)
	return err
}

func newImportInput() textinput.Model {
	in := textinput.New()
	in.CharLimit = 0 // unlimited: curl commands can be long
	in.Placeholder = "curl https://...   or   path/to/collection.postman_collection.json"
	return in
}

func (m *Model) openImport() tea.Cmd {
	m.modal = modalImport
	m.importConfirm = false
	m.importInput.SetValue("")
	return m.importInput.Focus()
}

func (m Model) updateImport(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.importInput.Blur()
		m.closeModal()
		return m, nil
	case "enter":
		text := strings.TrimSpace(m.importInput.Value())
		if text == "" {
			return m, nil
		}
		return m, m.runImport(text)
	}
	m.importConfirm = false
	var cmd tea.Cmd
	m.importInput, cmd = m.importInput.Update(msg)
	return m, cmd
}

func (m Model) renderImportModal() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("Import"))
	b.WriteString("\n\n")
	b.WriteString(helpStyle.Render("Paste a curl command (bash, cmd or PowerShell style) to open it in the editor,"))
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("or enter the path of a Postman v2.1 collection file to add it to your collections."))
	b.WriteString("\n\n")
	m.importInput.Width = max(m.termWidth-20, 20)
	b.WriteString(m.importInput.View())
	b.WriteString("\n\n")
	if m.importConfirm {
		b.WriteString(statusYellowStyle.Render("Unsaved changes in the editor. Press Enter again to replace them."))
		b.WriteString("\n\n")
	}
	b.WriteString(helpStyle.Render("Enter: Import • Esc: Cancel"))
	return m.placeModal("205", b.String())
}

// runImport imports a curl command into the editor, or a Postman collection
// file into the collections.
func (m *Model) runImport(text string) tea.Cmd {
	if looksLikeCurl(text) {
		req, warns, err := importer.ParseCurl(text)
		if err != nil {
			m.setStatus("Import failed: "+err.Error(), true)
			return nil
		}
		if m.isDirty() && !m.importConfirm {
			m.importConfirm = true
			return nil
		}
		m.loadRequest(req, nil, config.Request{})
		m.persistOrWarn()
		m.importInput.Blur()
		m.closeModal()
		m.setStatus(withWarnings("✓ Imported curl command", warns), len(warns) > 0)
		return m.setFocus(focusURL)
	}

	path := expandPath(text)
	data, err := os.ReadFile(path)
	if err != nil {
		m.setStatus("Import failed: "+err.Error(), true)
		return nil
	}
	res, err := importer.ImportPostman(data)
	if err != nil {
		m.setStatus("Import failed: "+err.Error(), true)
		return nil
	}
	envName := m.addImported(res)
	if err := m.persistCollections(); err != nil {
		m.setStatus("Could not save collections: "+err.Error(), true)
		return nil
	}
	m.persistOrWarn()

	msg := fmt.Sprintf("✓ Imported %s into %q", plural(len(res.Collection.Requests), "request"), res.Collection.Name)
	if envName != "" {
		msg += fmt.Sprintf(" + environment %q", envName)
	}
	m.importInput.Blur()
	m.openCollections()
	m.setStatus(withWarnings(msg, res.Warnings), len(res.Warnings) > 0)
	return nil
}

// addImported adds an imported collection (and its variables as a new
// environment) to the model. Returns the created environment name, if any.
func (m *Model) addImported(res importer.PostmanImport) string {
	c := res.Collection
	c.Name = m.uniqueCollectionName(c.Name)
	m.collections = append(m.collections, c)
	m.colExpanded[c.ID] = true
	for i, r := range m.colRows() {
		if r.ci == len(m.collections)-1 && r.ri < 0 {
			m.colCursor = i
		}
	}
	if len(res.Variables) == 0 {
		return ""
	}
	name := c.Name
	for n := 2; m.envNameTaken(name, -1); n++ {
		name = fmt.Sprintf("%s (%d)", c.Name, n)
	}
	m.cfg.Environments = append(m.cfg.Environments, config.Environment{Name: name, Vars: res.Variables})
	return name
}

func (m *Model) uniqueCollectionName(base string) string {
	taken := func(n string) bool {
		for _, c := range m.collections {
			if strings.EqualFold(c.Name, n) {
				return true
			}
		}
		return false
	}
	name := base
	for n := 2; taken(name); n++ {
		name = fmt.Sprintf("%s (%d)", base, n)
	}
	return name
}

func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}

func withWarnings(msg string, warns []string) string {
	switch len(warns) {
	case 0:
		return msg
	case 1:
		return msg + " • warning: " + warns[0]
	default:
		return fmt.Sprintf("%s • %d warnings, first: %s", msg, len(warns), warns[0])
	}
}

func looksLikeCurl(s string) bool {
	f := strings.Fields(s)
	return len(f) > 0 && (f[0] == "curl" || strings.EqualFold(f[0], "curl.exe"))
}

// expandPath trims quotes (from drag-and-drop) and expands a leading ~.
func expandPath(p string) string {
	p = strings.Trim(strings.TrimSpace(p), `"'`)
	if p == "~" || strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[1:])
		}
	}
	return p
}

// copyAsCurl copies the current request as a curl command. Environment
// variables are expanded; secrets stay as {{ secret.NAME }}.
func (m *Model) copyAsCurl() {
	cmd := importer.ToCurl(m.currentRequest(), m.env().Vars)
	if err := copyToClipboard(cmd); err != nil {
		m.setStatus("Copy failed: "+err.Error(), true)
		return
	}
	m.setStatus("✓ Copied as curl (secrets left as {{ secret.NAME }})", false)
}

var unsafeFileChars = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

// exportCollection writes collection ci as a Postman v2.1 file into the
// export directory and returns its path.
func (m *Model) exportCollection(ci int) (string, error) {
	c := m.collections[ci]
	data, err := importer.ExportPostman(c, nil)
	if err != nil {
		return "", err
	}
	dir := m.opts.ExportDir
	if dir == "" {
		if dir, err = os.Getwd(); err != nil {
			return "", err
		}
	}
	base := strings.Trim(unsafeFileChars.ReplaceAllString(c.Name, "_"), "_")
	if base == "" {
		base = "collection"
	}
	path := filepath.Join(dir, base+".postman_collection.json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return "", err
	}
	return path, nil
}

// ImportFile imports a Postman collection file into the collections stored
// in dir (used by the "postboy import" CLI). New variables are added as an
// environment in the config.
func ImportFile(dir, path string) (string, error) {
	data, err := os.ReadFile(expandPath(path))
	if err != nil {
		return "", err
	}
	res, err := importer.ImportPostman(data)
	if err != nil {
		return "", err
	}
	cols, err := collection.Load(dir)
	if err != nil {
		return "", err
	}
	cfg, err := config.Load(dir)
	if err != nil {
		return "", err
	}
	m := Model{collections: cols, cfg: cfg, colExpanded: map[string]bool{}}
	envName := m.addImported(res)
	if err := collection.Save(dir, m.collections); err != nil {
		return "", err
	}
	if envName != "" {
		if err := config.Save(dir, m.cfg); err != nil {
			return "", err
		}
	}
	name := m.collections[len(m.collections)-1].Name
	summary := fmt.Sprintf("Imported %s into collection %q", plural(len(res.Collection.Requests), "request"), name)
	if envName != "" {
		summary += fmt.Sprintf(" and created environment %q", envName)
	}
	for _, w := range res.Warnings {
		summary += "\n  warning: " + w
	}
	return summary, nil
}
