package tui

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	jsonKeyStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("#9CDCFE"))
	jsonStringStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6C37"))
	jsonNumberStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#00B894"))
	jsonBoolStyle        = lipgloss.NewStyle().Foreground(lipgloss.Color("#A855F7"))
	jsonPunctuationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#858585"))
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
			b.WriteRune(c)
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
			style := jsonStringStyle
			if i < len(runes) && runes[i] == ':' {
				style = jsonKeyStyle
			} else {
				// Look ahead past whitespace for a colon to detect object keys.
				j := i
				for j < len(runes) && (runes[j] == ' ' || runes[j] == '\t') {
					j++
				}
				if j < len(runes) && runes[j] == ':' {
					style = jsonKeyStyle
				}
			}
			b.WriteString(style.Render(token))
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

// prettyJSON indents JSON when possible and otherwise returns the input.
func prettyJSON(src string) string {
	trimmed := strings.TrimSpace(src)
	if trimmed == "" {
		return src
	}
	if !strings.HasPrefix(trimmed, "{") && !strings.HasPrefix(trimmed, "[") {
		return src
	}
	var buf strings.Builder
	if err := jsonIndent(&buf, []byte(trimmed)); err != nil {
		return src
	}
	return buf.String()
}

func jsonIndent(dst *strings.Builder, src []byte) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, src, "", "  "); err != nil {
		return err
	}
	_, err := dst.WriteString(buf.String())
	return err
}
