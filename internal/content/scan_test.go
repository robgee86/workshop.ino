package content

import (
	"os"
	"path/filepath"
	"testing"
)

// buildFixture writes a representative workshop tree and returns its root.
func buildFixture(t *testing.T) string {
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

	write("workshop.yaml", "title: My Workshop\nsubtitle: Learn blink\n")
	write("intro/01-welcome.md", "# Welcome\n")
	write("01-setup/01-install.md", "---\ntitle: Install Tools\n---\n# Install\n")
	write("01-setup/02-wiring.md", "---\ntitle: Wire the LED\nsummary: Connect the anode\nattachments:\n  - path: ./starter.zip\n    label: Starter\n---\nBody\n")
	write("01-setup/02-wiring.side.md", "# Extra wiring tips\n")
	write("02-blink/01-code.md", "# Blink code\n")
	write("02-blink/01-code.side/01-deep-dive.md", "---\ntitle: Deep Dive\n---\nmore\n")
	write("02-blink/01-code.side/02-timer-math.md", "# Timer Math\n")
	write("02-blink/02-upload.md", "# Upload\n")
	write("outro/01-wrap-up.md", "# Wrap Up\n")

	// Shared asset folder and an unprefixed stray folder: neither is a milestone.
	write("assets/logo.svg", "<svg/>\n")
	write("notes/scratch.md", "# scratch\n")

	return root
}

func TestScanWorkshopMeta(t *testing.T) {
	ws, err := Scan(buildFixture(t))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}
	if ws.Title != "My Workshop" {
		t.Errorf("Title = %q, want %q", ws.Title, "My Workshop")
	}
	if ws.Subtitle != "Learn blink" {
		t.Errorf("Subtitle = %q, want %q", ws.Subtitle, "Learn blink")
	}
}

func TestScanSectionsOrderAndKind(t *testing.T) {
	ws, err := Scan(buildFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if len(ws.Sections) != 4 {
		t.Fatalf("got %d sections, want 4", len(ws.Sections))
	}
	want := []struct {
		kind  SectionKind
		title string
		steps int
	}{
		{Intro, "Intro", 1},
		{Milestone, "Setup", 2},
		{Milestone, "Blink", 2},
		{Outro, "Outro", 1},
	}
	for i, w := range want {
		s := ws.Sections[i]
		if s.Kind != w.kind {
			t.Errorf("section %d Kind = %v, want %v", i, s.Kind, w.kind)
		}
		if s.Title != w.title {
			t.Errorf("section %d Title = %q, want %q", i, s.Title, w.title)
		}
		if len(s.Steps) != w.steps {
			t.Errorf("section %d has %d steps, want %d", i, len(s.Steps), w.steps)
		}
	}
}

func TestScanStepTitleResolution(t *testing.T) {
	ws, err := Scan(buildFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	if got := ws.FindStep("01-setup/02-wiring"); got == nil || got.Title != "Wire the LED" {
		t.Errorf("frontmatter title not used: %+v", got)
	}
	if got := ws.FindStep("02-blink/01-code"); got == nil || got.Title != "Blink code" {
		t.Errorf("H1 fallback title not used: %+v", got)
	}
	if got := ws.FindStep("intro/01-welcome"); got == nil || got.Title != "Welcome" {
		t.Errorf("intro step title wrong: %+v", got)
	}
}

func TestScanAttachments(t *testing.T) {
	ws, err := Scan(buildFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	step := ws.FindStep("01-setup/02-wiring")
	if step == nil || len(step.Attachments) != 1 {
		t.Fatalf("expected 1 attachment, got %+v", step)
	}
	if step.Attachments[0].Label != "Starter" {
		t.Errorf("attachment label = %q", step.Attachments[0].Label)
	}
}

func TestScanSideQuests(t *testing.T) {
	ws, err := Scan(buildFixture(t))
	if err != nil {
		t.Fatal(err)
	}

	// Shorthand: single .side.md sibling.
	wiring := ws.FindStep("01-setup/02-wiring")
	if wiring == nil || len(wiring.SideQuests) != 1 {
		t.Fatalf("shorthand side quest not found: %+v", wiring)
	}
	if sq := wiring.SideQuests[0]; ws.FindSideQuest(sq.Path) != sq {
		t.Errorf("shorthand side quest %q not resolvable via FindSideQuest", sq.Path)
	}

	// Folder form: multiple ordered side quests.
	code := ws.FindStep("02-blink/01-code")
	if code == nil || len(code.SideQuests) != 2 {
		t.Fatalf("folder side quests not found: %+v", code)
	}
	if code.SideQuests[0].Title != "Deep Dive" {
		t.Errorf("side quest 0 title = %q, want Deep Dive", code.SideQuests[0].Title)
	}
	if code.SideQuests[1].Title != "Timer Math" {
		t.Errorf("side quest 1 title = %q, want Timer Math", code.SideQuests[1].Title)
	}
	if code.SideQuests[0].ParentPath != "02-blink/01-code" {
		t.Errorf("side quest ParentPath = %q", code.SideQuests[0].ParentPath)
	}
}

func TestScanSideQuestsForIntroAndOutro(t *testing.T) {
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
	write("intro/01-hi.side.md", "---\ntitle: Intro Detour\n---\nmore\n") // shorthand under intro
	write("01-m/01-s.md", "---\ntitle: Step\n---\nbody\n")
	write("outro/01-bye.md", "---\ntitle: Bye\n---\nbye\n")
	write("outro/01-bye.side/01-extra.md", "---\ntitle: Outro Detour\n---\nmore\n") // folder form under outro

	ws, err := Scan(root)
	if err != nil {
		t.Fatal(err)
	}
	intro := ws.FindStep("intro/01-hi")
	if intro == nil || len(intro.SideQuests) != 1 || intro.SideQuests[0].Title != "Intro Detour" {
		t.Errorf("intro side quest not discovered: %+v", intro)
	}
	out := ws.FindStep("outro/01-bye")
	if out == nil || len(out.SideQuests) != 1 || out.SideQuests[0].Title != "Outro Detour" {
		t.Errorf("outro side quest not discovered: %+v", out)
	}
	if ws.FindSideQuest("outro/01-bye/01-extra") == nil {
		t.Errorf("outro side quest not resolvable via FindSideQuest")
	}
}

func TestScanCheckpointsExcludeExtras(t *testing.T) {
	ws, err := Scan(buildFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	got := ws.Checkpoints()
	want := []string{"01-setup/01-install", "01-setup/02-wiring", "02-blink/01-code", "02-blink/02-upload"}
	if len(got) != len(want) {
		t.Fatalf("checkpoints = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("checkpoint %d = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestScanNeighborsLinearOrder(t *testing.T) {
	ws, err := Scan(buildFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	// Linear order spans intro -> milestones -> outro.
	prev, next := ws.Neighbors("01-setup/02-wiring")
	if prev == nil || prev.Path != "01-setup/01-install" {
		t.Errorf("prev = %+v, want 01-setup/01-install", prev)
	}
	if next == nil || next.Path != "02-blink/01-code" {
		t.Errorf("next = %+v, want 02-blink/01-code", next)
	}

	// First step has no prev; last step has no next.
	if prev, _ := ws.Neighbors("intro/01-welcome"); prev != nil {
		t.Errorf("intro first step should have no prev, got %+v", prev)
	}
	if _, next := ws.Neighbors("outro/01-wrap-up"); next != nil {
		t.Errorf("outro last step should have no next, got %+v", next)
	}
}
