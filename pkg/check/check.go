// Package check is the CI gate: load specs, scan markers, join, and
// decide whether the run is objectively broken.
package check

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/pingedbrain/strata/pkg/graph"
	"github.com/pingedbrain/strata/pkg/ingest"
	"github.com/pingedbrain/strata/pkg/markers"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

// Options tunes the gate.
type Options struct {
	// MinCoverage, when > 0, fails the gate below this fraction.
	// Default 0 = coverage is advisory only.
	MinCoverage float64
}

// Report is the full gate result.
type Report struct {
	Formats []string
	Graph   *reqgraph.Graph
	Result  *graph.Result
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
	return &Report{Formats: formats, Graph: g, Result: graph.Join(g, ms), MinCov: opts.MinCoverage}, nil
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

// codeOnly excludes spec dirs, VCS metadata, and dependency dirs from
// marker scanning — specs are not code and markers inside them would
// self-link.
func codeOnly(p string, isDir bool) bool {
	if isDir {
		base := path.Base(p)
		return base == "openspec" || base == "vendor" || base == "node_modules" ||
			(strings.HasPrefix(base, ".") && base != ".")
	}
	return false
}
