package secrets

import (
	"errors"
	"testing"
)

func TestMemoryStore(t *testing.T) {
	s := NewMemory(map[string]string{"A": "1"})
	if s.Persistent() {
		t.Fatal("memory store reported persistent")
	}
	if v, err := s.Get("A"); err != nil || v != "1" {
		t.Fatalf("Get(A) = %q, %v", v, err)
	}
	if err := s.Set("B", "2"); err != nil {
		t.Fatal(err)
	}
	if v, _ := s.Get("B"); v != "2" {
		t.Fatalf("Get(B) = %q", v)
	}
	if err := s.Delete("B"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Get("B"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := s.Delete("missing"); err != nil {
		t.Fatalf("deleting missing secret errored: %v", err)
	}
}
