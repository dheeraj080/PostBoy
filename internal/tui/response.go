package tui

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/tidwall/gjson"

	"github.com/dheeraj080/PostBoy/internal/httpclient"
)

type respInputMode int

const (
	respInputNone respInputMode = iota
	respInputSearch
	respInputFilter
	respInputSave
)

// match is a search hit: line index and byte range within that line.
type match struct{ line, start, end int }

// responseView holds the last response and the state of the response tools.
type responseView struct {
	body        []byte
	headers     http.Header
	contentType string
	truncated   bool
	binary      bool

	// Cached renderings of the full body.
	prettyPlain   string
	prettyColored string

	rawMode bool

	filter      string
	filterPlain string // pretty JSON of the filter result
	filterErr   string

	search  string
	matches []match
	current int

	inputMode   respInputMode
	input       textinput.Model
	saveConfirm string // path awaiting overwrite confirmation
}

func newRespInput() textinput.Model {
	in := textinput.New()
	in.CharLimit = 1024
	return in
}

// setResponse loads a successful response, resetting tool state.
func (v *responseView) set(res *httpclient.Response) {
	input := v.input
	*v = responseView{input: input}
	v.body = res.Body
	v.headers = res.Headers
	v.contentType = res.Headers.Get("Content-Type")
	v.truncated = res.Truncated
	v.binary = len(res.Body) > 0 && (!utf8.Valid(res.Body) || isBinaryContentType(v.contentType))
	if !v.binary {
		v.prettyPlain, v.prettyColored = formatBody(res.Body, v.contentType)
	}
}

func (v *responseView) clear() {
	input := v.input
	*v = responseView{input: input}
}

func (v *responseView) hasBody() bool { return v.headers != nil }

func isBinaryContentType(ct string) bool {
	ct = strings.ToLower(ct)
	for _, p := range []string{"image/", "audio/", "video/", "application/octet-stream", "application/pdf", "application/zip", "font/"} {
		if strings.HasPrefix(ct, p) {
			return true
		}
	}
	return false
}

// formatBody returns plain and syntax-highlighted renderings of a body.
func formatBody(body []byte, contentType string) (plain, colored string) {
	if len(body) == 0 {
		return "", dimStyle.Render("(empty body)")
	}
	ct := strings.ToLower(contentType)
	if strings.Contains(ct, "json") || json.Valid(body) {
		pretty := prettyJSON(string(body))
		return pretty, highlightJSON(pretty)
	}
	s := string(body)
	switch {
	case strings.Contains(ct, "html"):
		return s, highlight(s, "html")
	case strings.Contains(ct, "xml"):
		return s, highlight(s, "xml")
	}
	return s, s
}

// bodyPlain is the body text currently shown (filter result, raw or pretty).
func (v *responseView) bodyPlain() string {
	switch {
	case v.filter != "" && v.filterErr == "":
		return v.filterPlain
	case v.rawMode:
		return string(v.body)
	default:
		return v.prettyPlain
	}
}

// exportBytes is what copy/save operate on: the filter result if a filter
// is active, otherwise the raw body as received.
func (v *responseView) exportBytes() []byte {
	if v.filter != "" && v.filterErr == "" {
		return []byte(v.filterPlain)
	}
	return v.body
}

func (v *responseView) applyFilter(path string) {
	v.filter = strings.TrimSpace(path)
	v.filterPlain, v.filterErr = "", ""
	if v.filter == "" {
		return
	}
	if !gjson.ValidBytes(v.body) {
		v.filterErr = "response is not valid JSON"
		return
	}
	r := gjson.GetBytes(v.body, v.filter)
	if !r.Exists() {
		v.filterErr = "no match for " + v.filter
		return
	}
	raw := []byte(r.Raw)
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, raw, "", "  "); err == nil {
		v.filterPlain = pretty.String()
	} else {
		v.filterPlain = r.Raw
	}
}

func plainHeaders(h http.Header) string {
	keys := make([]string, 0, len(h))
	for k := range h {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		fmt.Fprintf(&b, "%s: %s\n", k, strings.Join(h[k], ", "))
	}
	return b.String()
}

// findMatches returns case-insensitive occurrences of q in text.
func findMatches(text, q string) []match {
	if q == "" {
		return nil
	}
	lq := strings.ToLower(q)
	var out []match
	for i, line := range strings.Split(text, "\n") {
		ll := strings.ToLower(line)
		// Lowercasing can change byte lengths for some scripts; fall back
		// to no highlighting on that line rather than mis-slicing.
		if len(ll) != len(line) {
			continue
		}
		for off := 0; ; {
			j := strings.Index(ll[off:], lq)
			if j < 0 {
				break
			}
			out = append(out, match{line: i, start: off + j, end: off + j + len(lq)})
			off += j + len(lq)
		}
	}
	return out
}

var (
	matchStyle        = lipgloss.NewStyle().Background(lipgloss.Color("#614D1E")).Foreground(lipgloss.Color("#FFFFFF"))
	currentMatchStyle = lipgloss.NewStyle().Background(postmanOrange).Foreground(lipgloss.Color("#000000")).Bold(true)
)

