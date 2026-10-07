// Package httpclient builds and executes HTTP requests from PostBoy's
// request model (templated URL, headers, params and body).
package httpclient

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/interp"
)

// MaxBodyBytes caps how much of a response body is read into memory.
const MaxBodyBytes = 10 << 20

// Methods lists the supported HTTP methods in cycle order.
var Methods = []string{
	http.MethodGet, http.MethodPost, http.MethodPut, http.MethodPatch,
	http.MethodDelete, http.MethodHead, http.MethodOptions,
}

// SupportsBody reports whether the UI should offer a request body for method.
func SupportsBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	}
	return false
}

// Request is a templated request; templates are expanded at send time.
type Request struct {
	Method  string
	URL     string
	Body    string
	Headers []config.KeyValue
	Params  []config.KeyValue
	Auth    config.Auth
	Env     map[string]string
	Secrets interp.SecretGetter

	BodyMode config.BodyMode
	Form     []config.KeyValue
	BodyFile string
}

// FromConfig builds a Request from a saved/edited request.
func FromConfig(r config.Request, env map[string]string, secrets interp.SecretGetter) Request {
	return Request{
		Method: r.Method, URL: r.URL, Body: r.Body,
		Headers: r.Headers, Params: r.Params, Auth: r.Auth,
		BodyMode: r.BodyMode, Form: r.Form, BodyFile: r.BodyFile,
		Env: env, Secrets: secrets,
	}
}

// ErrMissingSecret is returned when a referenced secret cannot be resolved.
var ErrMissingSecret = errors.New("unresolved secret reference (check the Secrets manager)")

func (r Request) expand(s string) (string, error) {
	out := interp.Expand(s, r.Env, r.Secrets, false)
	if strings.Contains(out, interp.MissingSecret) {
		return out, ErrMissingSecret
	}
	return out, nil
}

// applyAuth sets authentication on httpReq. Auth values are never recorded in
// Response.RawHeaders, so they are never persisted to history.
func (r Request) applyAuth(httpReq *http.Request) error {
	a := r.Auth
	switch a.Type {
	case config.AuthNone:
		return nil
	case config.AuthBearer:
		tok, err := r.expand(a.Token)
		if err != nil {
			return fmt.Errorf("bearer token: %w", err)
		}
		if strings.TrimSpace(tok) == "" {
			return fmt.Errorf("bearer token is empty")
		}
		httpReq.Header.Set("Authorization", "Bearer "+strings.TrimSpace(tok))
	case config.AuthBasic:
		user, err := r.expand(a.Username)
		if err != nil {
			return fmt.Errorf("basic auth username: %w", err)
		}
		pass, err := r.expand(a.Password)
		if err != nil {
			return fmt.Errorf("basic auth password: %w", err)
		}
		httpReq.SetBasicAuth(user, pass)
	case config.AuthAPIKey:
		k := strings.TrimSpace(a.Key)
		if k == "" {
			return fmt.Errorf("API key name is empty")
		}
		v, err := r.expand(a.Value)
		if err != nil {
			return fmt.Errorf("API key value: %w", err)
		}
		if a.In == config.APIKeyInQuery {
			q := url.Values{}
			q.Set(k, v)
			if httpReq.URL.RawQuery != "" {
				httpReq.URL.RawQuery += "&" + q.Encode()
			} else {
				httpReq.URL.RawQuery = q.Encode()
			}
		} else {
			httpReq.Header.Set(k, v)
		}
	default:
		return fmt.Errorf("unknown auth type %q", a.Type)
	}
	return nil
}

// Response is the result of executing a Request. URL and RawHeaders are
// populated even when Do returns an error, when known.
type Response struct {
	// URL is the final, interpolated URL that was requested.
	URL string
	// RawHeaders are the enabled request headers before template expansion
	// (safe to persist after redaction; never contains resolved secrets).
	RawHeaders map[string]string

	StatusCode int
	Headers    http.Header
	Body       []byte
	Truncated  bool
	Duration   time.Duration
}

