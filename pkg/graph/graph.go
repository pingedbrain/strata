// Package graph joins the requirement graph with scanned markers into a
// traceability result: links, dangling references, stale bindings, and
// coverage. The join is deterministic — inference may suggest links, but
// only explicit markers decide them.
package graph

import (
	"strings"

	"github.com/pingedbrain/strata/pkg/markers"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// Edge is a resolved link between a marker and a requirement.
type Edge struct {
	markers.Marker
	Stale bool // bound hash no longer matches the requirement's acceptance text
}

// Result is the joined traceability picture.
type Result struct {
	Edges     []Edge                  // resolved marker→requirement links
	Dangling  []markers.Marker        // markers pointing at unknown requirement IDs
	Uncovered []*reqgraph.Requirement // requirements with no implementation link
}

// Join resolves every marker against the requirement graph.
// Impl-links are Spec/Implements kinds; Verifies counts separately.
func Join(g *reqgraph.Graph, ms []markers.Marker) *Result {
	res := &Result{}
	implLinked := map[string]bool{}

	for _, m := range ms {
		r, ok := g.Requirements[m.ReqID]
		if !ok {
			res.Dangling = append(res.Dangling, m)
			continue
		}
		e := Edge{Marker: m, Stale: m.Hash != "" && m.Hash != r.AcceptanceHash()}
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

// fileMatch accepts exact paths or basename suffixes ("main.go" matches
// "cmd/sub/main.go").
func fileMatch(got, want string) bool {
	return got == want || strings.HasSuffix(got, "/"+want)
}

// EdgesIn returns resolved links whose marker sits in file.
func (r *Result) EdgesIn(file string) []Edge {
	var out []Edge
	for _, e := range r.Edges {
		if fileMatch(e.File, file) {
			out = append(out, e)
		}
	}
	return out
}

// DanglingIn returns unresolved markers sitting in file.
func (r *Result) DanglingIn(file string) []markers.Marker {
	var out []markers.Marker
	for _, m := range r.Dangling {
		if fileMatch(m.File, file) {
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
