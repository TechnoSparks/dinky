package application

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindShell(t *testing.T) {
	// Ensure at least /bin/sh or a temp shell resolves.
	got := findShell()
	if got == "" {
		t.Fatal("findShell returned empty")
	}
	if info, err := os.Stat(got); err != nil || info.IsDir() {
		t.Fatalf("findShell returned unusable path %q: %v", got, err)
	}

	// PREFIX-based resolution takes priority when the file exists.
	dir := t.TempDir()
	sh := filepath.Join(dir, "bin", "sh")
	if err := os.MkdirAll(filepath.Dir(sh), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sh, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PREFIX", dir)
	if got := findShell(); got != sh {
		t.Errorf("findShell with PREFIX = %q, want %q", got, sh)
	}
}