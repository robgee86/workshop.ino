package content

import (
	"bytes"
	"strings"

	"gopkg.in/yaml.v3"
)

// Attachment is a downloadable file referenced by a step's frontmatter.
type Attachment struct {
	Path        string `yaml:"path"`
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
}

// Link is an external reference rendered in the step's "References" section.
type Link struct {
	URL         string `yaml:"url"`
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
}

// Frontmatter holds the optional YAML metadata at the top of a step document.
type Frontmatter struct {
	Title       string       `yaml:"title"`
	Summary     string       `yaml:"summary"`
	Attachments []Attachment `yaml:"attachments"`
	Patches     []Attachment `yaml:"patches"`
	Links       []Link       `yaml:"links"`
}

// SplitFrontmatter separates an optional leading YAML frontmatter block
// (delimited by "---" lines) from the Markdown body. When no frontmatter is
// present the returned Frontmatter is zero and body equals the input.
func SplitFrontmatter(source []byte) (Frontmatter, []byte, error) {
	var fm Frontmatter

	nl := bytes.IndexByte(source, '\n')
	if nl < 0 || strings.TrimRight(string(source[:nl]), "\r") != "---" {
		return fm, source, nil
	}
	rest := source[nl+1:]

	yamlEnd, bodyStart, ok := findClosingFence(rest)
	if !ok {
		return fm, source, nil
	}
	if err := yaml.Unmarshal(rest[:yamlEnd], &fm); err != nil {
		return Frontmatter{}, source, err
	}
	return fm, rest[bodyStart:], nil
}

// findClosingFence scans b line by line for a line that is exactly "---",
// returning the byte offset where that line begins (yamlEnd) and the offset of
// the first byte after it (bodyStart).
func findClosingFence(b []byte) (yamlEnd, bodyStart int, ok bool) {
	offset := 0
	for offset < len(b) {
		nl := bytes.IndexByte(b[offset:], '\n')
		lineStop := len(b) // index of the newline (or end of input)
		lineEnd := len(b)  // index of the first byte after this line
		if nl >= 0 {
			lineStop = offset + nl
			lineEnd = offset + nl + 1
		}
		if strings.TrimRight(string(b[offset:lineStop]), "\r") == "---" {
			return offset, lineEnd, true
		}
		if nl < 0 {
			break
		}
		offset = lineEnd
	}
	return 0, 0, false
}
