// Package ingest adapts existing spec formats to the canonical reqgraph.
// One file per format: openspec, spec-kit, gherkin, md-ids, gh-issues.
package ingest

import (
	"fmt"
	"io/fs"

	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// Adapter reads one spec format into the canonical graph.
type Adapter interface {
	// Name is the format identifier, e.g. "openspec".
	Name() string
	// Detect reports whether dir contains specs in this format.
	Detect(dir fs.FS) bool
	// Ingest parses all specs under dir into the graph.
	Ingest(dir fs.FS) (*reqgraph.Graph, error)
}

var adapters []Adapter

// Register adds an adapter; called from each adapter file's init.
func Register(a Adapter) { adapters = append(adapters, a) }

// DetectAll returns every adapter whose format is present under dir.
func DetectAll(dir fs.FS) []Adapter {
	var found []Adapter
	for _, a := range adapters {
		if a.Detect(dir) {
			found = append(found, a)
		}
	}
	return found
}

// Load runs every detected adapter and merges the graphs. Duplicate
// requirement IDs across formats surface as errors via Graph.Add.
func Load(dir fs.FS) (*reqgraph.Graph, error) {
	found := DetectAll(dir)
	if len(found) == 0 {
		return nil, fmt.Errorf("no spec format detected")
	}
	merged := reqgraph.New()
	for _, a := range found {
		g, err := a.Ingest(dir)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", a.Name(), err)
		}
		for _, id := range g.IDs() {
			if err := merged.Add(g.Requirements[id]); err != nil {
				return nil, err
			}
		}
	}
	return merged, nil
}
