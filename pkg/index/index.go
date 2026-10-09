// Package index provides the code side of the traceability map.
// v0.1 is file-level: which files exist and which contain markers.
// Symbol-level indexing lands via SCIP consumers, then tree-sitter/WASM.
package index

import "io/fs"

// Symbol is a named code entity with an exact location.
type Symbol struct {
	Name string
	Kind string // "func", "type", "method", ...
	File string
	Line int
}

// Index maps files to their symbols. File-level MVP populates Files
// with empty symbol slices; SCIP fills them in later.
type Index struct {
	Files   map[string][]Symbol
	Markers []string // files that contain at least one marker
}

// Build walks root and records every non-vendored source file.
// Marker detection happens in pkg/markers during the join — the index
// only tracks the file universe so coverage denominators are honest.
func Build(root fs.FS, isSource func(path string) bool) (*Index, error) {
	idx := &Index{Files: map[string][]Symbol{}}
	err := fs.WalkDir(root, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if isSource(path) {
			idx.Files[path] = nil
		}
		return nil
	})
	return idx, err
}
