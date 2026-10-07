// Package store persists request history in SQLite with versioned schema
// migrations.
package store

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/dheeraj080/PostBoy/internal/config"

	_ "modernc.org/sqlite" // pure-Go SQLite driver
)

const (
	// DBFileName is the database file name inside the app directory.
	DBFileName = "postboy.db"
	// MaxHistoryRows is the number of history rows retained.
	MaxHistoryRows = 500
	// RedactedPlaceholder replaces sensitive header values before storage.
	RedactedPlaceholder = "[REDACTED]"
)

var sensitiveHeaderKeys = map[string]bool{
	"authorization": true, "proxy-authorization": true,
	"x-api-key": true, "api-key": true,
	"cookie": true, "set-cookie": true, "x-auth-token": true,
}

// IsSensitiveHeader reports whether a header's value must never be persisted.
func IsSensitiveHeader(key string) bool {
	return sensitiveHeaderKeys[strings.ToLower(strings.TrimSpace(key))]
}

// RedactHeaders returns a copy of headers with sensitive values replaced.
func RedactHeaders(headers map[string]string) map[string]string {
	out := make(map[string]string, len(headers))
	for k, v := range headers {
		if IsSensitiveHeader(k) {
			v = RedactedPlaceholder
		}
		out[k] = v
	}
	return out
}

// HistoryRecord is one saved request.
type HistoryRecord struct {
	ID         int
	StatusCode int
	Method     string
	URL        string
	Body       string
	CreatedAt  string
	// Headers excludes redacted entries so they can be safely re-applied.
	Headers []config.KeyValue
}

// Store wraps the SQLite database.
type Store struct {
	db *sql.DB
}

// migration upgrades the schema by one version. Migrations must be
// idempotent with respect to databases created by pre-versioned builds.
type migration func(tx *sql.Tx) error

var migrations = []migration{
	// v1: base history table.
	func(tx *sql.Tx) error {
		_, err := tx.Exec(`CREATE TABLE IF NOT EXISTS history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			method TEXT,
			url TEXT,
			headers_json TEXT,
			status_code INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`)
		return err
	},
	// v2: request body column (may already exist from pre-versioned builds).
	func(tx *sql.Tx) error {
		_, err := tx.Exec(`ALTER TABLE history ADD COLUMN body TEXT`)
		if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
			return nil
		}
		return err
	},
}

// SchemaVersion is the schema version this build migrates to.
var SchemaVersion = len(migrations)

// Open opens (creating if needed) the database in dir and migrates it.
func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}
	path := filepath.Join(dir, DBFileName)

	// Create with restrictive permissions before SQLite touches it.
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("create db file: %w", err)
	}
	f.Close()
	_ = os.Chmod(path, 0o600)

	db, err := sql.Open("sqlite", path+"?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) migrate() error {
	var current int
	if err := s.db.QueryRow(`PRAGMA user_version`).Scan(&current); err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}
	if current > len(migrations) {
		return fmt.Errorf("database schema v%d is newer than supported v%d", current, len(migrations))
	}
	for v := current; v < len(migrations); v++ {
		tx, err := s.db.Begin()
		if err != nil {
			return err
		}
		if err := migrations[v](tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migrate to v%d: %w", v+1, err)
		}
		// PRAGMA does not support bound parameters.
		if _, err := tx.Exec(fmt.Sprintf(`PRAGMA user_version = %d`, v+1)); err != nil {
			tx.Rollback()
			return fmt.Errorf("set schema version: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Close closes the database. Safe to call on a nil Store.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// SaveHistory records a request, redacting sensitive headers, and prunes old
// rows beyond MaxHistoryRows.
func (s *Store) SaveHistory(method, url, body string, headers map[string]string, statusCode int) error {
	if s == nil || s.db == nil {
		return nil
	}
	headersJSON, err := json.Marshal(RedactHeaders(headers))
	if err != nil {
		return err
	}
	if _, err := s.db.Exec(
		`INSERT INTO history (method, url, body, headers_json, status_code) VALUES (?, ?, ?, ?, ?)`,
		method, url, body, string(headersJSON), statusCode); err != nil {
		return err
	}
	_, err = s.db.Exec(
		`DELETE FROM history WHERE id NOT IN (SELECT id FROM history ORDER BY id DESC LIMIT ?)`, MaxHistoryRows)
	return err
}

// LoadHistory returns up to limit most-recent records.
func (s *Store) LoadHistory(limit int) ([]HistoryRecord, error) {
	if s == nil || s.db == nil {
		return nil, nil
	}
	rows, err := s.db.Query(`SELECT id, COALESCE(method, ''), COALESCE(url, ''), COALESCE(body, ''),
		COALESCE(headers_json, '{}'), COALESCE(status_code, 0), COALESCE(created_at, '')
		FROM history ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []HistoryRecord
	for rows.Next() {
		var r HistoryRecord
		var headersJSON string
		if err := rows.Scan(&r.ID, &r.Method, &r.URL, &r.Body, &headersJSON, &r.StatusCode, &r.CreatedAt); err != nil {
			return records, err
		}
		r.Headers = parseHeaders(headersJSON)
		records = append(records, r)
	}
	return records, rows.Err()
}

// ClearHistory deletes all history.
func (s *Store) ClearHistory() error {
	if s == nil || s.db == nil {
		return nil
	}
	_, err := s.db.Exec(`DELETE FROM history`)
	return err
}

func parseHeaders(headersJSON string) []config.KeyValue {
	var raw map[string]string
	if err := json.Unmarshal([]byte(headersJSON), &raw); err != nil {
		return nil
	}
	keys := make([]string, 0, len(raw))
	for k := range raw {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]config.KeyValue, 0, len(keys))
	for _, k := range keys {
		if raw[k] == RedactedPlaceholder {
			continue
		}
		out = append(out, config.KeyValue{Key: k, Value: raw[k], Enabled: true})
	}
	return out
}
