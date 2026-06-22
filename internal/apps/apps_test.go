package apps

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"
)

func TestDir(t *testing.T) {
	root := "/apps"
	ok := []string{"blink", "my-app", "blink_1.0", "App2"}
	for _, n := range ok {
		got, err := Dir(root, n)
		if err != nil {
			t.Errorf("Dir(%q) unexpected error: %v", n, err)
		}
		if want := filepath.Join(root, n); got != want {
			t.Errorf("Dir(%q) = %q, want %q", n, got, want)
		}
	}
	bad := []string{
		"", ".", "..", "a/b", "../escape", "/abs", ".hidden", "a/../b",
		"/etc", "/etc/passwd", "../../etc", "etc/..", `C:\Windows`,
	}
	for _, n := range bad {
		if _, err := Dir(root, n); err == nil {
			t.Errorf("Dir(%q) should have errored", n)
		}
	}
}

// makeZip writes a .zip with the given path→content entries.
func makeZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	zw := zip.NewWriter(f)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
}

func makeTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	for name, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertRestored(t *testing.T, appDir string) {
	t.Helper()
	// Pre-existing stale file is gone (wipe-then-extract).
	if _, err := os.Stat(filepath.Join(appDir, "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("stale.txt should have been removed by restore")
	}
	if got, _ := os.ReadFile(filepath.Join(appDir, "blink.ino")); string(got) != "new code\n" {
		t.Errorf("blink.ino = %q, want %q", got, "new code\n")
	}
	if got, _ := os.ReadFile(filepath.Join(appDir, "lib", "helper.h")); string(got) != "header\n" {
		t.Errorf("lib/helper.h = %q, want %q", got, "header\n")
	}
}

func seedStale(t *testing.T, appDir string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(appDir, "stale.txt"), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestRestoreArchiveKeepsFolder proves the app folder is emptied in place, not
// removed and recreated: its identity (inode) and mode must survive a restore.
func TestRestoreArchiveKeepsFolder(t *testing.T) {
	appDir := t.TempDir()
	seedStale(t, appDir)
	if err := os.Chmod(appDir, 0o700); err != nil { // a non-default mode
		t.Fatal(err)
	}
	before, err := os.Stat(appDir)
	if err != nil {
		t.Fatal(err)
	}
	beforeIno := inode(t, before)

	archive := filepath.Join(t.TempDir(), "snap.zip")
	makeZip(t, archive, map[string]string{"blink.ino": "new code\n"})
	if err := RestoreArchive(appDir, archive); err != nil {
		t.Fatalf("RestoreArchive error: %v", err)
	}

	after, err := os.Stat(appDir)
	if err != nil {
		t.Fatalf("app folder is gone after restore: %v", err)
	}
	if inode(t, after) != beforeIno {
		t.Errorf("app folder was recreated (inode changed) — it should be emptied in place")
	}
	if after.Mode().Perm() != 0o700 {
		t.Errorf("app folder mode = %o, want 0700 preserved (a recreated dir would reset it)", after.Mode().Perm())
	}
	// Contents were still replaced.
	if _, err := os.Stat(filepath.Join(appDir, "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("stale.txt should have been removed")
	}
	if got, _ := os.ReadFile(filepath.Join(appDir, "blink.ino")); string(got) != "new code\n" {
		t.Errorf("blink.ino = %q, want %q", got, "new code\n")
	}
}

func TestRestoreArchiveZip(t *testing.T) {
	appDir := t.TempDir()
	seedStale(t, appDir)
	archive := filepath.Join(t.TempDir(), "snap.zip")
	makeZip(t, archive, map[string]string{"blink.ino": "new code\n", "lib/helper.h": "header\n"})

	if err := RestoreArchive(appDir, archive); err != nil {
		t.Fatalf("RestoreArchive error: %v", err)
	}
	assertRestored(t, appDir)
}

func TestRestoreArchiveTarGz(t *testing.T) {
	appDir := t.TempDir()
	seedStale(t, appDir)
	archive := filepath.Join(t.TempDir(), "snap.tar.gz")
	makeTarGz(t, archive, map[string]string{"blink.ino": "new code\n", "lib/helper.h": "header\n"})

	if err := RestoreArchive(appDir, archive); err != nil {
		t.Fatalf("RestoreArchive error: %v", err)
	}
	assertRestored(t, appDir)
}

// TestRestoreArchiveStripsWrapperDir proves an archive whose entries all live
// under one top-level folder is unwrapped: the files land directly in appDir,
// not in appDir/<wrapper>.
func TestRestoreArchiveStripsWrapperDir(t *testing.T) {
	cases := []struct {
		name string
		make func(*testing.T, string, map[string]string)
		ext  string
	}{
		{"zip", makeZip, ".zip"},
		{"tar.gz", makeTarGz, ".tar.gz"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			appDir := t.TempDir()
			seedStale(t, appDir)
			archive := filepath.Join(t.TempDir(), "snap"+tc.ext)
			tc.make(t, archive, map[string]string{
				"rss-reader/blink.ino":    "new code\n",
				"rss-reader/lib/helper.h": "header\n",
			})
			if err := RestoreArchive(appDir, archive); err != nil {
				t.Fatalf("RestoreArchive error: %v", err)
			}
			assertRestored(t, appDir)
			if _, err := os.Stat(filepath.Join(appDir, "rss-reader")); !os.IsNotExist(err) {
				t.Errorf("wrapper dir rss-reader/ should be stripped, not nested under appDir")
			}
		})
	}
}

// TestRestoreArchiveKeepsMultipleTopLevel proves an archive with more than one
// top-level entry is left as-is (no wrapper to strip).
func TestRestoreArchiveKeepsMultipleTopLevel(t *testing.T) {
	appDir := t.TempDir()
	archive := filepath.Join(t.TempDir(), "snap.zip")
	makeZip(t, archive, map[string]string{"a/x.txt": "1\n", "b/y.txt": "2\n"})
	if err := RestoreArchive(appDir, archive); err != nil {
		t.Fatalf("RestoreArchive error: %v", err)
	}
	for _, p := range []string{"a/x.txt", "b/y.txt"} {
		if _, err := os.Stat(filepath.Join(appDir, filepath.FromSlash(p))); err != nil {
			t.Errorf("expected %s to be extracted at root: %v", p, err)
		}
	}
}

// TestRestoreArchiveRejectsEscapeUnderWrapper proves stripping the wrapper can't
// be abused to escape: safeJoin still validates every stripped path.
func TestRestoreArchiveRejectsEscapeUnderWrapper(t *testing.T) {
	appDir := t.TempDir()
	archive := filepath.Join(t.TempDir(), "evil.zip")
	makeZip(t, archive, map[string]string{
		"app/keep.txt":         "ok\n",
		"app/../../escape.txt": "pwned\n",
	})
	if err := RestoreArchive(appDir, archive); err == nil {
		t.Fatal("RestoreArchive should reject an entry that escapes after wrapper stripping")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(appDir), "escape.txt")); !os.IsNotExist(err) {
		t.Errorf("escape wrote a file outside the app dir")
	}
}

func TestRestoreArchiveRejectsZipSlip(t *testing.T) {
	appDir := t.TempDir()
	archive := filepath.Join(t.TempDir(), "evil.zip")
	makeZip(t, archive, map[string]string{"../escape.txt": "pwned\n"})

	if err := RestoreArchive(appDir, archive); err == nil {
		t.Fatal("RestoreArchive should reject a zip-slip entry")
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(appDir), "escape.txt")); !os.IsNotExist(err) {
		t.Errorf("zip-slip wrote a file outside the app dir")
	}
}

func TestRestoreArchiveUnsupportedExtension(t *testing.T) {
	appDir := t.TempDir()
	archive := filepath.Join(t.TempDir(), "snap.rar")
	if err := os.WriteFile(archive, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := RestoreArchive(appDir, archive); err == nil {
		t.Error("RestoreArchive should reject an unsupported archive type")
	}
}
