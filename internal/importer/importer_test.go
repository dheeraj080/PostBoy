package importer

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/httpclient"
	"github.com/dheeraj080/PostBoy/internal/secrets"
)

func header(r config.Request, key string) string {
	for _, h := range r.Headers {
		if strings.EqualFold(h.Key, key) {
			return h.Value
		}
	}
	return ""
}

func TestParseCurlBasic(t *testing.T) {
	r, warns, err := ParseCurl(`curl -X POST 'https://api.test/items?x=1' -H 'Content-Type: application/json' -H "X-Quote: say \"hi\"" --data-raw '{"name":"it'\''s"}'`)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("warnings: %v", warns)
	}
	if r.Method != "POST" || r.URL != "https://api.test/items?x=1" {
		t.Fatalf("method/url = %s %s", r.Method, r.URL)
	}
	if header(r, "Content-Type") != "application/json" || header(r, "X-Quote") != `say "hi"` {
		t.Fatalf("headers = %+v", r.Headers)
	}
	if r.Body != `{"name":"it's"}` {
		t.Fatalf("body = %q", r.Body)
	}
}

func TestParseCurlVariants(t *testing.T) {
	tests := []struct {
		name  string
		cmd   string
		check func(t *testing.T, r config.Request)
	}{
		{"implicit GET", `curl https://a.test`, func(t *testing.T, r config.Request) {
			if r.Method != "GET" || r.URL != "https://a.test" || r.Body != "" {
				t.Fatalf("%+v", r)
			}
		}},
		{"data implies POST + form content type", `curl a.test -d a=1 -d b=2`, func(t *testing.T, r config.Request) {
			if r.Method != "POST" || r.BodyMode != config.BodyURLEncoded || len(r.Form) != 2 || r.Form[0].Key != "a" || header(r, "Content-Type") != "application/x-www-form-urlencoded" {
				t.Fatalf("%+v", r)
			}
		}},
		{"-G moves data to query", `curl -G a.test/s -d q=go --data-urlencode 'name=a b'`, func(t *testing.T, r config.Request) {
			if r.Method != "GET" || r.URL != "a.test/s?name=a+b&q=go" || r.Body != "" || r.BodyMode != config.BodyRaw {
				t.Fatalf("%+v", r)
			}
		}},
		{"--json", `curl --json '{"a":1}' a.test`, func(t *testing.T, r config.Request) {
			if r.Method != "POST" || header(r, "Content-Type") != "application/json" || header(r, "Accept") != "application/json" {
				t.Fatalf("%+v", r)
			}
		}},
		{"basic auth + combined flags", `curl -sSL -u bob:secret -XPUT a.test`, func(t *testing.T, r config.Request) {
			if r.Method != "PUT" || r.Auth.Type != config.AuthBasic || r.Auth.Username != "bob" || r.Auth.Password != "secret" {
				t.Fatalf("%+v", r)
			}
		}},
		{"long options with =", `curl --request=DELETE --url=https://a.test/1 --header='Accept: */*'`, func(t *testing.T, r config.Request) {
			if r.Method != "DELETE" || r.URL != "https://a.test/1" || header(r, "Accept") != "*/*" {
				t.Fatalf("%+v", r)
			}
		}},
		{"bash continuation", "curl 'https://a.test' \\\n  -H 'A: 1' \\\n  --compressed", func(t *testing.T, r config.Request) {
			if r.URL != "https://a.test" || header(r, "A") != "1" {
				t.Fatalf("%+v", r)
			}
		}},
		{"cmd.exe style (Chrome)", "curl ^\"https://a.test/x^\" ^\r\n  -H ^\"accept: */*^\"", func(t *testing.T, r config.Request) {
			if r.URL != "https://a.test/x" || header(r, "accept") != "*/*" {
				t.Fatalf("%+v", r)
			}
		}},
		{"powershell continuation", "curl.exe https://a.test `\n -H 'A: 1'", func(t *testing.T, r config.Request) {
			if r.URL != "https://a.test" || header(r, "A") != "1" {
				t.Fatalf("%+v", r)
			}
		}},
		{"continuations flattened to spaces", "curl \\  'https://a.test' \\  -H 'A: 1' `  -d x", func(t *testing.T, r config.Request) {
			if r.URL != "https://a.test" || header(r, "A") != "1" || r.Body != "x" {
				t.Fatalf("%+v", r)
			}
		}},
		{"ANSI-C quoting", `curl a.test --data-binary $'line1\nline2'`, func(t *testing.T, r config.Request) {
			if r.Body != "line1\nline2" {
				t.Fatalf("%q", r.Body)
			}
		}},
		{"user agent, cookie, referer, head", `curl -I -A ua/1 -b 'a=1' -e https://ref a.test`, func(t *testing.T, r config.Request) {
			if r.Method != "HEAD" || header(r, "User-Agent") != "ua/1" || header(r, "Cookie") != "a=1" || header(r, "Referer") != "https://ref" {
				t.Fatalf("%+v", r)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, _, err := ParseCurl(tt.cmd)
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, r)
		})
	}
}

