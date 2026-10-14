package check

import (
	"testing"
	"testing/fstest"
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
