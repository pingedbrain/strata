// Package graph joins the requirement graph with scanned markers into a
// traceability result: links, dangling references, stale bindings, and
// coverage. The join is deterministic — inference may suggest links, but
// only explicit markers decide them. When a symbol index is provided,
// markers resolve to the symbol declared directly below them.
package graph

import (
	"github.com/pingedbrain/strata/pkg/index"
	"github.com/pingedbrain/strata/pkg/markers"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// Edge is a resolved link between a marker and a requirement.
type Edge struct {
	markers.Marker
	Symbol string // symbol the marker annotates, "" if file-level
	Stale  bool   // bound hash no longer matches the requirement's acceptance text
}

// Result is the joined traceability picture.
type Result struct {
	Edges     []Edge                  // resolved marker→requirement links
	Dangling  []markers.Marker        // markers pointing at unknown requirement IDs
	Uncovered []*reqgraph.Requirement // requirements with no implementation link
	Unlinked  []index.Symbol          // exported symbols with no marker
	Verified  []string                // req IDs with a passing covering test (set by ApplyTestResults)
	Failing   []string                // req IDs with a failing covering test
}

// Join resolves every marker against the requirement graph. idx may be
// nil (file-level mode); when set, each marker binds to the symbol
// declared directly below it and unlinked symbols surface in Result.
// Impl-links are Spec/Implements kinds; Verifies counts separately.
func Join(g *reqgraph.Graph, ms []markers.Marker, idx *index.Index) *Result {
	res := &Result{}
	implLinked := map[string]bool{}
	linkedSyms := map[index.Symbol]bool{}

	for _, m := range ms {
		r, ok := g.Requirements[m.ReqID]
		if !ok {
			res.Dangling = append(res.Dangling, m)
			// a dangling marker still marks a symbol — it's broken
			// (reported above), not unlinked
			if idx != nil {
				if sym, ok := idx.SymbolBelow(m.File, m.Line); ok {
					linkedSyms[sym] = true
				}
			}
			continue
		}
		e := Edge{Marker: m, Stale: m.Hash != "" && m.Hash != r.AcceptanceHash()}
		if idx != nil {
			if sym, ok := idx.SymbolBelow(m.File, m.Line); ok {
				e.Symbol = sym.Name
				linkedSyms[sym] = true
			}
		}
		res.Edges = append(res.Edges, e)
		if m.Kind != markers.Verifies {
			implLinked[m.ReqID] = true
		}
	}

	for _, id := range g.IDs() {
		if !implLinked[id] {
			res.Uncovered = append(res.Uncovered, g.Requirements[id])
		}
	}
	if idx != nil {
		for _, syms := range idx.Files {
			for _, s := range syms {
				// only exported symbols are dead-code candidates —
				// private helpers don't need requirements
				if s.Exported && !linkedSyms[s] {
					res.Unlinked = append(res.Unlinked, s)
				}
			}
		}
	}
	return res
}

// Coverage returns implemented / total requirements.
func (r *Result) Coverage() (covered, total int) {
	linked := map[string]bool{}
	for _, e := range r.Edges {
		if e.Kind != markers.Verifies {
			linked[e.ReqID] = true
		}
	}
	return len(linked), len(linked) + len(r.Uncovered)
}

// EdgesIn returns resolved links whose marker sits in file.
func (r *Result) EdgesIn(file string) []Edge {
	var out []Edge
	for _, e := range r.Edges {
		if index.FileMatch(e.File, file) {
			out = append(out, e)
		}
	}
	return out
}

// DanglingIn returns unresolved markers sitting in file.
func (r *Result) DanglingIn(file string) []markers.Marker {
	var out []markers.Marker
	for _, m := range r.Dangling {
		if index.FileMatch(m.File, file) {
			out = append(out, m)
		}
	}
	return out
}

// EdgesFor returns resolved links pointing at requirement id.
func (r *Result) EdgesFor(id string) []Edge {
	var out []Edge
	for _, e := range r.Edges {
		if e.ReqID == id {
			out = append(out, e)
		}
	}
	return out
}