// Client executes requests.
type Client struct {
	hc *http.Client
}

// New returns a Client with sensible transport defaults.
func New() *Client {
	return &Client{hc: &http.Client{
		Transport: &http.Transport{
			Proxy:                 http.ProxyFromEnvironment,
			MaxIdleConns:          10,
			IdleConnTimeout:       60 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
		},
	}}
}

// NewWithHTTPClient wraps an existing *http.Client (useful for tests).
func NewWithHTTPClient(hc *http.Client) *Client { return &Client{hc: hc} }

// BuildURL interpolates the raw URL, defaults the scheme to https and
// appends enabled query params. The user's existing query string is
// preserved verbatim (not re-sorted or re-encoded).
func BuildURL(rawURL string, params []config.KeyValue, env map[string]string, secrets interp.SecretGetter) (string, error) {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return "", fmt.Errorf("URL cannot be empty")
	}
	realURL := interp.Expand(rawURL, env, secrets, false)
	if strings.Contains(realURL, interp.MissingSecret) {
		return "", fmt.Errorf("URL: %w", ErrMissingSecret)
	}
	lower := strings.ToLower(realURL)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		realURL = "https://" + realURL
	}
	u, err := url.Parse(realURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid URL: missing host")
	}
	extra := url.Values{}
	for _, p := range params {
		if key := strings.TrimSpace(p.Key); p.Enabled && key != "" {
			v := interp.Expand(p.Value, env, secrets, false)
			if strings.Contains(v, interp.MissingSecret) {
				return "", fmt.Errorf("param %s: %w", key, ErrMissingSecret)
			}
			extra.Add(key, v)
		}
	}
	if len(extra) > 0 {
		if u.RawQuery != "" {
			u.RawQuery += "&" + extra.Encode()
		} else {
			u.RawQuery = extra.Encode()
		}
	}
	return u.String(), nil
}

// Do expands and sends req. The returned Response is never nil.
func (c *Client) Do(ctx context.Context, req Request) (*Response, error) {
	start := time.Now()
	res := &Response{RawHeaders: map[string]string{}}

	finalURL, err := BuildURL(req.URL, req.Params, req.Env, req.Secrets)
	if err != nil {
		return res, err
	}
	// res.URL is shown to the user; it never includes auth query params.
	res.URL = finalURL

	built, err := req.buildBody()
	if err != nil {
		return res, err
	}
	var body io.Reader
	if built != nil {
		body = built.reader
		if built.closer != nil {
			// Closed by the transport after sending; this guards early
			// returns before the request is handed over.
			defer built.closer.Close()
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, req.Method, finalURL, body)
	if err != nil {
		return res, err
	}
	if built != nil && built.length >= 0 {
		httpReq.ContentLength = built.length
	}
	for _, h := range req.Headers {
		key := strings.TrimSpace(h.Key)
		if !h.Enabled || key == "" {
			continue
		}
		val, err := req.expand(h.Value)
		if err != nil {
			return res, fmt.Errorf("header %s: %w", key, err)
		}
		res.RawHeaders[key] = h.Value
		httpReq.Header.Set(key, val)
	}
	if built != nil && built.contentType != "" {
		if built.forceCT || (built.defaultCTOnly && httpReq.Header.Get("Content-Type") == "") {
			httpReq.Header.Set("Content-Type", built.contentType)
		}
	}
	if err := req.applyAuth(httpReq); err != nil {
		return res, fmt.Errorf("auth: %w", err)
	}

	resp, err := c.hc.Do(httpReq)
	if err != nil {
		res.Duration = time.Since(start)
		return res, err
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, MaxBodyBytes+1))
	res.Duration = time.Since(start)
	if err != nil {
		return res, err
	}
	if len(data) > MaxBodyBytes {
		data = data[:MaxBodyBytes]
		res.Truncated = true
	}
	res.StatusCode = resp.StatusCode
	res.Headers = resp.Header
	res.Body = data
	return res, nil
}
