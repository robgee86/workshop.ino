package content

import (
	"strings"
	"testing"
)

func TestResolveRel(t *testing.T) {
	cases := []struct {
		base, rel string
		want      string
		ok        bool
	}{
		{"01-setup", "./starter.zip", "01-setup/starter.zip", true},
		{"01-setup", "starter.zip", "01-setup/starter.zip", true},
		{"02-blink/01-code.side", "../assets/p.png", "02-blink/assets/p.png", true},
		{"01-setup", "../../etc/passwd", "", false}, // escapes root
		{"01-setup", "https://x/y.png", "", false},  // external
		{"01-setup", "data:image/png;base64,AAAA", "", false},
		{"01-setup", "", "", false},
	}
	for _, c := range cases {
		got, ok := ResolveRel(c.base, c.rel)
		if ok != c.ok || got != c.want {
			t.Errorf("ResolveRel(%q,%q) = (%q,%v), want (%q,%v)", c.base, c.rel, got, ok, c.want, c.ok)
		}
	}
}

func TestRenderRewritesRelativeImages(t *testing.T) {
	html, err := Render([]byte("![diagram](./diagram.png)\n\n![ext](https://x/y.png)\n"), "01-setup")
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	s := string(html)
	if !strings.Contains(s, `src="/dl/01-setup/diagram.png"`) {
		t.Errorf("relative image not rewritten to /dl path, got: %s", s)
	}
	if !strings.Contains(s, `src="https://x/y.png"`) {
		t.Errorf("external image must be left untouched, got: %s", s)
	}
}

func TestRenderBasicMarkdown(t *testing.T) {
	html, err := Render([]byte("# Title\n\nSome **bold** text.\n"), "")
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	s := string(html)
	if !strings.Contains(s, "<h1") || !strings.Contains(s, "Title</h1>") {
		t.Errorf("expected an <h1>Title</h1>, got: %s", s)
	}
	if !strings.Contains(s, "<strong>bold</strong>") {
		t.Errorf("expected bold to render, got: %s", s)
	}
}

func TestRenderMermaidPassthrough(t *testing.T) {
	src := "```mermaid\ngraph TD;\n  A-->B;\n```\n"
	html, err := Render([]byte(src), "")
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	s := string(html)
	if !strings.Contains(s, `<pre class="mermaid">`) {
		t.Errorf("expected mermaid pre wrapper, got: %s", s)
	}
	if !strings.Contains(s, "graph TD;") || !strings.Contains(s, "A--&gt;B;") {
		t.Errorf("expected raw (escaped) mermaid source preserved, got: %s", s)
	}
	if strings.Contains(s, "<span") {
		t.Errorf("mermaid block must not be syntax-highlighted into spans, got: %s", s)
	}
}

func TestRenderCodeBlockHighlighted(t *testing.T) {
	src := "```c\nint x = 1;\n```\n"
	html, err := Render([]byte(src), "")
	if err != nil {
		t.Fatalf("Render error: %v", err)
	}
	s := string(html)
	if !strings.Contains(s, `class="code-block"`) {
		t.Errorf("expected code-block wrapper for copy buttons, got: %s", s)
	}
	if !strings.Contains(s, `data-lang="c"`) {
		t.Errorf("expected language recorded on wrapper, got: %s", s)
	}
	if !strings.Contains(s, "int") || !strings.Contains(s, "x") {
		t.Errorf("expected code content preserved, got: %s", s)
	}
}

func TestSplitFrontmatter(t *testing.T) {
	src := []byte("---\ntitle: Wire the LED\nsummary: Connect the anode\nattachments:\n  - path: ./starter.zip\n    label: Starter\n    description: Initial sketch\n---\n# Heading\n\nBody text.\n")

	fm, body, err := SplitFrontmatter(src)
	if err != nil {
		t.Fatalf("SplitFrontmatter returned error: %v", err)
	}
	if fm.Title != "Wire the LED" {
		t.Errorf("Title = %q, want %q", fm.Title, "Wire the LED")
	}
	if fm.Summary != "Connect the anode" {
		t.Errorf("Summary = %q, want %q", fm.Summary, "Connect the anode")
	}
	if len(fm.Attachments) != 1 {
		t.Fatalf("got %d attachments, want 1", len(fm.Attachments))
	}
	if got := fm.Attachments[0]; got.Path != "./starter.zip" || got.Label != "Starter" || got.Description != "Initial sketch" {
		t.Errorf("attachment = %+v, want path=./starter.zip label=Starter description=Initial sketch", got)
	}
	if strings.Contains(string(body), "title:") {
		t.Errorf("body still contains frontmatter: %q", body)
	}
	if !strings.HasPrefix(strings.TrimSpace(string(body)), "# Heading") {
		t.Errorf("body = %q, want it to start with the heading", body)
	}
}

func TestSplitFrontmatterLinks(t *testing.T) {
	src := []byte("---\ntitle: Step\nlinks:\n  - url: https://example.com/ref\n    label: Example\n    description: A reference page\n  - url: https://example.com/bare\n---\nbody\n")
	fm, _, err := SplitFrontmatter(src)
	if err != nil {
		t.Fatalf("SplitFrontmatter error: %v", err)
	}
	if len(fm.Links) != 2 {
		t.Fatalf("got %d links, want 2", len(fm.Links))
	}
	if l := fm.Links[0]; l.URL != "https://example.com/ref" || l.Label != "Example" || l.Description != "A reference page" {
		t.Errorf("links[0] = %+v", l)
	}
	if l := fm.Links[1]; l.URL != "https://example.com/bare" || l.Label != "" {
		t.Errorf("links[1] = %+v (label should fall back later in the view, not the parse)", l)
	}
}

func TestSplitFrontmatterNoFrontmatter(t *testing.T) {
	src := []byte("# Just a heading\n\nNo frontmatter here.\n")

	fm, body, err := SplitFrontmatter(src)
	if err != nil {
		t.Fatalf("SplitFrontmatter returned error: %v", err)
	}
	if fm.Title != "" || len(fm.Attachments) != 0 {
		t.Errorf("expected empty frontmatter, got %+v", fm)
	}
	if string(body) != string(src) {
		t.Errorf("body = %q, want unchanged source", body)
	}
}
