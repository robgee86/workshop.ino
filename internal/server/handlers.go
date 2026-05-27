package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"workshopify/internal/content"
)

// baseData is shared by every page: workshop chrome and progress wiring.
type baseData struct {
	PageTitle     string
	WorkshopTitle string
	WorkshopSub   string
	ConfigJSON    template.JS // {id, checkpoints, stepPath} for client-side progress
	HasMermaid    bool        // only then is the (heavy) mermaid bundle loaded
}

type linkView struct {
	Title string
	URL   string
}

type attachmentView struct {
	Label, Description, URL string
}

type stepLink struct {
	linkView
	Path         string // bare checkpoint path, e.g. "01-setup/02-wiring"
	Summary      string
	IsCheckpoint bool
	SideQuests   int
}

type sectionView struct {
	Label string // "Intro", "Milestone 1", "Outro"
	Title string
	Steps []stepLink
}

type indexData struct {
	baseData
	Sections []sectionView
}

type stepData struct {
	baseData
	Eyebrow     string
	Heading     string
	Body        template.HTML
	Attachments []attachmentView
	SideQuests  []linkView
	Prev        *linkView
	Next        *linkView
}

type sideQuestData struct {
	baseData
	Heading   string
	Body      template.HTML
	BackURL   string
	BackTitle string
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	ws, err := content.Scan(s.contentRoot)
	if err != nil {
		http.Error(w, "cannot read workshop content", http.StatusInternalServerError)
		return
	}

	data := indexData{baseData: base(ws, "", "")}
	milestone := 0
	for _, sec := range ws.Sections {
		label := sec.Title
		if sec.Kind == content.Milestone {
			milestone++
			label = fmt.Sprintf("Milestone %d", milestone)
		}
		sv := sectionView{Label: label, Title: sec.Title}
		for _, st := range sec.Steps {
			sv.Steps = append(sv.Steps, stepLink{
				linkView:     linkView{Title: st.Title, URL: "/s/" + st.Path},
				Path:         st.Path,
				Summary:      st.Summary,
				IsCheckpoint: st.Kind == content.Milestone,
				SideQuests:   len(st.SideQuests),
			})
		}
		data.Sections = append(data.Sections, sv)
	}
	s.render(w, s.index, data)
}

func (s *Server) handleStep(w http.ResponseWriter, r *http.Request) {
	ws, err := content.Scan(s.contentRoot)
	if err != nil {
		http.Error(w, "cannot read workshop content", http.StatusInternalServerError)
		return
	}
	p := r.PathValue("path")
	step := ws.FindStep(p)
	if step == nil {
		http.NotFound(w, r)
		return
	}

	body, err := s.renderFile(step.FilePath)
	if err != nil {
		http.Error(w, "cannot read step", http.StatusInternalServerError)
		return
	}

	baseDir := s.baseDir(step.FilePath)
	data := stepData{
		baseData: base(ws, step.Title, step.Path),
		Eyebrow:  stepEyebrow(ws, p),
		Heading:  step.Title,
		Body:     body,
	}
	data.HasMermaid = hasMermaid(body)
	for _, a := range step.Attachments {
		url := a.Path
		if resolved, ok := content.ResolveRel(baseDir, a.Path); ok {
			url = "/dl/" + resolved
		}
		label := a.Label
		if label == "" {
			label = path.Base(a.Path)
		}
		data.Attachments = append(data.Attachments, attachmentView{Label: label, Description: a.Description, URL: url})
	}
	for _, sq := range step.SideQuests {
		data.SideQuests = append(data.SideQuests, linkView{Title: sq.Title, URL: "/q/" + sq.Path})
	}
	if prev, next := ws.Neighbors(p); prev != nil || next != nil {
		if prev != nil {
			data.Prev = &linkView{Title: prev.Title, URL: "/s/" + prev.Path}
		}
		if next != nil {
			data.Next = &linkView{Title: next.Title, URL: "/s/" + next.Path}
		}
	}
	s.render(w, s.step, data)
}

func (s *Server) handleSideQuest(w http.ResponseWriter, r *http.Request) {
	ws, err := content.Scan(s.contentRoot)
	if err != nil {
		http.Error(w, "cannot read workshop content", http.StatusInternalServerError)
		return
	}
	p := r.PathValue("path")
	sq := ws.FindSideQuest(p)
	if sq == nil {
		http.NotFound(w, r)
		return
	}

	body, err := s.renderFile(sq.FilePath)
	if err != nil {
		http.Error(w, "cannot read side quest", http.StatusInternalServerError)
		return
	}

	backTitle := "the step"
	if parent := ws.FindStep(sq.ParentPath); parent != nil {
		backTitle = parent.Title
	}
	data := sideQuestData{
		baseData:  base(ws, sq.Title, ""), // no StepPath: side quests don't count as checkpoints
		Heading:   sq.Title,
		Body:      body,
		BackURL:   "/s/" + sq.ParentPath,
		BackTitle: backTitle,
	}
	data.HasMermaid = hasMermaid(body)
	s.render(w, s.sideQuest, data)
}

func (s *Server) handleDownload(w http.ResponseWriter, r *http.Request) {
	abs, err := resolveDownload(s.contentRoot, r.PathValue("path"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	http.ServeFile(w, r, abs)
}

// renderFile reads a Markdown file, strips frontmatter, and renders the body,
// resolving relative image links against the file's directory.
func (s *Server) renderFile(absFile string) (template.HTML, error) {
	data, err := os.ReadFile(absFile)
	if err != nil {
		return "", err
	}
	_, body, err := content.SplitFrontmatter(data)
	if err != nil {
		return "", err
	}
	return content.Render(body, s.baseDir(absFile))
}

func (s *Server) baseDir(absFile string) string {
	rel, err := filepath.Rel(s.contentRoot, absFile)
	if err != nil {
		return ""
	}
	dir := path.Dir(filepath.ToSlash(rel))
	if dir == "." {
		return ""
	}
	return dir
}

func (s *Server) render(w http.ResponseWriter, t *template.Template, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func base(ws *content.Workshop, pageTitle, stepPath string) baseData {
	checkpoints := ws.Checkpoints()
	if checkpoints == nil {
		checkpoints = []string{}
	}
	cfg, _ := json.Marshal(map[string]any{
		"id":          slug(ws.Title),
		"checkpoints": checkpoints,
		"stepPath":    stepPath,
	})
	title := ws.Title
	if pageTitle != "" {
		title = pageTitle + " · " + ws.Title
	}
	return baseData{
		PageTitle:     title,
		WorkshopTitle: ws.Title,
		WorkshopSub:   ws.Subtitle,
		ConfigJSON:    template.JS(cfg),
	}
}

// stepEyebrow renders the "Milestone N · Step M" (or "Intro"/"Outro") label.
func stepEyebrow(ws *content.Workshop, p string) string {
	milestone := 0
	for _, sec := range ws.Sections {
		if sec.Kind == content.Milestone {
			milestone++
		}
		for i, st := range sec.Steps {
			if st.Path != p {
				continue
			}
			switch sec.Kind {
			case content.Intro:
				return "Intro"
			case content.Outro:
				return "Outro"
			default:
				return fmt.Sprintf("Milestone %d · Step %d", milestone, i+1)
			}
		}
	}
	return ""
}

// hasMermaid reports whether rendered HTML contains a Mermaid diagram block.
func hasMermaid(html template.HTML) bool {
	return strings.Contains(string(html), `<pre class="mermaid">`)
}

// slug makes a string safe for use as a localStorage namespace key.
func slug(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(s) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case !prevDash:
			b.WriteByte('-')
			prevDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}
