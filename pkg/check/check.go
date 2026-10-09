// Package check is the CI gate: load specs, scan markers, join, and
// decide whether the run is objectively broken.
package check

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/pingedbrain/strata/pkg/graph"
	"github.com/pingedbrain/strata/pkg/index"
	"github.com/pingedbrain/strata/pkg/ingest"
	"github.com/pingedbrain/strata/pkg/junit"
	"github.com/pingedbrain/strata/pkg/markers"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// Options tunes the gate.
type Options struct {
	// MinCoverage, when > 0, fails the gate below this fraction.
	// Default 0 = coverage is advisory only.
	MinCoverage float64
	// JUnit lists test-result XML files (relative to the scanned root)
	// to merge into Verified/Failing requirement status.
	JUnit []string
	// SCIP is an optional path to an index.scip file (relative to the
	// scanned root). Empty means auto-detect "index.scip" at the root;
	// richer indexer data overlays the native symbol extraction.
	SCIP string
}

// Report is the full gate result.
type Report struct {
	Formats []string
	Graph   *reqgraph.Graph
	Result  *graph.Result
	Index   *index.Index
	MinCov  float64
}

// Run executes the gate over fsys (a repo root).
func Run(fsys fs.FS, opts Options) (*Report, error) {
	g, err := ingest.Load(fsys)
	if err != nil {
		return nil, err
	}
	var formats []string
	for _, a := range ingest.DetectAll(fsys) {
		formats = append(formats, a.Name())
	}
	ms, err := markers.ScanDir(fsys, codeOnly)
	if err != nil {
		return nil, err
	}
	idx, err := index.Scan(fsys, codeOnly)
	if err != nil {
		return nil, err
	}
	scipPath := opts.SCIP
	if scipPath == "" {
		if _, err := fs.Stat(fsys, "index.scip"); err == nil {
			scipPath = "index.scip"
		}
	}
	if scipPath != "" {
		data, err := fs.ReadFile(fsys, scipPath)
		if err != nil {
			return nil, fmt.Errorf("scip: %w", err)
		}
		si, err := index.LoadSCIP(data)
		if err != nil {
			return nil, fmt.Errorf("scip %s: %w", scipPath, err)
		}
		idx.Overlay(si)
	}
	ms = append(ms, bddMarkers(g, idx)...)
	res := graph.Join(g, ms, idx)
	for _, jp := range opts.JUnit {
		data, err := fs.ReadFile(fsys, jp)
		if err != nil {
			return nil, fmt.Errorf("junit: %w", err)
		}
		cases, err := junit.Parse(data)
		if err != nil {
			return nil, fmt.Errorf("junit %s: %w", jp, err)
		}
		res.ApplyTestResults(g, idx, cases)
	}
	return &Report{
		Formats: formats, Graph: g, Index: idx,
		Result: res, MinCov: opts.MinCoverage,
	}, nil
}

// GateFailures lists the objectively-broken findings. Empty = pass.
func (r *Report) GateFailures() []string {
	var out []string
	for _, m := range r.Result.Dangling {
		out = append(out, fmt.Sprintf("dangling: %s:%d → %s (no such requirement)", m.File, m.Line, m.ReqID))
	}
	for _, e := range r.Result.Edges {
		if e.Stale {
			out = append(out, fmt.Sprintf("stale: %s:%d → %s (spec changed since annotation)", e.File, e.Line, e.ReqID))
		}
	}
	if r.MinCov > 0 {
		cov, tot := r.Result.Coverage()
		if tot > 0 && float64(cov)/float64(tot) < r.MinCov {
			out = append(out, fmt.Sprintf("coverage: %d/%d requirements linked (< %.0f%% minimum)",
				cov, tot, r.MinCov*100))
		}
	}
	return out
}

// StaleFixes converts stale edges into marker rewrites for `sync`.
func (r *Report) StaleFixes() []markers.Fix {
	var out []markers.Fix
	for _, e := range r.Result.Edges {
		if e.Stale {
			r2 := r.Graph.Requirements[e.ReqID]
			out = append(out, markers.Fix{
				File: e.File, Line: e.Line, ReqID: e.ReqID,
				NewHash: r2.AcceptanceHash(),
			})
		}
	}
	return out
}

// bddMarkers synthesizes `verifies` markers from BDD step definitions:
// a pytest-bdd @given/@when/@then or cucumber Given() whose text matches
// a scenario step counts as a verification link — no manual marker
// needed. A scenarios("x.feature") call links every requirement defined
// in that feature file.
func bddMarkers(g *reqgraph.Graph, idx *index.Index) []markers.Marker {
	seen := map[[3]string]bool{}
	var out []markers.Marker
	add := func(id string, sd index.Symbol) {
		k := [3]string{id, sd.File, fmt.Sprint(sd.Line)}
		if !seen[k] {
			seen[k] = true
			out = append(out, markers.Marker{
				Kind: markers.Verifies, ReqID: id, File: sd.File, Line: sd.Line,
			})
		}
	}
	for _, sd := range idx.StepDefs {
		if sd.Kind == "scenarios-ref" {
			for _, id := range g.IDs() {
				src := g.Requirements[id].Source
				if i := strings.IndexByte(src, ':'); i >= 0 {
					src = src[:i]
				}
				name := strings.TrimLeft(sd.Name, "./") // "../features/x" → "features/x"
				if index.FileMatch(src, name) {
					add(id, sd)
				}
			}
			continue
		}
		for _, id := range g.IDs() {
			for _, sc := range g.Requirements[id].Scenarios {
				for _, step := range sc.Steps {
					if index.BDDStepMatch(sd.Name, step) {
						add(id, sd)
						break
					}
				}
			}
		}
	}
	return out
}

// codeOnly excludes spec dirs (openspec/spec-kit/features), VCS
// metadata, and dependency dirs from marker and symbol scanning —
// specs are not code and markers inside them would self-link.
func codeOnly(p string, isDir bool) bool {
	if isDir {
		base := path.Base(p)
		return base == "openspec" || base == "specs" || base == "features" ||
			base == "vendor" || base == "node_modules" ||
			(strings.HasPrefix(base, ".") && base != ".")
	}
	return false
}
