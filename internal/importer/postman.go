package importer

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
)

const postmanSchemaV21 = "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"

// Postman v2.1 collection format (subset).
type pmCollection struct {
	Info     pmInfo   `json:"info"`
	Item     []pmItem `json:"item"`
	Auth     *pmAuth  `json:"auth,omitempty"`
	Variable []pmKV   `json:"variable,omitempty"`
}

type pmInfo struct {
	Name   string `json:"name"`
	Schema string `json:"schema"`
}

type pmItem struct {
	Name    string     `json:"name"`
	Item    []pmItem   `json:"item,omitempty"` // folder
	Request *pmRequest `json:"request,omitempty"`
	Auth    *pmAuth    `json:"auth,omitempty"` // folder-level auth
}

type pmRequest struct {
	Method string          `json:"method"`
	Header []pmKV          `json:"header"`
	URL    json.RawMessage `json:"url"`
	Body   *pmBody         `json:"body,omitempty"`
	Auth   *pmAuth         `json:"auth,omitempty"`
}

type pmKV struct {
	Key      string `json:"key"`
	Value    any    `json:"value"`
	Disabled bool   `json:"disabled,omitempty"`
	Type     string `json:"type,omitempty"`
	// Postman uses Type: "file" + Src for multipart file fields.
	Src string `json:"src,omitempty"`
}

func (kv pmKV) str() string {
	switch v := kv.Value.(type) {
	case nil:
		return ""
	case string:
		return v
	default:
		b, _ := json.Marshal(v)
		return string(b)
	}
}

type pmURL struct {
	Raw   string   `json:"raw"`
	Host  []string `json:"host,omitempty"`
	Path  []string `json:"path,omitempty"`
	Query []pmKV   `json:"query,omitempty"`
}

type pmBody struct {
	Mode       string `json:"mode"`
	Raw        string `json:"raw,omitempty"`
	URLEncoded []pmKV `json:"urlencoded,omitempty"`
	FormData   []pmKV `json:"formdata,omitempty"`
	GraphQL    *struct {
		Query     string `json:"query"`
		Variables string `json:"variables"`
	} `json:"graphql,omitempty"`
	File *struct {
		Src string `json:"src"`
	} `json:"file,omitempty"`
}

type pmAuth struct {
	Type   string `json:"type"`
	Bearer []pmKV `json:"bearer,omitempty"`
	Basic  []pmKV `json:"basic,omitempty"`
	APIKey []pmKV `json:"apikey,omitempty"`
}

func authParam(kvs []pmKV, key string) string {
	for _, kv := range kvs {
		if kv.Key == key {
			return kv.str()
		}
	}
	return ""
}

// PostmanImport is the result of importing a Postman collection.
type PostmanImport struct {
	Collection collection.Collection
	// Variables are the collection's variables (to create an environment).
	Variables map[string]string
	Warnings  []string
}

// ImportPostman parses a Postman v2.0/v2.1 collection. Folders are flattened
// into "Folder / Request" names; folder and collection auth is inherited.
func ImportPostman(data []byte) (PostmanImport, error) {
	var pc pmCollection
	if err := json.Unmarshal(data, &pc); err != nil {
		return PostmanImport{}, fmt.Errorf("parse Postman collection: %w", err)
	}
	if pc.Info.Name == "" && len(pc.Item) == 0 {
		return PostmanImport{}, errors.New("not a Postman collection (missing info/item)")
	}
	if pc.Info.Schema != "" && !strings.Contains(pc.Info.Schema, "v2.") {
		return PostmanImport{}, fmt.Errorf("unsupported Postman schema %q (export as v2.1)", pc.Info.Schema)
	}

	res := PostmanImport{
		Collection: collection.Collection{ID: collection.NewID(), Name: pc.Info.Name},
		Variables:  map[string]string{},
	}
	res.Collection.Variables = res.Variables
	if res.Collection.Name == "" {
		res.Collection.Name = "Imported collection"
	}
	for _, v := range pc.Variable {
		if v.Key != "" && !v.Disabled {
			res.Variables[v.Key] = v.str()
		}
	}
	var walk func(items []pmItem, prefix string, inherited *pmAuth)
	walk = func(items []pmItem, prefix string, inherited *pmAuth) {
		for _, it := range items {
			auth := inherited
			if it.Auth != nil {
				auth = it.Auth
			}
			name := it.Name
			if prefix != "" {
				name = prefix + " / " + name
			}
			if it.Request == nil {
				walk(it.Item, name, auth)
				continue
			}
			req, warns := convertPMRequest(*it.Request, auth)
			req.ID = collection.NewID()
			req.Name = name
			for _, w := range warns {
				res.Warnings = append(res.Warnings, name+": "+w)
			}
			res.Collection.Requests = append(res.Collection.Requests, req)
		}
	}
	walk(pc.Item, "", pc.Auth)
	return res, nil
}

