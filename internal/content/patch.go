package content

import (
	"bytes"
	"strings"

	"github.com/bluekeyes/go-gitdiff/gitdiff"
)

// DiffFile is one file touched by a patch, with its hunks of changes.
type DiffFile struct {
	Path      string // the resulting (new) file path; old path for deletions
	OldPath   string // the original path (differs from Path on a rename)
	Status    string // modified | added | deleted | renamed | copied
	Additions int
	Deletions int
	Binary    bool
	Hunks     []DiffHunk
}

// DiffHunk is a contiguous block of changed lines (one "@@ … @@" section).
type DiffHunk struct {
	Heading string // optional section context after the @@ header
	Lines   []DiffLine
}

// DiffLine is a single line in a hunk. OldNum/NewNum are 0 when not applicable
// (an addition has no old line number; a deletion has no new one).
type DiffLine struct {
	Kind   string // ctx | add | del
	OldNum int
	NewNum int
	Text   string
}

// ParsePatch parses a unified/git diff into a structured, render-friendly form.
func ParsePatch(src []byte) ([]DiffFile, error) {
	files, _, err := gitdiff.Parse(bytes.NewReader(src))
	if err != nil {
		return nil, err
	}

	out := make([]DiffFile, 0, len(files))
	for _, f := range files {
		df := DiffFile{
			Path:    f.NewName,
			OldPath: f.OldName,
			Status:  fileStatus(f),
			Binary:  f.IsBinary,
		}
		if df.Path == "" {
			df.Path = f.OldName // deletions carry only an old name
		}
		for _, frag := range f.TextFragments {
			df.Additions += int(frag.LinesAdded)
			df.Deletions += int(frag.LinesDeleted)
			h := DiffHunk{Heading: strings.TrimSpace(frag.Comment)}
			oldNum, newNum := int(frag.OldPosition), int(frag.NewPosition)
			for _, l := range frag.Lines {
				dl := DiffLine{Text: strings.TrimRight(l.Line, "\r\n")}
				switch l.Op {
				case gitdiff.OpContext:
					dl.Kind, dl.OldNum, dl.NewNum = "ctx", oldNum, newNum
					oldNum++
					newNum++
				case gitdiff.OpDelete:
					dl.Kind, dl.OldNum = "del", oldNum
					oldNum++
				case gitdiff.OpAdd:
					dl.Kind, dl.NewNum = "add", newNum
					newNum++
				}
				h.Lines = append(h.Lines, dl)
			}
			df.Hunks = append(df.Hunks, h)
		}
		out = append(out, df)
	}
	return out, nil
}

func fileStatus(f *gitdiff.File) string {
	switch {
	case f.IsNew:
		return "added"
	case f.IsDelete:
		return "deleted"
	case f.IsRename:
		return "renamed"
	case f.IsCopy:
		return "copied"
	default:
		return "modified"
	}
}
