package server

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveDownloadWithinRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "01-setup"), 0o755); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(root, "01-setup", "starter.zip")
	if err := os.WriteFile(want, []byte("zip"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, err := resolveDownload(root, "01-setup/starter.zip")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveDownloadRejectsTraversal(t *testing.T) {
	root := t.TempDir()
	// A secret living next to (outside) the content root must stay unreachable.
	secret := filepath.Join(filepath.Dir(root), "secret.txt")
	if err := os.WriteFile(secret, []byte("top secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Remove(secret) })

	for _, p := range []string{"../secret.txt", "../../secret.txt", "01-setup/../../secret.txt"} {
		if _, err := resolveDownload(root, p); err == nil {
			t.Errorf("resolveDownload(%q) returned no error; traversal not blocked", p)
		}
	}
}

func TestResolveDownloadRejectsDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "assets"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := resolveDownload(root, "assets"); err == nil {
		t.Error("expected error for a directory path, got nil")
	}
}

func TestResolveDownloadRejectsMissing(t *testing.T) {
	root := t.TempDir()
	if _, err := resolveDownload(root, "nope.zip"); err == nil {
		t.Error("expected error for a missing file, got nil")
	}
}
