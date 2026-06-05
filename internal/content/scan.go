package content

import (
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Scan walks a content directory and builds the Workshop model. It reads from
// disk every call, so edits to the tree are picked up without a restart.
func Scan(root string) (*Workshop, error) {
	ws := &Workshop{Title: titleize(stripPrefix(filepath.Base(root)))}
	loadWorkshopMeta(root, ws)

	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var introDir, outroDir string
	var milestones []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		switch strings.ToLower(e.Name()) {
		case "intro":
			introDir = e.Name()
		case "outro":
			outroDir = e.Name()
		default:
			// Milestones must carry a numeric prefix; unprefixed folders
			// (assets/, images/, …) are shared resources, not milestones.
			if prefixRe.MatchString(e.Name()) {
				milestones = append(milestones, e.Name())
			}
		}
	}
	sortByPrefix(milestones)

	if introDir != "" {
		ws.addSection(scanSection(root, introDir, Intro, ws.App))
	}
	for _, m := range milestones {
		ws.addSection(scanSection(root, m, Milestone, ws.App))
	}
	if outroDir != "" {
		ws.addSection(scanSection(root, outroDir, Outro, ws.App))
	}
	return ws, nil
}

// addSection appends a section only if it has at least one step, so empty or
// misnamed folders don't produce ghost sections in the navigation.
func (w *Workshop) addSection(sec *Section) {
	if len(sec.Steps) > 0 {
		w.Sections = append(w.Sections, sec)
	}
}

func loadWorkshopMeta(root string, ws *Workshop) {
	data, err := os.ReadFile(filepath.Join(root, "workshop.yaml"))
	if err != nil {
		return
	}
	var meta struct {
		Title    string `yaml:"title"`
		Subtitle string `yaml:"subtitle"`
		App      string `yaml:"app"`
	}
	if yaml.Unmarshal(data, &meta) == nil {
		if meta.Title != "" {
			ws.Title = meta.Title
		}
		ws.Subtitle = meta.Subtitle
		ws.App = meta.App
	}
}

func scanSection(root, dir string, kind SectionKind, defaultApp string) *Section {
	sec := &Section{Kind: kind, Slug: dir}
	switch kind {
	case Intro:
		sec.Title = "Intro"
	case Outro:
		sec.Title = "Outro"
	default:
		sec.Title = titleize(stripPrefix(dir))
	}

	entries, err := os.ReadDir(filepath.Join(root, dir))
	if err != nil {
		return sec
	}
	var stepFiles []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") || isSideFile(e.Name()) {
			continue
		}
		stepFiles = append(stepFiles, e.Name())
	}
	sortByPrefix(stepFiles)

	for _, f := range stepFiles {
		sec.Steps = append(sec.Steps, scanStep(root, dir, f, kind, defaultApp))
	}
	return sec
}

func scanStep(root, dir, file string, kind SectionKind, defaultApp string) *Step {
	seg := strings.TrimSuffix(file, ".md")
	stepPath := path.Join(dir, seg)
	abs := filepath.Join(root, dir, file)

	fm, body := readDoc(abs)
	app := defaultApp
	if fm.App != "" {
		app = fm.App
	}
	st := &Step{
		Title:       resolveTitle(fm.Title, body, seg),
		Summary:     fm.Summary,
		Path:        stepPath,
		FilePath:    abs,
		Kind:        kind,
		App:         app,
		Attachments: fm.Attachments,
		Patches:     fm.Patches,
		Links:       fm.Links,
	}
	st.SideQuests = scanSideQuests(root, dir, seg, stepPath)
	return st
}

func scanSideQuests(root, dir, seg, parentPath string) []*SideQuest {
	// Folder form wins: NN-slug.side/ holding ordered .md files.
	folder := filepath.Join(root, dir, seg+".side")
	if info, err := os.Stat(folder); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(folder)
		var files []string
		for _, e := range entries {
			if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
				files = append(files, e.Name())
			}
		}
		sortByPrefix(files)
		var quests []*SideQuest
		for _, f := range files {
			sqSeg := strings.TrimSuffix(f, ".md")
			abs := filepath.Join(folder, f)
			fm, body := readDoc(abs)
			quests = append(quests, &SideQuest{
				Title:      resolveTitle(fm.Title, body, sqSeg),
				Path:       path.Join(parentPath, sqSeg),
				FilePath:   abs,
				ParentPath: parentPath,
			})
		}
		return quests
	}

	// Shorthand: a single NN-slug.side.md sibling.
	short := filepath.Join(root, dir, seg+".side.md")
	if _, err := os.Stat(short); err == nil {
		fm, body := readDoc(short)
		title := fm.Title
		if title == "" {
			title = firstH1(body)
		}
		if title == "" {
			title = "Side quest"
		}
		return []*SideQuest{{
			Title:      title,
			Path:       path.Join(parentPath, "side"),
			FilePath:   short,
			ParentPath: parentPath,
		}}
	}
	return nil
}

// readDoc reads a Markdown file and splits off its frontmatter. Missing files
// yield zero values rather than an error so a partially-written tree still scans.
func readDoc(abs string) (Frontmatter, []byte) {
	data, err := os.ReadFile(abs)
	if err != nil {
		return Frontmatter{}, nil
	}
	fm, body, err := SplitFrontmatter(data)
	if err != nil {
		return Frontmatter{}, data
	}
	return fm, body
}

func resolveTitle(fmTitle string, body []byte, seg string) string {
	if fmTitle != "" {
		return fmTitle
	}
	if h := firstH1(body); h != "" {
		return h
	}
	return titleize(stripPrefix(seg))
}

// isSideFile reports whether a filename is the shorthand side-quest form.
func isSideFile(name string) bool {
	return strings.HasSuffix(name, ".side.md")
}

var prefixRe = regexp.MustCompile(`^(\d+)[-_. ]?`)

// stripPrefix removes a leading numeric ordering prefix from a name.
func stripPrefix(name string) string {
	return prefixRe.ReplaceAllString(name, "")
}

// prefixOrder returns the numeric value of a leading prefix, or a large number
// so unprefixed names sort last.
func prefixOrder(name string) int {
	m := prefixRe.FindStringSubmatch(name)
	if m == nil {
		return 1 << 30
	}
	n, _ := strconv.Atoi(m[1])
	return n
}

// sortByPrefix orders names by numeric prefix, then lexically as a tiebreaker.
func sortByPrefix(names []string) {
	sort.SliceStable(names, func(i, j int) bool {
		oi, oj := prefixOrder(names[i]), prefixOrder(names[j])
		if oi != oj {
			return oi < oj
		}
		return names[i] < names[j]
	})
}

// titleize turns a slug into a display title: "wire-the-led" -> "Wire The Led".
func titleize(slug string) string {
	slug = strings.NewReplacer("-", " ", "_", " ").Replace(slug)
	fields := strings.Fields(slug)
	for i, f := range fields {
		fields[i] = strings.ToUpper(f[:1]) + f[1:]
	}
	return strings.Join(fields, " ")
}

// firstH1 returns the text of the first level-1 ATX heading in a Markdown body.
func firstH1(body []byte) string {
	for _, line := range strings.Split(string(body), "\n") {
		t := strings.TrimSpace(line)
		if strings.HasPrefix(t, "# ") {
			return strings.TrimSpace(t[2:])
		}
	}
	return ""
}
