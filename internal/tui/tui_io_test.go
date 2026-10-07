package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/importer"
)

const testPostman = `{
  "info": {"name": "Pets", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
  "variable": [{"key": "base", "value": "https://pets.test"}],
  "item": [{"name": "List", "request": {"method": "GET", "header": [], "url": "{{base}}/pets"}},
           {"name": "Upload", "request": {"method": "POST", "header": [], "url": "{{base}}/up", "body": {"mode": "formdata", "formdata": []}}}]
}`

func TestImportCurlIntoEditor(t *testing.T) {
	m := newTestModel(t)
	m = press(t, m, "alt+i")
	if m.modal != modalImport {
		t.Fatal("import dialog not opened")
	}
	m.importInput.SetValue(`curl -X POST https://api.test/x -H 'A: 1' -d '{"k":1}'`)
	m = press(t, m, "enter")
	if m.modal != modalNone || m.method != "POST" || m.urlInput.Value() != "https://api.test/x" || m.bodyInput.Value() != `{"k":1}` {
		t.Fatalf("not imported: modal=%d %s %s %q", m.modal, m.method, m.urlInput.Value(), m.bodyInput.Value())
	}
	if m.origin != nil {
		t.Fatal("imported curl should be an untitled draft")
	}
}

func TestImportCurlConfirmsDiscard(t *testing.T) {
	m := newTestModel(t)
	m.urlInput.SetValue("unsaved")
	m = press(t, m, "alt+i")
	m.importInput.SetValue("curl https://api.test/y")
	m = press(t, m, "enter")
	if !m.importConfirm || m.urlInput.Value() != "unsaved" {
		t.Fatal("expected confirmation before replacing unsaved work")
	}
	m = press(t, m, "enter")
	if m.urlInput.Value() != "https://api.test/y" {
		t.Fatal("second Enter did not import")
	}
}

func TestImportCurlErrorKeepsDialog(t *testing.T) {
	m := newTestModel(t)
	m = press(t, m, "alt+i")
	m.importInput.SetValue("curl -H 'A: 1'")
	m = press(t, m, "enter")
	if m.modal != modalImport || !m.statusErr {
		t.Fatal("error should keep the dialog open and report")
	}
}

func TestImportPostmanFile(t *testing.T) {
	m := newTestModel(t)
	path := filepath.Join(t.TempDir(), "pets.json")
	if err := os.WriteFile(path, []byte(testPostman), 0o600); err != nil {
		t.Fatal(err)
	}
	envsBefore := len(m.cfg.Environments)
	m = press(t, m, "alt+i")
	m.importInput.SetValue(`"` + path + `"`) // quoted, as from drag-and-drop
	m = press(t, m, "enter")
	if m.modal != modalCollections {
		t.Fatalf("expected collections view after import, modal=%d status=%q", m.modal, m.status)
	}
	if len(m.collections) != 1 || len(m.collections[0].Requests) != 2 || m.collections[0].Name != "Pets" {
		t.Fatalf("collections = %+v", m.collections)
	}
	if len(m.cfg.Environments) != envsBefore+1 || m.cfg.Environments[envsBefore].Vars["base"] != "https://pets.test" {
		t.Fatalf("environment not created: %+v", m.cfg.Environments)
	}
	if m.collections[0].Requests[1].BodyMode != config.BodyMultipart {
		t.Fatalf("multipart body mode not imported: %q %q", m.collections[0].Requests[1].BodyMode, m.status)
	}
	// Persisted.
	cols, _ := collection.Load(m.dir)
	if len(cols) != 1 {
		t.Fatal("collections not saved")
	}

	// Importing again gets a unique name.
	m.closeModal()
	m = press(t, m, "alt+i")
	m.importInput.SetValue(path)
	m = press(t, m, "enter")
	if len(m.collections) != 2 || m.collections[1].Name != "Pets (2)" {
		t.Fatalf("duplicate import name: %+v", m.collections[1].Name)
	}
}

