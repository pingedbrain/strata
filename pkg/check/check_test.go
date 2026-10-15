package check

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/pingedbrain/strata/pkg/emit"
	"github.com/pingedbrain/strata/pkg/index"
	"github.com/pingedbrain/strata/pkg/mine"
)

var fixture = fstest.MapFS{
	"openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# Auth

## Requirements

### Requirement: Token expiry
Tokens expire.

#### Scenario: Expired rejected
- **WHEN** token is old
- **THEN** reject it
`)},
	"auth/handler.go": &fstest.MapFile{Data: []byte(`package auth

// @spec auth/token-expiry
func Validate() {}
`)},
	"auth/stale.go": &fstest.MapFile{Data: []byte(`package auth

// @spec auth/token-expiry #000000
func Old() {}
`)},
	"auth/ghost.go": &fstest.MapFile{Data: []byte(`package auth

// @spec auth/does-not-exist
func Ghost() {}
`)},
	"openspec/changes/whatever/notes.md": &fstest.MapFile{Data: []byte(
		"// @spec auth/token-expiry — spec dirs must not be scanned as code\n")},
}

func TestRunGate(t *testing.T) {
	rep, err := Run(fixture, Options{})
	if err != nil {
		t.Fatal(err)
	}
	fails := rep.GateFailures()
	if len(fails) != 2 {
		t.Fatalf("want 2 gate failures (stale + dangling), got %v", fails)
	}
	cov, tot := rep.Result.Coverage()
	if cov != 1 || tot != 1 {
		t.Fatalf("coverage %d/%d, want 1/1", cov, tot)
	}
}

func TestRunNoSpecs(t *testing.T) {
	_, err := Run(fstest.MapFS{"main.go": {}}, Options{})
	if err == nil {
		t.Fatal("repo without specs must error")
	}
}
func TestMultiRootSpecs(t *testing.T) {
	// spec trees anywhere under --root are discovered and merged;
	// .strataignore still prunes whole subtrees from discovery
	fsys := fstest.MapFS{
		".strataignore": &fstest.MapFile{Data: []byte("legacy/\n")},
		"a/openspec/specs/x/spec.md": &fstest.MapFile{Data: []byte(`# X
### Requirement: Alpha
a.
`)},
		"b/openspec/specs/y/spec.md": &fstest.MapFile{Data: []byte(`# Y
### Requirement: Beta
b.
`)},
		"legacy/openspec/specs/z/spec.md": &fstest.MapFile{Data: []byte(`# Z
### Requirement: Nope
ignored.
`)},
		"a/impl.go": &fstest.MapFile{Data: []byte(`package a

// @spec x/alpha
func A() {}
`)},
		"b/impl.go": &fstest.MapFile{Data: []byte(`package b

// @spec y/beta
func B() {}
`)},
	}
	rep, err := Run(fsys, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if rep.Graph.Requirements["x/alpha"] == nil || rep.Graph.Requirements["y/beta"] == nil {
		t.Fatalf("multi-root requirements missing: %v", rep.Graph.IDs())
	}
	if len(rep.Graph.IDs()) != 2 {
		t.Fatalf("ignored subtree leaked into graph: %v", rep.Graph.IDs())
	}
	if fails := rep.GateFailures(); len(fails) != 0 {
		t.Fatalf("gate should pass: %v", fails)
	}
	if cov, tot := rep.Result.Coverage(); cov != 2 || tot != 2 {
		t.Fatalf("coverage %d/%d, want 2/2", cov, tot)
	}
}

func TestSuggestedPinsMatchAcceptanceHash(t *testing.T) {
	// end-to-end: excavate pins a marker with the hash of the
	// requirement propose emits; after both are applied, check must
	// see a fresh (non-stale) edge — same normalization as graph.Join
	cap := mine.Capability{
		Name: "auth",
		Candidates: []mine.Candidate{{
			Name:     "Validate",
			Kind:     "func",
			Evidence: index.Symbol{Name: "Validate", File: "auth/validate.go", Line: 3},
			Tests:    []index.Symbol{{Name: "TestValidateRejectsExpired"}},
		}},
	}
	sugs := emit.SuggestMarkers([]mine.Capability{cap})
	if len(sugs) != 1 {
		t.Fatalf("want 1 suggestion, got %+v", sugs)
	}
	if !strings.Contains(sugs[0].Comment, " #") {
		t.Fatalf("suggestion not hash-pinned: %q", sugs[0].Comment)
	}
	fsys := fstest.MapFS{
		"openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(emit.Spec(cap, nil))},
		"auth/validate.go": &fstest.MapFile{Data: []byte(
			"package auth\n\n" + sugs[0].Comment + "\nfunc Validate() {}\n")},
	}
	rep, err := Run(fsys, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Result.Edges) != 1 {
		t.Fatalf("want 1 edge, got %+v", rep.Result.Edges)
	}
	if rep.Result.Edges[0].Stale {
		t.Fatalf("pinned marker must not be stale: %+v", rep.Result.Edges[0])
	}
	if fails := rep.GateFailures(); len(fails) != 0 {
		t.Fatalf("gate should pass: %v", fails)
	}
}

func TestFeaturesDirIsSourceNotSpec(t *testing.T) {
	// app/features is a common Python/TS source layout; only the
	// .feature extension marks a Cucumber spec. Markers under a
	// dir named "features" must be scanned and gated.
	fs := fstest.MapFS{
		"openspec/specs/auth/spec.md": &fstest.MapFile{Data: []byte(`# Auth

## Requirements

### Requirement: Token expiry
Tokens expire.
`)},
		"app/features/auth/router.py": &fstest.MapFile{Data: []byte(`# @spec auth/token-expiry
def validate(): ...
`)},
		"app/features/auth/ghost.py": &fstest.MapFile{Data: []byte(`# @spec auth/does-not-exist
def ghost(): ...
`)},
		"features/login.feature": &fstest.MapFile{Data: []byte(`Feature: Login
# @spec auth/self-link-must-not-scan
`)},
	}
	rep, err := Run(fs, Options{})
	if err != nil {
		t.Fatal(err)
	}
	fails := rep.GateFailures()
	if len(fails) != 1 || fails[0] == "" {
		t.Fatalf("want exactly 1 dangling failure (ghost.py), got %v", fails)
	}
	if cov, tot := rep.Result.Coverage(); cov != 1 || tot != 1 {
		t.Fatalf("coverage %d/%d, want 1/1", cov, tot)
	}
}