func convertPMRequest(p pmRequest, inherited *pmAuth) (config.Request, []string) {
	var warns []string
	r := config.Request{
		Method:   strings.ToUpper(p.Method),
		Headers:  []config.KeyValue{},
		Params:   []config.KeyValue{},
		BodyMode: config.BodyRaw,
		Form:     []config.KeyValue{},
	}
	if r.Method == "" {
		r.Method = http.MethodGet
	}

	// URL is either a string or an object.
	var s string
	if err := json.Unmarshal(p.URL, &s); err == nil {
		r.URL = s
	} else {
		var u pmURL
		if err := json.Unmarshal(p.URL, &u); err == nil {
			if u.Raw != "" {
				r.URL = u.Raw
			} else {
				r.URL = strings.Join(u.Host, ".")
				if len(u.Path) > 0 {
					r.URL += "/" + strings.Join(u.Path, "/")
				}
			}
			// Prefer the structured query list (it keeps disabled params);
			// strip the duplicated query string from the raw URL.
			if len(u.Query) > 0 {
				if i := strings.IndexByte(r.URL, '?'); i >= 0 {
					r.URL = r.URL[:i]
				}
				for _, q := range u.Query {
					r.Params = append(r.Params, config.KeyValue{Key: q.Key, Value: q.str(), Enabled: !q.Disabled})
				}
			}
		}
	}

	for _, h := range p.Header {
		r.Headers = append(r.Headers, config.KeyValue{Key: h.Key, Value: h.str(), Enabled: !h.Disabled})
	}

	if b := p.Body; b != nil {
		switch b.Mode {
		case "raw", "":
			r.Body = b.Raw
			r.BodyMode = config.BodyRaw
		case "urlencoded":
			r.BodyMode = config.BodyURLEncoded
			for _, kv := range b.URLEncoded {
				r.Form = append(r.Form, config.KeyValue{Key: kv.Key, Value: kv.str(), Enabled: !kv.Disabled, File: false})
			}
			setDefaultHeader(&r, "Content-Type", "application/x-www-form-urlencoded")
		case "graphql":
			if b.GraphQL != nil {
				payload := map[string]any{"query": b.GraphQL.Query}
				if strings.TrimSpace(b.GraphQL.Variables) != "" {
					var vars any
					if json.Unmarshal([]byte(b.GraphQL.Variables), &vars) == nil {
						payload["variables"] = vars
					}
				}
				out, _ := json.MarshalIndent(payload, "", "  ")
				r.Body = string(out)
				r.BodyMode = config.BodyRaw
				setDefaultHeader(&r, "Content-Type", "application/json")
			}
		case "formdata":
			r.BodyMode = config.BodyMultipart
			for _, kv := range b.FormData {
				entry := config.KeyValue{Key: kv.Key, Value: kv.str(), Enabled: !kv.Disabled}
				if kv.Type == "file" || kv.Src != "" {
					entry.File = true
					if kv.Src != "" {
						entry.Value = kv.Src
					}
				}
				r.Form = append(r.Form, entry)
			}
		case "file":
			r.BodyMode = config.BodyFile
			if b.File != nil {
				r.BodyFile = b.File.Src
			}
		default:
			warns = append(warns, fmt.Sprintf("body mode %q not supported; body dropped", b.Mode))
		}
	}

	auth := inherited
	if p.Auth != nil {
		auth = p.Auth
	}
	if auth != nil {
		switch auth.Type {
		case "bearer":
			r.Auth = config.Auth{Type: config.AuthBearer, Token: authParam(auth.Bearer, "token")}
		case "basic":
			r.Auth = config.Auth{Type: config.AuthBasic,
				Username: authParam(auth.Basic, "username"), Password: authParam(auth.Basic, "password")}
		case "apikey":
			in := config.APIKeyInHeader
			if authParam(auth.APIKey, "in") == "query" {
				in = config.APIKeyInQuery
			}
			r.Auth = config.Auth{Type: config.AuthAPIKey,
				Key: authParam(auth.APIKey, "key"), Value: authParam(auth.APIKey, "value"), In: in}
		case "noauth", "":
		default:
			warns = append(warns, fmt.Sprintf("auth type %q not supported; auth dropped", auth.Type))
		}
	}
	return r.Normalize(), warns
}

