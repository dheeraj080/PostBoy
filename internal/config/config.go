// Package config defines PostBoy's persisted configuration schema (config,
// requests, auth) and handles loading/saving it atomically.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dheeraj080/PostBoy/internal/fsutil"
)

const (
	// FileName is the config file name inside the app directory.
	FileName = "config.json"
	// CurrentVersion is the config schema version written by this build.
	//   v1: global headers/params
	//   v2: per-request draft (headers/params moved into Draft)
	CurrentVersion = 2
	// DefaultTimeout is the request timeout used when none is configured.
	DefaultTimeout = 15 * time.Second
)

// Environment is a named set of template variables.
type Environment struct {
	Name string            `json:"name"`
	Vars map[string]string `json:"vars"`
}

// KeyValue is a toggleable key/value pair used for headers, query params and
// form fields.
type KeyValue struct {
	Key     string `json:"key"`
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
	// File marks a multipart form field whose Value is a file path.
	File bool `json:"file,omitempty"`
}

// Config is the on-disk configuration.
type Config struct {
	Version        int           `json:"version"`
	Environments   []Environment `json:"environments"`
	ActiveEnv      int           `json:"active_env"`
	SecretNames    []string      `json:"secret_names"`
	TimeoutSeconds int           `json:"timeout_seconds"`

	// Draft is the request currently open in the editor.
	Draft Request `json:"draft"`
	// Origin links Draft to a saved request, if any.
	Origin *Origin `json:"origin,omitempty"`

	// Deprecated v1 fields, migrated into Draft on load.
	Headers []KeyValue `json:"headers,omitempty"`
	Params  []KeyValue `json:"params,omitempty"`
}

// Default returns the configuration used on first run.
func Default() Config {
	return Config{
		Version: CurrentVersion,
		Environments: []Environment{
			{Name: "Development", Vars: map[string]string{"BASE_URL": "http://localhost:8080"}},
			{Name: "Staging", Vars: map[string]string{"BASE_URL": "https://staging.api.github.com"}},
			{Name: "Production", Vars: map[string]string{"BASE_URL": "https://api.github.com"}},
		},
		ActiveEnv:      2,
		SecretNames:    []string{},
		TimeoutSeconds: int(DefaultTimeout / time.Second),
		Draft:          NewRequest(),
	}
}

// Normalize repairs a possibly partial or invalid config so the rest of the
// app can rely on its invariants, and migrates older schema versions.
func Normalize(c Config) Config {
	d := Default()
	if len(c.Environments) == 0 {
		c.Environments = d.Environments
		c.ActiveEnv = d.ActiveEnv
	}
	for i := range c.Environments {
		if c.Environments[i].Vars == nil {
			c.Environments[i].Vars = map[string]string{}
		}
	}
	if c.ActiveEnv < 0 || c.ActiveEnv >= len(c.Environments) {
		c.ActiveEnv = 0
	}
	if c.SecretNames == nil {
		c.SecretNames = []string{}
	}
	if c.TimeoutSeconds <= 0 {
		c.TimeoutSeconds = d.TimeoutSeconds
	}

	// v1 -> v2: global headers/params become the draft's.
	if c.Version < 2 && c.Draft.Method == "" && c.Draft.URL == "" {
		c.Draft = NewRequest()
		if c.Headers != nil {
			c.Draft.Headers = c.Headers
		}
		if c.Params != nil {
			c.Draft.Params = c.Params
		}
	}
	c.Headers, c.Params = nil, nil
	c.Draft = c.Draft.Normalize()
	if c.Origin != nil && (c.Origin.CollectionID == "" || c.Origin.RequestID == "") {
		c.Origin = nil
	}
	c.Version = CurrentVersion
	return c
}

// Timeout returns the configured request timeout.
func (c Config) Timeout() time.Duration {
	return time.Duration(c.TimeoutSeconds) * time.Second
}

// Dir returns the default PostBoy data directory.
func Dir() string {
	base, err := os.UserConfigDir()
	if err != nil || base == "" {
		base = "."
	}
	return filepath.Join(base, "postboy")
}

// Load reads the config from dir. A missing file yields Default() and no
// error; a corrupt file yields Default() and an error.
func Load(dir string) (Config, error) {
	data, err := os.ReadFile(filepath.Join(dir, FileName))
	if errors.Is(err, os.ErrNotExist) {
		return Default(), nil
	}
	if err != nil {
		return Default(), err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Default(), fmt.Errorf("parse %s: %w", FileName, err)
	}
	if c.Version > CurrentVersion {
		return Normalize(c), fmt.Errorf("%s was written by a newer PostBoy (schema v%d)", FileName, c.Version)
	}
	return Normalize(c), nil
}

// Save writes the config atomically with 0600 permissions.
func Save(dir string, c Config) error {
	c.Version = CurrentVersion
	c.Headers, c.Params = nil, nil
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return fsutil.WriteFileAtomic(dir, FileName, data)
}
