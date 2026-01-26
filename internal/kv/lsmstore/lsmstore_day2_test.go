//go:build day2

package lsmstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestLSMStore_PersistAcrossRestart(t *testing.T) {
	ctx := context.Background()
	dir := filepath.Join(t.TempDir(), "db")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}

	s, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if err := s.Put(ctx, []byte("a"), []byte("1")); err != nil {
		t.Fatalf("Put: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	s2, err := Open(Options{Dir: dir})
	if err != nil {
		t.Fatalf("Open2: %v", err)
	}
	defer s2.Close()

	got, err := s2.Get(ctx, []byte("a"))
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if string(got) != "1" {
		t.Fatalf("value mismatch: got=%q want=%q", string(got), "1")
	}
}
