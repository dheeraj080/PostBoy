package config

import (
	"encoding/json"
	"net/http"
)

// AuthType selects how a request is authenticated.
type AuthType string

// Supported auth types.
const (
	AuthNone   AuthType = ""
	AuthBearer AuthType = "bearer"
	AuthBasic  AuthType = "basic"
	AuthAPIKey AuthType = "apikey"
)

// AuthTypes lists auth types in UI cycle order.
var AuthTypes = []AuthType{AuthNone, AuthBearer, AuthBasic, AuthAPIKey}

// Label returns a human-readable name.
func (t AuthType) Label() string {
	switch t {
	case AuthBearer:
		return "Bearer Token"
	case AuthBasic:
		return "Basic Auth"
	case AuthAPIKey:
		return "API Key"
	default:
		return "No Auth"
	}
}

// API key placement.
const (
	APIKeyInHeader = "header"
	APIKeyInQuery  = "query"
)

// Auth holds request authentication settings. Values may contain
// templates such as {{ secret.TOKEN }}.
type Auth struct {
	Type     AuthType `json:"type,omitempty"`
	Token    string   `json:"token,omitempty"`
	Username string   `json:"username,omitempty"`
	Password string   `json:"password,omitempty"`
	Key      string   `json:"key,omitempty"`
	Value    string   `json:"value,omitempty"`
	In       string   `json:"in,omitempty"` // APIKeyInHeader (default) or APIKeyInQuery
}

// BodyMode selects how the request body is built.
type BodyMode string

// Supported body modes.
const (
	BodyRaw        BodyMode = ""           // Body as typed (JSON, text, XML…)
	BodyURLEncoded BodyMode = "urlencoded" // Form fields, application/x-www-form-urlencoded
	BodyMultipart  BodyMode = "multipart"  // Form fields and files, multipart/form-data
	BodyFile       BodyMode = "file"       // Contents of BodyFile
)

// BodyModes lists body modes in UI cycle order.
var BodyModes = []BodyMode{BodyRaw, BodyURLEncoded, BodyMultipart, BodyFile}

// Label returns a human-readable name.
func (b BodyMode) Label() string {
	switch b {
	case BodyURLEncoded:
		return "Form URL-Encoded"
	case BodyMultipart:
		return "Multipart Form"
	case BodyFile:
		return "Binary File"
	default:
		return "Raw"
	}
}

// Request is an editable/saved HTTP request.
type Request struct {
	ID      string     `json:"id,omitempty"`
	Name    string     `json:"name,omitempty"`
	Method  string     `json:"method"`
	URL     string     `json:"url"`
	Body    string     `json:"body,omitempty"`
	Headers []KeyValue `json:"headers"`
	Params  []KeyValue `json:"params"`
	Auth    Auth       `json:"auth"`

	BodyMode BodyMode `json:"body_mode,omitempty"`
	// Form holds fields for BodyURLEncoded and BodyMultipart. For multipart,
	// entries with File set have a file path as Value.
	Form []KeyValue `json:"form,omitempty"`
	// BodyFile is the file path sent for BodyFile.
	BodyFile       string `json:"body_file,omitempty"`
	TimeoutSeconds int    `json:"timeout_seconds,omitempty"`

	// Optional minimal assertion for the headless runner.
	ExpectedStatus int `json:"expected_status,omitempty"`
}

// Origin links the working draft to a saved request in a collection.
type Origin struct {
	CollectionID string `json:"collection_id"`
	RequestID    string `json:"request_id"`
}

// DefaultHeaders are applied to new requests.
func DefaultHeaders() []KeyValue {
	return []KeyValue{
		{Key: "User-Agent", Value: "PostBoy", Enabled: true},
		{Key: "Content-Type", Value: "application/json", Enabled: true},
	}
}

// NewRequest returns a blank request with default headers.
func NewRequest() Request {
	return Request{
		Method: http.MethodGet, Headers: DefaultHeaders(), Params: []KeyValue{},
		BodyMode: BodyRaw, Form: []KeyValue{},
	}
}

// Normalize fills nil slices and a default method.
func (r Request) Normalize() Request {
	if r.Method == "" {
		r.Method = http.MethodGet
	}
	if r.Headers == nil {
		r.Headers = []KeyValue{}
	}
	if r.Params == nil {
		r.Params = []KeyValue{}
	}
	if r.Form == nil {
		r.Form = []KeyValue{}
	}
	if r.BodyMode == "" {
		r.BodyMode = BodyRaw
	}
	if r.BodyMode != BodyRaw && r.BodyMode != BodyURLEncoded && r.BodyMode != BodyMultipart && r.BodyMode != BodyFile {
		r.BodyMode = BodyRaw
	}
	return r
}

// SameContent reports whether two requests are equivalent, ignoring
// identity fields (ID, Name).
func SameContent(a, b Request) bool {
	a, b = a.Normalize(), b.Normalize()
	a.ID, a.Name, b.ID, b.Name = "", "", "", ""
	ja, err1 := json.Marshal(a)
	jb, err2 := json.Marshal(b)
	return err1 == nil && err2 == nil && string(ja) == string(jb)
}
