// Package collection stores named groups of saved requests in
// collections.json (human-readable and VCS-friendly).
package collection

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/dheeraj080/PostBoy/internal/config"
	"github.com/dheeraj080/PostBoy/internal/fsutil"
)

const (
	// FileName is the collections file name inside the app directory.
	FileName = "collections.json"
	// CurrentVersion is the file schema version.
	CurrentVersion = 1
)

// Collection is a named group of requests.
type Collection struct {
	ID       string           `json:"id"`
	Name     string           `json:"name"`
	Requests []config.Request `json:"requests"`
	// Variables are collection-local variables. They overlay the active
	// environment when running/opening requests from this collection.
	Variables map[string]string `json:"variables,omitempty"`
}

type file struct {
	Version     int          `json:"version"`
	Collections []Collection `json:"collections"`
}

// NewID returns a random identifier.
func NewID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b[:])
}

// Load reads collections from dir. A missing file yields no collections.
func Load(dir string) ([]Collection, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var f file
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w", FileName, err)
	}
	if f.Version > CurrentVersion {
		return nil, fmt.Errorf("%s was written by a newer PostBoy (schema v%d)", FileName, f.Version)
	}
	return normalize(f.Collections), nil
}

// Save writes collections to dir atomically.
func Save(dir string, cols []Collection) error {
	data, err := json.MarshalIndent(file{Version: CurrentVersion, Collections: normalize(cols)}, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(dir, FileName, data)
}

// normalize ensures IDs exist and slices are non-nil.
func normalize(cols []Collection) []Collection {
	out := make([]Collection, len(cols))
	for i, c := range cols {
		if c.ID == "" {
			c.ID = NewID()
		}
		reqs := make([]config.Request, len(c.Requests))
		for j, r := range c.Requests {
			if r.ID == "" {
				r.ID = NewID()
			}
			reqs[j] = r.Normalize()
		}
		c.Requests = reqs
		if c.Variables == nil {
			c.Variables = map[string]string{}
		}
		out[i] = c
	}
	return out
}

// Find returns the indices of a collection and request by ID, or -1.
func Find(cols []Collection, colID, reqID string) (ci, ri int) {
	for i, c := range cols {
		if c.ID != colID {
			continue
		}
		for j, r := range c.Requests {
			if r.ID == reqID {
				return i, j
			}
		}
		return i, -1
	}
	return -1, -1
}