func TestParseCurlErrorsAndWarnings(t *testing.T) {
	if _, _, err := ParseCurl("wget x"); !errors.Is(err, ErrNotCurl) {
		t.Fatalf("expected ErrNotCurl, got %v", err)
	}
	if _, _, err := ParseCurl("curl -H 'A: 1'"); err == nil {
		t.Fatal("expected missing URL error")
	}
	if _, _, err := ParseCurl("curl 'unterminated"); err == nil {
		t.Fatal("expected quote error")
	}
	if _, _, err := ParseCurl("curl a.test -H"); err == nil {
		t.Fatal("expected missing value error")
	}
	r, warns, err := ParseCurl("curl a.test -F file=@x.png -d @body.json")
	if err != nil || len(warns) != 1 || r.BodyMode != config.BodyMultipart || len(r.Form) != 1 || !r.Form[0].File || r.Form[0].Value != "x.png" {
		t.Fatalf("r=%+v warns=%v err=%v", r, warns, err)
	}
}

func TestToCurlRoundTrip(t *testing.T) {
	in := config.Request{
		Method:  "PATCH",
		URL:     "{{BASE}}/items/1",
		Body:    `{"note":"it's"}`,
		Headers: []config.KeyValue{{Key: "Content-Type", Value: "application/json", Enabled: true}, {Key: "X-Off", Value: "1"}},
		Params:  []config.KeyValue{{Key: "q", Value: "a b", Enabled: true}},
		Auth:    config.Auth{Type: config.AuthBearer, Token: "{{ secret.TOKEN }}"},
	}
	out := ToCurl(in, map[string]string{"BASE": "https://api.test"})
	if !strings.Contains(out, "{{ secret.TOKEN }}") {
		t.Fatalf("secret reference should be kept, got:\n%s", out)
	}
	if strings.Contains(out, "X-Off") {
		t.Fatal("disabled header exported")
	}
	r, warns, err := ParseCurl(out)
	if err != nil || len(warns) != 0 {
		t.Fatalf("re-parse failed: %v %v\n%s", err, warns, out)
	}
	if r.Method != "PATCH" || r.URL != "https://api.test/items/1?q=a+b" || r.Body != in.Body {
		t.Fatalf("round trip: %+v\n%s", r, out)
	}
	if header(r, "Authorization") != "Bearer {{ secret.TOKEN }}" || header(r, "Content-Type") != "application/json" {
		t.Fatalf("headers: %+v", r.Headers)
	}
}