func TestImportFileCLI(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pets.json")
	if err := os.WriteFile(path, []byte(testPostman), 0o600); err != nil {
		t.Fatal(err)
	}
	summary, err := ImportFile(dir, path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(summary, "Imported 2 requests") {
		t.Fatalf("summary = %q", summary)
	}
	cols, _ := collection.Load(dir)
	cfg, _ := config.Load(dir)
	if len(cols) != 1 || cfg.Environments[len(cfg.Environments)-1].Name != "Pets" {
		t.Fatalf("not persisted: %+v %+v", cols, cfg.Environments)
	}
}

func TestCopyAsCurl(t *testing.T) {
	var copied string
	old := copyToClipboard
	copyToClipboard = func(s string) error { copied = s; return nil }
	defer func() { copyToClipboard = old }()

	m := newTestModel(t)
	m.urlInput.SetValue("{{BASE_URL}}/users")
	m.auth.set(config.Auth{Type: config.AuthBearer, Token: "{{ secret.TOKEN }}"})
	m = press(t, m, "alt+c")
	if !strings.Contains(copied, "https://api.github.com/users") {
		t.Fatalf("env not expanded:\n%s", copied)
	}
	if !strings.Contains(copied, "{{ secret.TOKEN }}") {
		t.Fatalf("secret should stay a reference:\n%s", copied)
	}
	if m.statusErr {
		t.Fatalf("status = %q", m.status)
	}
	// The copied command re-imports to the same request.
	r, _, err := importer.ParseCurl(copied)
	if err != nil || r.URL != "https://api.github.com/users" {
		t.Fatalf("re-import: %v %+v", err, r)
	}
}

func TestExportCollection(t *testing.T) {
	m := newTestModel(t)
	m.opts.ExportDir = t.TempDir()
	m.urlInput.SetValue("api.test/x")
	if err := m.saveAs("X", -1, "My API/v1"); err != nil {
		t.Fatal(err)
	}
	m = press(t, m, "alt+o", "e")
	want := filepath.Join(m.opts.ExportDir, "My_API_v1.postman_collection.json")
	if !strings.Contains(m.status, want) {
		t.Fatalf("status = %q", m.status)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	res, err := importer.ImportPostman(data)
	if err != nil || len(res.Collection.Requests) != 1 || res.Collection.Requests[0].Name != "X" {
		t.Fatalf("exported file does not re-import: %v %+v", err, res.Collection)
	}
}

func TestEnvEditorVariables(t *testing.T) {
	m := newTestModel(t)
	m = press(t, m, "alt+v")
	if m.modal != modalEnv || m.envView != m.envIndex {
		t.Fatal("env editor not opened on active env")
	}
	m = press(t, m, "n")
	m = typeText(t, m, "1bad=x")
	m = press(t, m, "enter")
	if !m.envVars.editing || m.envVars.err == "" {
		t.Fatal("invalid name accepted")
	}
	m.envVars.input.SetValue("TOKEN_URL={{BASE_URL}}/token")
	m = press(t, m, "enter")
	if m.envVars.editing {
		t.Fatalf("valid edit rejected: %s", m.envVars.err)
	}
	if m.env().Vars["TOKEN_URL"] != "{{BASE_URL}}/token" {
		t.Fatalf("var not stored: %v", m.env().Vars)
	}
	// Duplicate rejected.
	m = press(t, m, "n")
	m.envVars.input.SetValue("TOKEN_URL=other")
	m = press(t, m, "enter")
	if m.envVars.err == "" {
		t.Fatal("duplicate accepted")
	}
	m = press(t, m, "esc")
	// Persisted.
	cfg, _ := config.Load(m.dir)
	if cfg.Environments[cfg.ActiveEnv].Vars["TOKEN_URL"] == "" {
		t.Fatal("var not persisted")
	}
}

func TestEnvEditorManageEnvironments(t *testing.T) {
	m := newTestModel(t)
	n := len(m.cfg.Environments)
	m = press(t, m, "alt+v", "N")
	m = typeText(t, m, "Local")
	m = press(t, m, "enter")
	if len(m.cfg.Environments) != n+1 || m.cfg.Environments[m.envView].Name != "Local" {
		t.Fatal("environment not created / not viewed")
	}
	// Duplicate names rejected.
	m = press(t, m, "N")
	m = typeText(t, m, "local")
	m = press(t, m, "enter")
	if len(m.cfg.Environments) != n+1 || m.envInputMode == envInputNone {
		t.Fatal("duplicate environment accepted")
	}
	m = press(t, m, "esc")

	m = press(t, m, "a")
	if m.env().Name != "Local" {
		t.Fatal("'a' did not activate viewed env")
	}
	m = press(t, m, "R")
	m.envInput.SetValue("Localhost")
	m = press(t, m, "enter")
	if m.env().Name != "Localhost" {
		t.Fatal("rename failed")
	}
	m = press(t, m, "D")
	if len(m.cfg.Environments) != n+1 {
		t.Fatal("deleted without confirmation")
	}
	m = press(t, m, "D")
	if len(m.cfg.Environments) != n || m.envIndex != 0 {
		t.Fatalf("delete failed: envs=%d active=%d", len(m.cfg.Environments), m.envIndex)
	}

	// Cannot delete the last one.
	m.cfg.Environments = m.cfg.Environments[:1]
	m.envIndex = 0
	m.viewEnv(0)
	m = press(t, m, "D", "D")
	if len(m.cfg.Environments) != 1 {
		t.Fatal("deleted the only environment")
	}
}

func TestEnvEditorSwitchView(t *testing.T) {
	m := newTestModel(t)
	m = press(t, m, "alt+v")
	start := m.envView
	m = press(t, m, "right")
	if m.envView == start || m.envIndex != start {
		t.Fatal("arrow should change the viewed env without activating it")
	}
	if len(m.envVars.items) != len(m.cfg.Environments[m.envView].Vars) {
		t.Fatal("var list not reloaded")
	}
}
