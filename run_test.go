package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dheeraj080/PostBoy/internal/collection"
	"github.com/dheeraj080/PostBoy/internal/config"
)

func TestRunUsesCollectionVariables(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/check" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer srv.Close()

	dir := t.TempDir()
	cfg := config.Default()
	cfg.ActiveEnv = 0
	if err := config.Save(dir, cfg); err != nil {
		t.Fatal(err)
	}
	cols := []collection.Collection{{
		Name: "vars",
		Variables: map[string]string{
			"base": srv.URL,
		},
		Requests: []config.Request{{Method: "GET", URL: "{{base}}/check", ExpectedStatus: 200}},
	}}
	if err := collection.Save(dir, cols); err != nil {
		t.Fatal(err)
	}
	if code := runCmd([]string{"--data-dir", dir, "vars"}); code != 0 {
		t.Fatalf("runCmd code = %d", code)
	}
}

func TestLoadCollectionVariablesFromCollectionFile(t *testing.T) {
	dir := t.TempDir()
	data := `{"version":1,"collections":[{"id":"x","name":"C","requests":[],"variables":{"a":"1"}}]}`
	if err := os.WriteFile(filepath.Join(dir, "collections.json"), []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	cols, err := collection.Load(dir)
	if err != nil || len(cols) != 1 || cols[0].Variables["a"] != "1" {
		t.Fatalf("cols=%+v err=%v", cols, err)
	}
}