// renderMatches renders text with search hits highlighted.
func renderMatches(text string, matches []match, current int) string {
	lines := strings.Split(text, "\n")
	byLine := map[int][]int{} // line -> indexes into matches
	for i, mt := range matches {
		byLine[mt.line] = append(byLine[mt.line], i)
	}
	for ln, idxs := range byLine {
		line := lines[ln]
		var b strings.Builder
		pos := 0
		for _, i := range idxs {
			mt := matches[i]
			b.WriteString(line[pos:mt.start])
			style := matchStyle
			if i == current {
				style = currentMatchStyle
			}
			b.WriteString(style.Render(line[mt.start:mt.end]))
			pos = mt.end
		}
		b.WriteString(line[pos:])
		lines[ln] = b.String()
	}
	return strings.Join(lines, "\n")
}

// refreshResponse recomputes the viewport content from the response state.
// When keepOffset is false the view scrolls to the top.
func (m *Model) refreshResponse(keepOffset bool) {
	v := &m.resp
	offset := m.viewport.YOffset

	if v.hasBody() {
		m.respHeaders = formatResponseHeaders(v.headers)
		switch {
		case v.binary:
			m.respBody = statusYellowStyle.Render(fmt.Sprintf("Binary response (%s, %s) — press s to save it to a file.",
				nonEmpty(v.contentType, "unknown type"), formatSize(len(v.body))))
		case v.filter != "" && v.filterErr != "":
			m.respBody = statusRedStyle.Render("✗ Filter: " + v.filterErr)
		case v.filter != "":
			m.respBody = highlight(v.filterPlain, "json")
		case v.rawMode:
			m.respBody = string(v.body)
		default:
			m.respBody = v.prettyColored
		}
		if v.truncated && !v.binary {
			m.respBody += "\n" + statusYellowStyle.Render(fmt.Sprintf("\n[Response truncated at %d MB]", httpclient.MaxBodyBytes>>20))
		}
	}

	// Search (applies to the active tab's plain text).
	content := m.respBody
	if m.resTab == resTabHeaders {
		content = m.respHeaders
	}
	v.matches = nil
	if v.search != "" && v.hasBody() && !(m.resTab == resTabBody && v.binary) {
		plain := v.bodyPlain()
		if m.resTab == resTabHeaders {
			plain = plainHeaders(v.headers)
		}
		v.matches = findMatches(plain, v.search)
		if v.current >= len(v.matches) {
			v.current = 0
		}
		if len(v.matches) > 0 {
			content = renderMatches(plain, v.matches, v.current)
		}
	}
	m.viewport.SetContent(content)
	if keepOffset {
		m.viewport.SetYOffset(offset)
	} else {
		m.viewport.GotoTop()
	}
}

func (m *Model) scrollToCurrentMatch() {
	v := &m.resp
	if len(v.matches) == 0 {
		return
	}
	line := v.matches[v.current].line
	m.viewport.SetYOffset(max(line-m.viewport.Height/2, 0))
}

