package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newTestServer(t *testing.T) http.Handler {
	t.Helper()
	root := t.TempDir()
	write := func(rel, body string) {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("workshop.yaml", "title: Demo WS\nsubtitle: a sub\n")
	write("01-basics/01-hello.md", "---\ntitle: Hello Step\nattachments:\n  - path: ./starter.txt\n    label: Starter\n    description: begin here\n---\n# Hello\n\n```c\nint x = 1;\n```\n\n```mermaid\ngraph TD; A-->B;\n```\n")
	write("01-basics/01-hello.side.md", "# Bonus\n")
	write("01-basics/starter.txt", "STARTER")

	h, err := New(root)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return h
}

func get(t *testing.T, h http.Handler, path string) (*http.Response, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	res := rec.Result()
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

func TestIndexPage(t *testing.T) {
	h := newTestServer(t)
	res, body := get(t, h, "/")
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	for _, want := range []string{"Demo WS", "Hello Step", "Milestone 1"} {
		if !strings.Contains(body, want) {
			t.Errorf("index missing %q", want)
		}
	}
}

func TestStepPage(t *testing.T) {
	h := newTestServer(t)
	res, body := get(t, h, "/s/01-basics/01-hello")
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	wants := []string{
		"Hello Step",                 // heading
		`class="code-block"`,         // highlighted code
		`<pre class="mermaid">`,      // diagram passthrough
		"mermaid.min.js",             // mermaid bundle loaded only here
		`/dl/01-basics/starter.txt`,  // attachment URL resolved
		"begin here",                 // attachment description
		`/q/01-basics/01-hello/side`, // side quest link
	}
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("step page missing %q", w)
		}
	}
}

func TestSideQuestPage(t *testing.T) {
	h := newTestServer(t)
	res, body := get(t, h, "/q/01-basics/01-hello/side")
	if res.StatusCode != 200 {
		t.Fatalf("status %d", res.StatusCode)
	}
	if !strings.Contains(body, "Back to Hello Step") {
		t.Errorf("side quest missing back link, got: %s", body)
	}
}

func TestDownloadServesAndGuards(t *testing.T) {
	h := newTestServer(t)

	res, body := get(t, h, "/dl/01-basics/starter.txt")
	if res.StatusCode != 200 || body != "STARTER" {
		t.Errorf("download = (%d,%q), want (200,STARTER)", res.StatusCode, body)
	}

	// A traversal attempt must never serve a file (ServeMux cleans the path to a
	// redirect; the guard rejects anything that reaches the handler).
	if res, body := get(t, h, "/dl/..%2f..%2f..%2fetc%2fhosts"); res.StatusCode == 200 {
		t.Errorf("traversal returned 200 with body %q", body)
	}
}

func TestUnknownStep404(t *testing.T) {
	h := newTestServer(t)
	if res, _ := get(t, h, "/s/does/not/exist"); res.StatusCode != 404 {
		t.Errorf("status = %d, want 404", res.StatusCode)
	}
}
