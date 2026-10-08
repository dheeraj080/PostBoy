package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/secrets"
)

func TestBuildURL(t *testing.T) {
	env := map[string]string{"BASE_URL": "https://api.test"}

	tests := []struct {
		name    string
		raw     string
		params  []config.KeyValue
		want    string
		wantErr bool
	}{
		{name: "empty", raw: "  ", wantErr: true},
		{name: "default scheme", raw: "example.com/a", want: "https://example.com/a"},
		{name: "uppercase scheme kept", raw: "HTTP://example.com", want: "http://example.com"},
		{name: "env var", raw: "{{BASE_URL}}/users", want: "https://api.test/users"},
		{name: "preserves existing query order", raw: "example.com/?z=1&a=2", want: "https://example.com/?z=1&a=2"},
		{
			name:   "appends enabled params only",
			raw:    "example.com/?z=1",
			params: []config.KeyValue{{Key: "q", Value: "a b", Enabled: true}, {Key: "off", Value: "x"}},
			want:   "https://example.com/?z=1&q=a+b",
		},
		{name: "missing host", raw: "https://", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildURL(tt.raw, tt.params, env, nil)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDoSendsExpandedRequest(t *testing.T) {
	var gotAuth, gotBody, gotMethod, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotMethod = r.Method
		gotQuery = r.URL.RawQuery
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Header().Set("X-Reply", "yes")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	res, err := NewWithHTTPClient(srv.Client()).Do(context.Background(), Request{
		Method:  http.MethodPost,
		URL:     "{{BASE}}/items",
		Body:    `{"name":"{{NAME}}"}`,
		Headers: []config.KeyValue{{Key: "Authorization", Value: "Bearer {{ secret.TOKEN }}", Enabled: true}, {Key: "X-Off", Value: "1"}},
		Params:  []config.KeyValue{{Key: "page", Value: "2", Enabled: true}},
		Env:     map[string]string{"BASE": srv.URL, "NAME": "widget"},
		Secrets: secrets.NewMemory(map[string]string{"TOKEN": "abc"}),
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusCreated || string(res.Body) != `{"ok":true}` || res.Headers.Get("X-Reply") != "yes" {
		t.Fatalf("bad response: %+v", res)
	}
	if gotMethod != http.MethodPost || gotAuth != "Bearer abc" || gotBody != `{"name":"widget"}` || gotQuery != "page=2" {
		t.Fatalf("server saw method=%q auth=%q body=%q query=%q", gotMethod, gotAuth, gotBody, gotQuery)
	}
	if res.RawHeaders["Authorization"] != "Bearer {{ secret.TOKEN }}" {
		t.Fatalf("RawHeaders should hold the unexpanded template, got %q", res.RawHeaders["Authorization"])
	}
	if _, ok := res.RawHeaders["X-Off"]; ok {
		t.Fatal("disabled header was sent")
	}
}

func TestDoTruncatesLargeBody(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(strings.Repeat("a", MaxBodyBytes+10)))
	}))
	defer srv.Close()

	res, err := NewWithHTTPClient(srv.Client()).Do(context.Background(), Request{Method: http.MethodGet, URL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Truncated || len(res.Body) != MaxBodyBytes {
		t.Fatalf("truncated=%v len=%d", res.Truncated, len(res.Body))
	}
}

func TestDoHonoursContextTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
		}
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := NewWithHTTPClient(srv.Client()).Do(ctx, Request{Method: http.MethodGet, URL: srv.URL})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected deadline exceeded, got %v", err)
	}
}

func TestAuth(t *testing.T) {
	var got *http.Request
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Clone(context.Background())
	}))
	defer srv.Close()
	client := NewWithHTTPClient(srv.Client())
	sec := secrets.NewMemory(map[string]string{"TOKEN": "tok", "PASS": "p@ss"})

	tests := []struct {
		name  string
		auth  config.Auth
		check func(t *testing.T, r *http.Request)
	}{
		{"bearer", config.Auth{Type: config.AuthBearer, Token: "{{ secret.TOKEN }}"}, func(t *testing.T, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer tok" {
				t.Fatalf("Authorization = %q", r.Header.Get("Authorization"))
			}
		}},
		{"basic", config.Auth{Type: config.AuthBasic, Username: "bob", Password: "{{ secret.PASS }}"}, func(t *testing.T, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok || u != "bob" || p != "p@ss" {
				t.Fatalf("basic = %q %q %v", u, p, ok)
			}
		}},
		{"apikey header", config.Auth{Type: config.AuthAPIKey, Key: "X-Api-Key", Value: "{{ secret.TOKEN }}"}, func(t *testing.T, r *http.Request) {
			if r.Header.Get("X-Api-Key") != "tok" {
				t.Fatalf("X-Api-Key = %q", r.Header.Get("X-Api-Key"))
			}
		}},
		{"apikey query", config.Auth{Type: config.AuthAPIKey, Key: "api_key", Value: "tok", In: config.APIKeyInQuery}, func(t *testing.T, r *http.Request) {
			if r.URL.Query().Get("api_key") != "tok" || r.URL.Query().Get("a") != "1" {
				t.Fatalf("query = %q", r.URL.RawQuery)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got = nil
			res, err := client.Do(context.Background(), Request{Method: "GET", URL: srv.URL + "/?a=1", Auth: tt.auth, Secrets: sec})
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, got)
			// Auth must never leak into persisted headers or the displayed URL.
			if len(res.RawHeaders) != 0 || strings.Contains(res.URL, "tok") {
				t.Fatalf("auth leaked: headers=%v url=%s", res.RawHeaders, res.URL)
			}
		})
	}
}

