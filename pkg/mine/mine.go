// Package mine excavates candidate requirements from a repo's existing
// code: exported symbols become requirement seeds, tests become scenario
// seeds. Deterministic — no LLM. Output feeds pkg/emit to draft specs.
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
	Evidence index.Symbol   // where it lives
	Tests    []index.Symbol // tests that plausibly cover it (name-matched)
}

// Capability is a group of candidates, named after the directory.
type Capability struct {
	Name       string
	Candidates []Candidate
}

// extractor pulls exported symbols and test names from one language.
type extractor interface {
	match(path string) bool
	isTestFile(path string) bool
	parse(data []byte, filename string) (exports, tests []index.Symbol)
}

var extractors = []extractor{goExtractor{}, pyExtractor{}, tsExtractor{}}

// Scan walks fsys, extracts symbols, groups candidates by directory
// capability, and links tests to candidates by normalized name.
func Scan(fsys fs.FS) ([]Capability, error) {
	byCap := map[string][]Candidate{}
	var tests []index.Symbol

	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := path.Base(p)
			if strings.HasPrefix(base, ".") && base != "." || base == "vendor" || base == "node_modules" {
				return fs.SkipDir
			}
			return nil
		}
		for _, ex := range extractors {
			if !ex.match(p) {
				continue
			}
			data, err := fs.ReadFile(fsys, p)
			if err != nil {
				return err
			}
			exports, ts := ex.parse(data, p)
			cap := capabilityOf(p)
			if ex.isTestFile(p) {
				tests = append(tests, ts...)
			}
			for _, s := range exports {
				byCap[cap] = append(byCap[cap], Candidate{
					Name: s.Name, Kind: s.Kind, Evidence: s,
				})
			}
			break // first matching extractor owns the file
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// link tests to candidates globally (a tests/ dir tests other dirs)
	for cap, cands := range byCap {
		for i := range cands {
			for _, t := range tests {
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

// covers reports whether a test name plausibly exercises sym:
// normalized containment, case/underscore/space-insensitive.
func covers(testName, sym string) bool {
	n := func(s string) string {
		s = strings.ToLower(s)
		s = strings.TrimPrefix(s, "test")
		r := strings.NewReplacer("_", "", " ", "", "-", "")
		return r.Replace(s)
	}
	tn, sn := n(testName), n(sym)
	return sn != "" && strings.Contains(tn, sn)
}
