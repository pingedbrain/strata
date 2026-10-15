// Package ingest adapts existing spec formats to the canonical reqgraph.
// One file per format: openspec, spec-kit, gherkin, md-ids, gh-issues.
package ingest

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// Adapter reads one spec format into the canonical graph.
// @spec ingest/adapter
type Adapter interface {
	// Name is the format identifier, e.g. "openspec".
	Name() string
	// Detect reports whether dir contains specs in this format.
	Detect(dir fs.FS) bool
	// Ingest parses all specs under dir into the graph.
	Ingest(dir fs.FS) (*reqgraph.Graph, error)
}

var adapters []Adapter

// specSearchSkip prunes directories that can never host specs during
// adapter discovery walks: dependency trees and hidden/VCS dirs. This
// is the spec side's twin of check.codeOnly — a monorepo's vendored or
// tooling dirs must not leak requirements into the graph. Repo-specific
// exclusions come from .strataignore via check's filteredFS instead.
func specSearchSkip(p string, isDir bool) bool {
	if !isDir {
		return false
	}
	base := path.Base(p)
	return base == "vendor" || base == "node_modules" ||
		(strings.HasPrefix(base, ".") && base != ".")
}

// specTree reports whether p sits inside an openspec/ or specs/ tree at
// any depth — the openspec and spec-kit adapters own those files.
func specTree(p string) bool {
	return strings.HasPrefix(p, "openspec/") || strings.HasPrefix(p, "specs/") ||
		strings.Contains(p, "/openspec/") || strings.Contains(p, "/specs/")
}

// Register adds an adapter; called from each adapter file's init.
// @spec ingest/register
func Register(a Adapter) { adapters = append(adapters, a) }

// DetectAll returns every adapter whose format is present under dir.
// @spec ingest/detect-all
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
// @spec ingest/load
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
