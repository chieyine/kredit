package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSeedFileExists(t *testing.T) {
	root, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	// Navigate up if running from cmd/seed
	for {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			break
		}
		root = parent
	}
	seed := filepath.Join(root, "db", "seeds", "001_demo.sql")
	if info, err := os.Stat(seed); err != nil || info.Size() == 0 {
		t.Fatalf("expected seed file %s to exist and be non-empty", seed)
	}
}
