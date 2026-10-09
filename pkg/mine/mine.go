// Package mine excavates candidate requirements from a repo's existing
// code: exported symbols become requirement seeds, tests become scenario
// seeds. Deterministic — no LLM. Output feeds pkg/emit to draft specs.
// Symbol extraction lives in pkg/index; mine adds capability grouping
// and test→symbol linkage.
package mine

import (
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/pingedbrain/strata/pkg/index"
)

// Candidate is a symbol that plausibly backs a requirement.
type Candidate struct {
	Name     string         // symbol name, becomes the requirement title seed
	Kind     string         // "func", "type", "method", ...
	Doc      string         // doc comment/docstring, seeds the description
	Evidence index.Symbol   // where it lives
	Tests    []index.Symbol // tests that plausibly cover it (name-matched)
}

// Capability is a group of candidates, named after the directory.
type Capability struct {
	Name       string
	Candidates []Candidate
}

// Scan walks fsys, extracts symbols, groups candidates by directory
// capability, and links tests to candidates by normalized name.
func Scan(fsys fs.FS) ([]Capability, error) {
	idx, err := index.Scan(fsys, skipNoise)
	if err != nil {
		return nil, err
	}
	byCap := map[string][]Candidate{}
	for _, syms := range idx.Files {
		for _, s := range syms {
			if !s.Exported {
				continue // public surface = requirement seeds
			}
			cap := capabilityOf(s.File)
			byCap[cap] = append(byCap[cap], Candidate{Name: s.Name, Kind: s.Kind, Doc: s.Doc, Evidence: s})
		}
	}
	// link tests to candidates globally (a tests/ dir tests other dirs)
	for cap, cands := range byCap {
		for i := range cands {
			for _, t := range idx.Tests {
				if covers(t.Name, cands[i].Name) {
					cands[i].Tests = append(cands[i].Tests, t)
				}
			}
		}
		byCap[cap] = cands
	}

	names := make([]string, 0, len(byCap))
	for n := range byCap {
		names = append(names, n)
	}
	sort.Strings(names)
	out := make([]Capability, 0, len(names))
	for _, n := range names {
		out = append(out, Capability{Name: n, Candidates: byCap[n]})
	}
	return out, nil
}

func skipNoise(p string, isDir bool) bool {
	if !isDir {
		return false
	}
	base := path.Base(p)
	return base != "." && (strings.HasPrefix(base, ".") || base == "vendor" || base == "node_modules")
}

// capabilityOf names a capability after its directory, stripping common
// layout prefixes: pkg/auth → auth, cmd/tool → tool, . → root.
func capabilityOf(p string) string {
	dir := path.Dir(p)
	parts := strings.Split(dir, "/")
	last := parts[len(parts)-1]
	switch last {
	case ".", "pkg", "cmd", "src", "internal", "lib":
		return "root"
	default:
		return last
	}
}

// covers reports whether a test name plausibly exercises sym.
// Alias kept at the call sites via index.Covers.
func covers(testName, sym string) bool { return index.Covers(testName, sym) }
