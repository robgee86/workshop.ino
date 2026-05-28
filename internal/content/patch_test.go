package content

import "testing"

const samplePatch = `diff --git a/blink.ino b/blink.ino
index 1111111..2222222 100644
--- a/blink.ino
+++ b/blink.ino
@@ -1,3 +1,3 @@
 void setup() {
-  pinMode(13, 1);
+  pinMode(13, OUTPUT);
 }
diff --git a/README.md b/README.md
new file mode 100644
index 0000000..3333333
--- /dev/null
+++ b/README.md
@@ -0,0 +1,1 @@
+A demo.
`

func TestParsePatch(t *testing.T) {
	files, err := ParsePatch([]byte(samplePatch))
	if err != nil {
		t.Fatalf("ParsePatch error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("got %d files, want 2", len(files))
	}

	// File 0: a modified file with one deletion and one addition.
	mod := files[0]
	if mod.Path != "blink.ino" || mod.Status != "modified" {
		t.Errorf("file0 = {Path:%q Status:%q}, want blink.ino/modified", mod.Path, mod.Status)
	}
	if mod.Additions != 1 || mod.Deletions != 1 {
		t.Errorf("file0 counts = +%d -%d, want +1 -1", mod.Additions, mod.Deletions)
	}
	if len(mod.Hunks) != 1 {
		t.Fatalf("file0 hunks = %d, want 1", len(mod.Hunks))
	}
	got := mod.Hunks[0].Lines
	want := []DiffLine{
		{Kind: "ctx", OldNum: 1, NewNum: 1, Text: "void setup() {"},
		{Kind: "del", OldNum: 2, NewNum: 0, Text: "  pinMode(13, 1);"},
		{Kind: "add", OldNum: 0, NewNum: 2, Text: "  pinMode(13, OUTPUT);"},
		{Kind: "ctx", OldNum: 3, NewNum: 3, Text: "}"},
	}
	if len(got) != len(want) {
		t.Fatalf("file0 lines = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %+v, want %+v", i, got[i], want[i])
		}
	}

	// File 1: a brand-new file (all additions).
	nf := files[1]
	if nf.Path != "README.md" || nf.Status != "added" {
		t.Errorf("file1 = {Path:%q Status:%q}, want README.md/added", nf.Path, nf.Status)
	}
	if nf.Additions != 1 || nf.Deletions != 0 {
		t.Errorf("file1 counts = +%d -%d, want +1 -0", nf.Additions, nf.Deletions)
	}
	if len(nf.Hunks) != 1 || len(nf.Hunks[0].Lines) != 1 || nf.Hunks[0].Lines[0].Kind != "add" {
		t.Errorf("file1 should have a single added line, got %+v", nf.Hunks)
	}
	if l := nf.Hunks[0].Lines[0]; l.NewNum != 1 || l.OldNum != 0 {
		t.Errorf("file1 added line numbers = old %d/new %d, want old 0/new 1", l.OldNum, l.NewNum)
	}
}
