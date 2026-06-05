// Package apps resolves Arduino app directories and replaces an app's contents
// with a "solution" archive. The apps live under a root such as
// /home/arduino/ArduinoApps on the device running workshop.ino.
package apps

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// appNameRe allows a single safe path segment: no separators, no leading dot,
// no "."/".." — so an app name can never escape the apps root.
var appNameRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)

// Dir resolves an app name to its absolute directory under root, rejecting any
// name that isn't a single safe path segment.
func Dir(root, name string) (string, error) {
	if !appNameRe.MatchString(name) || name == "." || name == ".." || strings.Contains(name, "..") {
		return "", fmt.Errorf("invalid app name: %q", name)
	}
	return filepath.Join(root, name), nil
}

// RestoreArchive clears appDir and extracts archive into it, replacing the app
// with the archive's contents (a clean "apply this solution"). The archive type
// is detected by extension (.zip, .tar.gz/.tgz). Entries that would escape
// appDir are rejected.
func RestoreArchive(appDir, archive string) error {
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return err
	}
	switch {
	case strings.HasSuffix(archive, ".zip"):
		if err := clearDir(appDir); err != nil {
			return err
		}
		return extractZip(archive, appDir)
	case strings.HasSuffix(archive, ".tar.gz"), strings.HasSuffix(archive, ".tgz"):
		if err := clearDir(appDir); err != nil {
			return err
		}
		return extractTarGz(archive, appDir)
	default:
		return fmt.Errorf("unsupported archive type (use .zip or .tar.gz): %s", filepath.Base(archive))
	}
}

// clearDir removes everything inside dir but keeps dir itself (preserving its
// ownership and mode — important when it's owned by the arduino user).
func clearDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// safeJoin joins name onto dest and verifies the result stays within dest,
// guarding against archive path-traversal ("zip slip").
func safeJoin(dest, name string) (string, error) {
	target := filepath.Join(dest, name)
	rel, err := filepath.Rel(dest, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive entry escapes target: %q", name)
	}
	return target, nil
}

func extractZip(archive, dest string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		target, err := safeJoin(dest, f.Name)
		if err != nil {
			return err
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		err = writeFile(target, rc, f.Mode())
		rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func extractTarGz(archive, dest string) error {
	f, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		target, err := safeJoin(dest, hdr.Name)
		if err != nil {
			return err
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := writeFile(target, tr, os.FileMode(hdr.Mode)); err != nil {
				return err
			}
		}
	}
}

// writeFile creates target (and parent dirs) and copies r into it.
func writeFile(target string, r io.Reader, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	if mode == 0 {
		mode = 0o644
	}
	out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, r)
	return err
}
