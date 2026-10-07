package store

import (
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestSaveAndLoadHistoryRedacts(t *testing.T) {
	s := openTemp(t)
	err := s.SaveHistory("POST", "{{BASE}}/x", `{"a":1}`, map[string]string{
		"Authorization": "Bearer secret",
		"Accept":        "application/json",
	}, 201)
	if err != nil {
		t.Fatal(err)
	}
	recs, err := s.LoadHistory(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(recs) != 1 {
		t.Fatalf("got %d records", len(recs))
	}
	r := recs[0]
	if r.Method != "POST" || r.URL != "{{BASE}}/x" || r.Body != `{"a":1}` || r.StatusCode != 201 {
		t.Fatalf("bad record: %+v", r)
	}
	if len(r.Headers) != 1 || r.Headers[0].Key != "Accept" {
		t.Fatalf("redacted header leaked or others lost: %+v", r.Headers)
	}

	var raw string
	if err := s.db.QueryRow(`SELECT headers_json FROM history`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(raw, "Bearer secret") {
		t.Fatalf("secret stored in database: %s", raw)
	}
}

func TestHistoryPruned(t *testing.T) {
	s := openTemp(t)
	for i := 0; i < MaxHistoryRows+5; i++ {
		if err := s.SaveHistory("GET", "u", "", nil, 200); err != nil {
			t.Fatal(err)
		}
	}
	var n int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM history`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != MaxHistoryRows {
		t.Fatalf("rows = %d, want %d", n, MaxHistoryRows)
	}
}

func TestClearHistory(t *testing.T) {
	s := openTemp(t)
	_ = s.SaveHistory("GET", "u", "", nil, 200)
	if err := s.ClearHistory(); err != nil {
		t.Fatal(err)
	}
	recs, _ := s.LoadHistory(10)
	if len(recs) != 0 {
		t.Fatalf("history not cleared: %d", len(recs))
	}
}

// Databases created by pre-versioned builds (user_version 0, body column
// already present) must migrate cleanly.
func TestMigrateLegacyDatabase(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite", filepath.Join(dir, DBFileName))
	if err != nil {
		t.Fatal(err)
	}
	for _, q := range []string{
		`CREATE TABLE history (id INTEGER PRIMARY KEY AUTOINCREMENT, method TEXT, url TEXT,
			headers_json TEXT, status_code INTEGER, created_at DATETIME DEFAULT CURRENT_TIMESTAMP)`,
		`ALTER TABLE history ADD COLUMN body TEXT`,
		`INSERT INTO history (method, url, headers_json, status_code, body) VALUES ('GET', 'old', '{}', 200, '')`,
	} {
		if _, err := db.Exec(q); err != nil {
			t.Fatal(err)
		}
	}
	db.Close()

	s, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	var v int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&v); err != nil {
		t.Fatal(err)
	}
	if v != SchemaVersion {
		t.Fatalf("user_version = %d, want %d", v, SchemaVersion)
	}
	recs, err := s.LoadHistory(10)
	if err != nil || len(recs) != 1 || recs[0].URL != "old" {
		t.Fatalf("legacy data lost: %v %+v", err, recs)
	}
}

func TestNilStoreIsSafe(t *testing.T) {
	var s *Store
	if err := s.SaveHistory("GET", "u", "", nil, 200); err != nil {
		t.Fatal(err)
	}
	if _, err := s.LoadHistory(1); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
}
