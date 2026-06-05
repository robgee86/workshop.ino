package server

import (
	"archive/zip"
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

	h, err := New(root, t.TempDir())
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

func post(t *testing.T, h http.Handler, path, jsonBody string) (*http.Response, string) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	h.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res, string(body)
}

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeZip(t *testing.T, path string, files map[string]string) {
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

func TestIntroOutroHomeNav(t *testing.T) {
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
	write("workshop.yaml", "title: T\n")
	write("intro/01-hi.md", "---\ntitle: Hi\n---\nhello\n")
	write("01-m/01-s.md", "---\ntitle: Step\n---\nbody\n")
	write("outro/01-bye.md", "---\ntitle: Bye\n---\nbye\n")
	h, err := New(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	// Intro's first step has no real previous: its Prev goes home.
	if _, body := get(t, h, "/s/intro/01-hi"); !strings.Contains(body, `class="nav-btn prev" rel="prev" href="/"`) {
		t.Errorf("intro page missing home Prev button")
	}
	// Outro's last step has no real next: its Next goes home.
	if _, body := get(t, h, "/s/outro/01-bye"); !strings.Contains(body, `class="nav-btn next" rel="next" href="/"`) {
		t.Errorf("outro page missing home Next button")
	}
	// A milestone step in the middle keeps real linear neighbors, not home.
	_, body := get(t, h, "/s/01-m/01-s")
	if strings.Contains(body, `href="/"`) && !strings.Contains(body, `href="/s/`) {
		t.Errorf("milestone step should link to real steps, not home")
	}
	if !strings.Contains(body, `href="/s/intro/01-hi"`) || !strings.Contains(body, `href="/s/outro/01-bye"`) {
		t.Errorf("milestone step missing linear prev/next: %s", body)
	}
}

func TestStepPatchesRender(t *testing.T) {
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
	write("workshop.yaml", "title: T\napp: demo-app\n")
	write("01-m/01-s.md", "---\ntitle: Step\npatches:\n  - path: ./change.patch\n    label: My change\n---\nbody\n")
	write("01-m/change.patch", "--- a/x.go\n+++ b/x.go\n@@ -1,2 +1,2 @@\n keep me\n-old line\n+new line\n")

	h, err := New(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, body := get(t, h, "/s/01-m/01-s")
	wants := []string{
		"Code changes", "My change", "x.go", "diff-line add", "diff-line del", "new line",
		`href="/dl/01-m/change.patch"`, // file name downloads the .patch
	}
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("step page missing %q", w)
		}
	}
	// Code changes are display-only — no per-patch apply, no per-file copy.
	if strings.Contains(body, "apply-btn") || strings.Contains(body, "diff-copy") {
		t.Errorf("Code changes should show diffs only, with no apply/copy button")
	}
}

func TestApplySolutionEndpoint(t *testing.T) {
	root := t.TempDir()
	apps := t.TempDir()
	// A zip solution living in the content tree.
	zipPath := filepath.Join(root, "01-m", "solution.zip")
	if err := os.MkdirAll(filepath.Dir(zipPath), 0o755); err != nil {
		t.Fatal(err)
	}
	writeZip(t, zipPath, map[string]string{"blink.ino": "fresh\n"})
	writeFile(t, filepath.Join(root, "workshop.yaml"), "title: T\napp: blink\n")
	writeFile(t, filepath.Join(root, "01-m", "01-s.md"),
		"---\ntitle: Step\nattachments:\n  - path: ./solution.zip\n    label: Solution\n    solution: true\n---\nbody\n")
	// Device app with a stale file that applying the solution should remove.
	if err := os.MkdirAll(filepath.Join(apps, "blink"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(apps, "blink", "stale.txt"), "old")

	h, err := New(root, apps)
	if err != nil {
		t.Fatal(err)
	}

	// The solution attachment renders an Apply button and a solution tag.
	if _, body := get(t, h, "/s/01-m/01-s"); !strings.Contains(body, `class="action-btn solution-btn"`) || !strings.Contains(body, "solution-tag") {
		t.Errorf("solution attachment should render an Apply button and a solution tag")
	}

	res, body := post(t, h, "/apply-solution", `{"step":"01-m/01-s","index":0}`)
	if res.StatusCode != 200 || !strings.Contains(body, `"ok":true`) {
		t.Fatalf("apply solution = (%d, %s), want 200 ok", res.StatusCode, body)
	}
	if _, err := os.Stat(filepath.Join(apps, "blink", "stale.txt")); !os.IsNotExist(err) {
		t.Errorf("applying the solution should have wiped stale.txt")
	}
	if got, _ := os.ReadFile(filepath.Join(apps, "blink", "blink.ino")); string(got) != "fresh\n" {
		t.Errorf("applied blink.ino = %q, want %q", got, "fresh\n")
	}

	// A non-solution index is rejected.
	if res, _ := post(t, h, "/apply-solution", `{"step":"01-m/01-s","index":9}`); res.StatusCode == 200 {
		t.Errorf("apply-solution with out-of-range index should not succeed")
	}
}

func TestStepLinksRender(t *testing.T) {
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
	write("workshop.yaml", "title: T\n")
	write(
		"01-m/01-s.md",
		"---\ntitle: Step\nlinks:\n  - url: https://docs.example.com/ref\n    label: Example reference\n    description: Official docs\n  - url: https://example.com/bare\n---\nbody\n",
	)
	h, err := New(root, t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	_, body := get(t, h, "/s/01-m/01-s")
	wants := []string{
		"References",
		`href="https://docs.example.com/ref"`,
		`target="_blank"`,
		`rel="noopener noreferrer"`,
		"Example reference",
		"Official docs",
		// A bare-URL link falls back to showing the URL itself as the label.
		`href="https://example.com/bare"`,
		"https://example.com/bare",
	}
	for _, w := range wants {
		if !strings.Contains(body, w) {
			t.Errorf("step page missing %q", w)
		}
	}
}

func TestUnknownStep404(t *testing.T) {
	h := newTestServer(t)
	if res, _ := get(t, h, "/s/does/not/exist"); res.StatusCode != 404 {
		t.Errorf("status = %d, want 404", res.StatusCode)
	}
}
