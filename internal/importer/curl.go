// Package importer converts between PostBoy requests and external formats
// (curl command lines, Postman collections).
package importer

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/interp"
)

// ErrNotCurl is returned when the input does not start with "curl".
var ErrNotCurl = errors.New("not a curl command")

// Short curl options that consume a value.
const shortWithArg = "XHduAbeoFTmxwKcDErYyzCU"

// Long curl options that consume a value.
var longWithArg = map[string]bool{
	"request": true, "header": true, "data": true, "data-raw": true, "data-binary": true,
	"data-ascii": true, "data-urlencode": true, "json": true, "user": true, "user-agent": true,
	"cookie": true, "referer": true, "url": true, "output": true, "form": true, "form-string": true,
	"max-time": true, "connect-timeout": true, "proxy": true, "cert": true, "key": true,
	"cacert": true, "write-out": true, "retry": true, "config": true, "cookie-jar": true,
	"unix-socket": true, "resolve": true, "upload-file": true, "oauth2-bearer": true,
	"dump-header": true, "proxy-user": true, "range": true, "continue-at": true,
}

// ParseCurl converts a curl command line into a Request. It understands
// POSIX shell quoting (including $'...'), line continuations for bash (\),
// cmd.exe (^) and PowerShell (`), and the common curl options. Unsupported
// options that affect the request are reported as warnings.
func ParseCurl(cmd string) (config.Request, []string, error) {
	all, err := tokenize(normalizeContinuations(cmd))
	if err != nil {
		return config.Request{}, nil, err
	}
	// Drop leftovers from continuations whose newline was flattened to a
	// space by a single-line input (e.g. "\ " -> " ", "` ", "^ ").
	tokens := all[:0]
	for _, t := range all {
		if strings.TrimSpace(t) == "" || t == "`" || t == "^" {
			continue
		}
		tokens = append(tokens, t)
	}
	if len(tokens) == 0 || !(tokens[0] == "curl" || strings.EqualFold(tokens[0], "curl.exe")) {
		return config.Request{}, nil, ErrNotCurl
	}

	var (
		req      = config.Request{Headers: []config.KeyValue{}, Params: []config.KeyValue{}}
		warns    []string
		method   string
		rawURL   string
		data     []string
		formKV   []config.KeyValue
		mpartKV  []config.KeyValue
		filePath string
		getMode  bool
		jsonMode bool
	)

	addHeader := func(h string) {
		k, v, ok := strings.Cut(h, ":")
		k = strings.TrimSpace(k)
		if !ok {
			// "Name;" sends an empty header in curl.
			if strings.HasSuffix(k, ";") {
				req.Headers = append(req.Headers, config.KeyValue{Key: strings.TrimSuffix(k, ";"), Enabled: true})
			}
			return
		}
		v = strings.TrimSpace(v)
		if k == "" {
			return
		}
		if v == "" {
			return // "Name:" removes a header in curl
		}
		req.Headers = append(req.Headers, config.KeyValue{Key: k, Value: v, Enabled: true})
	}

	handle := func(name, val string) {
		switch name {
		case "X", "request":
			method = strings.ToUpper(val)
		case "H", "header":
			addHeader(val)
		case "d", "data", "data-ascii", "data-binary", "data-raw":
			if strings.HasPrefix(val, "@") && name != "data-raw" {
				warns = append(warns, fmt.Sprintf("file data (%s) is not supported; kept literally", val))
			}
			data = append(data, val)
		case "data-urlencode":
			kv := strings.SplitN(val, "=", 2)
			if len(kv) == 2 {
				formKV = append(formKV, config.KeyValue{Key: kv[0], Value: kv[1], Enabled: true})
			} else {
				formKV = append(formKV, config.KeyValue{Key: val, Enabled: true})
			}
		case "json":
			jsonMode = true
			data = append(data, val)
		case "u", "user":
			user, pass, _ := strings.Cut(val, ":")
			req.Auth = config.Auth{Type: config.AuthBasic, Username: user, Password: pass}
		case "oauth2-bearer":
			req.Auth = config.Auth{Type: config.AuthBearer, Token: val}
		case "A", "user-agent":
			addHeader("User-Agent: " + val)
		case "b", "cookie":
			if strings.Contains(val, "=") {
				addHeader("Cookie: " + val)
			} else {
				warns = append(warns, "cookie file (-b "+val+") is not supported")
			}
		case "e", "referer":
			addHeader("Referer: " + val)
		case "url":
			rawURL = val
		case "G", "get":
			getMode = true
		case "I", "head":
			method = http.MethodHead
		case "F", "form", "form-string":
			// -F 'name=value' or 'file=@/path[;type=...]'.
			parts := strings.SplitN(val, "=", 2)
			entry := config.KeyValue{Enabled: true}
			if len(parts) == 2 {
				entry.Key = parts[0]
				v := parts[1]
				if strings.HasPrefix(v, "@") {
					entry.File = true
					if i := strings.Index(v, ";"); i > 0 {
						v = v[:i]
					}
					entry.Value = strings.TrimPrefix(v, "@")
				} else {
					entry.Value = v
				}
			} else {
				entry.Key = val
			}
			mpartKV = append(mpartKV, entry)
		case "T", "upload-file":
			filePath = strings.TrimPrefix(val, "@")
			if method == "" {
				method = http.MethodPut
			}
		case "x", "proxy", "cert", "key", "cacert", "E", "resolve", "unix-socket":
			warns = append(warns, fmt.Sprintf("option --%s ignored", name))
		}
	}

	for i := 1; i < len(tokens); i++ {
		tok := tokens[i]
		switch {
		case tok == "--":
			if i+1 < len(tokens) && rawURL == "" {
				rawURL = tokens[i+1]
			}
			i = len(tokens)
		case strings.HasPrefix(tok, "--"):
			name, val, hasVal := strings.Cut(tok[2:], "=")
			if longWithArg[name] {
				if !hasVal {
					if i+1 >= len(tokens) {
						return req, warns, fmt.Errorf("option --%s requires a value", name)
					}
					i++
					val = tokens[i]
				}
			}
			handle(name, val)
		case strings.HasPrefix(tok, "-") && len(tok) > 1:
			// Combined short flags, e.g. -sSL or -XPOST.
			for j := 1; j < len(tok); j++ {
				c := tok[j : j+1]
				if strings.Contains(shortWithArg, c) {
					val := tok[j+1:]
					if val == "" {
						if i+1 >= len(tokens) {
							return req, warns, fmt.Errorf("option -%s requires a value", c)
						}
						i++
						val = tokens[i]
					}
					handle(c, val)
					break
				}
				handle(c, "")
			}
		default:
			if rawURL == "" {
				rawURL = tok
			}
		}
	}

	if rawURL == "" {
		return req, warns, errors.New("no URL found in curl command")
	}

	// -d values are form fields when every one is a k=v pair; otherwise treat as raw.
	allKV := true
	for _, d := range data {
		kv := strings.SplitN(d, "=", 2)
		if len(kv) != 2 || strings.TrimSpace(kv[0]) == "" {
			allKV = false
			break
		}
	}
	if allKV && len(data) > 0 && !jsonMode {
		for _, d := range data {
			kv := strings.SplitN(d, "=", 2)
			formKV = append(formKV, config.KeyValue{Key: strings.TrimSpace(kv[0]), Value: strings.TrimSpace(kv[1]), Enabled: true})
		}
		data = nil
	}

	body := strings.Join(data, "&")
	switch {
	case len(mpartKV) > 0:
		req.BodyMode = config.BodyMultipart
		req.Form = append(req.Form, mpartKV...)
		if method == "" {
			method = http.MethodPost
		}
	case filePath != "":
		req.BodyMode = config.BodyFile
		req.BodyFile = filePath
		if method == "" {
			method = http.MethodPost
		}
	case len(formKV) > 0 && getMode:
		var qp []string
		for _, kv := range formKV {
			if kv.Enabled && kv.Key != "" {
				qp = append(qp, url.QueryEscape(kv.Key)+"="+url.QueryEscape(kv.Value))
			}
		}
		if len(qp) > 0 {
			sep := "?"
			if strings.Contains(rawURL, "?") {
				sep = "&"
			}
			rawURL += sep + strings.Join(qp, "&")
		}
		req.BodyMode = config.BodyRaw
		req.Form = []config.KeyValue{}
		if method == "" {
			method = http.MethodGet
		}
	case len(formKV) > 0:
		req.BodyMode = config.BodyURLEncoded
		req.Form = append(req.Form, formKV...)
		setDefaultHeader(&req, "Content-Type", "application/x-www-form-urlencoded")
		if method == "" {
			method = http.MethodPost
		}
	case getMode && body != "":
		sep := "?"
		if strings.Contains(rawURL, "?") {
			sep = "&"
		}
		rawURL += sep + body
		body = ""
		if method == "" {
			method = http.MethodGet
		}
	case body != "":
		req.BodyMode = config.BodyRaw
		if jsonMode {
			setDefaultHeader(&req, "Content-Type", "application/json")
			setDefaultHeader(&req, "Accept", "application/json")
		} else {
			setDefaultHeader(&req, "Content-Type", "application/x-www-form-urlencoded")
		}
		if method == "" {
			method = http.MethodPost
		}
	}
	if method == "" {
		method = http.MethodGet
	}

	req.Method = method
	req.URL = rawURL
	req.Body = body
	return req.Normalize(), warns, nil
}