// ExportPostman renders a collection (and optional variables) as a Postman
// v2.1 collection.
func ExportPostman(c collection.Collection, vars map[string]string) ([]byte, error) {
	pc := pmCollection{
		Info: pmInfo{Name: c.Name, Schema: postmanSchemaV21},
		Item: []pmItem{},
	}
	for _, k := range sortedKeys(vars) {
		pc.Variable = append(pc.Variable, pmKV{Key: k, Value: vars[k]})
	}
	for _, r := range c.Requests {
		pr := pmRequest{Method: r.Method, Header: []pmKV{}}
		for _, h := range r.Headers {
			pr.Header = append(pr.Header, pmKV{Key: h.Key, Value: h.Value, Disabled: !h.Enabled, Type: "text"})
		}
		u := pmURL{Raw: r.URL}
		// Mirror any query string typed into the URL into the structured
		// list, so importers that prefer it (including ours) keep it.
		if i := strings.IndexByte(r.URL, '?'); i >= 0 && len(r.Params) > 0 {
			for _, pair := range strings.Split(r.URL[i+1:], "&") {
				if pair == "" {
					continue
				}
				k, v, _ := strings.Cut(pair, "=")
				if uk, err := url.QueryUnescape(k); err == nil {
					k = uk
				}
				if uv, err := url.QueryUnescape(v); err == nil {
					v = uv
				}
				u.Query = append(u.Query, pmKV{Key: k, Value: v})
			}
		}
		for _, p := range r.Params {
			u.Query = append(u.Query, pmKV{Key: p.Key, Value: p.Value, Disabled: !p.Enabled})
		}
		if len(u.Query) > 0 {
			// Postman's raw URL must include enabled query params.
			var q []string
			for _, p := range r.Params {
				if p.Enabled {
					q = append(q, queryPair(p.Key, p.Value))
				}
			}
			if len(q) > 0 {
				sep := "?"
				if strings.Contains(u.Raw, "?") {
					sep = "&"
				}
				u.Raw += sep + strings.Join(q, "&")
			}
		}
		raw, err := json.Marshal(u)
		if err != nil {
			return nil, err
		}
		pr.URL = raw
		switch r.BodyMode {
		case config.BodyURLEncoded:
			pr.Body = &pmBody{Mode: "urlencoded"}
			for _, f := range r.Form {
				pr.Body.URLEncoded = append(pr.Body.URLEncoded, pmKV{Key: f.Key, Value: f.Value, Disabled: !f.Enabled, Type: "text"})
			}
		case config.BodyMultipart:
			pr.Body = &pmBody{Mode: "formdata"}
			for _, f := range r.Form {
				entry := pmKV{Key: f.Key, Value: f.Value, Disabled: !f.Enabled, Type: "text"}
				if f.File {
					entry.Type = "file"
					entry.Value = ""
					entry.Src = f.Value
				}
				pr.Body.FormData = append(pr.Body.FormData, entry)
			}
		case config.BodyFile:
			pr.Body = &pmBody{Mode: "file"}
			if r.BodyFile != "" {
				pr.Body.File = &struct {
					Src string `json:"src"`
				}{Src: r.BodyFile}
			}
		default:
			if r.Body != "" {
				pr.Body = &pmBody{Mode: "raw", Raw: r.Body}
			}
		}
		switch r.Auth.Type {
		case config.AuthBearer:
			pr.Auth = &pmAuth{Type: "bearer", Bearer: []pmKV{{Key: "token", Value: r.Auth.Token, Type: "string"}}}
		case config.AuthBasic:
			pr.Auth = &pmAuth{Type: "basic", Basic: []pmKV{
				{Key: "username", Value: r.Auth.Username, Type: "string"},
				{Key: "password", Value: r.Auth.Password, Type: "string"}}}
		case config.AuthAPIKey:
			in := "header"
			if r.Auth.In == config.APIKeyInQuery {
				in = "query"
			}
			pr.Auth = &pmAuth{Type: "apikey", APIKey: []pmKV{
				{Key: "key", Value: r.Auth.Key, Type: "string"},
				{Key: "value", Value: r.Auth.Value, Type: "string"},
				{Key: "in", Value: in, Type: "string"}}}
		}
		pc.Item = append(pc.Item, pmItem{Name: r.Name, Request: &pr})
	}
	return json.MarshalIndent(pc, "", "  ")
}
