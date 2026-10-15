package emit

import (
	"fmt"
	"io/fs"
	"sort"
	"strings"
)

// diffContext is the number of unchanged lines shown around each
// insertion — unified-diff convention.
const diffContext = 3

// DiffMarkers renders marker suggestions as a unified diff per file —
// the reviewable form of `suggest-markers` before --write applies it.
// Insertions keep --write semantics: a suggestion for line N lands
// before that line; out-of-range lines are skipped.
// @spec emit/diff-markers
func DiffMarkers(fsys fs.FS, sugs []MarkerSuggestion) (string, error) {
	byFile := map[string][]MarkerSuggestion{}
	var order []string
	for _, s := range sugs {
		if _, seen := byFile[s.File]; !seen {
			order = append(order, s.File)
		}
		byFile[s.File] = append(byFile[s.File], s)
	}
	var b strings.Builder
	for _, file := range order {
		data, err := fs.ReadFile(fsys, file)
		if err != nil {
			return "", err
		}
		lines := strings.Split(string(data), "\n")
		if n := len(lines); n > 0 && lines[n-1] == "" {
			lines = lines[:n-1] // trailing newline is not a line
		}
		// insertion positions in original-line coordinates
		inserts := byFile[file]
		sort.SliceStable(inserts, func(i, j int) bool { return inserts[i].Line < inserts[j].Line })
		type ins struct {
			pos     int // insert before original line pos (0-based)
			comment string
		}
		var insAt []ins
		for _, s := range inserts {
			idx := s.Line - 1
			if idx < 0 || idx > len(lines) {
				continue
			}
			insAt = append(insAt, ins{idx, s.Comment})
		}
		if len(insAt) == 0 {
			continue
		}
		// group insertions into hunks: two inserts share a hunk when
		// their context windows overlap or touch
		type hunk struct {
			start, end int   // original-line range, exclusive end
			inserts    []ins // sorted by pos
		}
		var hunks []hunk
		for _, in := range insAt {
			if n := len(hunks); n > 0 && in.pos <= hunks[n-1].end+diffContext {
				hunks[n-1].inserts = append(hunks[n-1].inserts, in)
				if e := in.pos + diffContext; e > hunks[n-1].end {
					hunks[n-1].end = e
				}
				if hunks[n-1].end > len(lines) {
					hunks[n-1].end = len(lines)
				}
				continue
			}
			h := hunk{start: in.pos - diffContext, end: in.pos + diffContext}
			if h.start < 0 {
				h.start = 0
			}
			if h.end > len(lines) {
				h.end = len(lines)
			}
			h.inserts = []ins{in}
			hunks = append(hunks, h)
		}
		fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", file, file)
		insertedSoFar := 0
		for _, h := range hunks {
			origCount := h.end - h.start
			newCount := origCount + len(h.inserts)
			origFrom := h.start + 1
			newFrom := h.start + 1 + insertedSoFar
			if origCount == 0 {
				origFrom = 0 // empty file
			}
			fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n", origFrom, origCount, newFrom, newCount)
			next := 0
			for i := h.start; i <= h.end; i++ {
				for next < len(h.inserts) && h.inserts[next].pos == i {
					b.WriteString("+" + h.inserts[next].comment + "\n")
					next++
				}
				if i < h.end {
					b.WriteString(" " + lines[i] + "\n")
				}
			}
			insertedSoFar += len(h.inserts)
		}
	}
	return b.String(), nil
}
