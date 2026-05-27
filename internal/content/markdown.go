package content

import (
	"bytes"
	"html/template"
	"path"
	"strings"

	"github.com/alecthomas/chroma/v2/quick"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
)

// chromaStyle is the chroma syntax-highlighting theme. It uses inline styles so
// rendered code is self-contained and needs no extra stylesheet (offline-safe).
const chromaStyle = "github"

// Render converts a Markdown body (frontmatter already stripped) to HTML.
// baseDir is the document's directory relative to the content root; it is used
// to rewrite relative image paths to /dl/ download URLs. Fenced "mermaid"
// blocks pass through as <pre class="mermaid"> for client-side rendering; other
// code blocks are highlighted server-side.
func Render(body []byte, baseDir string) (template.HTML, error) {
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
			parser.WithASTTransformers(util.Prioritized(imageRewriter{baseDir}, 100)),
		),
		goldmark.WithRendererOptions(
			gmhtml.WithUnsafe(), // author-trusted content; allows raw HTML in docs
			renderer.WithNodeRenderers(util.Prioritized(&codeRenderer{}, 10)),
		),
	)
	var buf bytes.Buffer
	if err := md.Convert(body, &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

// ResolveRel resolves a relative reference (from frontmatter or Markdown) against
// baseDir into a clean, content-root-relative slash path. It returns ok=false for
// empty, external (scheme/protocol-relative/data), or root-escaping references.
func ResolveRel(baseDir, rel string) (string, bool) {
	if rel == "" {
		return "", false
	}
	if strings.Contains(rel, "://") || strings.HasPrefix(rel, "//") || strings.HasPrefix(rel, "data:") || strings.HasPrefix(rel, "mailto:") {
		return "", false
	}
	cleaned := path.Clean(path.Join(baseDir, rel))
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

// imageRewriter rewrites relative <img> sources to absolute /dl/ download URLs.
type imageRewriter struct{ baseDir string }

func (t imageRewriter) Transform(doc *ast.Document, reader text.Reader, pc parser.Context) {
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		img, ok := n.(*ast.Image)
		if !ok {
			return ast.WalkContinue, nil
		}
		if resolved, ok := ResolveRel(t.baseDir, string(img.Destination)); ok {
			img.Destination = []byte("/dl/" + resolved)
		}
		return ast.WalkContinue, nil
	})
}

// codeRenderer overrides goldmark's default fenced/indented code block output.
type codeRenderer struct{}

func (r *codeRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindFencedCodeBlock, r.renderFenced)
	reg.Register(ast.KindCodeBlock, r.renderIndented)
}

func (r *codeRenderer) renderFenced(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.FencedCodeBlock)
	lang := ""
	if n.Info != nil {
		if fields := strings.Fields(string(n.Info.Segment.Value(source))); len(fields) > 0 {
			lang = fields[0] // first token of the info string, e.g. "c" in "c title=x"
		}
	}
	return r.writeCode(w, lang, blockText(source, n))
}

func (r *codeRenderer) renderIndented(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	return r.writeCode(w, "", blockText(source, node))
}

func (r *codeRenderer) writeCode(w util.BufWriter, lang string, code []byte) (ast.WalkStatus, error) {
	if lang == "mermaid" {
		w.WriteString(`<pre class="mermaid">`)
		template.HTMLEscape(w, code)
		w.WriteString("</pre>\n")
		return ast.WalkSkipChildren, nil
	}
	w.WriteString(`<div class="code-block" data-lang="`)
	w.WriteString(template.HTMLEscapeString(lang))
	w.WriteString(`">`)
	// quick.Highlight falls back to a plaintext lexer for unknown/empty langs.
	if err := quick.Highlight(w, string(code), lang, "html", chromaStyle); err != nil {
		// Degrade gracefully to an escaped preformatted block.
		w.WriteString("<pre>")
		template.HTMLEscape(w, code)
		w.WriteString("</pre>")
	}
	w.WriteString("</div>\n")
	return ast.WalkSkipChildren, nil
}

// blockText reassembles the raw text of a code block from its source segments.
func blockText(source []byte, n ast.Node) []byte {
	var buf bytes.Buffer
	lines := n.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		buf.Write(seg.Value(source))
	}
	return buf.Bytes()
}