func setDefaultHeader(r *config.Request, key, value string) {
	for _, h := range r.Headers {
		if strings.EqualFold(h.Key, key) {
			return
		}
	}
	r.Headers = append(r.Headers, config.KeyValue{Key: key, Value: value, Enabled: true})
}

// normalizeContinuations joins lines continued with \, ^ (cmd.exe) or `
// (PowerShell) and strips cmd.exe ^-escapes.
func normalizeContinuations(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	for _, c := range []string{"\\\n", "^\n", "`\n"} {
		s = strings.ReplaceAll(s, c, " ")
	}
	// Chrome "Copy as cURL (cmd)" escapes quotes and specials with ^.
	if strings.Contains(s, `^"`) {
		var b strings.Builder
		for i := 0; i < len(s); i++ {
			if s[i] == '^' && i+1 < len(s) {
				i++
			}
			b.WriteByte(s[i])
		}
		s = b.String()
	}
	return s
}

// tokenize splits a command line using POSIX shell quoting rules.
func tokenize(s string) ([]string, error) {
	var (
		tokens []string
		cur    strings.Builder
		inTok  bool
	)
	flush := func() {
		if inTok {
			tokens = append(tokens, cur.String())
			cur.Reset()
			inTok = false
		}
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n':
			flush()
		case c == '\'':
			inTok = true
			end := strings.IndexByte(s[i+1:], '\'')
			if end < 0 {
				return nil, errors.New("unterminated single quote")
			}
			cur.WriteString(s[i+1 : i+1+end])
			i += end + 1
		case c == '$' && i+1 < len(s) && s[i+1] == '\'':
			inTok = true
			i += 2
			for ; i < len(s) && s[i] != '\''; i++ {
				if s[i] == '\\' && i+1 < len(s) {
					i++
					switch s[i] {
					case 'n':
						cur.WriteByte('\n')
					case 't':
						cur.WriteByte('\t')
					case 'r':
						cur.WriteByte('\r')
					default:
						cur.WriteByte(s[i])
					}
					continue
				}
				cur.WriteByte(s[i])
			}
			if i >= len(s) {
				return nil, errors.New("unterminated $'...' quote")
			}
		case c == '"':
			inTok = true
			i++
			for ; i < len(s) && s[i] != '"'; i++ {
				if s[i] == '\\' && i+1 < len(s) && strings.IndexByte("\"\\$`\n", s[i+1]) >= 0 {
					i++
				}
				cur.WriteByte(s[i])
			}
			if i >= len(s) {
				return nil, errors.New("unterminated double quote")
			}
		case c == '\\' && i+1 < len(s):
			inTok = true
			i++
			cur.WriteByte(s[i])
		default:
			inTok = true
			cur.WriteByte(c)
		}
	}
	flush()
	return tokens, nil
}

