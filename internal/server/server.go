// Package server hosts the workshop handbook over HTTP. Content is read from
// disk on every request so instructors can fix typos live without a restart;
// templates and static assets are embedded for single-binary distribution.
package server

import (
	"embed"
	"html/template"
	"io/fs"
	"net/http"
)

//go:embed templates
var templatesFS embed.FS

//go:embed assets
var assetsFS embed.FS

// Server serves a single workshop rooted at contentRoot.
type Server struct {
	contentRoot string
	index       *template.Template
	step        *template.Template
	sideQuest   *template.Template
}

// New builds an HTTP handler serving the workshop at contentRoot.
func New(contentRoot string) (http.Handler, error) {
	page := func(name string) (*template.Template, error) {
		return template.New("layout").ParseFS(templatesFS, "templates/layout.html", "templates/"+name)
	}
	idx, err := page("index.html")
	if err != nil {
		return nil, err
	}
	stp, err := page("step.html")
	if err != nil {
		return nil, err
	}
	sq, err := page("sidequest.html")
	if err != nil {
		return nil, err
	}

	s := &Server{contentRoot: contentRoot, index: idx, step: stp, sideQuest: sq}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleIndex)
	mux.HandleFunc("GET /s/{path...}", s.handleStep)
	mux.HandleFunc("GET /q/{path...}", s.handleSideQuest)
	mux.HandleFunc("GET /dl/{path...}", s.handleDownload)

	static, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		return nil, err
	}
	mux.Handle("GET /static/{path...}", http.StripPrefix("/static/", http.FileServer(http.FS(static))))

	return mux, nil
}
