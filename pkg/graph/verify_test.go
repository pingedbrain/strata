// SPDX-License-Identifier: Apache-2.0
package graph

import (
	"testing"
	"testing/fstest"

	"github.com/pingedbrain/strata/pkg/index"
	"github.com/pingedbrain/strata/pkg/junit"
	"github.com/pingedbrain/strata/pkg/markers"
	"github.com/pingedbrain/strata/pkg/reqgraph"
)

func TestApplyTestResults(t *testing.T) {
	fsys := fstest.MapFS{
		"a.go": &fstest.MapFile{Data: []byte(`package x

func Validate() {}
func Reset() {}
`)},
	}
	idx, err := index.Scan(fsys, nil)
	if err != nil {
		t.Fatal(err)
	}
	g := reqgraph.New()
	_ = g.Add(&reqgraph.Requirement{ID: "R-1"})
	_ = g.Add(&reqgraph.Requirement{ID: "R-2"})

	res := Join(g, []markers.Marker{
		{Kind: markers.Spec, ReqID: "R-1", File: "a.go", Line: 2}, // → Validate
		{Kind: markers.Spec, ReqID: "R-2", File: "a.go", Line: 3}, // → Reset
	}, idx)

	cases := []junit.Case{
		{Name: "TestValidate"},            // passes → R-1 verified
		{Name: "TestReset", Failed: true}, // fails  → R-2 failing
		{Name: "TestValidateExtra"},       // extra passing cover for Validate
		{Name: "TestUnrelated"},
	}
	res.ApplyTestResults(idx, cases)

	if len(res.Verified) != 1 || res.Verified[0] != "R-1" {
		t.Fatalf("verified: %+v", res.Verified)
	}
	if len(res.Failing) != 1 || res.Failing[0] != "R-2" {
		t.Fatalf("failing: %+v", res.Failing)
	}

	// failing trumps passing on the same requirement
	res.ApplyTestResults(idx, []junit.Case{
		{Name: "TestValidate", Failed: true},
		{Name: "TestValidate2"},
	})
	if len(res.Failing) != 1 || res.Failing[0] != "R-1" || len(res.Verified) != 0 {
		t.Fatalf("failing should trump: v=%+v f=%+v", res.Verified, res.Failing)
	}
}
