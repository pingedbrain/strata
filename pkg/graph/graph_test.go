package graph

import (
	"testing"

	"github.com/pingedbrain/strata/pkg/markers"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

func TestJoinClassifiesEdges(t *testing.T) {
	g := reqgraph.New()
	r := &reqgraph.Requirement{ID: "A-1", Title: "x",
		Scenarios: []reqgraph.Scenario{{Name: "s", Steps: []string{"step"}}}}
	_ = g.Add(r)
	_ = g.Add(&reqgraph.Requirement{ID: "A-2", Title: "orphan req"})

	ms := []markers.Marker{
		{Kind: markers.Spec, ReqID: "A-1", Hash: r.AcceptanceHash(), File: "a.go", Line: 1},
		{Kind: markers.Spec, ReqID: "GHOST", File: "b.go", Line: 2},      // dangling
		{Kind: markers.Spec, ReqID: "A-1", Hash: "ffffff", File: "c.go"}, // stale
	}
	res := Join(g, ms)

	if len(res.Dangling) != 1 || res.Dangling[0].ReqID != "GHOST" {
		t.Fatalf("dangling: %+v", res.Dangling)
	}
	stale := 0
	for _, e := range res.Edges {
		if e.Stale {
			stale++
		}
	}
	if stale != 1 {
		t.Fatalf("want 1 stale edge, got %d", stale)
	}
	cov, tot := res.Coverage()
	if cov != 1 || tot != 2 {
		t.Fatalf("coverage = %d/%d, want 1/2", cov, tot)
	}
	if len(res.Uncovered) != 1 || res.Uncovered[0].ID != "A-2" {
		t.Fatalf("uncovered: %+v", res.Uncovered)
	}
}

func TestLookups(t *testing.T) {
	g := reqgraph.New()
	_ = g.Add(&reqgraph.Requirement{ID: "A-1"})

	ms := []markers.Marker{
		{Kind: markers.Spec, ReqID: "A-1", File: "cmd/x/main.go", Line: 1},
		{Kind: markers.Spec, ReqID: "A-1", File: "lib.go", Line: 2},
		{Kind: markers.Spec, ReqID: "NOPE", File: "cmd/x/main.go", Line: 9},
	}
	res := Join(g, ms)

	if got := res.EdgesFor("A-1"); len(got) != 2 {
		t.Fatalf("EdgesFor: %+v", got)
	}
	if got := res.EdgesIn("main.go"); len(got) != 1 {
		t.Fatalf("EdgesIn basename match: %+v", got)
	}
	if got := res.EdgesIn("cmd/x/main.go"); len(got) != 1 {
		t.Fatalf("EdgesIn path match: %+v", got)
	}
	if got := res.DanglingIn("main.go"); len(got) != 1 || got[0].ReqID != "NOPE" {
		t.Fatalf("DanglingIn: %+v", got)
	}
	if got := res.EdgesIn("nope.go"); len(got) != 0 {
		t.Fatalf("EdgesIn should not match: %+v", got)
	}
}
