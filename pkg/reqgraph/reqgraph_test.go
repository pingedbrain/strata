package reqgraph

import "testing"

func TestAcceptanceHashStableAcrossWhitespace(t *testing.T) {
	r1 := &Requirement{ID: "A-1", Title: "tokens expire",
		Scenarios: []Scenario{{Name: "expired", Steps: []string{"issued at T", "rejected at T+24h"}}}}
	r2 := &Requirement{ID: "A-1", Title: "tokens   expire",
		Scenarios: []Scenario{{Name: "expired", Steps: []string{"issued  at   T", "rejected at T+24h"}}}}
	if r1.AcceptanceHash() != r2.AcceptanceHash() {
		t.Fatal("whitespace-only edits must not change the acceptance hash")
	}
	r3 := &Requirement{ID: "A-1", Title: "tokens expire",
		Scenarios: []Scenario{{Name: "expired", Steps: []string{"issued at T", "rejected at T+1h"}}}}
	if r1.AcceptanceHash() == r3.AcceptanceHash() {
		t.Fatal("a semantic edit must change the acceptance hash")
	}
}

func TestAddRejectsDuplicates(t *testing.T) {
	g := New()
	if err := g.Add(&Requirement{ID: "A-1"}); err != nil {
		t.Fatal(err)
	}
	if err := g.Add(&Requirement{ID: "A-1"}); err == nil {
		t.Fatal("duplicate ID must error")
	}
}
