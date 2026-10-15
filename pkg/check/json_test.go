package check

import (
	"encoding/json"
	"testing"
)

// The JSON report is a stable machine-readable contract — pin the shape.
func TestJSONReport(t *testing.T) {
	rep, err := Run(fixture, Options{})
	if err != nil {
		t.Fatal(err)
	}
	b, err := rep.JSON()
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Tool    string   `json:"tool"`
		Version int      `json:"version"`
		Formats []string `json:"formats"`
		Reqs    []struct {
			ID     string `json:"id"`
			Title  string `json:"title"`
			Source string `json:"source"`
			Hash   string `json:"hash"`
			Linked bool   `json:"linked"`
		} `json:"requirements"`
		Edges []struct {
			ReqID  string `json:"req_id"`
			Kind   string `json:"kind"`
			File   string `json:"file"`
			Line   int    `json:"line"`
			Hash   string `json:"hash"`
			Symbol string `json:"symbol"`
			Stale  bool   `json:"stale"`
		} `json:"edges"`
		Dangling []struct {
			ReqID string `json:"req_id"`
			File  string `json:"file"`
			Line  int    `json:"line"`
		} `json:"dangling"`
		Coverage struct {
			Linked   int     `json:"linked"`
			Total    int     `json:"total"`
			Fraction float64 `json:"fraction"`
		} `json:"coverage"`
		Gate struct {
			Pass     bool     `json:"pass"`
			Failures []string `json:"failures"`
		} `json:"gate"`
	}
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, b)
	}
	if doc.Tool != "spec-blame" || doc.Version != 1 {
		t.Fatalf("envelope wrong: %+v", doc)
	}
	if len(doc.Formats) != 1 || doc.Formats[0] != "openspec" {
		t.Fatalf("formats: %v", doc.Formats)
	}
	if len(doc.Reqs) != 1 || doc.Reqs[0].ID != "auth/token-expiry" || !doc.Reqs[0].Linked {
		t.Fatalf("requirements: %+v", doc.Reqs)
	}
	if doc.Reqs[0].Hash == "" {
		t.Fatal("requirement hash should be the acceptance hash")
	}
	// fixture: handler.go clean link + stale.go stale link, ghost.go dangling
	if len(doc.Edges) != 2 {
		t.Fatalf("want 2 edges, got %+v", doc.Edges)
	}
	var stale, clean bool
	for _, e := range doc.Edges {
		if e.ReqID != "auth/token-expiry" || e.Kind != "spec" {
			t.Fatalf("edge fields wrong: %+v", e)
		}
		if e.Stale {
			stale = true
			if e.File != "auth/stale.go" || e.Hash != "000000" {
				t.Fatalf("stale edge wrong: %+v", e)
			}
		} else {
			clean = true
			if e.File != "auth/handler.go" || e.Symbol != "Validate" {
				t.Fatalf("clean edge wrong: %+v", e)
			}
		}
	}
	if !stale || !clean {
		t.Fatalf("want one stale + one clean edge: %+v", doc.Edges)
	}
	if len(doc.Dangling) != 1 || doc.Dangling[0].ReqID != "auth/does-not-exist" {
		t.Fatalf("dangling: %+v", doc.Dangling)
	}
	if doc.Coverage.Linked != 1 || doc.Coverage.Total != 1 || doc.Coverage.Fraction != 1 {
		t.Fatalf("coverage: %+v", doc.Coverage)
	}
	if doc.Gate.Pass || len(doc.Gate.Failures) != 2 {
		t.Fatalf("gate should fail on stale+dangling: %+v", doc.Gate)
	}
}
