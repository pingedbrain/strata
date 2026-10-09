// Package index builds the code side of the traceability map: exported
// symbols and tests per source file, extracted per language. This is the
// seam where SCIP consumers land later — the Index contract is already
// symbol-shaped so blame/check don't care who produced it.
package index

import (
	"io/fs"
	"strings"
)

// Symbol is a named code entity with an exact location.
type Symbol struct {
	Name     string
	Kind     string // "func", "type", "method", "test"
	File     string
	Line     int
	Exported bool // public surface; unexported symbols bind markers but aren't requirement candidates
}

// Index maps files to their exported symbols, plus test symbols.
type Index struct {
	Files map[string][]Symbol
	Tests []Symbol
}

// extractor pulls exported symbols and test names from one language.
type extractor interface {
	match(path string) bool
	isTestFile(path string) bool
	parse(data []byte, filename string) (exports, tests []Symbol)
}

var extractors = []extractor{goExtractor{}, pyExtractor{}, tsExtractor{}}

// Scan walks fsys and extracts symbols for every recognized source file
// not skipped. skip(path, isDir)=true prunes dirs and ignores files;
// nil scans everything.
func Scan(fsys fs.FS, skip func(path string, isDir bool) bool) (*Index, error) {
	idx := &Index{Files: map[string][]Symbol{}}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if skip != nil && skip(p, true) {
				return fs.SkipDir
			}
			return nil
		}
		if skip != nil && skip(p, false) {
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
			exports, tests := ex.parse(data, p)
			if ex.isTestFile(p) {
				idx.Tests = append(idx.Tests, tests...)
			} else if len(exports) > 0 {
				idx.Files[p] = exports
			}
			break // first matching extractor owns the file
		}
		return nil
	})
	return idx, err
}

// SymbolBelow returns the first declared symbol after line — the symbol a
// comment marker annotates.
func (i *Index) SymbolBelow(file string, line int) (Symbol, bool) {
	var best Symbol
	found := false
	for _, s := range i.Files[file] {
		if s.Line > line && (!found || s.Line < best.Line) {
			best, found = s, true
		}
	}
	return best, found
}

// FileMatch accepts exact paths or basename suffixes ("main.go" matches
// "cmd/sub/main.go").
func FileMatch(got, want string) bool {
	return got == want || (want != "" && len(got) > len(want) &&
		got[len(got)-len(want)-1] == '/' && hasSuffix(got, want))
}

func hasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}

// Covers reports whether a test name plausibly exercises a symbol:
// normalized containment, case/underscore/space-insensitive.
// "TestValidateToken" covers "ValidateToken" and "validate_token".
func Covers(testName, sym string) bool {
	n := func(s string) string {
		s = strings.ToLower(s)
		s = strings.TrimPrefix(s, "test")
		r := strings.NewReplacer("_", "", " ", "", "-", "")
		return r.Replace(s)
	}
	tn, sn := n(testName), n(sym)
	return sn != "" && strings.Contains(tn, sn)
}

// FilesMatching returns index keys matching a path or basename.
func (i *Index) FilesMatching(want string) []string {
	var out []string
	for f := range i.Files {
		if FileMatch(f, want) {
			out = append(out, f)
		}
	}
	return out
}
