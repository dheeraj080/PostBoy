package tui

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// JSON token colours from the spec. Every style carries the background so a
// token never falls back to the terminal default when nested in a styled
// viewport/panel line.
var (
	jsonKeyStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#9cdcfe")).Background(darkBg)
	jsonStringStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#ce9178")).Background(darkBg)
	jsonNumberStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#b5cea8")).Background(darkBg)
	jsonBoolStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#569cd6")).Background(darkBg)
	jsonPunctuationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#d4d4d4")).Background(darkBg)
)

// highlightJSON colorizes JSON tokens with a small, dependency-free tokenizer.
func highlightJSON(src string) string {
	var b strings.Builder
	runes := []rune(src)
	i := 0
	for i < len(runes) {
		c := runes[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			// Indent/whitespace cells carry the background explicitly: the
			// previous token's style ends with a reset, so a raw space here
			// would fall back to the terminal default background.
			b.WriteString(fillStyle.Render(string(c)))
			i++
		case c == '"':
			start := i
			i++
			for i < len(runes) {
				if runes[i] == '\\' && i+1 < len(runes) {
					i += 2
					continue
				}
				if runes[i] == '"' {
					i++
					break
				}
				i++
			}
			token := string(runes[start:i])
			// Look ahead past whitespace for a colon to detect object keys.
			j := i
			for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t') {
				j++
			}
			if j < len(runes) && runes[j] == ':' {
				b.WriteString(jsonKeyStyle.Render(token))
			} else {
				b.WriteString(jsonStringStyle.Render(token))
			}
		case c == '{' || c == '}' || c == '[' || c == ']' || c == ':' || c == ',':
			b.WriteString(jsonPunctuationStyle.Render(string(c)))
			i++
		case c >= '0' && c <= '9' || c == '-' || c == '+' || c == '.':
			start := i
			for i < len(runes) {
				ch := runes[i]
				if (ch >= '0' && ch <= '9') || ch == '.' || ch == 'e' || ch == 'E' || ch == '+' || ch == '-' {
					i++
					continue
				}
				break
			}
			b.WriteString(jsonNumberStyle.Render(string(runes[start:i])))
		default:
			start := i
			for i < len(runes) {
				ch := runes[i]
				if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' ||
					ch == '{' || ch == '}' || ch == '[' || ch == ']' || ch == ':' || ch == ',' || ch == '"' {
					break
				}
				i++
			}
			token := string(runes[start:i])
			switch token {
			case "true", "false", "null":
				b.WriteString(jsonBoolStyle.Render(token))
			default:
				b.WriteString(token)
			}
		}
	}
	return b.String()
}

// prettyJSON formats JSON deterministically: objects are expanded with a
// two-space indent, but an array whose elements are all scalars stays on one
// line ("labels": ["bug", "terminal"]). Non-JSON input is returned unchanged.
func prettyJSON(src string) string {
	trimmed := strings.TrimSpace(src)
	if trimmed == "" {
		return src
	}
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return src
	}
	dec := json.NewDecoder(strings.NewReader(trimmed))
	dec.UseNumber()
	node, err := parseJSONValue(dec)
	if err != nil {
		return src
	}
	var b strings.Builder
	writeJSONValue(&b, node, 0)
	return b.String()
}

// jsonKind classifies a parsed node for formatting.
type jsonKind int

const (
	jsonScalarKind jsonKind = iota
	jsonObjectKind
	jsonArrayKind
)

// jsonNode is an ordered JSON tree. Object key order is preserved exactly as
// decoded (never a map) so the rendering is deterministic.
type jsonNode struct {
	kind   jsonKind
	scalar string      // rendered scalar token
	keys   []string    // object keys in order
	kids   []*jsonNode // object values / array elements
}

func parseJSONValue(dec *json.Decoder) (*jsonNode, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch t := tok.(type) {
	case json.Delim:
		switch t {
		case '{':
			n := &jsonNode{kind: jsonObjectKind}
			for dec.More() {
				keyTok, err := dec.Token()
				if err != nil {
					return nil, err
				}
				key, ok := keyTok.(string)
				if !ok {
					return nil, fmt.Errorf("expected object key, got %v", keyTok)
				}
				val, err := parseJSONValue(dec)
				if err != nil {
					return nil, err
				}
				n.keys = append(n.keys, key)
				n.kids = append(n.kids, val)
			}
			if _, err := dec.Token(); err != nil { // closing }
				return nil, err
			}
			return n, nil
		case '[':
			n := &jsonNode{kind: jsonArrayKind}
			for dec.More() {
				val, err := parseJSONValue(dec)
				if err != nil {
					return nil, err
				}
				n.kids = append(n.kids, val)
			}
			if _, err := dec.Token(); err != nil { // closing ]
				return nil, err
			}
			return n, nil
		}
		return nil, fmt.Errorf("unexpected delimiter %v", t)
	default:
		// Re-encode scalars so strings keep valid JSON escaping and
		// json.Number keeps its original digits.
		b, err := json.Marshal(tok)
		if err != nil {
			return nil, err
		}
		return &jsonNode{kind: jsonScalarKind, scalar: string(b)}, nil
	}
}

// allScalars reports whether every node in kids is a scalar.
func allScalars(kids []*jsonNode) bool {
	for _, k := range kids {
		if k.kind != jsonScalarKind {
			return false
		}
	}
	return true
}

func writeJSONValue(b *strings.Builder, n *jsonNode, indent int) {
	switch n.kind {
	case jsonObjectKind:
		if len(n.keys) == 0 {
			b.WriteString("{}")
			return
		}
		b.WriteString("{\n")
		for i, key := range n.keys {
			b.WriteString(spaces((indent + 1) * 2))
			kb, err := json.Marshal(key)
			if err != nil {
				kb = []byte(`"` + key + `"`)
			}
			b.Write(kb)
			b.WriteString(": ")
			writeJSONValue(b, n.kids[i], indent+1)
			if i < len(n.keys)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString(spaces(indent*2) + "}")
	case jsonArrayKind:
		if len(n.kids) == 0 {
			b.WriteString("[]")
			return
		}
		if allScalars(n.kids) {
			b.WriteString("[")
			for i, k := range n.kids {
				if i > 0 {
					b.WriteString(", ")
				}
				b.WriteString(k.scalar)
			}
			b.WriteString("]")
			return
		}
		b.WriteString("[\n")
		for i, k := range n.kids {
			b.WriteString(spaces((indent + 1) * 2))
			writeJSONValue(b, k, indent+1)
			if i < len(n.kids)-1 {
				b.WriteString(",")
			}
			b.WriteString("\n")
		}
		b.WriteString(spaces(indent*2) + "]")
	default:
		b.WriteString(n.scalar)
	}
}