func TestAuthErrors(t *testing.T) {
	client := NewWithHTTPClient(http.DefaultClient)
	for name, a := range map[string]config.Auth{
		"empty bearer":   {Type: config.AuthBearer},
		"missing secret": {Type: config.AuthBearer, Token: "{{ secret.NOPE }}"},
		"empty key name": {Type: config.AuthAPIKey, Value: "x"},
		"unknown type":   {Type: "digest"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := client.Do(context.Background(), Request{Method: "GET", URL: "http://127.0.0.1:1", Auth: a, Secrets: secrets.NewMemory(nil)})
			if err == nil || !strings.Contains(err.Error(), "auth") {
				t.Fatalf("expected auth error, got %v", err)
			}
		})
	}
}

func TestMissingSecretIsAnError(t *testing.T) {
	client := NewWithHTTPClient(http.DefaultClient)
	cases := map[string]Request{
		"url":    {Method: "GET", URL: "http://127.0.0.1:1/{{ secret.X }}"},
		"param":  {Method: "GET", URL: "http://127.0.0.1:1", Params: []config.KeyValue{{Key: "k", Value: "{{ secret.X }}", Enabled: true}}},
		"header": {Method: "GET", URL: "http://127.0.0.1:1", Headers: []config.KeyValue{{Key: "K", Value: "{{ secret.X }}", Enabled: true}}},
		"body":   {Method: "POST", URL: "http://127.0.0.1:1", Body: "{{ secret.X }}"},
	}
	for name, req := range cases {
		t.Run(name, func(t *testing.T) {
			req.Secrets = secrets.NewMemory(nil)
			if _, err := client.Do(context.Background(), req); !errors.Is(err, ErrMissingSecret) {
				t.Fatalf("expected ErrMissingSecret, got %v", err)
			}
		})
	}
}

func TestBodyURLEncoded(t *testing.T) {
	var gotCT, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
	}))
	defer srv.Close()
	if _, err := NewWithHTTPClient(srv.Client()).Do(context.Background(), Request{
		Method: http.MethodPost, URL: srv.URL, BodyMode: config.BodyURLEncoded,
		Form: []config.KeyValue{{Key: "user", Value: "a b", Enabled: true}, {Key: "skip", Value: "x"}},
		Env:  map[string]string{}, Secrets: secrets.NewMemory(nil),
	}); err != nil {
		t.Fatal(err)
	}
	if gotCT != "application/x-www-form-urlencoded" || gotBody != "user=a+b" {
		t.Fatalf("ct=%q body=%q", gotCT, gotBody)
	}
}

func TestBodyMultipartFileUpload(t *testing.T) {
	var gotCT string
	var gotName, gotFile string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		if strings.HasPrefix(gotCT, "multipart/form-data; boundary=") {
			if err := r.ParseMultipartForm(1 << 20); err == nil {
				gotName = r.FormValue("name")
				fh, _, _ := r.FormFile("file")
				if fh != nil {
					defer fh.Close()
					b, _ := io.ReadAll(fh)
					gotFile = string(b)
				}
			}
		}
	}))
	defer srv.Close()
	dir := t.TempDir()
	p := filepath.Join(dir, "hello.txt")
	if err := os.WriteFile(p, []byte("hello multipart"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewWithHTTPClient(srv.Client()).Do(context.Background(), Request{
		Method: http.MethodPost, URL: srv.URL, BodyMode: config.BodyMultipart,
		Form: []config.KeyValue{{Key: "name", Value: "value", Enabled: true}, {Key: "file", Value: p, Enabled: true, File: true}},
		Env:  map[string]string{}, Secrets: secrets.NewMemory(nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(gotCT, "multipart/form-data; boundary=") || gotName != "value" || gotFile != "hello multipart" {
		t.Fatalf("ct=%q name=%q file=%q", gotCT, gotName, gotFile)
	}
}

func TestBodyFile(t *testing.T) {
	var gotCT string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
	}))
	defer srv.Close()
	dir := t.TempDir()
	p := filepath.Join(dir, "data.bin")
	if err := os.WriteFile(p, []byte{1, 2, 3, 4}, 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := NewWithHTTPClient(srv.Client()).Do(context.Background(), Request{
		Method: http.MethodPost, URL: srv.URL, BodyMode: config.BodyFile, BodyFile: p,
		Env: map[string]string{}, Secrets: secrets.NewMemory(nil),
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotCT != "application/octet-stream" || string(gotBody) != string([]byte{1, 2, 3, 4}) {
		t.Fatalf("ct=%q body=%v", gotCT, gotBody)
	}
}

func TestInsecureSkipVerify(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(200) }))
	defer ts.Close()
	if _, err := NewWithConfig(config.Default()).Do(context.Background(), Request{Method: http.MethodGet, URL: ts.URL}); err == nil {
		t.Fatal("expected TLS certificate error")
	}
	c := aConfig()
	c.InsecureSkipVerify = true
	res, err := NewWithConfig(c).Do(context.Background(), Request{Method: http.MethodGet, URL: ts.URL})
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("insecure GET: %v %v", res, err)
	}
}

