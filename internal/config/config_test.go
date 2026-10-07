package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingReturnsDefault(t *testing.T) {
	c, err := Load(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(c.Environments) == 0 || c.TimeoutSeconds <= 0 || c.Draft.Method != "GET" {
		t.Fatalf("unexpected default: %+v", c)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	in := Default()
	in.ActiveEnv = 1
	in.Draft.URL = "{{BASE_URL}}/x"
	in.Draft.Headers = append(in.Draft.Headers, KeyValue{Key: "X-Test", Value: "1", Enabled: true})
	in.Draft.Auth = Auth{Type: AuthBearer, Token: "{{ secret.T }}"}
	in.Origin = &Origin{CollectionID: "c", RequestID: "r"}
	in.SecretNames = []string{"TOKEN"}
	if err := Save(dir, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if out.ActiveEnv != 1 || out.SecretNames[0] != "TOKEN" || out.Origin == nil || out.Origin.RequestID != "r" {
		t.Fatalf("round trip mismatch: %+v", out)
	}
	if !SameContent(in.Draft, out.Draft) {
		t.Fatalf("draft mismatch: %+v vs %+v", in.Draft, out.Draft)
	}
	if out.Version != CurrentVersion {
		t.Fatalf("version = %d", out.Version)
	}
}

func TestLoadCorruptReturnsError(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte("{not json"), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if len(c.Environments) == 0 {
		t.Fatal("expected default config on error")
	}
}

func TestNormalize(t *testing.T) {
	c := Normalize(Config{
		Environments: []Environment{{Name: "x"}},
		ActiveEnv:    5,
		Origin:       &Origin{CollectionID: "only-collection"},
	})
	if c.ActiveEnv != 0 {
		t.Fatalf("ActiveEnv = %d", c.ActiveEnv)
	}
	if c.Environments[0].Vars == nil || c.SecretNames == nil || c.Draft.Headers == nil || c.Draft.Params == nil {
		t.Fatal("nil collections not initialized")
	}
	if c.TimeoutSeconds <= 0 {
		t.Fatal("timeout not defaulted")
	}
	if c.Origin != nil {
		t.Fatal("incomplete origin not cleared")
	}
}

// v1 configs (unversioned, global headers/params) migrate into the draft.
func TestLoadLegacyConfigMigratesToDraft(t *testing.T) {
	dir := t.TempDir()
	legacy := `{"environments":[{"name":"Dev","vars":{"A":"1"}}],"active_env":0,
		"headers":[{"key":"Accept","value":"*/*","enabled":true}],
		"params":[{"key":"page","value":"1","enabled":true}],
		"secret_names":null,"timeout_seconds":0}`
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c.Environments[0].Vars["A"] != "1" {
		t.Fatalf("env lost: %+v", c)
	}
	if len(c.Draft.Headers) != 1 || c.Draft.Headers[0].Key != "Accept" {
		t.Fatalf("headers not migrated: %+v", c.Draft.Headers)
	}
	if len(c.Draft.Params) != 1 || c.Draft.Params[0].Key != "page" {
		t.Fatalf("params not migrated: %+v", c.Draft.Params)
	}
	if c.Headers != nil || c.Params != nil {
		t.Fatal("legacy fields not cleared")
	}
	if c.Draft.Method != "GET" {
		t.Fatalf("method = %q", c.Draft.Method)
	}
}

func TestSameContent(t *testing.T) {
	a := Request{ID: "1", Name: "a", Method: "GET", URL: "x"}
	b := Request{ID: "2", Name: "b", Method: "GET", URL: "x", Headers: []KeyValue{}, Params: nil}
	if !SameContent(a, b) {
		t.Fatal("expected equal ignoring id/name/nil-vs-empty")
	}
	b.URL = "y"
	if SameContent(a, b) {
		t.Fatal("expected different")
	}
}
