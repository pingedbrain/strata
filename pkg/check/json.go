package check

import (
	"encoding/json"

	"github.com/pingedbrain/strata/pkg/markers"
)

// JSON renders the full gate report as a stable, machine-readable
// document — the companion to SARIF for tooling that wants the graph,
// not just findings. Schema (version 1):
//
//	{tool, version, formats,
//	 requirements: [{id, title, source, hash, linked, status?}],
//	 edges: [{req_id, kind, file, line, hash, symbol, stale}],
//	 dangling: [{req_id, kind, file, line, hash}],
//	 unlinked: [{name, kind, file, line}],
//	 coverage: {linked, total, fraction},
//	 verified: [req_id...], failing: [req_id...],
//	 gate: {pass, failures}}
//
// @spec check/report-json
func (r *Report) JSON() ([]byte, error) {
	type reqJSON struct {
		ID     string `json:"id"`
		Title  string `json:"title"`
		Source string `json:"source"`
		Hash   string `json:"hash"`
		Linked bool   `json:"linked"`
		Status string `json:"status,omitempty"`
	}
	type markerJSON struct {
		ReqID string `json:"req_id"`
		Kind  string `json:"kind"`
		File  string `json:"file"`
		Line  int    `json:"line"`
		Hash  string `json:"hash,omitempty"`
	}
	type edgeJSON struct {
		markerJSON
		Symbol string `json:"symbol,omitempty"`
		Stale  bool   `json:"stale"`
	}
	type symJSON struct {
		Name string `json:"name"`
		Kind string `json:"kind"`
		File string `json:"file"`
		Line int    `json:"line"`
	}

	mkMarker := func(m markers.Marker) markerJSON {
		return markerJSON{ReqID: m.ReqID, Kind: m.Kind.String(),
			File: m.File, Line: m.Line, Hash: m.Hash}
	}

	linked := map[string]bool{}
	for _, e := range r.Result.Edges {
		if e.Kind != markers.Verifies {
			linked[e.ReqID] = true
		}
	}
	status := map[string]string{}
	for _, id := range r.Result.Verified {
		status[id] = "verified"
	}
	for _, id := range r.Result.Failing {
		status[id] = "failing"
	}

	var doc struct {
		Tool     string       `json:"tool"`
		Version  int          `json:"version"`
		Formats  []string     `json:"formats"`
		Reqs     []reqJSON    `json:"requirements"`
		Edges    []edgeJSON   `json:"edges"`
		Dangling []markerJSON `json:"dangling"`
		Unlinked []symJSON    `json:"unlinked"`
		Coverage struct {
			Linked   int     `json:"linked"`
			Total    int     `json:"total"`
			Fraction float64 `json:"fraction"`
		} `json:"coverage"`
		Verified []string `json:"verified"`
		Failing  []string `json:"failing"`
		Gate     struct {
			Pass     bool     `json:"pass"`
			Failures []string `json:"failures"`
		} `json:"gate"`
	}
	doc.Tool = "spec-blame"
	doc.Version = 1
	doc.Formats = r.Formats
	doc.Reqs = []reqJSON{}
	for _, id := range r.Graph.IDs() {
		req := r.Graph.Requirements[id]
		doc.Reqs = append(doc.Reqs, reqJSON{
			ID: id, Title: req.Title, Source: req.Source,
			Hash: req.AcceptanceHash(), Linked: linked[id],
			Status: status[id],
		})
	}
	doc.Edges = []edgeJSON{}
	for _, e := range r.Result.Edges {
		doc.Edges = append(doc.Edges, edgeJSON{
			markerJSON: mkMarker(e.Marker), Symbol: e.Symbol, Stale: e.Stale,
		})
	}
	doc.Dangling = []markerJSON{}
	for _, m := range r.Result.Dangling {
		doc.Dangling = append(doc.Dangling, mkMarker(m))
	}
	doc.Unlinked = []symJSON{}
	for _, s := range r.Result.Unlinked {
		doc.Unlinked = append(doc.Unlinked, symJSON{
			Name: s.Name, Kind: s.Kind, File: s.File, Line: s.Line,
		})
	}
	cov, tot := r.Result.Coverage()
	doc.Coverage.Linked = cov
	doc.Coverage.Total = tot
	if tot > 0 {
		doc.Coverage.Fraction = float64(cov) / float64(tot)
	}
	doc.Verified = r.Result.Verified
	doc.Failing = r.Result.Failing
	if doc.Verified == nil {
		doc.Verified = []string{}
	}
	if doc.Failing == nil {
		doc.Failing = []string{}
	}
	doc.Gate.Failures = r.GateFailures()
	doc.Gate.Pass = len(doc.Gate.Failures) == 0
	if doc.Gate.Failures == nil {
		doc.Gate.Failures = []string{}
	}
	return json.MarshalIndent(doc, "", "  ")
}
