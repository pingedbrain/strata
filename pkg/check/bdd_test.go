// SPDX-License-Identifier: Apache-2.0
package check

import (
	"testing"
	"testing/fstest"

	"github.com/pingedbrain/strata/pkg/graph"
	"github.com/pingedbrain/strata/pkg/index"
	"github.com/pingedbrain/strata/pkg/markers"
)

func TestImpact(t *testing.T) {
	// lib.go defines Parse (bound to R-1); app.go references Parse and is
	// bound to R-2 — refactoring lib.go must flag both.
	idx := &index.Index{
		Files: map[string][]index.Symbol{
			"lib.go": {{Name: "Parse", File: "lib.go", Line: 3}},
			"app.go": {{Name: "Run", File: "app.go", Line: 2}},
		},
		Refs: map[string][]index.Symbol{
			"app.go": {{Name: "Parse", Kind: "ref", File: "app.go", Line: 9}},
		},
	}
	rep := &Report{
		Index: idx,
		Result: &graph.Result{Edges: []graph.Edge{
			{Marker: markers.Marker{ReqID: "R-1", File: "lib.go", Line: 2}, Symbol: "Parse"},
			{Marker: markers.Marker{ReqID: "R-2", File: "app.go", Line: 1}, Symbol: "Run"},
			{Marker: markers.Marker{ReqID: "R-3", File: "other.go", Line: 1}, Symbol: "Unrelated"},
		}},
	}
	got := rep.Impact("lib.go")
	if len(got) != 2 || got[0] != "R-1" || got[1] != "R-2" {
		t.Fatalf("impact(lib.go) = %v, want [R-1 R-2]", got)
	}
	if got := rep.Impact("other.go"); len(got) != 1 || got[0] != "R-3" {
		t.Fatalf("impact(other.go) = %v", got)
	}
}

func TestBDDStepDefSynthesis(t *testing.T) {
	fsys := fstest.MapFS{
		"features/login.feature": &fstest.MapFile{Data: []byte(`Feature: Login

  Scenario: Valid login
    Given valid credentials
    When the login request is posted
    Then a session token is returned
`)},
		"tests/test_login.py": &fstest.MapFile{Data: []byte(`from pytest_bdd import given, when, then, scenarios

scenarios("features/login.feature")

@given("valid credentials")
def creds(): ...

@when("the login request is posted")
def post(): ...

@then("a session token is returned")
def check(): ...
`)},
	}
	rep, err := Run(fsys, Options{})
	if err != nil {
		t.Fatal(err)
	}
	verifies := 0
	for _, e := range rep.Result.Edges {
		if e.Kind == markers.Verifies {
			verifies++
			if e.ReqID != "login/valid-login" {
				t.Fatalf("verifies edge to wrong req: %+v", e)
			}
		}
	}
	// 3 step-defs + 1 scenarios() ref, all bound to the same requirement
	if verifies != 4 {
		t.Fatalf("want 4 synthesized verifies edges, got %d: %+v", verifies, rep.Result.Edges)
	}
	// no dangling, no gate failures — implicit links are clean
	if len(rep.GateFailures()) != 0 {
		t.Fatalf("gate: %+v", rep.GateFailures())
	}
}

func TestBDDParamMatch(t *testing.T) {
	// pytest-bdd parser templates must match concrete scenario steps
	fsys := fstest.MapFS{
		"features/cukes.feature": &fstest.MapFile{Data: []byte(`Feature: Cukes

  Scenario: Eating
    Given there are 4 cucumbers
    When I eat 2
`)},
		"tests/test_cukes.py": &fstest.MapFile{Data: []byte(`from pytest_bdd import given, parsers

@given(parsers.parse("there are {n} cucumbers"))
def cukes(n): ...
`)},
	}
	rep, err := Run(fsys, Options{})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range rep.Result.Edges {
		if e.Kind == markers.Verifies && e.ReqID == "cukes/eating" {
			found = true
		}
	}
	if !found {
		t.Fatal("parametrized step-def should verify cukes/eating")
	}
}
