package check

import (
	"encoding/json"
	"fmt"
)

// SARIF renders the gate failures as SARIF 2.1.0 — feed it to GitHub
// code scanning via `upload-sarif` and findings land on the PR diff.
// @spec check/report-sarif
func (r *Report) SARIF() ([]byte, error) {
	type artifactLoc struct {
		URI string `json:"uri"`
	}
	type region struct {
		StartLine int `json:"startLine"`
	}
	type location struct {
		PhysicalLocation struct {
			ArtifactLocation artifactLoc `json:"artifactLocation"`
			Region           region      `json:"region"`
		} `json:"physicalLocation"`
	}
	type result struct {
		RuleID    string                `json:"ruleId"`
		Level     string                `json:"level"`
		Message   struct{ Text string } `json:"message"`
		Locations []location            `json:"locations,omitempty"`
	}
	type rule struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		HelpURI string `json:"helpUri"`
	}
	type run struct {
		Tool struct {
			Driver struct {
				Name    string `json:"name"`
				Version string `json:"version"`
				Rules   []rule `json:"rules"`
			} `json:"driver"`
		} `json:"tool"`
		Results []result `json:"results"`
	}
	type sarif struct {
		Schema  string `json:"$schema"`
		Version string `json:"version"`
		Runs    []run  `json:"runs"`
	}

	var results []result
	add := func(ruleID, level, file string, line int, msg string) {
		var res result
		res.RuleID = ruleID
		res.Level = level
		res.Message.Text = msg
		var loc location
		loc.PhysicalLocation.ArtifactLocation.URI = file
		loc.PhysicalLocation.Region.StartLine = line
		res.Locations = []location{loc}
		results = append(results, res)
	}
	for _, m := range r.Result.Dangling {
		add("dangling-marker", "error", m.File, m.Line,
			fmt.Sprintf("marker references %s, which does not exist in any loaded spec", m.ReqID))
	}
	for _, e := range r.Result.Edges {
		if e.Stale {
			add("stale-marker", "warning", e.File, e.Line,
				fmt.Sprintf("spec %s changed since this marker's bound hash was written (run spec-blame sync after review)", e.ReqID))
		}
	}
	for _, s := range r.Result.Unlinked {
		add("unlinked-symbol", "note", s.File, s.Line,
			fmt.Sprintf("exported %s %q has no requirement marker", s.Kind, s.Name))
	}
	if r.MinCov > 0 {
		cov, tot := r.Result.Coverage()
		if tot > 0 && float64(cov)/float64(tot) < r.MinCov {
			var res result
			res.RuleID = "low-coverage"
			res.Level = "error"
			res.Message.Text = fmt.Sprintf("coverage %d/%d below minimum %.0f%%", cov, tot, r.MinCov*100)
			results = append(results, res)
		}
	}

	var doc sarif
	doc.Schema = "https://json.schemastore.org/sarif-2.1.0.json"
	doc.Version = "2.1.0"
	rn := run{}
	rn.Tool.Driver.Name = "spec-blame"
	rn.Tool.Driver.Version = "0.2.0"
	rn.Tool.Driver.Rules = []rule{
		{ID: "dangling-marker", Name: "DanglingMarker", HelpURI: "https://github.com/pingedbrain/strata#markers"},
		{ID: "stale-marker", Name: "StaleMarker", HelpURI: "https://github.com/pingedbrain/strata#markers"},
		{ID: "unlinked-symbol", Name: "UnlinkedSymbol", HelpURI: "https://github.com/pingedbrain/strata"},
		{ID: "low-coverage", Name: "LowCoverage", HelpURI: "https://github.com/pingedbrain/strata"},
	}
	rn.Results = results
	doc.Runs = []run{rn}
	return json.MarshalIndent(doc, "", "  ")
}
