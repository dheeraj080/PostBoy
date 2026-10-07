package collection

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/dheeraj080/PostBoy/internal/config"
)

func TestLoadMissing(t *testing.T) {
	cols, err := Load(t.TempDir())
	if err != nil || cols != nil {
		t.Fatalf("got %v, %v", cols, err)
	}
}

func TestRoundTripAssignsIDs(t *testing.T) {
	dir := t.TempDir()
	in := []Collection{{Name: "API", Requests: []config.Request{{Name: "List", Method: "GET", URL: "x"}}}}
	if err := Save(dir, in); err != nil {
		t.Fatal(err)
	}
	out, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].ID == "" || out[0].Requests[0].ID == "" {
		t.Fatalf("ids not assigned: %+v", out)
	}
	if out[0].Requests[0].Headers == nil {
		t.Fatal("requests not normalized")
	}
	ci, ri := Find(out, out[0].ID, out[0].Requests[0].ID)
	if ci != 0 || ri != 0 {
		t.Fatalf("Find = %d,%d", ci, ri)
	}
	if ci, ri := Find(out, "nope", "x"); ci != -1 || ri != -1 {
		t.Fatalf("Find missing = %d,%d", ci, ri)
	}
}

func TestLoadCorrupt(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, FileName), []byte("nope"), 0o600)
	if _, err := Load(dir); err == nil {
		t.Fatal("expected error")
	}
}

func TestNewIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 1000; i++ {
		id := NewID()
		if seen[id] {
			t.Fatal("duplicate id")
		}
		seen[id] = true
	}
}
