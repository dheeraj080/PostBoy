// Package secrets stores secret values in the OS keychain, falling back to
// an in-memory store when no keychain is available.
package secrets

import (
	"errors"
	"sync"

	"github.com/zalando/go-keyring"
)

const serviceName = "postboy"

// ErrNotFound is returned when a secret does not exist.
var ErrNotFound = errors.New("secret not found")

// Store reads and writes secrets.
type Store struct {
	mu       sync.Mutex
	mem      map[string]string
	inMemory bool
}

// New returns a Store backed by the OS keychain if it is usable, otherwise an
// in-memory store (see Persistent).
func New() *Store {
	s := &Store{mem: map[string]string{}}
	if err := keyring.Set(serviceName, "__test_ping__", "pong"); err != nil {
		s.inMemory = true
	} else {
		_ = keyring.Delete(serviceName, "__test_ping__")
	}
	return s
}

// NewMemory returns a non-persistent store, useful for tests.
func NewMemory(initial map[string]string) *Store {
	s := &Store{mem: map[string]string{}, inMemory: true}
	for k, v := range initial {
		s.mem[k] = v
	}
	return s
}

// Persistent reports whether secrets survive process exit.
func (s *Store) Persistent() bool { return !s.inMemory }

// Set stores a secret.
func (s *Store) Set(name, value string) error {
	if s.inMemory {
		s.mu.Lock()
		s.mem[name] = value
		s.mu.Unlock()
		return nil
	}
	return keyring.Set(serviceName, name, value)
}

// Get returns a secret or ErrNotFound.
func (s *Store) Get(name string) (string, error) {
	if s.inMemory {
		s.mu.Lock()
		defer s.mu.Unlock()
		if v, ok := s.mem[name]; ok {
			return v, nil
		}
		return "", ErrNotFound
	}
	v, err := keyring.Get(serviceName, name)
	if errors.Is(err, keyring.ErrNotFound) {
		return "", ErrNotFound
	}
	return v, err
}

// Delete removes a secret. Deleting a missing secret is not an error.
func (s *Store) Delete(name string) error {
	if s.inMemory {
		s.mu.Lock()
		delete(s.mem, name)
		s.mu.Unlock()
		return nil
	}
	err := keyring.Delete(serviceName, name)
	if errors.Is(err, keyring.ErrNotFound) {
		return nil
	}
	return err
}
