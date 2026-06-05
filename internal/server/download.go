package server

import (
	"errors"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// errInvalidPath marks a content-relative reference that can't be resolved
// (empty, external, or escaping the content root).
var errInvalidPath = errors.New("invalid content path")

// resolveDownload maps a content-root-relative URL path to an absolute file
// path, guaranteeing the result stays inside root and points at a regular file.
// It is the single chokepoint protecting against path-traversal in /dl/.
func resolveDownload(root, reqPath string) (string, error) {
	// Anchor at "/" so path.Clean collapses any ".." without escaping above it.
	clean := path.Clean("/" + reqPath)
	rel := strings.TrimPrefix(clean, "/")
	abs := filepath.Join(root, filepath.FromSlash(rel))

	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	fileAbs, err := filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	within, err := filepath.Rel(rootAbs, fileAbs)
	if err != nil || within == ".." || strings.HasPrefix(within, ".."+string(filepath.Separator)) {
		return "", errors.New("path escapes content root")
	}

	info, err := os.Stat(fileAbs)
	if err != nil {
		return "", err
	}
	if info.IsDir() {
		return "", errors.New("not a file")
	}
	return fileAbs, nil
}