// ToCurl renders r as a POSIX-shell curl command. Environment and dynamic
// variables are expanded; secret references are left as {{ secret.NAME }}
// so the output is safe to share.
func ToCurl(r config.Request, env map[string]string) string {
	exp := func(s string) string { return interp.ExpandEnv(s, env) }

	u := strings.TrimSpace(exp(r.URL))
	lower := strings.ToLower(u)
	if u != "" && !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		u = "https://" + u
	}
	var query []string
	for _, p := range r.Params {
		if p.Enabled && strings.TrimSpace(p.Key) != "" {
			query = append(query, queryPair(strings.TrimSpace(p.Key), exp(p.Value)))
		}
	}
	if r.Auth.Type == config.AuthAPIKey && r.Auth.In == config.APIKeyInQuery && r.Auth.Key != "" {
		query = append(query, queryPair(r.Auth.Key, exp(r.Auth.Value)))
	}
	if len(query) > 0 {
		sep := "?"
		if strings.Contains(u, "?") {
			sep = "&"
		}
		u += sep + strings.Join(query, "&")
	}

	parts := []string{"curl"}
	method := r.Method
	if method == "" {
		method = http.MethodGet
	}
	hasBody := false
	switch r.BodyMode {
	case config.BodyURLEncoded:
		for _, f := range r.Form {
			if f.Enabled && strings.TrimSpace(f.Key) != "" {
				hasBody = true
				break
			}
		}
	case config.BodyMultipart:
		hasBody = len(r.Form) > 0
	case config.BodyFile:
		hasBody = r.BodyFile != ""
	default:
		hasBody = r.Body != ""
	}
	if method == http.MethodGet || method == http.MethodHead {
		hasBody = false
	}
	if method == http.MethodHead {
		parts = append(parts, "--head")
	} else if method != http.MethodGet && !(method == http.MethodPost && hasBody) {
		parts = append(parts, "-X "+method)
	}
	parts = append(parts, shellQuote(u))

	for _, h := range r.Headers {
		if h.Enabled && strings.TrimSpace(h.Key) != "" {
			parts = append(parts, "-H "+shellQuote(strings.TrimSpace(h.Key)+": "+exp(h.Value)))
		}
	}
	switch r.Auth.Type {
	case config.AuthBearer:
		parts = append(parts, "-H "+shellQuote("Authorization: Bearer "+exp(r.Auth.Token)))
	case config.AuthBasic:
		parts = append(parts, "-u "+shellQuote(exp(r.Auth.Username)+":"+exp(r.Auth.Password)))
	case config.AuthAPIKey:
		if r.Auth.In != config.APIKeyInQuery && r.Auth.Key != "" {
			parts = append(parts, "-H "+shellQuote(r.Auth.Key+": "+exp(r.Auth.Value)))
		}
	}
	switch r.BodyMode {
	case config.BodyURLEncoded:
		for _, f := range r.Form {
			if f.Enabled && strings.TrimSpace(f.Key) != "" {
				parts = append(parts, "--data-urlencode "+shellQuote(f.Key+"="+exp(f.Value)))
			}
		}
	case config.BodyMultipart:
		for _, f := range r.Form {
			if !f.Enabled || strings.TrimSpace(f.Key) == "" {
				continue
			}
			if f.File {
				parts = append(parts, "-F "+shellQuote(f.Key+"=@"+exp(f.Value)))
			} else {
				parts = append(parts, "-F "+shellQuote(f.Key+"="+exp(f.Value)))
			}
		}
	case config.BodyFile:
		if r.BodyFile != "" {
			parts = append(parts, "--data-binary @"+shellQuote(exp(r.BodyFile)))
		}
	default:
		if hasBody {
			parts = append(parts, "--data-raw "+shellQuote(exp(r.Body)))
		}
	}
	return strings.Join(parts, " \\\n  ")
}

// queryPair encodes a query parameter, keeping {{ ... }} templates readable.
func queryPair(k, v string) string {
	if strings.Contains(v, "{{") {
		return url.QueryEscape(k) + "=" + v
	}
	return url.QueryEscape(k) + "=" + url.QueryEscape(v)
}

func shellQuote(s string) string {
	if s == "" {
		return "''"
	}
	safe := true
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9' || strings.ContainsRune("-_./:=@%+,", c)) {
			safe = false
			break
		}
	}
	if safe {
		return s
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// sortedKeys returns map keys in order (used by Postman import/export).
func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