func aConfig() config.Config {
	return config.Config{TimeoutSeconds: 15, Environments: []config.Environment{}, SecretNames: []string{}}
}

func TestRedirectPolicy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/target", http.StatusFound)
			return
		}
		fmt.Fprintln(w, "target")
	}))
	defer srv.Close()
	res, err := New().Do(context.Background(), Request{Method: http.MethodGet, URL: srv.URL + "/redirect"})
	if err != nil || res.StatusCode != 200 || !strings.Contains(string(res.Body), "target") {
		t.Fatalf("default redirect: %v %v", res, err)
	}
	c := aConfig()
	c.DisableRedirects = true
	res, err = NewWithConfig(c).Do(context.Background(), Request{Method: http.MethodGet, URL: srv.URL + "/redirect"})
	if err != nil || res.StatusCode != 302 {
		t.Fatalf("redirect disabled: %v %v", res, err)
	}
}


func TestProxyURLSchemes(t *testing.T) {
	for _, proxy := range []string{"http://127.0.0.1:1080", "https://user:pass@127.0.0.1:1080", "socks5://127.0.0.1:1080"} {
		c := aConfig()
		c.ProxyURL = proxy
		if NewWithConfig(c) == nil {
			t.Fatalf("proxy %s did not build client", proxy)
		}
	}
}

func TestCookieJarConfig(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/set" {
			http.SetCookie(w, &http.Cookie{Name: "sid", Value: "abc"})
		}
		if r.URL.Path == "/check" {
			fmt.Fprintln(w, r.Header.Get("Cookie"))
		}
	}))
	defer srv.Close()

	no := NewWithConfig(aConfig())
	_, _ = no.Do(context.Background(), Request{Method: http.MethodGet, URL: srv.URL + "/set"})
	res, _ := no.Do(context.Background(), Request{Method: http.MethodGet, URL: srv.URL + "/check"})
	if strings.Contains(string(res.Body), "sid=abc") {
		t.Fatal("cookies should be disabled by default")
	}

	c := aConfig()
	c.EnableCookies = true
	yes := NewWithConfig(c)
	_, _ = yes.Do(context.Background(), Request{Method: http.MethodGet, URL: srv.URL + "/set"})
	res, err := yes.Do(context.Background(), Request{Method: http.MethodGet, URL: srv.URL + "/check"})
	if err != nil || !strings.Contains(string(res.Body), "sid=abc") {
		t.Fatalf("cookies enabled: %v %s", err, res.Body)
	}
}

func TestProxyURL(t *testing.T) {
	backendCalled := false
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		backendCalled = true
		fmt.Fprintln(w, "backend")
	}))
	defer backend.Close()

	proxyCalled := false
	proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		proxyCalled = true
		fmt.Fprintln(w, "proxy")
	}))
	defer proxy.Close()

	c := aConfig()
	c.ProxyURL = proxy.URL
	res, err := NewWithConfig(c).Do(context.Background(), Request{Method: http.MethodGet, URL: backend.URL})
	if err != nil || !proxyCalled || backendCalled || string(res.Body) != "proxy\n" {
		t.Fatalf("proxy res=%v err=%v proxy=%v backend=%v body=%q", res, err, proxyCalled, backendCalled, res.Body)
	}
}

func TestSupportsBody(t *testing.T) {
	for _, m := range Methods {
		want := m == "POST" || m == "PUT" || m == "PATCH" || m == "DELETE"
		if SupportsBody(m) != want {
			t.Errorf("SupportsBody(%s) = %v", m, SupportsBody(m))
		}
	}
}