func nonEmpty(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func (m *Model) startRespInput(mode respInputMode, value, placeholder string) tea.Cmd {
	v := &m.resp
	v.inputMode = mode
	v.saveConfirm = ""
	v.input.SetValue(value)
	v.input.Placeholder = placeholder
	v.input.CursorEnd()
	v.input.Width = max(m.usableWidth()-40, 20)
	return v.input.Focus()
}

func (m *Model) stopRespInput() {
	m.resp.inputMode = respInputNone
	m.resp.saveConfirm = ""
	m.resp.input.Blur()
}

// updateResponseKeys handles keys while the response panel is focused.
// handled is false when the key should go to the viewport.
func (m *Model) updateResponseKeys(msg tea.KeyMsg) (cmd tea.Cmd, handled bool) {
	v := &m.resp
	if v.inputMode != respInputNone {
		switch msg.String() {
		case "esc":
			m.stopRespInput()
		case "enter":
			m.commitRespInput()
		default:
			v.input, cmd = v.input.Update(msg)
			if v.inputMode == respInputSave {
				v.saveConfirm = ""
			}
		}
		return cmd, true
	}

	if !v.hasBody() {
		return nil, false
	}
	switch msg.String() {
	case "/":
		return m.startRespInput(respInputSearch, v.search, "search…"), true
	case "f":
		if m.resTab == resTabBody {
			return m.startRespInput(respInputFilter, v.filter, "JSON path, e.g. data.#.id"), true
		}
	case "n", "N":
		if len(v.matches) > 0 {
			if msg.String() == "n" {
				v.current = (v.current + 1) % len(v.matches)
			} else {
				v.current = (v.current - 1 + len(v.matches)) % len(v.matches)
			}
			m.refreshResponse(true)
			m.scrollToCurrentMatch()
			return nil, true
		}
	case "r":
		if m.resTab == resTabBody && !v.binary {
			v.rawMode = !v.rawMode
			m.refreshResponse(false)
			if v.rawMode {
				m.setStatus("Showing raw body", false)
			} else {
				m.setStatus("Showing formatted body", false)
			}
			return nil, true
		}
	case "y":
		data := v.exportBytes()
		if m.resTab == resTabHeaders {
			data = []byte(plainHeaders(v.headers))
		}
		if v.binary && m.resTab == resTabBody {
			m.setStatus("Cannot copy a binary body; press s to save it", true)
			return nil, true
		}
		if err := copyToClipboard(string(data)); err != nil {
			m.setStatus("Copy failed: "+err.Error(), true)
		} else {
			m.setStatus(fmt.Sprintf("✓ Copied %s to clipboard", formatSize(len(data))), false)
		}
		return nil, true
	case "s":
		return m.startRespInput(respInputSave, m.defaultSaveName(), "file name"), true
	case "esc":
		if v.search != "" || v.filter != "" {
			v.search, v.filter, v.filterErr, v.filterPlain = "", "", "", ""
			m.refreshResponse(false)
			m.setStatus("Search and filter cleared", false)
			return nil, true
		}
	}
	return nil, false
}

func (m *Model) commitRespInput() {
	v := &m.resp
	val := v.input.Value()
	switch v.inputMode {
	case respInputSearch:
		v.search = val
		v.current = 0
		m.stopRespInput()
		m.refreshResponse(true)
		switch {
		case val == "":
			m.setStatus("Search cleared", false)
		case len(v.matches) == 0:
			m.setStatus(fmt.Sprintf("No matches for %q", val), true)
		default:
			m.scrollToCurrentMatch()
			m.setStatus(fmt.Sprintf("%d matches for %q • n/N: next/previous", len(v.matches), val), false)
		}
	case respInputFilter:
		v.applyFilter(val)
		m.stopRespInput()
		m.refreshResponse(false)
		switch {
		case v.filter == "":
			m.setStatus("Filter cleared", false)
		case v.filterErr != "":
			m.setStatus("Filter: "+v.filterErr, true)
		default:
			m.setStatus("Filter applied: "+v.filter+" • Esc to clear", false)
		}
	case respInputSave:
		path := m.resolveSavePath(val)
		if path == "" {
			m.setStatus("File name cannot be empty", true)
			return
		}
		if _, err := os.Stat(path); err == nil && v.saveConfirm != path {
			v.saveConfirm = path
			m.setStatus(fmt.Sprintf("%s exists — press Enter again to overwrite", path), true)
			return
		}
		data := v.exportBytes()
		if err := os.WriteFile(path, data, 0o644); err != nil {
			m.setStatus("Save failed: "+err.Error(), true)
			return
		}
		m.stopRespInput()
		m.setStatus(fmt.Sprintf("✓ Saved %s to %s", formatSize(len(data)), path), false)
	}
}

func (m *Model) resolveSavePath(name string) string {
	name = expandPath(name)
	if name == "" {
		return ""
	}
	if filepath.IsAbs(name) {
		return name
	}
	dir := m.opts.ExportDir
	if dir == "" {
		if wd, err := os.Getwd(); err == nil {
			dir = wd
		}
	}
	return filepath.Join(dir, name)
}

func (m *Model) defaultSaveName() string {
	ct := strings.ToLower(m.resp.contentType)
	ext := ".txt"
	switch {
	case m.resp.filter != "" && m.resp.filterErr == "":
		ext = ".json"
	case strings.Contains(ct, "json"):
		ext = ".json"
	case strings.Contains(ct, "html"):
		ext = ".html"
	case strings.Contains(ct, "xml"):
		ext = ".xml"
	case strings.HasPrefix(ct, "image/png"):
		ext = ".png"
	case strings.HasPrefix(ct, "image/jpeg"):
		ext = ".jpg"
	case strings.HasPrefix(ct, "image/gif"):
		ext = ".gif"
	case strings.HasPrefix(ct, "application/pdf"):
		ext = ".pdf"
	case m.resp.binary:
		ext = ".bin"
	}
	return "response" + ext
}

// renderResponseToolbar returns the input line or tool hints shown next to
// the response tabs.
func (m Model) renderResponseToolbar() string {
	v := m.resp
	switch v.inputMode {
	case respInputSearch:
		return cursorStyle.Render("Search: ") + v.input.View()
	case respInputFilter:
		return cursorStyle.Render("Filter: ") + v.input.View()
	case respInputSave:
		return cursorStyle.Render("Save as: ") + v.input.View()
	}
	var tags []string
	if v.filter != "" {
		tags = append(tags, statusYellowStyle.Render("filter: "+truncate(v.filter, 30)))
	}
	if v.search != "" {
		n := len(v.matches)
		pos := 0
		if n > 0 {
			pos = v.current + 1
		}
		tags = append(tags, statusYellowStyle.Render(fmt.Sprintf("search: %q %d/%d", truncate(v.search, 20), pos, n)))
	}
	if v.rawMode {
		tags = append(tags, statusYellowStyle.Render("raw"))
	}
	if m.focus == focusResContent && v.hasBody() {
		tags = append(tags, helpStyle.Render("/ search  f filter  r raw  y copy  s save"))
	}
	return strings.Join(tags, "  ")
}
