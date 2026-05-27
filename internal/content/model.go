package content

// SectionKind distinguishes the optional intro/outro framing from the graded
// milestone sections that count toward progress.
type SectionKind int

const (
	Milestone SectionKind = iota
	Intro
	Outro
)

// Workshop is the whole handbook: ordered sections plus display metadata.
type Workshop struct {
	Title    string
	Subtitle string
	Sections []*Section
}

// Section is an intro folder, a milestone folder, or an outro folder.
type Section struct {
	Kind  SectionKind
	Title string // display title, numeric prefix stripped
	Slug  string // on-disk folder name, e.g. "01-setup" or "intro"
	Steps []*Step
}

// Step is one Markdown checkpoint page.
type Step struct {
	Title       string
	Summary     string
	Path        string // stable URL id, e.g. "01-setup/02-wiring"
	FilePath    string // absolute path to the .md on disk
	Kind        SectionKind
	Attachments []Attachment
	SideQuests  []*SideQuest
}

// SideQuest is an optional detour page hanging off a parent step.
type SideQuest struct {
	Title      string
	Path       string // stable URL id, e.g. "02-blink/01-code/02-timer-math"
	FilePath   string
	ParentPath string // the originating step's Path
}

// orderedSteps returns every step in linear reading order (intro, then each
// milestone, then outro). Side quests are excluded — they sit off this path.
func (w *Workshop) orderedSteps() []*Step {
	var steps []*Step
	for _, s := range w.Sections {
		steps = append(steps, s.Steps...)
	}
	return steps
}

// Checkpoints returns the paths of milestone steps only, in order. These form
// the denominator for progress; intro, outro and side quests don't count.
func (w *Workshop) Checkpoints() []string {
	var ids []string
	for _, s := range w.Sections {
		if s.Kind != Milestone {
			continue
		}
		for _, st := range s.Steps {
			ids = append(ids, st.Path)
		}
	}
	return ids
}

// FindStep returns the step with the given path, or nil.
func (w *Workshop) FindStep(path string) *Step {
	for _, st := range w.orderedSteps() {
		if st.Path == path {
			return st
		}
	}
	return nil
}

// FindSideQuest returns the side quest with the given path, or nil.
func (w *Workshop) FindSideQuest(path string) *SideQuest {
	for _, st := range w.orderedSteps() {
		for _, sq := range st.SideQuests {
			if sq.Path == path {
				return sq
			}
		}
	}
	return nil
}

// Neighbors returns the previous and next steps in linear order for the given
// step path. Either may be nil at the ends of the workshop.
func (w *Workshop) Neighbors(path string) (prev, next *Step) {
	steps := w.orderedSteps()
	for i, st := range steps {
		if st.Path != path {
			continue
		}
		if i > 0 {
			prev = steps[i-1]
		}
		if i < len(steps)-1 {
			next = steps[i+1]
		}
		return prev, next
	}
	return nil, nil
}