func TestToCurlAuthVariants(t *testing.T) {
	basic := ToCurl(config.Request{Method: "GET", URL: "a.test", Auth: config.Auth{Type: config.AuthBasic, Username: "u", Password: "p w"}}, nil)
	if !strings.Contains(basic, `-u 'u:p w'`) {
		t.Fatalf("basic: %s", basic)
	}
	q := ToCurl(config.Request{Method: "GET", URL: "a.test", Auth: config.Auth{Type: config.AuthAPIKey, Key: "k", Value: "v", In: config.APIKeyInQuery}}, nil)
	if !strings.Contains(q, "https://a.test?k=v") {
		t.Fatalf("apikey query: %s", q)
	}
	head := ToCurl(config.Request{Method: "HEAD", URL: "a.test"}, nil)
	if !strings.Contains(head, "--head") {
		t.Fatalf("head: %s", head)
	}
}

// The exported curl command, when executed by our own client after
// re-import, sends the same request.
func TestCurlImportedRequestSends(t *testing.T) {
	var gotBody, gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody, gotCT = string(b), r.Header.Get("Content-Type")
	}))
	defer srv.Close()

	r, _, err := ParseCurl("curl " + srv.URL + ` -H 'Content-Type: application/json' -d '{"a":1}'`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = httpclient.NewWithHTTPClient(srv.Client()).Do(context.Background(), httpclient.Request{
		Method: r.Method, URL: r.URL, Body: r.Body, Headers: r.Headers, Params: r.Params, Auth: r.Auth, Secrets: secrets.NewMemory(nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotBody != `{"a":1}` || gotCT != "application/json" {
		t.Fatalf("server saw body=%q ct=%q", gotBody, gotCT)
	}
}

const samplePostman = `{
  "info": {"name": "Sample API", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
  "auth": {"type": "bearer", "bearer": [{"key": "token", "value": "{{token}}", "type": "string"}]},
  "variable": [{"key": "baseUrl", "value": "https://api.test"}, {"key": "off", "value": "x", "disabled": true}],
  "item": [
    {"name": "Users", "item": [
      {"name": "List users", "request": {
        "method": "GET",
        "header": [{"key": "Accept", "value": "application/json"}, {"key": "X-Off", "value": "1", "disabled": true}],
        "url": {"raw": "{{baseUrl}}/users?page=1&debug=1", "host": ["{{baseUrl}}"], "path": ["users"],
                "query": [{"key": "page", "value": "1"}, {"key": "debug", "value": "1", "disabled": true}]}
      }},
      {"name": "Create user", "request": {
        "method": "POST", "header": [],
        "url": "{{baseUrl}}/users",
        "body": {"mode": "raw", "raw": "{\"name\":\"a\"}"},
        "auth": {"type": "basic", "basic": [{"key": "username", "value": "u"}, {"key": "password", "value": "p"}]}
      }}
    ]},
    {"name": "Login", "request": {
      "method": "POST", "header": [], "url": "{{baseUrl}}/login",
      "body": {"mode": "urlencoded", "urlencoded": [{"key": "user", "value": "a b"}, {"key": "skip", "value": "x", "disabled": true}]},
      "auth": {"type": "noauth"}
    }},
    {"name": "Upload", "request": {"method": "POST", "header": [], "url": "{{baseUrl}}/up", "body": {"mode": "formdata", "formdata": []}}},
    {"name": "GQL", "request": {"method": "POST", "header": [], "url": "{{baseUrl}}/graphql",
      "body": {"mode": "graphql", "graphql": {"query": "{ me { id } }", "variables": "{\"a\":1}"}}}}
  ]
}`

func TestImportPostman(t *testing.T) {
	res, err := ImportPostman([]byte(samplePostman))
	if err != nil {
		t.Fatal(err)
	}
	c := res.Collection
	if c.Name != "Sample API" || len(c.Requests) != 5 {
		t.Fatalf("collection = %s, %d requests", c.Name, len(c.Requests))
	}
	if res.Variables["baseUrl"] != "https://api.test" || res.Variables["off"] != "" {
		t.Fatalf("variables = %v", res.Variables)
	}

	list := c.Requests[0]
	if list.Name != "Users / List users" || list.URL != "{{baseUrl}}/users" {
		t.Fatalf("list = %+v", list)
	}
	if len(list.Params) != 2 || list.Params[1].Enabled {
		t.Fatalf("params = %+v", list.Params)
	}
	if len(list.Headers) != 2 || list.Headers[1].Enabled {
		t.Fatalf("headers = %+v", list.Headers)
	}
	if list.Auth.Type != config.AuthBearer || list.Auth.Token != "{{token}}" {
		t.Fatalf("inherited auth = %+v", list.Auth)
	}

	create := c.Requests[1]
	if create.URL != "{{baseUrl}}/users" || create.Body != `{"name":"a"}` || create.Auth.Type != config.AuthBasic || create.Auth.Password != "p" {
		t.Fatalf("create = %+v", create)
	}

	login := c.Requests[2]
	if login.BodyMode != config.BodyURLEncoded || login.Body != "" || len(login.Form) != 2 || login.Form[0].Value != "a b" || login.Form[1].Enabled || header(login, "Content-Type") != "application/x-www-form-urlencoded" || login.Auth.Type != "" {
		t.Fatalf("login = %v\n", login)
	}

	gql := c.Requests[4]
	var payload map[string]any
	if err := json.Unmarshal([]byte(gql.Body), &payload); err != nil || payload["query"] != "{ me { id } }" {
		t.Fatalf("graphql body = %q", gql.Body)
	}

	upload := c.Requests[3]
	if upload.BodyMode != config.BodyMultipart || len(upload.Form) != 0 {
		t.Fatalf("upload = %v", upload)
	}
	if len(res.Warnings) != 0 {
		t.Fatalf("warnings = %v", res.Warnings)
	}
	for _, r := range c.Requests {
		if r.ID == "" {
			t.Fatal("request without ID")
		}
	}
}

func TestImportPostmanRejectsGarbage(t *testing.T) {
	for _, in := range []string{`nope`, `{}`, `{"info":{"name":"x","schema":"https://schema.getpostman.com/json/collection/v1.0.0/collection.json"},"item":[]}`} {
		if _, err := ImportPostman([]byte(in)); err == nil {
			t.Fatalf("expected error for %s", in)
		}
	}
}

func TestPostmanRoundTrip(t *testing.T) {
	in := collection.Collection{Name: "RT", Requests: []config.Request{
		{Name: "One", Method: "PUT", URL: "{{B}}/x?keep=1", Body: "data",
			Headers: []config.KeyValue{{Key: "A", Value: "1", Enabled: true}, {Key: "Off", Value: "0"}},
			Params:  []config.KeyValue{{Key: "p", Value: "v", Enabled: true}, {Key: "d", Value: "x"}},
			Auth:    config.Auth{Type: config.AuthAPIKey, Key: "k", Value: "{{ secret.K }}", In: config.APIKeyInQuery}},
	}}
	data, err := ExportPostman(in, map[string]string{"B": "https://b.test"})
	if err != nil {
		t.Fatal(err)
	}
	res, err := ImportPostman(data)
	if err != nil {
		t.Fatal(err)
	}
	if res.Variables["B"] != "https://b.test" {
		t.Fatalf("vars = %v", res.Variables)
	}
	out := res.Collection.Requests[0]
	if out.Name != "One" || out.Method != "PUT" || out.Body != "data" || out.URL != "{{B}}/x" {
		t.Fatalf("out = %+v", out)
	}
	// URL query + params, disabled param preserved.
	if len(out.Params) != 3 || out.Params[0].Key != "keep" || out.Params[2].Enabled {
		t.Fatalf("params = %+v", out.Params)
	}
	if out.Auth != in.Requests[0].Auth {
		t.Fatalf("auth = %+v", out.Auth)
	}
	if len(out.Headers) != 2 || out.Headers[1].Enabled {
		t.Fatalf("headers = %+v", out.Headers)
	}
}
