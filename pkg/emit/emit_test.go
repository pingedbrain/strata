package emit

import (
	"strings"
	"testing"

	"github.com/pingedbrain/strata/pkg/index"
	"github.com/pingedbrain/strata/pkg/mine"
)

func TestSpecRendersOpenSpecShape(t *testing.T) {
	cap := mine.Capability{
		Name: "auth",
		Candidates: []mine.Candidate{{
			Name: "ValidateToken", Kind: "func",
			Evidence: index.Symbol{Name: "ValidateToken", File: "auth.go", Line: 10},
			Tests:    []index.Symbol{{Name: "TestValidateTokenRejectsExpired"}},
		}},
	}
	out := Spec(cap)
	for _, want := range []string{
		"# Auth Specification",
		"### Requirement: Validate Token",
		"_Evidence: auth.go:10_",
		"#### Scenario: Validate Token Rejects Expired",
		"- **WHEN** TODO",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("spec missing %q:\n%s", want, out)
		}
	}
}

func TestSuggestMarkers(t *testing.T) {
	caps := []mine.Capability{{
		Name: "auth",
		Candidates: []mine.Candidate{
			{Name: "Validate", Evidence: index.Symbol{File: "a.go", Line: 5}},
			{Name: "forward_request", Evidence: index.Symbol{File: "f.py", Line: 3}},
		},
	}}
	ms := SuggestMarkers(caps)
	if len(ms) != 2 {
		t.Fatalf("want 2 suggestions, got %+v", ms)
	}
	if ms[0].Comment != "// @spec auth/validate" {
		t.Fatalf("go comment wrong: %q", ms[0].Comment)
	}
	if ms[1].Comment != "# @spec auth/forward-request" {
		t.Fatalf("py comment wrong: %q", ms[1].Comment)
	}
}

func TestSameNameCollisionGetsFileStem(t *testing.T) {
	caps := []mine.Capability{{
		Name: "effects",
		Candidates: []mine.Candidate{
			{Name: "apply", Evidence: index.Symbol{File: "effects/latency.py", Line: 10}},
			{Name: "apply", Evidence: index.Symbol{File: "effects/reset.py", Line: 9}},
		},
	}}
	ms := SuggestMarkers(caps)
	if ms[0].Comment != "# @spec effects/apply" {
		t.Fatalf("first occurrence keeps plain id: %q", ms[0].Comment)
	}
	if ms[1].Comment != "# @spec effects/reset-apply" {
		t.Fatalf("collision not disambiguated: %q", ms[1].Comment)
	}
	spec := Spec(caps[0])
	if !strings.Contains(spec, "### Requirement: Reset Apply") {
		t.Fatalf("spec title not disambiguated:\n%s", spec)
	}
}
