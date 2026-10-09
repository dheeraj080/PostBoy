package tui

import (
	"net/http"
	"time"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/httpclient"
	"github.com/dheeraj080/PostBoy/internal/secrets"
)

// demoRequestBody is the request body shown by --demo.
const demoRequestBody = `{
  "title": "CLI headless test runner fails in CI",
  "body": "Steps to reproduce:\n1. Run postboy run col.json in a container",
  "assignee": "{{ ASSIGNEE }}",
  "labels": ["bug", "terminal"],
  "milestone": null
}`

// demoResponseBody is the canned 201 response shown by --demo.
const demoResponseBody = `{"id":84019284,"number":42,"state":"open","title":"CLI headless test runner fails in CI","user":{"login":"dheeraj080","id":102938},"labels":[{"name":"bug","color":"d73a4a"},{"name":"terminal","color":"0052cc"}],"assignee":{"login":"dk","id":55501},"created_at":"2026-10-09T09:14:22Z"}`

// applyDemo loads the canned request/response pair so the UI can be viewed
// (and screenshotted, and golden-tested) without any config, keychain,
// database or network. It never touches the filesystem.
func (m *Model) applyDemo() {
	m.cfg = config.Default()
	m.cfg.ActiveEnv = 2 // Production
	m.envIndex = m.cfg.ActiveEnv
	m.secrets = secrets.NewMemory(nil)
	m.store = nil
	m.ready = true
	m.status = ""

	req := config.Request{
		Name:     "Create issue",
		Method:   http.MethodPost,
		URL:      "https://api.github.com/repos/owner/api/issues",
		Body:     demoRequestBody,
		BodyMode: config.BodyRaw,
		Headers: []config.KeyValue{
			{Key: "User-Agent", Value: "PostBoy", Enabled: true},
			{Key: "Content-Type", Value: "application/json", Enabled: true},
			{Key: "Accept", Value: "application/vnd.github+json", Enabled: true},
			{Key: "Authorization", Value: "Bearer {{ secret.GITHUB_TOKEN }}", Enabled: true},
		},
		Params: []config.KeyValue{},
		Form:   []config.KeyValue{},
	}
	m.loadRequest(req, nil, req)
	// loadRequest clears the name for unlinked drafts; restore it and mark
	// the draft clean so the title shows without the dirty dot.
	m.requestName = req.Name
	m.savedSnapshot = m.currentRequest()

	m.reqTab = reqTabBody
	m.resTab = resTabBody
	m.setFocus(focusURL)

	res := &httpclient.Response{
		URL:        req.URL,
		StatusCode: http.StatusCreated,
		Headers: http.Header{
			"Content-Type":          []string{"application/json; charset=utf-8"},
			"X-RateLimit-Remaining": []string{"4999"},
			"Location":              []string{"/repos/owner/api/issues/42"},
		},
		Body:     []byte(demoResponseBody),
		Duration: 142 * time.Millisecond,
	}
	m.handleResponse(responseMsg{id: 0, res: res})
	// Show the size from the spec's target data (1.4 KB) rather than the
	// shortened canned body.
	m.respSize = 1434
	m.refreshResponse(true)
	m.reqID = 1 // any later interaction starts at id >= 1; 0 marks "not in flight"
}
